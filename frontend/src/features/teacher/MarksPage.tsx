import { useQuery } from "@tanstack/react-query";
import { academicsApi } from "../../api/academics";
import { Card, PageHeader, Table } from "../../components/ui";

export function MarksPage() {
  const { data: assessments } = useQuery({ queryKey: ["assessments"], queryFn: () => academicsApi.assessments() });
  const { data: marks } = useQuery({ queryKey: ["marks"], queryFn: () => academicsApi.marks() });
  return (
    <div>
      <PageHeader title="Marks entry / CBC gradebook" />
      <Card className="mb-4">
        <p className="text-sm text-slate-600">{(assessments ?? []).length} assessment(s) this term. Scores feed Academic Reports and parent progress.</p>
      </Card>
      <Table headers={["Learner", "Assessment", "Score", "Competency"]}>
        {(marks ?? []).map((m: any) => (
          <tr key={m.id}>
            <td className="px-4 py-3">{m.student_name}</td>
            <td className="px-4 py-3">{m.assessment_title}</td>
            <td className="px-4 py-3">{m.score}</td>
            <td className="px-4 py-3">{m.competency_level}</td>
          </tr>
        ))}
      </Table>
    </div>
  );
}
