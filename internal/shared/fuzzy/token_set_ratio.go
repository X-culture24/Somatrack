package fuzzy

import (
	"sort"
	"strings"
	"unicode"
)

func tokenize(s string) []string {
	f := func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}
	raw := strings.FieldsFunc(strings.ToUpper(s), f)
	seen := map[string]struct{}{}
	out := make([]string, 0, len(raw))
	for _, t := range raw {
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

func intersectionCount(a, b []string) int {
	i, j, count := 0, 0, 0
	for i < len(a) && j < len(b) {
		switch {
		case a[i] == b[j]:
			count++
			i++
			j++
		case a[i] < b[j]:
			i++
		default:
			j++
		}
	}
	return count
}

func unionCount(a, b []string) int {
	i, j, count := 0, 0, 0
	for i < len(a) && j < len(b) {
		switch {
		case a[i] == b[j]:
			count++
			i++
			j++
		case a[i] < b[j]:
			count++
			i++
		default:
			count++
			j++
		}
	}
	count += len(a) - i
	count += len(b) - j
	return count
}

func tversky(s0, s1 []string, alpha, beta float64) float64 {
	inter := float64(intersectionCount(s0, s1))
	onlyS0 := float64(len(s0)) - inter
	onlyS1 := float64(len(s1)) - inter
	denom := inter + alpha*onlyS0 + beta*onlyS1
	if denom == 0 {
		return 0
	}
	return inter / denom
}

func TokenSetRatio(s1, s2 string) float64 {
	t1 := tokenize(s1)
	t2 := tokenize(s2)
	if len(t1) == 0 || len(t2) == 0 {
		return 0
	}
	interN := intersectionCount(t1, t2)
	unionN := unionCount(t1, t2)
	interSet := make([]string, 0, interN)
	i, j := 0, 0
	for i < len(t1) && j < len(t2) {
		switch {
		case t1[i] == t2[j]:
			interSet = append(interSet, t1[i])
			i++
			j++
		case t1[i] < t2[j]:
			i++
		default:
			j++
		}
	}
	a := interSet
	b := t1
	if len(a) < len(b) {
		a, b = b, a
	}
	r0 := tversky(a, b, 0.5, 0.5)
	a = interSet
	b = t2
	if len(a) < len(b) {
		a, b = b, a
	}
	r1 := tversky(a, b, 0.5, 0.5)
	a = t1
	b = t2
	if len(a) < len(b) {
		a, b = b, a
	}
	r2 := tversky(a, b, 0.5, 0.5)
	max := r0
	if r1 > max {
		max = r1
	}
	if r2 > max {
		max = r2
	}
	_ = unionN
	return max * 100.0
}

func TokenSetRatioInt(s1, s2 string) int {
	return int(TokenSetRatio(s1, s2) + 0.5)
}
