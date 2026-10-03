import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { api } from "../../api/client";
import { schoolApi } from "../../api/school";
import { studentsApi } from "../../api/students";
import { Button, Card, Field, Input, PageHeader, Select, Stat, Table, useConfirm } from "../../components/ui";

export function LibraryManagementPage() {
  const qc = useQueryClient();
  const { confirm, dialog } = useConfirm();
  const [tab, setTab] = useState<"books" | "loans" | "overdue">("books");
  const [search, setSearch] = useState("");

  const { data: books } = useQuery({ queryKey: ["library-books"], queryFn: schoolApi.library });
  const { data: loans } = useQuery({ queryKey: ["library-loans"], queryFn: () => schoolApi.libraryLoans() });
  const { data: overdue } = useQuery({ queryKey: ["library-overdue"], queryFn: schoolApi.overdueLoans });
  const { data: students } = useQuery({ queryKey: ["students", ""], queryFn: () => studentsApi.list() });

  // Checkout form
  const [checkoutForm, setCheckoutForm] = useState({ book: "", student: "", due_at: "", borrower_type: "student" });

  const EMPTY_BOOK = { title: "", author: "", isbn: "", publisher: "", shelf_location: "", total_copies: "1", price: "0" };
  const [bookForm, setBookForm] = useState(EMPTY_BOOK);
  const [editingBookId, setEditingBookId] = useState<string | null>(null);

  const addBook = useMutation({
    mutationFn: () => schoolApi.createBook({ ...bookForm, total_copies: Number(bookForm.total_copies), price: Number(bookForm.price) }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["library-books"] }); setBookForm(EMPTY_BOOK); },
    onError: (err: any) => window.alert(err?.response?.data?.message ?? "Could not add this book — check the fields above."),
  });
  const updateBook = useMutation({
    mutationFn: () => schoolApi.updateBook(editingBookId!, { ...bookForm, total_copies: Number(bookForm.total_copies), price: Number(bookForm.price) }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["library-books"] }); setBookForm(EMPTY_BOOK); setEditingBookId(null); },
    onError: (err: any) => window.alert(err?.response?.data?.message ?? "Could not save changes."),
  });

  function startEditBook(b: any) {
    setEditingBookId(b.id);
    setBookForm({
      title: b.title, author: b.author ?? "", isbn: b.isbn ?? "", publisher: b.publisher ?? "",
      shelf_location: b.shelf_location ?? "", total_copies: String(b.total_copies), price: String(b.price ?? 0),
    });
  }
  function cancelEditBook() {
    setEditingBookId(null);
    setBookForm(EMPTY_BOOK);
  }
  async function handleSaveBook() {
    const isEdit = Boolean(editingBookId);
    const ok = await confirm({
      title: isEdit ? "Save book changes" : "Add book",
      message: isEdit ? `Save changes to "${bookForm.title}"?` : `Add "${bookForm.title}" with ${bookForm.total_copies} cop${bookForm.total_copies === "1" ? "y" : "ies"}?`,
      confirmLabel: isEdit ? "Save changes" : "Add book",
    });
    if (ok) (isEdit ? updateBook : addBook).mutate();
  }

  const checkout = useMutation({
    mutationFn: () => api.post("/library/loans/", checkoutForm).then((r: any) => r.data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["library-loans"] });
      qc.invalidateQueries({ queryKey: ["library-books"] });
      setCheckoutForm({ book: "", student: "", due_at: "", borrower_type: "student" });
    },
    onError: (err: any) => {
      window.alert(err?.response?.data?.message ?? "Could not issue this book. Check the details and try again.");
    },
  });

  const returnBook = useMutation({
    mutationFn: (loanId: string) => schoolApi.returnBook(loanId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["library-loans"] });
      qc.invalidateQueries({ queryKey: ["library-overdue"] });
      qc.invalidateQueries({ queryKey: ["library-books"] });
    },
  });

  const renewLoan = useMutation({
    mutationFn: (loanId: string) => schoolApi.renewLoan(loanId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["library-loans"] });
      qc.invalidateQueries({ queryKey: ["library-overdue"] });
    },
  });

  async function handleRenew(loan: any) {
    const ok = await confirm({
      title: "Renew loan",
      message: `Extend the due date for "${loan.book_title}" by 7 days?`,
      confirmLabel: "Renew",
    });
    if (ok) renewLoan.mutate(loan.id);
  }

  async function handleCheckout() {
    const book = (books as any[] ?? []).find((b: any) => String(b.id) === checkoutForm.book);
    const student = (students as any[] ?? []).find((s: any) => String(s.id) === checkoutForm.student);
    const ok = await confirm({
      title: "Checkout book",
      message: `Issue "${book?.title}" to ${student?.full_name ?? "borrower"}, due ${checkoutForm.due_at}?`,
      confirmLabel: "Issue book",
    });
    if (ok) checkout.mutate();
  }

  async function handleReturn(loan: any) {
    const ok = await confirm({
      title: "Return book",
      message: `Mark "${loan.book_title}" as returned${loan.is_overdue ? " — fine will be calculated" : ""}?`,
      confirmLabel: "Confirm return",
      variant: loan.is_overdue ? "danger" : "primary",
    });
    if (ok) returnBook.mutate(loan.id);
  }

  const filtered = (books as any[] ?? []).filter((b: any) =>
    !search || b.title?.toLowerCase().includes(search.toLowerCase()) || b.author?.toLowerCase().includes(search.toLowerCase())
  );

  const activeLoans = (loans as any[] ?? []).filter((l: any) => !l.returned_at);
  const overdueCount = (overdue as any[] ?? []).length;

  return (
    <div className="space-y-5">
      {dialog}
      <PageHeader title="Library management" subtitle="Book catalogue, loans, returns, and overdue tracking." />

      {/* Stats */}
      <div className="grid gap-3 sm:grid-cols-4">
        <Stat label="Total titles" value={(books as any[] ?? []).length} />
        <Stat label="Active loans" value={activeLoans.length} />
        <Stat label="Overdue" value={overdueCount} />
        <Stat label="Available" value={(books as any[] ?? []).filter((b: any) => b.available_copies > 0).length} />
      </div>

      {/* Tabs */}
      <div className="flex rounded-xl bg-slate-100 p-1 w-full max-w-sm">
        {(["books", "loans", "overdue"] as const).map((t) => (
          <button key={t}
            className={`flex-1 rounded-lg py-1.5 text-sm font-medium transition capitalize ${tab === t ? "bg-white text-[#1b365d] shadow-sm" : "text-slate-500"}`}
            onClick={() => setTab(t)}>
            {t}{t === "overdue" && overdueCount > 0 ? ` (${overdueCount})` : ""}
          </button>
        ))}
      </div>

      {/* Books tab */}
      {tab === "books" && (
        <>
          <Input className="max-w-xs" placeholder="Search title or author…" value={search} onChange={(e) => setSearch(e.target.value)} />

          {/* Add/edit book form */}
          <Card>
            <div className="mb-3 flex items-center justify-between">
              <h3 className="font-semibold text-[#1b365d]">{editingBookId ? "Edit book" : "Add book"}</h3>
              {editingBookId && <Button variant="ghost" onClick={cancelEditBook}>Cancel edit</Button>}
            </div>
            <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
              <Field label="Title"><Input value={bookForm.title} onChange={(e) => setBookForm({ ...bookForm, title: e.target.value })} /></Field>
              <Field label="Author"><Input value={bookForm.author} onChange={(e) => setBookForm({ ...bookForm, author: e.target.value })} /></Field>
              <Field label="ISBN"><Input value={bookForm.isbn} onChange={(e) => setBookForm({ ...bookForm, isbn: e.target.value })} /></Field>
              <Field label="Publisher"><Input value={bookForm.publisher} onChange={(e) => setBookForm({ ...bookForm, publisher: e.target.value })} /></Field>
              <Field label="Shelf location"><Input value={bookForm.shelf_location} onChange={(e) => setBookForm({ ...bookForm, shelf_location: e.target.value })} placeholder="e.g. A-12" /></Field>
              <Field label="Total copies"><Input type="number" min="1" value={bookForm.total_copies} onChange={(e) => setBookForm({ ...bookForm, total_copies: e.target.value })} /></Field>
              <Field label="Price (KES)"><Input type="number" min="0" value={bookForm.price} onChange={(e) => setBookForm({ ...bookForm, price: e.target.value })} /></Field>
            </div>
            <Button className="mt-3" onClick={handleSaveBook} disabled={addBook.isPending || updateBook.isPending || !bookForm.title}>
              {addBook.isPending || updateBook.isPending ? "Saving…" : editingBookId ? "Save changes" : "Add book"}
            </Button>
          </Card>

          {/* Checkout form */}
          <Card>
            <h3 className="mb-3 font-semibold text-[#1b365d]">Issue a book</h3>
            <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
              <Field label="Book">
                <Select value={checkoutForm.book} onChange={(e) => setCheckoutForm({ ...checkoutForm, book: e.target.value })}>
                  <option value="">Select book…</option>
                  {filtered.filter((b: any) => b.available_copies > 0).map((b: any) => (
                    <option key={b.id} value={b.id}>{b.title} ({b.available_copies} avail.)</option>
                  ))}
                </Select>
              </Field>
              <Field label="Borrower">
                <Select value={checkoutForm.student} onChange={(e) => setCheckoutForm({ ...checkoutForm, student: e.target.value })}>
                  <option value="">Select learner…</option>
                  {(students as any[] ?? []).map((s: any) => (
                    <option key={s.id} value={s.id}>{s.full_name} ({s.admission_no})</option>
                  ))}
                </Select>
              </Field>
              <Field label="Due date">
                <Input type="date" value={checkoutForm.due_at} onChange={(e) => setCheckoutForm({ ...checkoutForm, due_at: e.target.value })} />
              </Field>
              <div className="flex items-end">
                <Button onClick={handleCheckout} disabled={!checkoutForm.book || !checkoutForm.student || !checkoutForm.due_at || checkout.isPending}>
                  Issue book
                </Button>
              </div>
            </div>
          </Card>

          <Table headers={["Title", "Author", "Shelf", "Copies", "Available", "Status", ""]}>
            {filtered.map((b: any) => (
              <tr key={b.id} className="hover:bg-slate-50">
                <td className="px-4 py-3 font-medium">{b.title}</td>
                <td className="px-4 py-3 text-sm text-slate-600">{b.author || "—"}</td>
                <td className="px-4 py-3 text-xs">{b.shelf_location || "—"}</td>
                <td className="px-4 py-3">{b.total_copies}</td>
                <td className="px-4 py-3">
                  <span className={`font-medium ${b.available_copies === 0 ? "text-red-600" : "text-green-600"}`}>
                    {b.available_copies}
                  </span>
                </td>
                <td className="px-4 py-3">
                  <span className={`rounded-full px-2 py-0.5 text-xs ${b.available_copies > 0 ? "bg-green-100 text-green-700" : "bg-slate-100 text-slate-600"}`}>
                    {b.available_copies > 0 ? "available" : "all checked out"}
                  </span>
                </td>
                <td className="px-4 py-3">
                  <Button variant="ghost" onClick={() => startEditBook(b)}>Edit</Button>
                </td>
              </tr>
            ))}
            {filtered.length === 0 && (
              <tr><td colSpan={7} className="px-4 py-8 text-center text-slate-400">No books yet — add one above.</td></tr>
            )}
          </Table>
        </>
      )}

      {/* Active loans tab */}
      {tab === "loans" && (
        <Table headers={["Borrower", "Book", "Issued", "Due", "Days left", ""]}>
          {activeLoans.map((l: any) => {
            const daysLeft = Math.ceil((new Date(l.due_at).getTime() - Date.now()) / 86400000);
            return (
              <tr key={l.id} className="hover:bg-slate-50">
                <td className="px-4 py-3">{l.student_name || "—"}</td>
                <td className="px-4 py-3 font-medium">{l.book_title}</td>
                <td className="px-4 py-3 text-xs text-slate-500">{l.borrowed_at}</td>
                <td className="px-4 py-3 text-xs">{l.due_at}</td>
                <td className="px-4 py-3">
                  <span className={`font-medium ${daysLeft < 0 ? "text-red-600" : daysLeft <= 3 ? "text-orange-500" : "text-green-600"}`}>
                    {daysLeft < 0 ? `${Math.abs(daysLeft)}d overdue` : `${daysLeft}d`}
                  </span>
                </td>
                <td className="px-4 py-3 flex gap-2">
                  <Button variant="ghost" onClick={() => handleRenew(l)} disabled={renewLoan.isPending} className="py-1 text-xs">
                    Renew
                  </Button>
                  <Button variant="ghost" onClick={() => handleReturn(l)} disabled={returnBook.isPending} className="py-1 text-xs">
                    Return
                  </Button>
                </td>
              </tr>
            );
          })}
          {activeLoans.length === 0 && (
            <tr><td colSpan={6} className="px-4 py-8 text-center text-slate-400">No active loans.</td></tr>
          )}
        </Table>
      )}

      {/* Overdue tab */}
      {tab === "overdue" && (
        <Table headers={["Borrower", "Book", "Due", "Days overdue", "Fine (KES)", ""]}>
          {(overdue as any[] ?? []).map((l: any) => (
            <tr key={l.id} className="bg-red-50 hover:bg-red-100">
              <td className="px-4 py-3 font-medium">{l.student_name || "—"}</td>
              <td className="px-4 py-3">{l.book_title}</td>
              <td className="px-4 py-3 text-xs">{l.due_at}</td>
              <td className="px-4 py-3 text-red-700 font-medium">{l.days_overdue ?? "—"} days</td>
              <td className="px-4 py-3 text-red-700">{l.fine_amount ?? 0}</td>
              <td className="px-4 py-3 flex gap-2">
                <Button variant="ghost" onClick={() => handleRenew(l)} className="py-1 text-xs">
                  Renew
                </Button>
                <Button variant="ghost" onClick={() => handleReturn(l)} className="py-1 text-xs border-red-300 text-red-700 hover:bg-red-50">
                  Return
                </Button>
              </td>
            </tr>
          ))}
          {(overdue as any[] ?? []).length === 0 && (
            <tr><td colSpan={6} className="px-4 py-8 text-center text-green-600">No overdue loans.</td></tr>
          )}
        </Table>
      )}
    </div>
  );
}
