package porting

import "testing"

func TestSourceTimezoneDefaultsToStockholm(t *testing.T) {
	for _, tc := range []struct{ input, want string }{{"", DefaultSourceTimezone}, {"  ", DefaultSourceTimezone}, {"UTC", "UTC"}, {" Europe/Stockholm ", DefaultSourceTimezone}} {
		opts := Options{SourceTimezone: tc.input}.normalized()
		if opts.SourceTimezone != tc.want || opts.Validate() != nil {
			t.Fatalf("unexpected timezone for %q", tc.input)
		}
	}
	if (Options{SourceTimezone: "Not/AZone"}).Validate() == nil {
		t.Fatal("invalid explicit timezone accepted")
	}
}

func TestDeleteUnusedPlansOnlyUnmappedExercisesWithoutPerformance(t *testing.T) {
	user := fixtureRow(t, "User", "a", `{"mail":"a"}`)
	ex := fixtureRow(t, "Exercise", "a", `{"ex_id":1,"ex_name":"Unused","_normalized_name":"unused","auto_inc":null}`)
	all := []Row{user, ex}
	if p := buildPlan(all, "a", emptyState(), Options{}); p.Ready || len(p.DeleteUnused) != 0 {
		t.Fatal("unused exercise silently accepted without selecting deletion")
	}
	p := buildPlan(all, "a", emptyState(), Options{DeleteUnusedExercises: true})
	if !p.Ready || !p.DeleteUnused[ex.identity()] || p.Counts["Exercise"].DeletePlanned != 1 || len(p.Groups) != 0 {
		t.Fatal(p.OwnerReport)
	}
	withHistory := append(append([]Row{}, all...), fixtureRow(t, "History", "a", `{"ex_id":1,"save_date":"2026-01-15T12:00:00","avg_rep":10,"avg_weight":40,"no_of_sets":2}`))
	p = buildPlan(withHistory, "a", emptyState(), Options{DeleteUnusedExercises: true})
	if p.Ready || len(p.DeleteUnused) != 0 {
		t.Fatal("exercise with history selected for deletion")
	}
	s := emptyState()
	s.Mapped[ex.identity()] = References{Exercise: "already-imported"}
	p = buildPlan(all, "a", s, Options{DeleteUnusedExercises: true})
	if !p.Ready || len(p.DeleteUnused) != 0 || p.Counts["Exercise"].Mapped != 1 {
		t.Fatal("mapped source exercise selected for deletion")
	}
}

func TestMissingRestDefaultsToZeroAndHistoryUsesDefaultTimezone(t *testing.T) {
	all := []Row{
		fixtureRow(t, "User", "a", `{"mail":"a"}`),
		fixtureRow(t, "Exercise", "a", `{"ex_id":1,"ex_name":"Bench","_normalized_name":"bench","auto_inc":2.5}`),
		fixtureRow(t, "Session", "a", `{"ses_id":1,"ses_name":"Push"}`),
		fixtureRow(t, "History", "a", `{"ex_id":1,"save_date":"2026-01-15T12:00:00","avg_rep":10,"avg_weight":40,"no_of_sets":2}`),
	}
	for _, tc := range []struct {
		rest  string
		want  int
		ready bool
	}{{"null", 0, true}, {"45", 45, true}, {"-1", -1, false}} {
		placement := fixtureRow(t, "SesExercise", "a", `{"ses_id":1,"ex_id":1,"set_count":2,"rep_per_set":[12,10],"weight_per_set":[40,50],"rep_threshold":12,"order":1,"rest_seconds":`+tc.rest+`}`)
		p := buildPlan(append(append([]Row{}, all...), placement), "a", emptyState(), Options{})
		if p.Ready != tc.ready || p.Assignments[0].RestSeconds != tc.want {
			t.Fatal(p.OwnerReport)
		}
	}
}

func TestUnusedHistoryDeletionRequiresExplicitOptInAndPreviewsBothCounts(t *testing.T) {
	if (Options{DeleteUnusedHistory: true}).Validate() == nil {
		t.Fatal("history cleanup accepted without exercise cleanup")
	}
	all := []Row{
		fixtureRow(t, "User", "a", `{"mail":"a"}`),
		fixtureRow(t, "Exercise", "a", `{"ex_id":1,"ex_name":"Unused","_normalized_name":"unused","auto_inc":null}`),
		fixtureRow(t, "History", "a", `{"ex_id":1,"save_date":"2026-01-15T12:00:00","avg_rep":10,"avg_weight":40,"no_of_sets":2}`),
		fixtureRow(t, "History", "a", `{"ex_id":1,"save_date":"2026-01-16T12:00:00","avg_rep":11,"avg_weight":40,"no_of_sets":2}`),
	}
	opts := Options{DeleteUnusedExercises: true, DeleteUnusedHistory: true}
	p := buildPlan(all, "a", emptyState(), opts)
	if !p.Ready || len(p.DeleteUnused) != 3 || p.Counts["Exercise"].DeletePlanned != 1 || p.Counts["History"].DeletePlanned != 2 || p.Counts["History"].Retained != 0 || len(p.Groups) != 0 {
		t.Fatal(p.OwnerReport)
	}
	for _, issue := range p.Issues {
		if issue.Severity == "error" || issue.Code == "unassigned_legacy_history" {
			t.Fatal("history marked for deletion was validated for import", issue)
		}
	}
	state := emptyState()
	state.Mapped[all[2].identity()] = References{LegacyHistory: "already-imported"}
	p = buildPlan(all, "a", state, opts)
	if p.Ready || len(p.DeleteUnused) != 0 {
		t.Fatal("already imported history selected for deletion")
	}
	withPlacement := append(append([]Row{}, all...), fixtureRow(t, "Session", "a", `{"ses_id":1,"ses_name":"Push"}`), fixtureRow(t, "SesExercise", "a", `{"ses_id":1,"ex_id":1,"set_count":2,"rep_per_set":[12,10],"weight_per_set":[40,50],"rep_threshold":12,"rest_seconds":0,"order":1}`))
	p = buildPlan(withPlacement, "a", emptyState(), opts)
	if len(p.DeleteUnused) != 0 {
		t.Fatal("exercise used by a session selected for deletion")
	}
}

func TestHistoryDeletionRequiresExactSourceKeys(t *testing.T) {
	a := fixtureRow(t, "History", "a", `{"ex_id":1,"save_date":"2026-01-15T12:00:00.123456"}`)
	b := fixtureRow(t, "History", "a", `{"ex_id":1,"save_date":"2026-01-15T12:00:00.123457"}`)
	c := fixtureRow(t, "History", "a", `{"ex_id":2,"save_date":"2026-01-15T12:00:00.123456"}`)
	if !sameSourceKeys([]Row{a, b}, []Row{b, a}) {
		t.Fatal("source order affected key comparison")
	}
	if sameSourceKeys([]Row{a}, []Row{b}) || sameSourceKeys([]Row{a}, []Row{c}) || sameSourceKeys([]Row{a}, []Row{a, b}) || sameSourceKeys([]Row{a, b}, []Row{a, a}) {
		t.Fatal("changed history keys accepted for deletion")
	}
}
