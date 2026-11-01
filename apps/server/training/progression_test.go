package training_test

import (
	"github.com/gustavegv/resilnce/apps/server/training"
	"reflect"
	"testing"
)

func TestAverageProgressionWorkedExample(t *testing.T) {
	v := training.Variant{ProgressionMode: "average", RepThreshold: 11, AutoIncrement: "2.50", PrescribedSets: 3, TargetSets: []training.TargetSet{
		{ID: "a", Position: 1, TargetReps: 11, TargetWeight: "40.00"},
		{ID: "b", Position: 2, TargetReps: 11, TargetWeight: "50.00"},
		{ID: "c", Position: 3, TargetReps: 11, TargetWeight: "50.00"},
	}}
	sets := []training.WorkoutSet{}
	for i, reps := range []int{12, 11, 10} {
		id := v.TargetSets[i].ID
		r := reps
		sets = append(sets, training.WorkoutSet{TargetSetID: &id, Kind: "prescribed", State: "performed", RecordedReps: &r})
	}
	got, err := training.NextTargets(v, sets)
	want := []training.TargetSet{{ID: "a", Position: 1, TargetReps: 6, TargetWeight: "42.50"}, {ID: "b", Position: 2, TargetReps: 6, TargetWeight: "52.50"}, {ID: "c", Position: 3, TargetReps: 6, TargetWeight: "52.50"}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("worked average: %+v %v", got, err)
	}
	if v.TargetSets[0].TargetWeight != "40.00" || *sets[0].RecordedReps != 12 {
		t.Fatal("progression mutated its inputs")
	}
}

func TestAverageProgressionResolutionBoundariesAndOverrides(t *testing.T) {
	for _, tc := range []struct {
		name        string
		threshold   int
		reps        []int
		skipped     int
		override    int
		weights     []string
		wantReps    []int
		wantWeights []string
		invalid     bool
	}{
		{name: "below unrounded average", threshold: 12, reps: []int{12, 11, 12}, skipped: -1, override: -1, weights: []string{"40.00", "50.00", "50.00"}, wantReps: []int{12, 11, 12}, wantWeights: []string{"40.00", "50.00", "50.00"}},
		{name: "zero participates", threshold: 11, reps: []int{0, 22}, skipped: -1, override: -1, weights: []string{"0.00", "50.00"}, wantReps: []int{6, 6}, wantWeights: []string{"2.50", "52.50"}},
		{name: "zero below", threshold: 12, reps: []int{0, 12}, skipped: -1, override: -1, weights: []string{"0.00", "50.00"}, wantReps: []int{0, 12}, wantWeights: []string{"0.00", "50.00"}},
		{name: "skip blocks even high average", threshold: 12, reps: []int{99, 99, 0}, skipped: 2, override: -1, weights: []string{"40.00", "50.00", "50.00"}, wantReps: []int{99, 99, 7}, wantWeights: []string{"40.00", "50.00", "50.00"}},
		{name: "fully skipped", threshold: 12, reps: []int{0}, skipped: 0, override: -1, weights: []string{"40.00"}, wantReps: []int{7}, wantWeights: []string{"40.00"}},
		{name: "late override suppresses whole increase and performed reset", threshold: 11, reps: []int{12, 11, 10}, skipped: -1, override: 1, weights: []string{"40.00", "45.00", "50.00"}, wantReps: []int{12, 11, 10}, wantWeights: []string{"40.00", "45.00", "50.00"}},
		{name: "actual reps above threshold bound remain reps", threshold: 99, reps: []int{2147483647}, skipped: -1, override: 0, weights: []string{"0.00"}, wantReps: []int{2147483647}, wantWeights: []string{"0.00"}},
		{name: "upper weight accepted exactly", threshold: 11, reps: []int{11}, skipped: -1, override: -1, weights: []string{"9996.50"}, wantReps: []int{6}, wantWeights: []string{"9999.00"}},
		{name: "one overflowing unequal weight rejects all", threshold: 11, reps: []int{11, 11}, skipped: -1, override: -1, weights: []string{"40.00", "9997.00"}, invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := training.Variant{ProgressionMode: "average", RepThreshold: tc.threshold, AutoIncrement: "2.50", PrescribedSets: len(tc.reps)}
			sets := []training.WorkoutSet{}
			for i, r := range tc.reps {
				id := string(rune('a' + i))
				v.TargetSets = append(v.TargetSets, training.TargetSet{ID: id, Position: i + 1, TargetReps: tc.threshold, TargetWeight: tc.weights[i]})
				state := "performed"
				var reps *int = &r
				if i == tc.skipped {
					state = "skipped"
					reps = nil
				}
				sets = append(sets, training.WorkoutSet{TargetSetID: &id, Kind: "prescribed", State: state, RecordedReps: reps, LateWeightOverride: i == tc.override})
			}
			got, err := training.NextTargets(v, sets)
			if tc.invalid {
				if err == nil || got != nil {
					t.Fatalf("bounds must reject all: %+v %v", got, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			for i, target := range got {
				if target.TargetReps != tc.wantReps[i] || target.TargetWeight != tc.wantWeights[i] {
					t.Fatalf("next target %d: %+v", i, target)
				}
			}
		})
	}
}

func TestAverageProgressionPositiveJavaScriptResetExamples(t *testing.T) {
	// Independently worked positive-threshold division/Math.round results.
	for _, tc := range []struct{ threshold, want int }{{1, 1}, {2, 1}, {3, 2}, {4, 2}, {5, 3}, {6, 4}, {7, 4}, {8, 5}, {9, 5}, {10, 6}, {11, 6}, {12, 7}, {16, 9}, {17, 10}, {18, 11}, {98, 58}, {99, 58}} {
		id := "target"
		reps := tc.threshold
		v := training.Variant{ProgressionMode: "average", RepThreshold: tc.threshold, AutoIncrement: "0.25", PrescribedSets: 1, TargetSets: []training.TargetSet{{ID: id, TargetReps: tc.threshold, TargetWeight: "0.00"}}}
		next, e := training.NextTargets(v, []training.WorkoutSet{{TargetSetID: &id, Kind: "prescribed", State: "performed", RecordedReps: &reps}})
		if e != nil || next[0].TargetReps != tc.want {
			t.Fatalf("threshold %d: %+v %v", tc.threshold, next, e)
		}
	}
}

func TestAverageProgressionPendingNeverBecomesActual(t *testing.T) {
	id := "target"
	v := training.Variant{ProgressionMode: "average", RepThreshold: 12, AutoIncrement: "2.50", PrescribedSets: 1, TargetSets: []training.TargetSet{{ID: id, TargetReps: 12, TargetWeight: "40.00"}}}
	next, e := training.NextTargets(v, []training.WorkoutSet{{TargetSetID: &id, Kind: "prescribed", State: "pending", TargetReps: 12}})
	if e == nil || next != nil {
		t.Fatalf("unresolved: %+v %v", next, e)
	}
}
