package training

import "time"

type StartWorkout struct {
	OperationID             string `json:"operationId"`
	SessionID               string `json:"sessionId"`
	ExpectedSessionRevision string `json:"expectedSessionRevision"`
}

type WorkoutSource struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Workout contains server-confirmed performance. CurrentVariant is separate from
// captured attempt settings/targets, and may have a newer configuration revision.
type Workout struct {
	ID            string            `json:"id"`
	Status        string            `json:"status"`
	Revision      string            `json:"revision"`
	Origin        string            `json:"origin"`
	SourceSession *WorkoutSource    `json:"sourceSession"`
	StartedAt     time.Time         `json:"startedAt"`
	EndedAt       *time.Time        `json:"endedAt"`
	Exercises     []WorkoutExercise `json:"exercises"`
}
type WorkoutExercise struct {
	ID              string       `json:"id"`
	VariantID       string       `json:"variantId"`
	Position        int          `json:"position"`
	State           string       `json:"state"`
	Origin          string       `json:"origin"`
	ConfigRevision  string       `json:"configRevision"`
	ProgressionMode string       `json:"progressionMode"`
	RepThreshold    int          `json:"repThreshold"`
	AutoIncrement   string       `json:"autoIncrement"`
	RestSeconds     int          `json:"restSeconds"`
	CompletedAt     *time.Time   `json:"completedAt"`
	RecordedRPE     *int         `json:"recordedRpe"`
	Notes           *string      `json:"notes"`
	Exercise        Exercise     `json:"exercise"`
	CurrentVariant  Variant      `json:"currentVariant"`
	Sets            []WorkoutSet `json:"sets"`
}
type WorkoutSet struct {
	ID                 string     `json:"id"`
	TargetSetID        *string    `json:"targetSetId"`
	Position           int        `json:"position"`
	Kind               string     `json:"kind"`
	State              string     `json:"state"`
	TargetReps         int        `json:"targetReps"`
	TargetWeight       string     `json:"targetWeight"`
	RecordedReps       *int       `json:"recordedReps"`
	RecordedWeight     *string    `json:"recordedWeight"`
	RecordedAt         *time.Time `json:"recordedAt"`
	LateWeightOverride bool       `json:"lateWeightOverride"`
}
type WorkoutResult struct {
	Operation Acknowledgment `json:"operation"`
	Workout   Workout        `json:"workout"`
}
