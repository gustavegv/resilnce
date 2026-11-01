// Package training owns exercise-centric application operations and persistence.
package training

import "time"

type Configuration struct {
	PrescribedSets int     `json:"prescribedSets"`
	InitialWeight  string  `json:"initialWeight"`
	RepThreshold   *int    `json:"repThreshold,omitempty"`
	AutoIncrement  *string `json:"autoIncrement,omitempty"`
	RestSeconds    *int    `json:"restSeconds,omitempty"`
}

// SessionItem is a tagged intent. Further creation/reuse intents are added here,
// rather than accepting configuration overrides on references implicitly.
type SessionItem struct {
	Kind             string        `json:"kind"`
	Name             string        `json:"name"`
	MuscleGroupCodes Codes         `json:"muscleGroupCodes,omitempty"`
	Configuration    Configuration `json:"configuration"`
}
type CreateSession struct {
	OperationID string        `json:"operationId"`
	Name        string        `json:"name"`
	Items       []SessionItem `json:"items"`
}
type TargetSet struct {
	ID           string `json:"id"`
	Position     int    `json:"position"`
	TargetReps   int    `json:"targetReps"`
	TargetWeight string `json:"targetWeight"`
}
type Exercise struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Revision         string   `json:"revision"`
	MuscleGroupCodes []string `json:"muscleGroupCodes"`
	Archived         bool     `json:"archived"`
}
type Variant struct {
	ID              string      `json:"id"`
	Name            string      `json:"name"`
	Revision        string      `json:"revision"`
	ProgressionMode string      `json:"progressionMode"`
	RepThreshold    int         `json:"repThreshold"`
	AutoIncrement   string      `json:"autoIncrement"`
	RestSeconds     int         `json:"restSeconds"`
	PrescribedSets  int         `json:"prescribedSets"`
	Archived        bool        `json:"archived"`
	TargetSets      []TargetSet `json:"targetSets"`
}
type Placement struct {
	ID        string   `json:"id"`
	VariantID string   `json:"variantId"`
	Position  int      `json:"position"`
	Exercise  Exercise `json:"exercise"`
	Variant   Variant  `json:"variant"`
}
type Session struct {
	ID              string      `json:"id"`
	Name            string      `json:"name"`
	Revision        string      `json:"revision"`
	CreatedAt       time.Time   `json:"createdAt"`
	LastPerformedAt *time.Time  `json:"lastPerformedAt"`
	Placements      []Placement `json:"placements"`
}
type Acknowledgment struct {
	OperationID      string    `json:"operationId"`
	AcceptedRevision string    `json:"acceptedRevision"`
	CommittedAt      time.Time `json:"committedAt"`
}
type SessionResult struct {
	Operation Acknowledgment `json:"operation"`
	Session   Session        `json:"session"`
}
