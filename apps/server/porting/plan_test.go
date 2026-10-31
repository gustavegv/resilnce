package porting

import (
	"encoding/json"
	"reflect"
	"testing"
)

func fixtureRow(t *testing.T, table, owner, raw string) Row {
	t.Helper()
	r, err := sourceRow(table, json.RawMessage(raw), owner)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestDecimalNeverRounds(t *testing.T) {
	for _, tc := range []struct {
		source, want string
		ok           bool
	}{
		{"2.5", "2.50", true}, {"5e1", "50.00", true}, {"0", "0.00", true}, {"12.345", "", false}, {"-1", "", false}, {"NaN", "", false}, {"9999.01", "", false},
	} {
		t.Run(tc.source, func(t *testing.T) {
			n := json.Number(tc.source)
			got, ok := decimal(&n, "0", "9999")
			if got != tc.want || ok != tc.ok {
				t.Fatalf("got %q,%v", got, ok)
			}
		})
	}
}

func TestOccurrenceRequiresUniqueInstant(t *testing.T) {
	for _, tc := range []struct {
		raw, zone string
		ok        bool
	}{
		{"2026-01-15T14:12:10.123456", "Europe/Stockholm", true},
		{"2026-03-29T02:30:00", "Europe/Stockholm", false},
		{"2026-10-25T02:30:00", "Europe/Stockholm", false},
		{"2026-10-25T02:30:00", "UTC", true}, {"infinity", "UTC", false},
	} {
		t.Run(tc.raw+tc.zone, func(t *testing.T) {
			_, err := occurrence(tc.raw, tc.zone)
			if (err == nil) != tc.ok {
				t.Fatalf("unexpected timestamp outcome: %v", err)
			}
		})
	}
}

func TestPlannerGroupingValidationAndStableNames(t *testing.T) {
	all := []Row{
		fixtureRow(t, "User", "a", `{"mail":"a"}`),
		fixtureRow(t, "Exercise", "a", `{"ex_id":1,"ex_name":" Bench ","_normalized_name":"bench","auto_inc":2.5}`),
		fixtureRow(t, "Exercise", "a", `{"ex_id":2,"ex_name":"BENCH","_normalized_name":"bench","auto_inc":2.5}`),
		fixtureRow(t, "Session", "a", `{"ses_id":1,"ses_name":"Push"}`),
		fixtureRow(t, "SesExercise", "a", `{"ses_id":1,"ex_id":1,"set_count":2,"rep_per_set":[12,11],"weight_per_set":[40,50],"rep_threshold":12,"rest_seconds":0,"order":null}`),
		fixtureRow(t, "SesExercise", "a", `{"ses_id":1,"ex_id":2,"set_count":2,"rep_per_set":[12,11],"weight_per_set":[40,50],"rep_threshold":12,"rest_seconds":0,"order":null}`),
	}
	p := buildPlan(all, "a", emptyState(), Options{})
	if !p.Ready || len(p.Groups) != 1 || len(p.Assignments) != 2 {
		t.Fatalf("unexpected plan: %+v", p.OwnerReport)
	}
	if p.Assignments[0].Name != "Variant 1" || p.Assignments[1].Name != "Variant 2" || p.Assignments[1].Position != 2 {
		t.Fatal(p.Assignments)
	}
	if !reflect.DeepEqual(p.Assignments[0].Weights, []string{"40.00", "50.00"}) {
		t.Fatal(p.Assignments)
	}
	s := emptyState()
	s.Groups["bench"] = Group{"bench", "exercise", 3}
	s.Exercises["exercise"] = "renamed"
	s.VariantNames["exercise"] = map[string]bool{"variant 3": true}
	s.Mapped[all[4].identity()] = References{Variant: "variant"}
	p = buildPlan(all, "a", s, Options{})
	if !p.Ready || len(p.Assignments) != 1 || p.Assignments[0].Name != "Variant 4" {
		t.Fatal(p.OwnerReport)
	}
	all[5] = fixtureRow(t, "SesExercise", "a", `{"ses_id":1,"ex_id":2,"set_count":2,"rep_per_set":[12,null],"weight_per_set":[40,50.001],"rep_threshold":12,"rest_seconds":0}`)
	p = buildPlan(all, "a", s, Options{})
	if p.Ready {
		t.Fatal("invalid targets accepted")
	}
}
