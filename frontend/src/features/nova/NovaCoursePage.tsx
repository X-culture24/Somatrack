import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate, useParams } from "react-router-dom";
import { ClipboardList, FileText, Layers, Settings } from "lucide-react";
import { novaApi } from "../../api/nova";
import { useAuth } from "../../store/auth";
import { Button, Card, PageHeader, Table } from "../../components/ui";

export function NovaCoursePage() {
  const { id } = useParams<{ id: string }>();
  const user = useAuth((s) => s.user);
  const navigate = useNavigate();
  const qc = useQueryClient();

  const isTeacher = user?.portal === "teacher";
  const isStudent = user?.portal === "student";

  const { data: course } = useQuery({ queryKey: ["course", id], queryFn: () => novaApi.course(id!) });
  const { data: materials } = useQuery({ queryKey: ["mats", id], queryFn: () => novaApi.materials({ course: id! }) });
  const { data: assignments } = useQuery({ queryKey: ["asg", id], queryFn: () => novaApi.assignments({ course: id! }) });
  const { data: quizzes } = useQuery({ queryKey: ["quiz", id], queryFn: () => novaApi.quizzes({ course: id! }) });
  const { data: discussions } = useQuery({ queryKey: ["disc", id], queryFn: () => novaApi.discussions({ course: id! }) });

  const post = useMutation({
    mutationFn: (body: string) => novaApi.postDiscussion({ course: id, body }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["disc", id] }),
  });

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-4">
        <PageHeader title={course?.title ?? "Course"} subtitle={course?.description} />
        {isStudent && (
          <Button variant="ghost" onClick={() => navigate("/nova/my-work")}>
            My work →
          </Button>
        )}
      </div>

      {/* Teacher: manage course tools */}
      {isTeacher && (
        <Card>
          <div className="mb-3 flex items-center gap-2 text-[#1b365d]">
            <Settings size={16} />
            <h3 className="font-semibold">Manage course</h3>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button onClick={() => navigate(`/nova/teacher/${id}`)}>
              <Layers size={14} className="mr-1.5 inline" />
              Materials &amp; content
            </Button>
            <Button variant="ghost" onClick={() => navigate(`/nova/teacher/${id}/submissions`)}>
              <ClipboardList size={14} className="mr-1.5 inline" />
              Grade submissions
            </Button>
          </div>
        </Card>
      )}

      {/* Learning materials */}
      <section>
        <h3 className="mb-2 font-semibold text-[#1b365d]">Learning materials</h3>
        <Table headers={["Title", "Strand", "Notes"]}>
          {(materials ?? []).map((m: any) => (
            <tr key={m.id}>
              <td className="px-4 py-3">{m.title}</td>
              <td className="px-4 py-3">{m.strand}</td>
              <td className="px-4 py-3">{m.description}</td>
            </tr>
          ))}
        </Table>
      </section>

      {/* Assignments */}
      <section>
        <h3 className="mb-2 font-semibold text-[#1b365d]">Assignments</h3>
        <Table headers={["Title", "Due", "Max marks", ""]}>
          {(assignments ?? []).map((a: any) => (
            <tr key={a.id}>
              <td className="px-4 py-3 font-medium">{a.title}</td>
              <td className="px-4 py-3 text-sm text-slate-600">{new Date(a.due_at).toLocaleString()}</td>
              <td className="px-4 py-3 text-sm">{a.max_score}</td>
              <td className="px-4 py-3">
                {isStudent ? (
                  <Button className="py-1 text-xs" onClick={() => navigate(`/nova/assignment/${a.id}`)}>
                    <FileText size={12} className="mr-1 inline" />
                    Submit
                  </Button>
                ) : isTeacher ? (
                  <Button variant="ghost" className="py-1 text-xs" onClick={() => navigate(`/nova/teacher/${id}/submissions`)}>
                    Edit
                  </Button>
                ) : null}
              </td>
            </tr>
          ))}
        </Table>
      </section>

      {/* Quizzes */}
      <section>
        <h3 className="mb-2 font-semibold text-[#1b365d]">Quizzes</h3>
        <Table headers={["Title", "Duration", "Due", ""]}>
          {(quizzes ?? []).map((q: any) => (
            <tr key={q.id}>
              <td className="px-4 py-3 font-medium">{q.title}</td>
              <td className="px-4 py-3 text-sm text-slate-600">
                {q.duration_minutes ? `${q.duration_minutes} min` : "—"}
              </td>
              <td className="px-4 py-3 text-sm text-slate-600">
                {q.due_at ? new Date(q.due_at).toLocaleString() : "—"}
              </td>
              <td className="px-4 py-3">
                {isStudent && (
                  <Button className="py-1 text-xs" onClick={() => navigate(`/nova/quiz/${q.id}/instructions`)}>
                    Start quiz
                  </Button>
                )}
              </td>
            </tr>
          ))}
        </Table>
      </section>

      {/* Discussion */}
      <section>
        <h3 className="mb-2 font-semibold text-[#1b365d]">Discussion</h3>
        <Card>
          {(discussions ?? []).map((d: any) => (
            <p key={d.id} className="border-b border-slate-100 py-2 text-sm">
              <strong>{d.author_name}:</strong> {d.body}
            </p>
          ))}
          {user?.portal !== "parent" ? (
            <Button className="mt-3" onClick={() => post.mutate("Please clarify today's homework. Thank you.")}>
              Post a question
            </Button>
          ) : null}
        </Card>
      </section>
    </div>
  );
}
