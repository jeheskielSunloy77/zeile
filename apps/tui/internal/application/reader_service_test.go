package application

import "testing"

func TestComputeSectionTokenStarts(t *testing.T) {
	sections := []string{
		"alpha beta",
		"gamma",
		"delta epsilon zeta",
	}

	starts := computeSectionTokenStarts(sections)
	if len(starts) != 3 {
		t.Fatalf("expected 3 starts, got %d", len(starts))
	}
	if starts[0] != 0 {
		t.Fatalf("expected first section to start at 0, got %d", starts[0])
	}
	if !(starts[0] < starts[1] && starts[1] < starts[2]) {
		t.Fatalf("expected monotonic starts, got %v", starts)
	}
}
