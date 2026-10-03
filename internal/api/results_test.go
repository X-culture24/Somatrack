package api

import (
	"math"
	"testing"
)

func near(a, b float64) bool { return math.Abs(a-b) < 0.01 }

func TestCBCDefaultGrading(t *testing.T) {
	cases := map[float64]string{100: "EE", 80: "EE", 79.9: "ME", 50: "ME", 49.95: "AE", 30: "AE", 29.99: "BE", 0: "BE", -5: "BE", 120: "EE"}
	for pct, want := range cases {
		if got := cbcDefault.grade(pct).Letter; got != want {
			t.Errorf("grade(%v) = %s, want %s", pct, got, want)
		}
	}
}

func TestConfiguredBandsWithGapsFallBack(t *testing.T) {
	// Like the seed data: only an EE band configured.
	g := grading{{80, 100, "EE", 4, "Exceeds"}}
	if g.grade(90).Letter != "EE" || g.grade(60).Letter != "ME" {
		t.Errorf("gap should fall back to CBC levels: %v %v", g.grade(90), g.grade(60))
	}
}

func TestSubjectPercentageWeightedMean(t *testing.T) {
	got := subjectPercentage([]scoredAssessment{
		{pct: 80, weight: 100, kind: "cat1"},
		{pct: 40, weight: 300, kind: "endterm"},
	})
	if !near(got, 50) { // (80*100 + 40*300) / 400
		t.Errorf("weighted mean = %v, want 50", got)
	}
}

func TestSubjectPercentageExamThread(t *testing.T) {
	th := func(pct float64, kind string) scoredAssessment {
		return scoredAssessment{pct: pct, weight: 100, kind: kind, hasThread: true, catsWeight: 30, mainWght: 70}
	}
	// CATs average 60, main exam 80 → 0.3*60 + 0.7*80 = 74.
	got := subjectPercentage([]scoredAssessment{th(50, "cat1"), th(70, "cat2"), th(80, "endterm")})
	if !near(got, 74) {
		t.Errorf("thread blend = %v, want 74", got)
	}
	// Main exam not sat yet: CATs alone, not dragged to 30% of their value.
	if got := subjectPercentage([]scoredAssessment{th(50, "cat1"), th(70, "cat2")}); !near(got, 60) {
		t.Errorf("CATs only = %v, want 60", got)
	}
	if got := subjectPercentage([]scoredAssessment{th(90, "endterm")}); !near(got, 90) {
		t.Errorf("main only = %v, want 90", got)
	}
}
