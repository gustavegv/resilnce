// Package predefined contains the backend's predefined reference data.
package predefined

import "slices"

// MuscleGroup describes a selectable muscle group in the exercise library.
type MuscleGroup struct {
	Code      string `json:"code"`
	Label     string `json:"label"`
	SortOrder int    `json:"sort_order"`
}

var muscleGroups = [...]MuscleGroup{
	{Code: "chest", Label: "Chest", SortOrder: 1},
	{Code: "back", Label: "Back", SortOrder: 1},
	{Code: "shoulders", Label: "Shoulders", SortOrder: 1},
	{Code: "biceps", Label: "Biceps", SortOrder: 1},
	{Code: "triceps", Label: "Triceps", SortOrder: 1},
	{Code: "quadriceps", Label: "Quadriceps", SortOrder: 1},
	{Code: "hamstrings", Label: "Hamstrings", SortOrder: 1},
	{Code: "glutes", Label: "Glutes", SortOrder: 1},
	{Code: "abs", Label: "Abs", SortOrder: 1},
	{Code: "core", Label: "Core", SortOrder: 1},
	{Code: "calves", Label: "Calves", SortOrder: 1},

	{Code: "forearms", Label: "Forearms", SortOrder: 2},
	{Code: "traps", Label: "Traps", SortOrder: 2},
	{Code: "lats", Label: "Lats", SortOrder: 2},
	{Code: "lower_back", Label: "Lower Back", SortOrder: 2},
	{Code: "obliques", Label: "Obliques", SortOrder: 2},
	{Code: "adductors", Label: "Adductors", SortOrder: 2},
	{Code: "abductors", Label: "Abductors", SortOrder: 2},

	{Code: "rear_delts", Label: "Rear Delts", SortOrder: 3},
	{Code: "front_delts", Label: "Front Delts", SortOrder: 3},
	{Code: "side_delts", Label: "Side Delts", SortOrder: 3},
	{Code: "hip_flexors", Label: "Hip Flexors", SortOrder: 3},
	{Code: "rotator_cuff", Label: "Rotator Cuff", SortOrder: 3},
	{Code: "serratus", Label: "Serratus", SortOrder: 3},

	{Code: "tibialis", Label: "Tibialis", SortOrder: 4},
	{Code: "neck", Label: "Neck", SortOrder: 4},
}

// MuscleGroups returns a copy of the catalog in display order.
// Groups with equal sort_order retain their order from the predefined list.
func MuscleGroups() []MuscleGroup {
	return slices.Clone(muscleGroups[:])
}
