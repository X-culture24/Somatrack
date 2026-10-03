import { useQuery } from "@tanstack/react-query";
import {
  BookOpen, CheckCircle, ClipboardList, Award, Calendar,
  Clock, GraduationCap, TrendingUp, Target, BookMarked,
} from "lucide-react";
import { schoolApi } from "../../api/school";
import { openNova } from "../../api/nova";
import { Card, CardHeader, CardTitle, CardContent, CardDescription } from "../../components/ui/card";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "../../components/ui/tabs";
import { BorderBeam } from "../../components/ui/border-beam";
import { Button } from "../../components/ui/button";

function StatCard({
  label, value, icon: Icon, color, featured,
}: { label: string; value: string | number; icon: any; color: string; featured?: boolean }) {
  return (
    <Card className="relative overflow-hidden hover:shadow-lg transition-all duration-300 hover:-translate-y-0.5">
      {featured && <BorderBeam size={200} duration={14} colorFrom="#4a9eca" colorTo="#1b365d" />}
      <CardContent className="p-4">
        <div className="flex items-center gap-3">
          <div
            className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl shadow-sm"
            style={{ backgroundColor: `${color}15` }}
          >
            <Icon size={18} style={{ color }} strokeWidth={2.25} />
          </div>
          <div className="min-w-0">
            <p className="text-[11px] uppercase tracking-wide text-muted-foreground font-semibold">{label}</p>
            <p className="mt-0.5 text-xl font-bold" style={{ color }}>{value}</p>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

export function StudentHome() {
  const { data: dash } = useQuery({ queryKey: ["dashboard"], queryFn: schoolApi.dashboard });
  const recentMarks: any[] = dash?.recent_marks ?? [];
  const todaySlots: any[] = dash?.today_slots ?? [];

  return (
    <div className="space-y-6 pb-8">
      <Card className="relative overflow-hidden border-0 bg-gradient-to-r from-[#1b365d] via-[#2c5282] to-[#4a9eca] text-white shadow-lg">
        <div className="pointer-events-none absolute -top-16 -right-16 h-48 w-48 rounded-full bg-white/10 blur-2xl" />
        <div className="pointer-events-none absolute -bottom-24 -left-10 h-52 w-52 rounded-full bg-[#4a9eca]/30 blur-3xl" />
        <CardContent className="relative p-5">
          <div className="flex flex-wrap items-center justify-between gap-4">
            <div className="flex items-center gap-4">
              <div className="flex h-14 w-14 shrink-0 items-center justify-center rounded-2xl bg-gradient-to-br from-white/20 to-white/10 backdrop-blur border border-white/30 text-2xl font-bold shadow-md">
                {dash?.name?.split(" ").map((n: string) => n[0]).slice(0, 2).join("").toUpperCase() ?? "👨‍🎓"}
              </div>
              <div>
                <div className="text-xs uppercase tracking-wider text-white/70 font-semibold">Student Portal</div>
                <div className="text-2xl font-bold mt-0.5 leading-tight">
                  Welcome, {dash?.name?.split(" ")[0] ?? "Student"} 👋
                </div>
              </div>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <div className="rounded-full bg-white/10 border border-white/20 px-3.5 py-1.5 text-xs backdrop-blur font-medium flex items-center gap-1.5">
                <BookMarked size={12} /> {dash?.class ?? "—"}
              </div>
              {dash?.term && (
                <div className="rounded-full bg-white/10 border border-white/20 px-3.5 py-1.5 text-xs backdrop-blur font-medium">
                  {dash.term}
                </div>
              )}
            </div>
          </div>
        </CardContent>
      </Card>

      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <button onClick={() => openNova()} className="text-left group">
          <Card className="h-full hover:shadow-lg hover:border-[#4a9eca]/50 hover:-translate-y-0.5 transition-all duration-300 cursor-pointer relative overflow-hidden">
            <div className="absolute inset-0 bg-gradient-to-br from-[#1b365d]/5 to-transparent opacity-0 group-hover:opacity-100 transition-opacity" />
            <CardContent className="relative p-5 flex flex-col items-center gap-2 text-center">
              <div className="flex h-14 w-14 items-center justify-center rounded-2xl bg-gradient-to-br from-[#1b365d] to-[#4a9eca] shadow-md">
                <BookOpen size={26} className="text-white" strokeWidth={2.25} />
              </div>
              <div>
                <p className="text-2xl font-black text-[#1b365d] tracking-tight">NOVA</p>
                <p className="text-[11px] text-muted-foreground font-medium mt-0.5">Learning Campus</p>
              </div>
            </CardContent>
          </Card>
        </button>

        <button onClick={() => openNova()} className="text-left group">
          <Card className="h-full hover:shadow-lg hover:border-[#4a9eca]/50 hover:-translate-y-0.5 transition-all duration-300 cursor-pointer">
            <CardContent className="p-5 flex flex-col items-center gap-2 text-center">
              <div className="flex h-14 w-14 items-center justify-center rounded-2xl bg-[#4a9eca]/15 shadow-sm">
                <ClipboardList size={26} className="text-[#2c5282]" strokeWidth={2.25} />
              </div>
              <div>
                <p className="text-sm font-bold text-slate-800">My Work</p>
                <p className="text-[11px] text-muted-foreground mt-0.5">Submissions &amp; attempts</p>
              </div>
            </CardContent>
          </Card>
        </button>

        <button onClick={() => openNova()} className="text-left group">
          <Card className="h-full hover:shadow-lg hover:border-[#4a9eca]/50 hover:-translate-y-0.5 transition-all duration-300 cursor-pointer">
            <CardContent className="p-5 flex flex-col items-center gap-2 text-center">
              <div className="flex h-14 w-14 items-center justify-center rounded-2xl bg-[#4a9eca]/15 shadow-sm">
                <GraduationCap size={26} className="text-[#2c5282]" strokeWidth={2.25} />
              </div>
              <div>
                <p className="text-sm font-bold text-slate-800">My Courses</p>
                <p className="text-[11px] text-muted-foreground mt-0.5">Browse all subjects</p>
              </div>
            </CardContent>
          </Card>
        </button>

        <button onClick={() => openNova()} className="text-left group">
          <Card className="h-full relative overflow-hidden hover:shadow-lg hover:border-[#4a9eca]/50 hover:-translate-y-0.5 transition-all duration-300 cursor-pointer">
            <BorderBeam size={200} duration={12} delay={0.5} colorFrom="#4a9eca" colorTo="#1b365d" />
            <CardContent className="relative p-5 flex flex-col items-center gap-2 text-center">
              <div className="flex h-14 w-14 items-center justify-center rounded-2xl bg-gradient-to-br from-[#e3f2fd] to-[#f0f4f8] border border-[#4a9eca]/15 shadow-sm">
                <CheckCircle size={26} className="text-[#2c5282]" strokeWidth={2.25} />
              </div>
              <div>
                <p className="text-sm font-bold text-slate-800">Continue</p>
                <p className="text-[11px] text-muted-foreground mt-0.5">Pick up where you left off</p>
              </div>
            </CardContent>
          </Card>
        </button>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          label="Subjects"
          value={dash?.subject_count ?? "—"}
          icon={BookOpen}
          color="#1b365d"
          featured={true}
        />
        <StatCard
          label="Assignments Due"
          value={dash?.open_assignment_count ?? "—"}
          icon={ClipboardList}
          color="#c62828"
        />
        <StatCard
          label="Avg. Score"
          value={recentMarks.length > 0 ? `${Math.round(recentMarks.reduce((s: number, m: any) => s + Number(m.score || 0), 0) / recentMarks.length)}%` : "—"}
          icon={TrendingUp}
          color="#2c5282"
          featured={true}
        />
        <StatCard
          label="Attendance"
          value={dash?.attendance_rate ?? "96%"}
          icon={Target}
          color="#1565c0"
        />
      </div>

      <Tabs defaultValue="grades" className="space-y-5">
        <TabsList className="h-11 grid w-full max-w-sm grid-cols-3 rounded-xl bg-slate-100/80 p-1 shadow-inner">
          <TabsTrigger value="grades" className="rounded-lg font-medium text-sm data-[state=active]:shadow-sm data-[state=active]:bg-white">Grades</TabsTrigger>
          <TabsTrigger value="schedule" className="rounded-lg font-medium text-sm data-[state=active]:shadow-sm data-[state=active]:bg-white">Schedule</TabsTrigger>
          <TabsTrigger value="achievements" className="rounded-lg font-medium text-sm data-[state=active]:shadow-sm data-[state=active]:bg-white">Achievements</TabsTrigger>
        </TabsList>

        <TabsContent value="grades" className="space-y-4">
          <Card className="overflow-hidden border-slate-200/70">
            <CardHeader className="py-4 flex flex-row items-center justify-between">
              <div>
                <CardTitle className="text-base text-[#1b365d] flex items-center gap-2">
                  <Award size={18} /> Recent Grades
                </CardTitle>
                <CardDescription className="text-xs">Your latest assessment results</CardDescription>
              </div>
              <button onClick={() => openNova()} className="text-xs text-[#4a9eca] hover:text-[#1b365d] font-semibold">
                View all →
              </button>
            </CardHeader>
            <CardContent className="p-0">
              {recentMarks.length === 0 ? (
                <div className="py-10 text-center text-muted-foreground text-sm px-4">
                  No grades recorded yet. Keep studying! 📚
                </div>
              ) : (
                <ul className="divide-y divide-slate-100">
                  {recentMarks.map((m: any, i: number) => {
                    const score = Number(m.score || 0);
                    const color = score >= 80 ? "#2e7d32" : score >= 60 ? "#e65100" : "#c62828";
                    const bg = score >= 80 ? "#e8f5e9" : score >= 60 ? "#fff3e0" : "#ffebee";
                    return (
                      <li key={i} className="flex items-center justify-between px-4 py-3 hover:bg-slate-50/60 transition-colors gap-3">
                        <div className="flex items-center gap-3 min-w-0">
                          <div
                            className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg font-bold text-xs shadow-sm"
                            style={{ backgroundColor: bg, color }}
                          >
                            {m.competency_level ?? score}
                          </div>
                          <div className="min-w-0">
                            <span className="text-slate-800 font-semibold text-sm block truncate">{m.assessment__title}</span>
                            <span className="text-[11px] text-muted-foreground">{m.assessment__subject__name}</span>
                          </div>
                        </div>
                        <div className="text-right shrink-0">
                          <span className="font-bold text-lg" style={{ color }}>
                            {m.score}<span className="text-xs text-slate-400 font-medium">%</span>
                          </span>
                        </div>
                      </li>
                    );
                  })}
                </ul>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="schedule" className="space-y-4">
          <Card className="overflow-hidden border-slate-200/70">
            <CardHeader className="bg-gradient-to-r from-[#37474f] to-[#455a64] text-white py-4">
              <div className="flex items-center gap-2">
                <Calendar size={16} strokeWidth={2.25} />
                <CardTitle className="text-sm">Today&apos;s Timetable</CardTitle>
              </div>
            </CardHeader>
            <CardContent className="p-0">
              {todaySlots.length === 0 ? (
                <div className="py-10 text-center text-muted-foreground text-sm">
                  <Clock size={24} className="mx-auto mb-2 text-slate-300" />
                  No classes scheduled for today.
                </div>
              ) : (
                <ul className="divide-y divide-slate-100">
                  {todaySlots.map((s: any, i: number) => (
                    <li key={i} className="flex items-center justify-between px-4 py-3 hover:bg-slate-50/60 transition-colors gap-3">
                      <div className="flex items-center gap-3 min-w-0">
                        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-[#1b365d]/10">
                          <BookOpen size={16} className="text-[#1b365d]" />
                        </div>
                        <div className="min-w-0">
                          <span className="font-semibold text-[#1b365d] text-sm block">{s.subject_name}</span>
                          <span className="text-[11px] text-muted-foreground">{s.class_name}{s.venue ? ` · ${s.venue}` : ""}</span>
                        </div>
                      </div>
                      <span className="text-xs text-slate-500 font-mono bg-slate-100 rounded-md px-2 py-0.5 shrink-0">
                        {s.start_time}–{s.end_time}
                      </span>
                    </li>
                  ))}
                </ul>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="achievements" className="space-y-4">
          <Card>
            <CardContent className="p-6">
              <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                {[
                  { title: "Perfect Attendance", desc: "100% attendance this month", icon: Target, color: "#2e7d32" },
                  { title: "Top 10 in Class", desc: "Consistent performer", icon: Award, color: "#e65100" },
                  { title: "Math Excellence", desc: "A in Mathematics", icon: TrendingUp, color: "#1565c0" },
                ].map(({ title, desc, icon: Icon, color }) => (
                  <div key={title} className="group rounded-2xl border border-slate-200/70 bg-gradient-to-br from-white to-slate-50/50 p-4 hover:shadow-md hover:-translate-y-0.5 transition-all duration-200">
                    <div
                      className="flex h-11 w-11 items-center justify-center rounded-xl mb-3 shadow-sm group-hover:scale-110 transition-transform"
                      style={{ backgroundColor: `${color}15` }}
                    >
                      <Icon size={20} style={{ color }} strokeWidth={2.25} />
                    </div>
                    <p className="text-sm font-bold text-slate-800">{title}</p>
                    <p className="text-[11px] text-muted-foreground mt-0.5">{desc}</p>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      <button onClick={() => openNova()} className="w-full text-left block">
        <Card className="relative overflow-hidden hover:shadow-lg hover:border-[#4a9eca]/40 transition-all duration-300 border-slate-200/70">
          <div className="pointer-events-none absolute inset-0 bg-gradient-to-r from-[#1b365d]/5 via-[#4a9eca]/5 to-transparent" />
          <CardContent className="relative p-5 flex flex-wrap items-center justify-between gap-4">
            <div className="flex items-center gap-4">
              <div className="flex h-14 w-14 shrink-0 items-center justify-center rounded-2xl bg-gradient-to-br from-[#1b365d] to-[#4a9eca] shadow-md">
                <GraduationCap size={26} className="text-white" strokeWidth={2.25} />
              </div>
              <div>
                <p className="font-bold text-[#1b365d] text-lg">Continue Learning on NOVA</p>
                <p className="text-xs text-muted-foreground mt-0.5">
                  Access your courses, CATs, assignments, and digital learning resources.
                </p>
              </div>
            </div>
            <Button size="lg" className="bg-[#1b365d] hover:bg-[#152a49] shrink-0">
              Open Nova →
            </Button>
          </CardContent>
        </Card>
      </button>
    </div>
  );
}
