//go:build integration

package api

import (
	"encoding/json"
	"fmt"
	"github.com/gustavegv/resilnce/apps/server/training"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/gustavegv/resilnce/apps/server/application"
	"github.com/gustavegv/resilnce/apps/server/internal/testdb"
	"github.com/gustavegv/resilnce/apps/server/migrations"
)

func TestStatsSumsActualSetsIncludingCompletedWorkInActiveWorkout(t *testing.T) {
	pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
	if e := migrations.Apply(ctx, pool); e != nil {
		t.Fatal(e)
	}
	h, e := NewHandler(ctx, Options{Pool: pool, Origin: "https://app.example", Application: application.Config{DevelopmentMode: true}})
	if e != nil {
		t.Fatal(e)
	}
	w := configuredWorkout(t, h)
	w = configurationSaved(t, h, configurationIntent(w, 1, map[string]any{"prescribedSets": 3, "applyWeightToAllSets": "40"})).Workout
	// Fixture unequal targets; completion and recording use the real HTTP commands.
	a := w.Exercises[0]
	for i := 1; i < 3; i++ {
		if _, e = pool.Exec(ctx, `UPDATE public.variant_target_set SET target_weight=50 WHERE id=$1`, *a.Sets[i].TargetSetID); e != nil {
			t.Fatal(e)
		}
		if _, e = pool.Exec(ctx, `UPDATE public.workout_set SET target_weight=50 WHERE id=$1`, a.Sets[i].ID); e != nil {
			t.Fatal(e)
		}
	}
	for i, r := range []int{12, 11, 10} {
		w = completionRecord(t, h, w, 0, i, i+1, "performed", r)
	}
	w = completed(t, h, completionIntent(w, 0, 1)).Workout
	if w.Status != "active" {
		t.Fatal("fixture must still be active")
	}
	p := request(t, h, "GET", "/api/training/stats", "", "", nil)
	if p.Code != http.StatusOK {
		t.Fatalf("stats: %d %s", p.Code, p.Body.String())
	}
	var out struct {
		Tracks []struct {
			VariantID *string `json:"variantId"`
			Entries   []struct {
				ID                                 string
				SetCount                           *int
				AverageReps, AverageWeight, Volume *string
				Representation                     string
			}
		}
	}
	if e = json.Unmarshal(p.Body.Bytes(), &out); e != nil {
		t.Fatal(e)
	}
	found := false
	for _, track := range out.Tracks {
		if track.VariantID != nil && *track.VariantID == a.VariantID {
			found = true
			if len(track.Entries) != 1 {
				t.Fatalf("entries: %s", p.Body.String())
			}
			v := track.Entries[0]
			if v.ID != a.ID || v.SetCount == nil || *v.SetCount != 3 || !statsAverageIs(v.AverageReps, 11) || v.Volume == nil || *v.Volume != "1530.00" || v.Representation != "set_derived" {
				t.Fatalf("summary: %s", p.Body.String())
			}
		}
	}
	if !found {
		t.Fatal("missing variant track")
	}
}

func statsRead(t *testing.T, h http.Handler) []training.StatsTrack {
	t.Helper()
	p := request(t, h, "GET", "/api/training/stats?mail=other@example.com", "", "", nil)
	var out struct {
		Tracks []training.StatsTrack `json:"tracks"`
	}
	if e := json.Unmarshal(p.Body.Bytes(), &out); p.Code != 200 || e != nil {
		t.Fatalf("stats %d: %s", p.Code, p.Body.String())
	}
	return out.Tracks
}
func statsVariant(t *testing.T, h http.Handler, id string) training.StatsTrack {
	t.Helper()
	for _, track := range statsRead(t, h) {
		if track.VariantID != nil && *track.VariantID == id {
			return track
		}
	}
	t.Fatal("track absent")
	return training.StatsTrack{}
}
func TestStatsFractionalPerformedCountZeroAndSkipsExcludeDrafts(t *testing.T) {
	for _, allSkipped := range []bool{false, true} {
		t.Run(fmt.Sprint(allSkipped), func(t *testing.T) {
			pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
			if e := migrations.Apply(ctx, pool); e != nil {
				t.Fatal(e)
			}
			h, e := NewHandler(ctx, Options{Pool: pool, Origin: "https://app.example", Application: application.Config{DevelopmentMode: true}})
			if e != nil {
				t.Fatal(e)
			}
			w := configuredWorkout(t, h)
			w = configurationSaved(t, h, configurationIntent(w, 1, map[string]any{"prescribedSets": 3})).Workout
			for i, r := range []int{0, 1, 0} {
				state := "performed"
				if i == 2 || allSkipped {
					state = "skipped"
				}
				w = completionRecord(t, h, w, 0, i, i+1, state, r)
			}
			if len(statsVariant(t, h, w.Exercises[0].VariantID).Entries) != 0 {
				t.Fatal("draft leaked into Stats")
			}
			w = completed(t, h, completionIntent(w, 0, 1)).Workout
			got := statsVariant(t, h, w.Exercises[0].VariantID).Entries[0]
			if allSkipped {
				if *got.SetCount != 0 || got.AverageReps != nil || got.AverageWeight != nil || *got.Volume != "0.00" {
					t.Fatalf("skipped: %+v", got)
				}
			} else {
				if *got.SetCount != 2 || !statsAverageIs(got.AverageReps, 0.5) || !statsAverageIs(got.AverageWeight, 40.5) || *got.Volume != "40.50" {
					t.Fatalf("fractional: %+v", got)
				}
			}
			w = completionRecord(t, h, w, 1, 0, 4, "performed", 0)
			w = completed(t, h, completionIntent(w, 1, 2)).Workout
			zero := statsVariant(t, h, w.Exercises[1].VariantID).Entries[0]
			if *zero.SetCount != 1 || *zero.Volume != "0.00" || zero.AverageReps == nil || zero.AverageWeight == nil {
				t.Fatalf("actual zero was lost: %+v", zero)
			}
		})
	}
}

func TestStatsPreservesImportedAggregatesUnassignedHistoryAndEqualTimes(t *testing.T) {
	pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
	if e := migrations.Apply(ctx, pool); e != nil {
		t.Fatal(e)
	}
	h, e := NewHandler(ctx, Options{Pool: pool, Origin: "https://app.example", Application: application.Config{DevelopmentMode: true}})
	if e != nil {
		t.Fatal(e)
	}
	w := configuredWorkout(t, h)
	w = configurationSaved(t, h, configurationIntent(w, 1, map[string]any{"exerciseName": "Current press", "variantName": "Heavy now"})).Workout
	a := w.Exercises[0]
	// Imported and malformed historical input is fixture data. No raw import JSON is public.
	for i := 1; i <= 5; i++ {
		_, e = pool.Exec(ctx, `INSERT INTO public.legacy_history_entry(id,owner_mail,exercise_id,variant_id,source_ex_id,source_save_date,occurred_at,source_timezone,avg_rep,avg_weight,no_of_sets,source_data)
  VALUES($1,$2,$3,CASE WHEN $4=1 THEN $5::uuid ELSE NULL END,$4,'2020-01-02 03:04:05.123456',CASE WHEN $4=4 THEN NULL ELSE '2020-01-02T02:04:05.123456Z'::timestamptz END,'Europe/Stockholm',CASE WHEN $4=2 THEN -1 WHEN $4=3 THEN NULL ELSE 7 END,CASE WHEN $4=5 THEN 'NaN'::numeric WHEN $4=3 THEN 0 ELSE 42.25 END,CASE WHEN $4=2 THEN -2 ELSE 3 END,'{"private":"secret-source"}')`, fmt.Sprintf("a3600000-0000-4000-8000-%012d", i), application.DevelopmentEmail, a.Exercise.ID, i, a.VariantID)
		if e != nil {
			t.Fatal(e)
		}
	}
	tracks := statsRead(t, h)
	var unassigned training.StatsTrack
	for _, track := range tracks {
		if track.ExerciseID == a.Exercise.ID && track.VariantID == nil {
			unassigned = track
		}
	}
	attributed := statsVariant(t, h, a.VariantID)
	if len(attributed.Entries) != 1 || attributed.ExerciseName != "Current press" || *attributed.VariantName != "Heavy now" || *attributed.Entries[0].Volume != "887.25" || !attributed.Entries[0].VolumeEstimated {
		t.Fatalf("attributed: %+v", attributed)
	}
	if len(unassigned.Entries) != 4 {
		t.Fatalf("unassigned IDs collapsed: %+v", unassigned)
	}
	byID := map[string]training.StatsEntry{}
	for _, v := range unassigned.Entries {
		byID[v.ID] = v
		if v.Representation != "legacy_aggregate" || !v.VolumeEstimated || v.SourceTimestamp == nil || *v.SourceTimestamp != "2020-01-02T03:04:05.123456" || *v.SourceTimezone != "Europe/Stockholm" {
			t.Fatalf("provenance: %+v", v)
		}
	}
	nan := byID["a3600000-0000-4000-8000-000000000005"]
	if nan.Volume != nil {
		t.Fatalf("invalid NaN generated volume: %+v", nan)
	}
	invalid := byID["a3600000-0000-4000-8000-000000000002"]
	missing := byID["a3600000-0000-4000-8000-000000000003"]
	undated := byID["a3600000-0000-4000-8000-000000000004"]
	if *invalid.AverageReps != "-1" || *invalid.SetCount != -2 || invalid.Volume != nil || missing.AverageReps != nil || *missing.AverageWeight != "0.00" || missing.Volume != nil || undated.OccurredAt != nil {
		t.Fatal("original nullable/invalid metrics were changed or fabricated")
	}
	if !invalid.OccurredAt.Equal(*missing.OccurredAt) {
		t.Fatal("equal timestamps jittered")
	}
	raw := request(t, h, "GET", "/api/training/stats", "", "", nil).Body.String()
	if strings.Contains(raw, "secret-source") || strings.Contains(raw, "sourceSession") || strings.Contains(raw, "workoutId") {
		t.Fatal("private bookkeeping or fabricated history association exposed")
	}
	// Simulate archived, unlinked library history; Stats must not use selectable search.
	if _, e = pool.Exec(ctx, `DELETE FROM public.session_variant WHERE variant_id=$1`, a.VariantID); e != nil {
		t.Fatal(e)
	}
	if _, e = pool.Exec(ctx, `UPDATE public.exercise_variant SET archived_at=now() WHERE id=$1`, a.VariantID); e != nil {
		t.Fatal(e)
	}
	if _, e = pool.Exec(ctx, `UPDATE public.exercise SET archived_at=now() WHERE id=$1`, a.Exercise.ID); e != nil {
		t.Fatal(e)
	}
	tracks = statsRead(t, h)
	found := false
	for _, track := range tracks {
		if track.ExerciseID == a.Exercise.ID && track.VariantID == nil {
			found = len(track.Entries) == 4
		}
	}
	if !found || len(statsVariant(t, h, a.VariantID).Entries) != 1 {
		t.Fatal("archived/unlinked history vanished")
	}
	// Request query parameters cannot select another owner's data.
	if _, e = pool.Exec(ctx, `INSERT INTO public."User"(mail,name) VALUES('other@example.com','Other'); INSERT INTO public.exercise(id,owner_mail,name) VALUES('a3690000-0000-4000-8000-000000000001','other@example.com','Private lift'); INSERT INTO public.legacy_history_entry(id,owner_mail,exercise_id,source_ex_id,source_save_date,source_data) VALUES('a3690000-0000-4000-8000-000000000002','other@example.com','a3690000-0000-4000-8000-000000000001',99,'2020-01-01','{}')`); e != nil {
		t.Fatal(e)
	}
	for _, track := range statsRead(t, h) {
		if track.ExerciseName == "Private lift" {
			t.Fatal("cross-owner history leak")
		}
	}
}

func TestStatsSharesVariantAcrossSessionsWithCurrentNamesAfterRemoval(t *testing.T) {
	pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
	if e := migrations.Apply(ctx, pool); e != nil {
		t.Fatal(e)
	}
	h, e := NewHandler(ctx, Options{Pool: pool, Origin: "https://app.example", Application: application.Config{DevelopmentMode: true}})
	if e != nil {
		t.Fatal(e)
	}
	w := configuredWorkout(t, h)
	a := w.Exercises[0]
	p := request(t, h, "POST", "/api/training/sessions/create", `{"operationId":"a3610000-0000-4000-8000-000000000001","name":"Second plan","items":[{"kind":"reuseVariant","exerciseId":"`+a.Exercise.ID+`","variantId":"`+a.VariantID+`","expectedVariantRevision":"`+a.CurrentVariant.Revision+`"}]}`, "https://app.example", nil)
	var second training.SessionResult
	if e = json.Unmarshal(p.Body.Bytes(), &second); p.Code != 200 || e != nil {
		t.Fatal(p.Body.String())
	}
	w = completionRecord(t, h, w, 0, 0, 1, "performed", 1)
	w = completionRecord(t, h, w, 0, 1, 2, "performed", 2)
	w = completed(t, h, completionIntent(w, 0, 1)).Workout
	first := statsVariant(t, h, a.VariantID).Entries[0]
	w = completionRecord(t, h, w, 1, 0, 3, "skipped", 0)
	w = completed(t, h, completionIntent(w, 1, 2)).Workout
	p = request(t, h, "POST", "/api/training/workouts/start", `{"operationId":"a3610000-0000-4000-8000-000000000002","sessionId":"`+second.Session.ID+`","expectedSessionRevision":"1"}`, "https://app.example", nil)
	var started training.WorkoutResult
	if e = json.Unmarshal(p.Body.Bytes(), &started); p.Code != 200 || e != nil {
		t.Fatal(p.Body.String())
	}
	w = started.Workout
	w = configurationSaved(t, h, configurationIntent(w, 1, map[string]any{"exerciseName": "Renamed press", "variantName": "Renamed heavy", "applyWeightToAllSets": "0"})).Workout
	w = completionRecord(t, h, w, 0, 0, 4, "performed", 0)
	w = completionRecord(t, h, w, 0, 1, 5, "skipped", 0)
	w = completed(t, h, completionIntent(w, 0, 3)).Workout
	if _, e = pool.Exec(ctx, `UPDATE public.workout_exercise SET completed_at=$1 WHERE id=$2`, first.OccurredAt, w.Exercises[0].ID); e != nil {
		t.Fatal(e)
	}
	track := statsVariant(t, h, a.VariantID)
	if track.ExerciseName != "Renamed press" || *track.VariantName != "Renamed heavy" || len(track.Entries) != 2 {
		t.Fatalf("shared current track: %+v", track)
	}
	byID := map[string]training.StatsEntry{}
	for _, v := range track.Entries {
		byID[v.ID] = v
	}
	if *byID[first.ID].Volume != "121.50" || *byID[w.Exercises[0].ID].Volume != "0.00" || !track.Entries[0].OccurredAt.Equal(*track.Entries[1].OccurredAt) || track.Entries[0].ID == track.Entries[1].ID {
		t.Fatal("immutable actuals or stable equal-time identities lost")
	}
	// Deleting both saved sessions leaves the same immutable history track.
	p = request(t, h, "GET", "/api/training/sessions", "", "", nil)
	var plans struct{ Sessions []training.Session }
	if e = json.Unmarshal(p.Body.Bytes(), &plans); e != nil {
		t.Fatal(e)
	}
	for i, plan := range plans.Sessions {
		p = request(t, h, "POST", "/api/training/sessions/"+plan.ID+"/delete", fmt.Sprintf(`{"operationId":"a3610000-0000-4000-8000-%012d","expectedRevision":"%s"}`, i+10, plan.Revision), "https://app.example", nil)
		if p.Code != 200 {
			t.Fatal(p.Body.String())
		}
	}
	if len(statsVariant(t, h, a.VariantID).Entries) != 2 {
		t.Fatal("session deletion removed history")
	}
}

// Compare average values independently of PostgreSQL's decimal output scale.
func statsAverageIs(value *string, want float64) bool {
	if value == nil {
		return false
	}
	number, err := strconv.ParseFloat(*value, 64)
	return err == nil && number == want
}
