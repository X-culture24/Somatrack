import * as React from "react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { BorderBeam } from "@/components/ui/border-beam";
import {
  GraduationCap,
  Users,
  BookOpen,
  ArrowRight,
  ShieldCheck,
  Sparkles,
} from "lucide-react";

interface Hero195Props {
  className?: string;
  schoolName?: string;
  tagline?: string;
  stats?: Array<{ label: string; value: string | number; icon?: any }>;
  primaryAction?: { label: string; onClick?: () => void };
  secondaryAction?: { label: string; onClick?: () => void };
}

const DEFAULT_STATS = [
  { label: "Students", value: "1,200+", icon: Users },
  { label: "Teachers", value: "85+", icon: GraduationCap },
  { label: "Courses", value: "40+", icon: BookOpen },
  { label: "Accredited", value: "MoE", icon: ShieldCheck },
];

const Hero195 = React.forwardRef<HTMLDivElement, Hero195Props>(
  (
    {
      className,
      schoolName = "ACK St. Mary's School Kabete",
      tagline = "Excellence in education, rooted in faith and tradition.",
      stats,
      primaryAction = { label: "Get started" },
      secondaryAction = { label: "Learn more" },
    },
    ref,
  ) => {
    const displayStats = stats ?? DEFAULT_STATS;
    return (
      <section
        ref={ref}
        className={cn(
          "relative w-full overflow-hidden bg-gradient-to-br from-white via-[#f0f4f8] to-[#e3f2fd] py-12 sm:py-16 lg:py-20",
          className,
        )}
      >
        {/* Decorative blobs */}
        <div className="pointer-events-none absolute -top-24 -left-24 h-64 w-64 rounded-full bg-[#4a9eca]/20 blur-3xl" />
        <div className="pointer-events-none absolute -bottom-24 -right-24 h-72 w-72 rounded-full bg-[#1b365d]/20 blur-3xl" />
        <div className="pointer-events-none absolute top-1/3 right-1/4 h-40 w-40 rounded-full bg-[#4a9eca]/10 blur-2xl" />

        <div className="relative mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="grid items-center gap-10 lg:grid-cols-2">
            {/* Copy */}
            <div className="animate-fade-in">
              <div className="mb-5 inline-flex items-center gap-2 rounded-full border border-[#4a9eca]/30 bg-white/60 px-3 py-1 text-xs font-medium text-[#1b365d] backdrop-blur">
                <Sparkles size={14} className="text-[#4a9eca]" />
                Welcome to the Parent &amp; Staff Portal
              </div>
              <h1 className="text-3xl font-bold tracking-tight text-[#1b365d] sm:text-4xl lg:text-5xl">
                {schoolName}
              </h1>
              <p className="mt-4 max-w-xl text-base leading-relaxed text-slate-600 sm:text-lg">
                {tagline}
              </p>
              <div className="mt-8 flex flex-wrap gap-3">
                {primaryAction && (
                  <Button
                    size="lg"
                    onClick={primaryAction.onClick}
                    className="group bg-[#1b365d] text-white hover:bg-[#152a49]"
                  >
                    {primaryAction.label}
                    <ArrowRight
                      size={16}
                      className="ml-2 transition-transform group-hover:translate-x-1"
                    />
                  </Button>
                )}
                {secondaryAction && (
                  <Button
                    size="lg"
                    variant="outline"
                    onClick={secondaryAction.onClick}
                    className="border-[#1b365d]/20 text-[#1b365d] hover:bg-[#1b365d]/5"
                  >
                    {secondaryAction.label}
                  </Button>
                )}
              </div>

              {/* Stats strip */}
              <dl className="mt-10 grid grid-cols-2 gap-3 sm:grid-cols-4">
                {displayStats.map((s) => {
                  const Icon = s.icon ?? ShieldCheck;
                  return (
                    <div
                      key={s.label}
                      className="rounded-xl border border-white/60 bg-white/70 p-3 text-center shadow-sm backdrop-blur"
                    >
                      <dt className="text-[11px] uppercase tracking-wide text-slate-500">
                        {s.label}
                      </dt>
                      <dd className="mt-1 flex items-center justify-center gap-1.5 text-lg font-bold text-[#1b365d]">
                        <Icon size={16} className="text-[#4a9eca]" />
                        {s.value}
                      </dd>
                    </div>
                  );
                })}
              </dl>
            </div>

            {/* Illustration / feature card */}
            <div className="relative animate-fade-in [animation-delay:200ms]">
              <div className="relative mx-auto max-w-md">
                <div className="relative overflow-hidden rounded-3xl border border-white/60 bg-white p-6 shadow-2xl">
                  <BorderBeam
                    size={300}
                    duration={18}
                    delay={2}
                    colorFrom="#4a9eca"
                    colorTo="#1b365d"
                    borderWidth={2}
                  />
                  <div className="aspect-[4/3] w-full overflow-hidden rounded-2xl bg-gradient-to-br from-[#1b365d] via-[#2c5282] to-[#4a9eca]">
                    <img
                      src="https://images.unsplash.com/photo-1523050854058-8df90110c9f1?auto=format&fit=crop&w=1200&q=80"
                      alt="Students in classroom"
                      className="h-full w-full object-cover opacity-90 mix-blend-luminosity"
                      onError={(e) => {
                        (e.target as HTMLImageElement).style.display = "none";
                      }}
                    />
                  </div>
                  <div className="mt-5 flex items-start justify-between gap-3">
                    <div>
                      <h3 className="text-lg font-semibold text-[#1b365d]">
                        Live Dashboard
                      </h3>
                      <p className="mt-0.5 text-sm text-slate-500">
                        Attendance, marks, fees, and announcements — all in one place.
                      </p>
                    </div>
                    <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-[#4a9eca]/15 text-[#4a9eca]">
                      <ShieldCheck size={20} />
                    </div>
                  </div>
                </div>

                {/* Floating mini cards */}
                <div className="absolute -left-4 top-1/3 hidden animate-fade-in rounded-2xl border border-white/60 bg-white/95 px-3 py-2 shadow-xl backdrop-blur sm:block [animation-delay:500ms]">
                  <div className="flex items-center gap-2">
                    <span className="flex h-7 w-7 items-center justify-center rounded-lg bg-[#e3f2fd] text-[#1b365d]">
                      <Users size={14} />
                    </span>
                    <div>
                      <div className="text-[10px] uppercase tracking-wide text-slate-400">
                        Present today
                      </div>
                      <div className="text-sm font-bold text-[#1b365d]">96.4%</div>
                    </div>
                  </div>
                </div>
                <div className="absolute -right-3 bottom-8 hidden animate-fade-in rounded-2xl border border-white/60 bg-white/95 px-3 py-2 shadow-xl backdrop-blur sm:block [animation-delay:700ms]">
                  <div className="flex items-center gap-2">
                    <span className="flex h-7 w-7 items-center justify-center rounded-lg bg-[#4a9eca]/15 text-[#4a9eca]">
                      <BookOpen size={14} />
                    </span>
                    <div>
                      <div className="text-[10px] uppercase tracking-wide text-slate-400">
                        Mean score
                      </div>
                      <div className="text-sm font-bold text-[#1b365d]">B+</div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>
    );
  },
);
Hero195.displayName = "Hero195";

export { Hero195 };
export default Hero195;
