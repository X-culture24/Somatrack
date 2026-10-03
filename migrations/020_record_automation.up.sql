-- Book-keeping that must hold no matter which screen or API call changed the
-- data, so it lives in the database. Violations raise SQLSTATE 23514 with a
-- named constraint, which the resource engine turns into a form message.

-- ------------------------------------------------ Inventory stock movements
-- Movements carry a positive quantity; only adjustments may be signed.
ALTER TABLE inventory_transactions
    ADD CONSTRAINT inventory_txn_quantity_sign CHECK (transaction_type = 'adjustment' OR quantity > 0);

CREATE OR REPLACE FUNCTION inventory_txn_delta(txn_type TEXT, qty INTEGER) RETURNS INTEGER AS $$
    SELECT CASE WHEN txn_type IN ('purchase', 'return', 'adjustment') THEN qty ELSE -qty END;
$$ LANGUAGE sql IMMUTABLE;

CREATE OR REPLACE FUNCTION apply_inventory_txn() RETURNS trigger AS $$
DECLARE
    new_qty INTEGER;
BEGIN
    IF TG_OP IN ('UPDATE', 'DELETE') THEN
        UPDATE inventory_items
           SET quantity_in_stock = quantity_in_stock - inventory_txn_delta(OLD.transaction_type, OLD.quantity),
               updated_at = now()
         WHERE id = OLD.item_id;
    END IF;
    IF TG_OP IN ('INSERT', 'UPDATE') THEN
        UPDATE inventory_items
           SET quantity_in_stock = quantity_in_stock + inventory_txn_delta(NEW.transaction_type, NEW.quantity),
               updated_at = now()
         WHERE id = NEW.item_id
        RETURNING quantity_in_stock INTO new_qty;
        IF new_qty < 0 THEN
            RAISE EXCEPTION 'not enough stock (would be %)', new_qty
                USING ERRCODE = 'check_violation', CONSTRAINT = 'inventory_stock_nonnegative';
        END IF;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_inventory_txn ON inventory_transactions;
CREATE TRIGGER trg_inventory_txn AFTER INSERT OR UPDATE OR DELETE ON inventory_transactions
    FOR EACH ROW EXECUTE FUNCTION apply_inventory_txn();

-- ------------------------------------------------ Sales reduce stock
CREATE OR REPLACE FUNCTION apply_sale() RETURNS trigger AS $$
DECLARE
    left_qty INTEGER;
BEGIN
    IF TG_OP IN ('UPDATE', 'DELETE') THEN
        UPDATE sales_items SET stock_qty = stock_qty + OLD.quantity, updated_at = now() WHERE id = OLD.item_id;
    END IF;
    IF TG_OP IN ('INSERT', 'UPDATE') THEN
        UPDATE sales_items SET stock_qty = stock_qty - NEW.quantity, updated_at = now()
         WHERE id = NEW.item_id RETURNING stock_qty INTO left_qty;
        IF left_qty < 0 THEN
            RAISE EXCEPTION 'not enough stock (would be %)', left_qty
                USING ERRCODE = 'check_violation', CONSTRAINT = 'sales_stock_nonnegative';
        END IF;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_sale ON sales;
CREATE TRIGGER trg_sale AFTER INSERT OR UPDATE OR DELETE ON sales
    FOR EACH ROW EXECUTE FUNCTION apply_sale();

-- ------------------------------------------------ Bed status follows allocations
CREATE OR REPLACE FUNCTION apply_bed_allocation() RETURNS trigger AS $$
BEGIN
    IF TG_OP IN ('INSERT', 'UPDATE') AND NEW.status = 'active' THEN
        IF EXISTS (SELECT 1 FROM hostel_beds WHERE id = NEW.bed_id AND status = 'maintenance') THEN
            RAISE EXCEPTION 'bed is under maintenance'
                USING ERRCODE = 'check_violation', CONSTRAINT = 'bed_not_in_maintenance';
        END IF;
        UPDATE hostel_beds SET status = 'occupied' WHERE id = NEW.bed_id;
    END IF;
    -- A bed freed by ending, moving or deleting an allocation becomes available
    -- again unless another active allocation still holds it.
    IF TG_OP IN ('UPDATE', 'DELETE') THEN
        UPDATE hostel_beds b SET status = 'available'
         WHERE b.id = OLD.bed_id AND b.status = 'occupied'
           AND NOT EXISTS (SELECT 1 FROM bed_allocations a WHERE a.bed_id = OLD.bed_id AND a.status = 'active');
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_bed_allocation ON bed_allocations;
CREATE TRIGGER trg_bed_allocation AFTER INSERT OR UPDATE OR DELETE ON bed_allocations
    FOR EACH ROW EXECUTE FUNCTION apply_bed_allocation();

-- ------------------------------------------------ Pocket money balance
CREATE OR REPLACE VIEW pocket_money_balances AS
SELECT student_id,
       SUM(CASE WHEN txn_type = 'deposit' THEN amount ELSE -amount END) AS balance,
       MAX(txn_date) AS last_txn_date
FROM pocket_money_transactions
GROUP BY student_id;

CREATE OR REPLACE FUNCTION check_pocket_money() RETURNS trigger AS $$
DECLARE
    bal NUMERIC;
BEGIN
    SELECT COALESCE(SUM(CASE WHEN txn_type = 'deposit' THEN amount ELSE -amount END), 0)
      INTO bal FROM pocket_money_transactions WHERE student_id = NEW.student_id;
    IF bal < 0 THEN
        RAISE EXCEPTION 'withdrawal exceeds balance'
            USING ERRCODE = 'check_violation', CONSTRAINT = 'pocket_money_nonnegative';
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_pocket_money ON pocket_money_transactions;
CREATE CONSTRAINT TRIGGER trg_pocket_money AFTER INSERT OR UPDATE ON pocket_money_transactions
    FOR EACH ROW EXECUTE FUNCTION check_pocket_money();

-- ------------------------------------------------ Supplier invoice status
ALTER TABLE supplier_invoices ADD CONSTRAINT supplier_invoice_not_overpaid CHECK (amount_paid <= amount);

CREATE OR REPLACE FUNCTION set_supplier_invoice_status() RETURNS trigger AS $$
BEGIN
    NEW.status := CASE WHEN NEW.amount_paid >= NEW.amount AND NEW.amount > 0 THEN 'paid'
                       WHEN NEW.amount_paid > 0 THEN 'partial'
                       ELSE 'unpaid' END;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_supplier_invoice_status ON supplier_invoices;
CREATE TRIGGER trg_supplier_invoice_status BEFORE INSERT OR UPDATE ON supplier_invoices
    FOR EACH ROW EXECUTE FUNCTION set_supplier_invoice_status();
