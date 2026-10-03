//go:build integration

package integration

import (
	"context"
	"strings"
	"testing"

	"github.com/stmaryskabete/lms/internal/testutil"
)

func TestInventoryMovementsUpdateStock(t *testing.T) {
	e := newEnv(t)
	a := e.admin()
	item := id(t, want(t, a.post("/api/inventory/", map[string]interface{}{"name": "Chalk", "sku": "CH-1", "category": "stationery"}), 201))
	stock := func() float64 { return want(t, a.get("/api/inventory/"+item+"/"), 200).JSON(t)["quantity_in_stock"].(float64) }
	move := func(kind string, qty int) resp {
		return a.post("/api/inventory-transactions/", map[string]interface{}{
			"item_id": item, "transaction_type": kind, "quantity": qty, "transaction_date": "2026-02-02"})
	}

	want(t, move("purchase", 50), 201)
	issue := id(t, want(t, move("issue", 20), 201))
	if s := stock(); s != 30 {
		t.Fatalf("stock after +50 -20 = %v", s)
	}
	r := want(t, move("issue", 100), 400)
	if !strings.Contains(string(r.Body), "Not enough stock") {
		t.Errorf("overdraw message: %s", r.Body)
	}
	want(t, move("issue", -5), 400) // only adjustments may be negative
	want(t, move("adjustment", -2), 201)
	want(t, a.del("/api/inventory-transactions/"+issue+"/"), 204)
	if s := stock(); s != 48 { // 50 - 2, issue reversed
		t.Errorf("stock after reversal = %v", s)
	}
}

func TestSalesReduceStock(t *testing.T) {
	e := newEnv(t)
	a := e.admin()
	item := id(t, want(t, a.post("/api/sales-items/", map[string]interface{}{"name": "Exercise book", "unit_price": 50, "stock_qty": 10}), 201))
	want(t, a.post("/api/sales/", map[string]interface{}{"item_id": item, "quantity": 4, "unit_price": 50}), 201)
	if q := want(t, a.get("/api/sales-items/"+item+"/"), 200).JSON(t)["stock_qty"]; q != 6.0 {
		t.Errorf("stock after sale = %v", q)
	}
	r := want(t, a.post("/api/sales/", map[string]interface{}{"item_id": item, "quantity": 7, "unit_price": 50}), 400)
	if !strings.Contains(string(r.Body), "Not enough of this item") {
		t.Errorf("oversell message: %s", r.Body)
	}
}

func TestBedStatusFollowsAllocations(t *testing.T) {
	e := newEnv(t)
	a := e.admin()
	h := id(t, want(t, a.post("/api/hostels/", map[string]interface{}{"name": "Kilimanjaro"}), 201))
	room := id(t, want(t, a.post("/api/hostel-rooms/", map[string]interface{}{"hostel_id": h, "room_no": "1"}), 201))
	bed := id(t, want(t, a.post("/api/hostel-beds/", map[string]interface{}{"room_id": room, "bed_no": "1"}), 201))
	status := func() string { return want(t, a.get("/api/hostel-beds/"+bed+"/"), 200).JSON(t)["status"].(string) }

	alloc := id(t, want(t, a.post("/api/bed-allocations/", map[string]interface{}{"bed_id": bed, "student_id": faithID}), 201))
	if s := status(); s != "occupied" {
		t.Errorf("after allocation: %s", s)
	}
	want(t, a.post("/api/bed-allocations/", map[string]interface{}{"bed_id": bed, "student_id": brianID}), 409) // bed taken
	want(t, a.patch("/api/bed-allocations/"+alloc+"/", map[string]interface{}{"status": "ended"}), 200)
	if s := status(); s != "available" {
		t.Errorf("after ending: %s", s)
	}
	want(t, a.patch("/api/hostel-beds/"+bed+"/", map[string]interface{}{"status": "maintenance"}), 200)
	r := want(t, a.post("/api/bed-allocations/", map[string]interface{}{"bed_id": bed, "student_id": brianID}), 400)
	if !strings.Contains(string(r.Body), "under maintenance") {
		t.Errorf("maintenance message: %s", r.Body)
	}
}

func TestPocketMoneyBalance(t *testing.T) {
	e := newEnv(t)
	a := e.admin()
	want(t, a.post("/api/pocket-money/", map[string]interface{}{"student_id": brianID, "txn_type": "deposit", "amount": 500}), 201)
	want(t, a.post("/api/pocket-money/", map[string]interface{}{"student_id": brianID, "txn_type": "withdrawal", "amount": 120}), 201)
	r := want(t, a.post("/api/pocket-money/", map[string]interface{}{"student_id": brianID, "txn_type": "withdrawal", "amount": 1000}), 400)
	if !strings.Contains(string(r.Body), "more than the student's pocket money balance") {
		t.Errorf("overdraw message: %s", r.Body)
	}
	want(t, a.post("/api/pocket-money/", map[string]interface{}{"student_id": faithID, "txn_type": "deposit", "amount": 50}), 201)

	bals := want(t, e.parent().get("/api/pocket-money-balances/"), 200).JSON(t)["results"].([]interface{})
	if len(bals) != 1 || bals[0].(map[string]interface{})["balance"] != 380.0 {
		t.Errorf("parent balances (Brian only, 380): %v", bals)
	}
	if n := len(want(t, a.get("/api/pocket-money-balances/"), 200).JSON(t)["results"].([]interface{})); n != 2 {
		t.Errorf("admin sees %d balances, want 2", n)
	}
}

func TestSupplierInvoiceStatusIsComputed(t *testing.T) {
	e := newEnv(t)
	f := e.finance()
	sup := id(t, want(t, f.post("/api/suppliers/", map[string]interface{}{"name": "Nairobi Stationers"}), 201))
	inv := want(t, f.post("/api/supplier-invoices/", map[string]interface{}{
		"supplier_id": sup, "invoice_no": "NS-1", "invoice_date": "2026-02-01", "amount": 10000, "amount_paid": 4000,
		"status": "paid", // read-only: ignored
	}), 201).JSON(t)
	if inv["status"] != "partial" {
		t.Errorf("status = %v, want partial", inv["status"])
	}
	iid := inv["id"].(string)
	if s := want(t, f.patch("/api/supplier-invoices/"+iid+"/", map[string]interface{}{"amount_paid": 10000}), 200).JSON(t)["status"]; s != "paid" {
		t.Errorf("status = %v, want paid", s)
	}
	want(t, f.patch("/api/supplier-invoices/"+iid+"/", map[string]interface{}{"amount_paid": 12000}), 400)
}

func TestResultsCannotBeDeletedWithTheirTerm(t *testing.T) {
	e := newEnv(t)
	aid := id(t, want(t, e.westT().post("/api/assessments/", map[string]interface{}{
		"name": "CAT", "class_id": westID, "subject_id": mathID}), 201))
	want(t, e.westT().put("/api/assessments/"+aid+"/marks/", map[string]interface{}{
		"marks": []map[string]interface{}{{"student_id": brianID, "score": 50}}}), 200)
	for _, sql := range []string{
		`DELETE FROM terms WHERE id = $1`, `DELETE FROM class_groups WHERE id = $2`,
		`DELETE FROM subjects WHERE id = $3`, `DELETE FROM students WHERE id = $4`,
	} {
		if _, err := e.pool.Exec(context.Background(), sql, termID, westID, mathID, brianID); err == nil {
			t.Errorf("%s succeeded — results would be lost", sql)
		}
	}
	if n := testutil.Scalar[int](t, e.pool, `SELECT count(*) FROM marks`); n != 1 {
		t.Errorf("marks = %d", n)
	}
}
