package predefined

import "slices"

// exerciseNames suggests common lifts when building a personal exercise library.
// These names have no shared identity, targets, variants, or history.
var exerciseNames = [...]string{
	"Bench press",
	"Squat",
	"Deadlift",
	"Overhead press",
	"Barbell row",
	"Pull-up",
	"Lat pulldown",
	"Leg press",
	"Biceps curl",
	"Triceps pushdown",
}

// ExerciseNames returns a copy of the suggested names in display order.
// Choosing a name creates a private exercise only when its session is saved.
func ExerciseNames() []string {
	return slices.Clone(exerciseNames[:])
}
