import { useQuery } from "@tanstack/react-query";
import { unreconciledApi } from "../../api/finance";
import { PageHeader } from "../../components/ui";

export function UnreconciledPage() {
  const { data } = useQuery({ queryKey: ["unrec"], queryFn: unreconciledApi.list });

  return (
    <div>
      <PageHeader
        title="Unreconciled M-Pesa"
        subtitle="Payments the fee listener couldn't confidently match to a learner — manual linking isn't wired up yet."
      />
      <p className="rounded-2xl border border-slate-200 bg-white px-4 py-8 text-center text-sm text-slate-500">
        {(data ?? []).length === 0
          ? "No unreconciled payments to show yet."
          : `${data!.length} payment(s) need review.`}
      </p>
    </div>
  );
}
