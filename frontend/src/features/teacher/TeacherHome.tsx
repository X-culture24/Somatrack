import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import {
  CalendarCheck, NotebookPen, FileText, Clock, CheckCircle2, Circle,
  BookOpen, Users, GraduationCap, TrendingUp, ClipboardList, Award,
} from "lucide-react";
import { schoolApi } from "../../api/school";
import { teacherApi } from "../../api/teacher";
import { Badge } from "../../components/ui";
import { Card, CardHeader, CardTitle, CardContent, CardDescription } from "../../components/ui/card";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "../../components/ui/tabs";
import { BorderBeam } from "../../components/ui/border-beam";
import { Button } from "../../components/ui/button";

const QUICK_ACTIONS = [
  { to: "/teacher/attendance", label: "Take attendance", icon: CalendarCheck, color: "#1565c0", bg: "#e3f2fd" },
  { to: "/teacher/marks", label: "Enter marks", icon: NotebookPen, color: "#1b365d", bg: "#dbeafe" },
  { to: "/teacher/lesson-plans", label: "New lesson plan", icon: FileText, color: "#2c5282", bg: "#eff6fb" },
];

function StatCard({
  label,
  value,
  icon: Icon,
  color,
  trend,
  featured,
}: {
  label: string;
  value: string | number;
  icon: any;
  color: string;
  trend?: { value: number; up: boolean };
  featured?: boolean;
}) {
  return (
    <Card className="relative overflow-hidden hover:shadow-lg transition-shadow duration-300">
      {featured && (
        <BorderBeam size={200} duration={14} colorFrom="#4a9eca" colorTo="#1b365d" delay={1} />
      )}
      <CardContent className="p-5">
        <div className="flex items-start justify-between">
          <div>
            <p className="text-xs text-muted-foreground uppercase tracking-wide font-medium">{label}</p>
            <p className="mt-1.5 text-2xl font-bold" style={{ color }}>{value}</p>
            {trend && (
              <div className="mt-1.5 flex items-center gap-1 text-xs">
                <TrendingUp size={12} className="text-green-600" />
                <span className="text-green-600 font-semibold">{trend.value}%</span>
                <span className="text-muted-foreground">vs last week</span>
              </div>
            )}
          </div>
          <div
            className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl shadow-sm"
            style={{ backgroundColor: `${color}15` }}
          >
            <Icon size={20} style={{ color }} strokeWidth={2.25} />
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

export function TeacherHome() {
  const { data } = useQuery({ queryKey: ["dashboard"], queryFn: schoolApi.dashboard });
  const { data: myClasses } = useQuery({ queryKey: ["teacher-my-classes"], queryFn: teacherApi.myClasses });
  const recentPlans: any[] = data?.recent_lesson_plans ?? [];
  const classes: any[] = (myClasses as any)?.results ?? (myClasses as any[]) ?? [];
  const todaySlots: any[] = data?.today_slots ?? [];
  const attendanceMarkedToday: number = data?.attendance_marked_today ?? 0;

  return (
    <div className="space-y-6 pb-8">
      <Card className="relative overflow-hidden border-0 bg-gradient-to-r from-[#1b365d] via-[#2c5282] to-[#4a9eca] text-white shadow-lg">
        <div className="pointer-events-none absolute -top-16 -right-16 h-48 w-48 rounded-full bg-white/10 blur-2xl" />
        <CardContent className="relative p-5">
          <div className="flex flex-wrap items-center justify-between gap-4">
            <div className="flex items-center gap-4">
              <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-white/15 backdrop-blur border border-white/20">
                <GraduationCap size={24} />
              </div>
              <div>
                <div className="text-xs uppercase tracking-wider text-white/70 font-semibold">Teacher Portal</div>
                <div className="text-xl font-bold mt-0.5">Welcome back 👋</div>
              </div>
            </div>
            {data?.term && (
              <div className="rounded-full bg-white/10 border border-white/20 px-3.5 py-1.5 text-xs backdrop-blur font-medium">
                {data.term}
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          label="Learners (Homeroom)"
          value={data?.learner_count ?? "—"}
          icon={Users}
          color="#1b365d"
          featured={true}
        />
        <StatCard
          label="Nova Courses"
          value={data?.nova_course_count ?? "—"}
          icon={BookOpen}
          color="#2c5282"
          trend={{ value: 15, up: true }}
        />
        <StatCard
          label="Assignments Due"
          value={data?.open_assignment_count ?? "—"}
          icon={ClipboardList}
          color="#4a9eca"
        />
        <StatCard
          label="Attendance Today"
          value={`${attendanceMarkedToday}`}
          icon={Award}
          color="#1565c0"
          trend={{ value: 5, up: true }}
          featured={true}
        />
      </div>

      <Card className="relative overflow-hidden">
        <CardHeader className="pb-2">
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="text-lg font-semibold text-[#1b365d]">Quick Actions</CardTitle>
              <CardDescription className="mt-0.5 text-xs">Start your day here</CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <div className="grid gap-3 sm:grid-cols-3">
            {QUICK_ACTIONS.map(({ to, label, icon: Icon, color, bg }) => (
              <Link
                key={to}
                to={to}
                className="group flex items-center gap-3 rounded-2xl border border-slate-200/70 bg-gradient-to-br from-white to-slate-50/50 p-4 shadow-sm hover:shadow-lg hover:border-[#4a9eca]/40 hover:-translate-y-0.5 transition-all duration-200"
              >
                <div
                  className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl shadow-sm group-hover:scale-110 transition-transform"
                  style={{ background: bg }}
                >
                  <Icon size={22} style={{ color }} strokeWidth={2.25} />
                </div>
                <div className="min-w-0">
                  <span className="text-sm font-bold text-slate-800 group-hover:text-[#1b365d] transition-colors">{label}</span>
                  <p className="text-[11px] text-muted-foreground mt-0.5">Click to open →</p>
                </div>
              </Link>
            ))}
          </div>
        </CardContent>
      </Card>

      <Tabs defaultValue="schedule" className="space-y-5">
        <TabsList className="h-11 grid w-full max-w-md grid-cols-3 rounded-xl bg-slate-100/80 p-1 shadow-inner">
          <TabsTrigger value="schedule" className="rounded-lg font-medium text-sm data-[state=active]:shadow-sm data-[state=active]:bg-white">Schedule</TabsTrigger>
          <TabsTrigger value="classes" className="rounded-lg font-medium text-sm data-[state=active]:shadow-sm data-[state=active]:bg-white">My Classes</TabsTrigger>
          <TabsTrigger value="plans" className="rounded-lg font-medium text-sm data-[state=active]:shadow-sm data-[state=active]:bg-white">Lesson Plans</TabsTrigger>
        </TabsList>

        <TabsContent value="schedule" className="space-y-4">
          <Card className="overflow-hidden border-slate-200/70">
            <CardHeader className="bg-gradient-to-r from-[#37474f] to-[#455a64] text-white py-4">
              <div className="flex items-center gap-2">
                <Clock size={16} strokeWidth={2.25} />
                <CardTitle className="text-sm">Today&apos;s Schedule</CardTitle>
              </div>
            </CardHeader>
            <CardContent className="p-0">
              {todaySlots.length === 0 ? (
                <div className="py-10 text-center text-muted-foreground text-sm">
                  No classes scheduled for today.
                </div>
              ) : (
                <ul className="divide-y divide-slate-100">
                  {todaySlots.map((s: any, i: number) => (
                    <li
                      key={i}
                      className="flex items-center justify-between px-4 py-3 hover:bg-slate-50/60 transition-colors"
                    >
                      <div className="flex items-center gap-3">
                        <div className="flex h-9 w-9 shrink-0 items-center justify-center">
                          {s.attendance_done ? (
                            <CheckCircle2 size={18} className="text-green-500" strokeWidth={2.5} />
                          ) : (
                            <Circle size={18} className="text-slate-300" strokeWidth={2} />
                          )}
                        </div>
                        <div>
                          <span className="font-semibold text-[#1b365d] text-sm">{s.subject_name}</span>
                          <span className="ml-1.5 text-xs text-slate-500">· {s.class_name}</span>
                          {s.venue && <span className="ml-1.5 text-[11px] text-muted-foreground font-medium">· {s.venue}</span>}
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <span className="text-xs text-slate-500 font-mono bg-slate-100 rounded-md px-2 py-0.5">
                          {s.start_time}–{s.end_time}
                        </span>
                        {!s.attendance_done && (
                          <Button size="sm" variant="outline" asChild className="h-7 px-2.5 text-[11px] border-slate-200">
                            <Link to="/teacher/attendance">Mark</Link>
                          </Button>
                        )}
                      </div>
                    </li>
                  ))}
                </ul>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="classes" className="space-y-4">
          <Card className="overflow-hidden border-slate-200/70">
            <CardHeader className="py-4">
              <CardTitle className="text-base text-[#1b365d]">My Classes &amp; Subjects</CardTitle>
              <CardDescription className="text-xs">Classes and subjects assigned to you this term</CardDescription>
            </CardHeader>
            <CardContent className="p-0">
              {classes.length === 0 ? (
                <div className="py-10 text-center text-muted-foreground text-sm px-4">
                  Not assigned to any class yet — an admin can assign you on the Classes or Staff page.
                </div>
              ) : (
                <ul className="divide-y divide-slate-100">
                  {classes.map((c: any) => (
                    <li
                      key={c.id}
                      className="flex items-center justify-between px-4 py-3 hover:bg-slate-50/60 transition-colors"
                    >
                      <div className="flex items-center gap-3 min-w-0">
                        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-[#1b365d]/10">
                          <BookOpen size={18} className="text-[#1b365d]" />
                        </div>
                        <div className="min-w-0">
                          <span className="font-semibold text-[#1b365d] text-sm flex items-center gap-2">
                            {c.name}
                            {c.stream_name && <span className="text-xs text-slate-500 font-normal">· {c.stream_name}</span>}
                          </span>
                          {c.subjects_taught?.length > 0 && (
                            <p className="text-[11px] text-muted-foreground mt-0.5 truncate">
                              {c.subjects_taught.join(", ")}
                            </p>
                          )}
                        </div>
                      </div>
                      <div className="flex items-center gap-2 shrink-0">
                        <span className="text-xs text-slate-500 font-medium">
                          {c.student_count} learners
                        </span>
                        {c.is_homeroom && <Badge variant="navy">Homeroom</Badge>}
                      </div>
                    </li>
                  ))}
                </ul>
              )}
            </CardContent>
          </Card>

          {classes.some((c: any) => c.is_homeroom) && attendanceMarkedToday >= 0 && (
            <Card className="border-[#4a9eca]/25 bg-gradient-to-br from-[#e3f2fd]/60 to-white">
              <CardContent className="p-4">
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div className="flex items-center gap-3">
                    <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-[#4a9eca]/15">
                      <CalendarCheck size={18} className="text-[#1b365d]" />
                    </div>
                    <div>
                      <p className="text-sm font-semibold text-slate-800">Homeroom Attendance</p>
                      <p className="text-xs text-muted-foreground">
                        Attendance marked today: <span className="font-bold text-[#1b365d]">{attendanceMarkedToday}</span> learner{attendanceMarkedToday !== 1 ? "s" : ""}
                      </p>
                    </div>
                  </div>
                  {attendanceMarkedToday === 0 && (
                    <Button size="sm" asChild className="bg-green-700 hover:bg-green-800">
                      <Link to="/teacher/attendance">Take attendance now →</Link>
                    </Button>
                  )}
                </div>
              </CardContent>
            </Card>
          )}
        </TabsContent>

        <TabsContent value="plans" className="space-y-4">
          <Card className="overflow-hidden border-slate-200/70">
            <CardHeader className="py-4 flex flex-row items-center justify-between">
              <div>
                <CardTitle className="text-base text-[#1b365d]">Recent Lesson Plans</CardTitle>
                <CardDescription className="text-xs">Your recently created plans</CardDescription>
              </div>
              <Button size="sm" variant="outline" asChild className="border-slate-200">
                <Link to="/teacher/lesson-plans">+ New plan</Link>
              </Button>
            </CardHeader>
            <CardContent className="p-0">
              {recentPlans.length === 0 ? (
                <div className="py-10 text-center text-muted-foreground text-sm">
                  No lesson plans yet.
                </div>
              ) : (
                <ul className="divide-y divide-slate-100">
                  {recentPlans.map((p: any, i: number) => (
                    <li
                      key={i}
                      className="flex items-start justify-between px-4 py-3 hover:bg-slate-50/60 transition-colors gap-3"
                    >
                      <div className="flex items-start gap-3 min-w-0">
                        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-[#4a9eca]/12 mt-0.5">
                          <FileText size={16} className="text-[#2c5282]" />
                        </div>
                        <div className="min-w-0">
                          <span className="text-slate-800 font-semibold text-sm">{p.topic}</span>
                          <p className="text-[11px] text-muted-foreground mt-0.5">
                            {p.class_name} · {p.subject_name}
                          </p>
                        </div>
                      </div>
                      <span className="text-[11px] text-slate-500 shrink-0 bg-slate-100 rounded-md px-2 py-0.5 font-mono">
                        {p.lesson_date}
                      </span>
                    </li>
                  ))}
                </ul>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}
