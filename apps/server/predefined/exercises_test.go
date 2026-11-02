package predefined

import (
	"strings"
	"testing"
)

func TestExerciseNamesAreUniqueSuggestionsReturnedByCopy(t *testing.T) {
	names := ExerciseNames()
	if len(names) != 10 {
		t.Fatalf("suggested exercise names: got %d, want 10", len(names))
	}
	seen := map[string]bool{}
	for _, name := range names {
		key := strings.ToLower(strings.TrimSpace(name))
		if key == "" || name != strings.TrimSpace(name) || seen[key] {
			t.Fatalf("blank, untrimmed or duplicate exercise name %q", name)
		}
		seen[key] = true
	}
	first := names[0]
	names[0] = "changed by caller"
	if ExerciseNames()[0] != first {
		t.Fatal("caller changed the shared suggestion catalog")
	}
}
