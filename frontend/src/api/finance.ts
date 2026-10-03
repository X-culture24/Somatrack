import { api, downloadFile, unwrap } from "./client";

// IDs are UUID strings throughout the real backend.
export const invoicesApi = {
  list: (params?: Record<string, string>) =>
    api.get("/invoices/", { params }).then((r) => unwrap(r.data)),
  get: (id: string) => api.get(`/invoices/${id}/`).then((r) => r.data),
};

export type ManualPaymentInput = {
  student_id: string;
  amount: number;
  payment_method: "cash" | "cheque" | "bank_transfer" | "card" | "other";
  payment_date?: string;
  reference?: string;
  bank_name?: string;
  cheque_no?: string;
  payer_name?: string;
  payer_phone?: string;
  notes?: string;
  idempotency_key: string;
};

export const paymentsApi = {
  list: (params?: Record<string, string>) =>
    api.get("/payments/", { params }).then((r) => unwrap(r.data)),
  // Cash / cheque / bank slip at the bursar's desk. Posted, receipted and
  // confirmed to the family by SMS, like an M-Pesa payment.
  recordManual: (data: ManualPaymentInput) =>
    api.post<{ payment_id: string; receipt_no: string; duplicate: boolean }>("/finance/manual-payment/", data).then((r) => r.data),
  downloadReceipt: (id: string, receiptNo: string) => downloadFile(`/payments/${id}/receipt/`, `receipt-${receiptNo}.pdf`),
  // Fires a Safaricom-shaped payment through the real fee-listener pipeline
  // (match student by bill_ref_number, post payment, recompute invoice) —
  // the local equivalent of the M-Pesa C2B webhook, for demos/manual QA.
  simulate: (data: { amount: number; bill_ref_number: string; phone?: string; payer_name?: string }) =>
    api.post("/finance/simulate-payment/", data).then((r) => r.data),
  sendReminders: () => api.post("/finance/send-reminders/").then((r) => r.data),
};

export interface FeeStructure {
  id: string;
  academic_year_id: string;
  grade: number; // Playgroup=-2, PP1=-1, PP2=0, Grade N=N
  term_number: number;
  admission_fee: number; // one-off, Term 1 only, new intake only
  tuition_fee: number;
  boarding_fee: number;
  transport_fee: number;
  medical_fee: number;
  activity_fee: number;
  exam_fee: number;
  library_fee: number;
  computer_fee: number;
  uniform_fee: number;
  diary_book_fee: number; // one-off, Term 1 only, every student
  other_fee: number;
  notes: string;
  total_day: number;
  total_boarding: number;
  is_active: boolean;
}

export const GRADE_LABELS: Record<number, string> = {
  "-2": "Playgroup", "-1": "PP1", "0": "PP2",
  "1": "Grade 1", "2": "Grade 2", "3": "Grade 3", "4": "Grade 4", "5": "Grade 5",
  "6": "Grade 6", "7": "Grade 7", "8": "Grade 8", "9": "Grade 9",
};

export const feeStructuresApi = {
  list: (params?: Record<string, string>) =>
    api.get("/fee-structures/", { params }).then((r) => unwrap<FeeStructure>(r.data)),
  get: (id: string) => api.get(`/fee-structures/${id}/`).then((r) => r.data),
  create: (data: Partial<FeeStructure>) => api.post("/fee-structures/", data).then((r) => r.data),
  update: (id: string, data: Partial<FeeStructure>) =>
    api.put(`/fee-structures/${id}/`, data).then((r) => r.data),
  remove: (id: string) => api.delete(`/fee-structures/${id}/`).then((r) => r.data),
};

export interface FeeAnalytics {
  by_term: { term_id: string; label: string; billed: number; collected: number; outstanding: number }[];
  by_grade: { grade: number; billed: number; collected: number; outstanding: number }[];
  by_status: { status: string; count: number; total: number }[];
  payment_methods: { method: string; total: number }[];
}

export const feeAnalyticsApi = {
  get: () => api.get("/fee-analytics/").then((r) => r.data as FeeAnalytics),
};

export interface Discount {
  id: string;
  student_id: string;
  term_id: string;
  waiver_type: string;
  amount: number;
  percentage?: number;
  reason?: string;
  effective_date: string;
}

export const discountsApi = {
  list: (params?: Record<string, string>) =>
    api.get("/discounts/", { params }).then((r) => unwrap<Discount>(r.data)),
  create: (data: {
    student_id: string;
    term_id: string;
    waiver_type: string;
    amount?: number;
    percentage?: number;
    reason?: string;
    effective_date: string;
  }) => api.post("/discounts/", data).then((r) => r.data),
};

// /unreconciled-payments/ is still a backend placeholder stub — this page
// degrades gracefully (empty list) until that endpoint is built for real.
export const unreconciledApi = {
  list: () => api.get("/unreconciled-payments/").then((r) => (Array.isArray(r.data) ? r.data : [])),
};

export const billingApi = {
  // Creates/refreshes one invoice per active student for the given term,
  // from the matching fee_structure (by academic year + grade + term
  // number) plus any discount_waivers for that student+term.
  generate: (termId: string) =>
    api.post("/billing/generate/", { term_id: termId }).then((r) => r.data as {
      term_id: string;
      generated: number;
      skipped: { student_id: string; reason: string }[] | null;
    }),
};

export const feePaymentApi = {
  initiateMpesa: (data: { student_id: string; phone: string; amount: number }) =>
    api.post("/finance/pay/", data).then((r) => r.data as { id: string; status: string }),
  mpesaStatus: (id: string) =>
    api.get(`/finance/pay/${id}/status/`).then((r) => r.data),
  initiateEquity: (data: { student_id: string; phone: string; amount: number; wallet_name?: string }) =>
    api.post("/finance/pay/equity/", data).then((r) => r.data as { id: string; status: string; description?: string }),
  equityStatus: (id: string) =>
    api.get(`/finance/pay/equity/${id}/status/`).then((r) => r.data),
};
