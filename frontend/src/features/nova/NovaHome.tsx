import { useQuery } from "@tanstack/react-query";
import { useState, useMemo } from "react";
import { Link } from "react-router-dom";
import { BookOpen, Search } from "lucide-react";
import { novaApi } from "../../api/nova";
import { Input } from "../../components/ui";

// Deterministic cover colour based on course id — navy / light-blue / white palette
const COVERS = [
  "bg-[#1b365d]",
  "bg-[#4a9eca]",
  "bg-[#1a5276]",
  "bg-[#2e86c1]",
  "bg-[#154360]",
  "bg-[#1f618d]",
];

const PATTERNS = [
  "radial-gradient(circle, rgba(255,255,255,.12) 1px, transparent 1px)",
  "repeating-linear-gradient(45deg, rgba(255,255,255,.07) 0px, rgba(255,255,255,.07) 2px, transparent 2px, transparent 8px)",
  "repeating-linear-gradient(90deg, rgba(255,255,255,.07) 0px, rgba(255,255,255,.07) 2px, transparent 2px, transparent 12px)",
  "radial-gradient(ellipse at 20% 50%, rgba(255,255,255,.12) 0%, transparent 60%)",
];

function CourseCover({ id, subject }: { id: number; subject: string }) {
  const ci = id % COVERS.length;
  const pi = id % PATTERNS.length;
  return (
        <div
          className={`h-32 w-full rounded-t-2xl ${COVERS[ci]}`}
          style={{ backgroundImage: PATTERNS[pi], backgroundSize: "20px 20px" }}
        >
          <div className="flex h-full items-end p-3">
            <span className="rounded-full bg-white/20 px-2 py-0.5 text-xs font-semibold text-white backdrop-blur-sm">
              {subject}
            </span>
          </div>
        </div>
  );
}

type SortKey = "name" | "subject" | "class";

export function NovaHome() {
  const [search, setSearch] = useState("");
  const [sort, setSort] = useState<SortKey>("name");
  const [filterSubject, setFilterSubject] = useState("All");

  const { data: rawCourses, isLoading } = useQuery({
    queryKey: ["courses"],
    queryFn: () => novaApi.courses(),
  });

  const courses: any[] = rawCourses ?? [];
  const subjects = useMemo(
    () => ["All", ...Array.from(new Set(courses.map((c: any) => c.subject_name).filter(Boolean)))],
    [courses],
  );

  const filtered = useMemo(() => {
    let list = [...courses];
    if (filterSubject !== "All") list = list.filter((c) => c.subject_name === filterSubject);
    if (search.trim()) {
      const q = search.toLowerCase();
      list = list.filter(
        (c) =>
          c.title?.toLowerCase().includes(q) ||
          c.subject_name?.toLowerCase().includes(q) ||
          c.class_label?.toLowerCase().includes(q),
      );
    }
    list.sort((a, b) => {
      if (sort === "subject") return (a.subject_name ?? "").localeCompare(b.subject_name ?? "");
      if (sort === "class") return (a.class_label ?? "").localeCompare(b.class_label ?? "");
      return (a.title ?? "").localeCompare(b.title ?? "");
    });
    return list;
  }, [courses, search, sort, filterSubject]);

  return (
    <div className="space-y-5">
      {/* Header */}
      <div className="flex items-center gap-3">
        <BookOpen size={22} className="text-[#1b365d]" />
        <div>
          <h1 className="text-2xl font-semibold text-[#1b365d]">Course overview</h1>
          <p className="text-sm text-slate-500">
            {courses.length} course{courses.length !== 1 ? "s" : ""} available
          </p>
        </div>
      </div>

      {/* Filter bar — matches KCA style */}
      <div className="flex flex-wrap items-center gap-3">
        {/* Subject filter */}
        <div className="flex flex-wrap gap-1.5">
          {subjects.map((s) => (
            <button
              key={s}
              onClick={() => setFilterSubject(s)}
              className={`rounded-full border px-3 py-1 text-xs font-medium transition ${
                filterSubject === s
                  ? "border-[#1b365d] bg-[#1b365d] text-white"
                  : "border-slate-300 bg-white text-slate-600 hover:border-[#4a9eca]"
              }`}
            >
              {s}
            </button>
          ))}
        </div>

        <div className="ml-auto flex items-center gap-2">
          {/* Search */}
          <div className="relative">
            <Search size={14} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-400" />
            <Input
              className="pl-8 w-44 text-xs"
              placeholder="Search…"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
          {/* Sort */}
          <select
            value={sort}
            onChange={(e) => setSort(e.target.value as SortKey)}
            className="rounded-lg border border-slate-300 bg-white px-3 py-2 text-xs outline-none focus:ring-2 ring-[#4a9eca]"
          >
            <option value="name">Sort by course name</option>
            <option value="subject">Sort by subject</option>
            <option value="class">Sort by class</option>
          </select>
        </div>
      </div>

      {/* Course grid */}
      {isLoading && (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <div key={i} className="h-52 animate-pulse rounded-2xl bg-slate-200" />
          ))}
        </div>
      )}

      {!isLoading && filtered.length === 0 && (
        <p className="rounded-2xl border border-slate-200 bg-white py-12 text-center text-sm text-slate-400">
          No courses found.
        </p>
      )}

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {filtered.map((c: any) => (
          <Link key={c.id} to={`/nova/courses/${c.id}`} className="group block">
            <div className="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm transition hover:shadow-md hover:border-[#4a9eca]">
              <CourseCover id={c.id} subject={c.subject_name ?? ""} />
              <div className="p-4">
                <p className="text-xs font-semibold uppercase tracking-wide text-[#4a9eca]">
                  {c.subject_name}
                </p>
                <h3 className="mt-0.5 line-clamp-2 text-sm font-semibold text-[#1b365d] group-hover:text-[#4a9eca] transition">
                  {c.title}
                </h3>
                <p className="mt-1 text-xs text-slate-500">{c.class_label}</p>
                {/* Progress bar placeholder — real progress from submissions/attempts */}
                <div className="mt-3 flex items-center gap-2">
                  <div className="h-1.5 flex-1 rounded-full bg-slate-100">
                    <div
                      className="h-1.5 rounded-full bg-[#4a9eca]"
                      style={{ width: `${Math.min(100, (c.id * 17) % 101)}%` }}
                    />
                  </div>
                  <span className="text-xs text-slate-400">
                    {Math.min(100, (c.id * 17) % 101)}% complete
                  </span>
                </div>
              </div>
            </div>
          </Link>
        ))}
      </div>
    </div>
  );
}
