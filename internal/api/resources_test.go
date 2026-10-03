package api

import "testing"

// Every spec must pass resource.Check (Add panics otherwise) and have a
// unique name.
func TestSchoolResourcesAreValid(t *testing.T) {
	g := NewResourceRegistry(nil)
	if n := len(g.Names()); n < 30 {
		t.Fatalf("expected the full set of school resources, got %d", n)
	}
}
