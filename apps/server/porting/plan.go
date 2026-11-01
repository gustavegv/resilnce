package porting

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"
	_ "time/tzdata"
	"unicode/utf8"
)

const DefaultSourceTimezone = "Europe/Stockholm"

type Options struct {
	SourceTimezone        string `json:"sourceTimezone,omitempty"`
	DeleteUnusedExercises bool   `json:"deleteUnusedExercises,omitempty"`
	DeleteUnusedHistory   bool   `json:"deleteUnusedHistory,omitempty"`
}

func (o Options) normalized() Options {
	o.SourceTimezone = strings.TrimSpace(o.SourceTimezone)
	if o.SourceTimezone == "" {
		o.SourceTimezone = DefaultSourceTimezone
	}
	return o
}

func (o Options) Validate() error {
	if o.DeleteUnusedHistory && !o.DeleteUnusedExercises {
		return errors.New("deleting unused history also requires deleting unused exercises")
	}
	_, err := time.LoadLocation(o.normalized().SourceTimezone)
	return err
}

type References struct {
	Exercise      string   `json:"exercise,omitempty"`
	Variant       string   `json:"variant,omitempty"`
	Session       string   `json:"session,omitempty"`
	Placement     string   `json:"placement,omitempty"`
	Targets       []string `json:"targets,omitempty"`
	LegacyHistory string   `json:"legacyHistory,omitempty"`
	LegacySet     string   `json:"legacySet,omitempty"`
	Account       string   `json:"account,omitempty"`
	DeletedUnused bool     `json:"deletedUnused,omitempty"`
}
type Group struct {
	Name     string `json:"normalizedName"`
	Exercise string `json:"exercise"`
	Next     int    `json:"nextVariantNumber"`
}
type State struct {
	Mapped       map[string]References
	Groups       map[string]Group
	Exercises    map[string]string // id -> current normalized name
	Variants     map[string]string // id -> parent id
	Sessions     map[string]bool
	VariantNames map[string]map[string]bool
	LastPosition map[string]int
}

func emptyState() State {
	return State{map[string]References{}, map[string]Group{}, map[string]string{}, map[string]string{}, map[string]bool{}, map[string]map[string]bool{}, map[string]int{}}
}

type Issue struct {
	Table    string          `json:"table"`
	Key      json.RawMessage `json:"key"`
	Code     string          `json:"code"`
	Severity string          `json:"severity"`
	Details  string          `json:"details"`
	Source   json.RawMessage `json:"source"`
}
type Assignment struct {
	SourceKey   json.RawMessage `json:"sourceKey"`
	Group       string          `json:"group"`
	Name        string          `json:"name"`
	Position    int             `json:"position"`
	Reps        []int           `json:"reps"`
	Weights     []string        `json:"weights"`
	RestSeconds int             `json:"restSeconds"`
}
type Count struct {
	Source        int `json:"source"`
	Mapped        int `json:"mapped"`
	Pending       int `json:"pending"`
	Imported      int `json:"imported"`
	Retained      int `json:"retained"`
	DeletePlanned int `json:"deletePlanned"`
	Deleted       int `json:"deleted"`
}
type OwnerReport struct {
	Owner       string            `json:"owner"`
	Ready       bool              `json:"ready"`
	Counts      map[string]*Count `json:"counts"`
	Issues      []Issue           `json:"issues"`
	Assignments []Assignment      `json:"assignments"`
	Created     int               `json:"destinationObjectsCreated"`
	Error       string            `json:"error,omitempty"`
}
type plan struct {
	OwnerReport
	Rows             []Row
	AssignmentsByKey map[string]Assignment
	Groups           map[string][]Row
	DeleteUnused     map[string]bool
}

// decimal accepts exact cents only; it never rounds source floating point data.
func decimal(n *json.Number, min, max string) (string, bool) {
	if n == nil {
		return "", false
	}
	v, ok := new(big.Rat).SetString(n.String())
	if !ok {
		return "", false
	}
	lo, _ := new(big.Rat).SetString(min)
	hi, _ := new(big.Rat).SetString(max)
	if v.Cmp(lo) < 0 || v.Cmp(hi) > 0 {
		return "", false
	}
	cents := new(big.Rat).Mul(v, big.NewRat(100, 1))
	if !cents.IsInt() {
		return "", false
	}
	return v.FloatString(2), true
}
func validName(s *string) bool {
	return s != nil && strings.Trim(*s, " ") != "" && utf8.RuneCountInString(strings.Trim(*s, " ")) <= 100
}
func number(n *int, min, max int) bool { return n != nil && *n >= min && *n <= max }

func restSeconds(n *int) int {
	if n == nil {
		return 0
	}
	return *n
}

func buildPlan(all []Row, owner string, state State, opts Options) plan {
	opts = opts.normalized()
	p := plan{OwnerReport: OwnerReport{Owner: owner, Ready: true, Counts: map[string]*Count{}, Issues: []Issue{}, Assignments: []Assignment{}}, AssignmentsByKey: map[string]Assignment{}, Groups: map[string][]Row{}, DeleteUnused: map[string]bool{}}
	users := map[string]bool{}
	exs := map[int]Row{}
	sessions := map[int]Row{}
	placements := map[int][]Row{}
	performance := map[int]bool{}
	history := map[int][]Row{}
	sets := map[int]bool{}
	for _, r := range all {
		switch r.Table {
		case "User":
			users[r.Owner] = true
		case "Exercise":
			exs[r.Data.ExID] = r
		case "Session":
			sessions[r.Data.SesID] = r
		case "SesExercise":
			placements[r.Data.ExID] = append(placements[r.Data.ExID], r)
		case "History":
			history[r.Data.ExID] = append(history[r.Data.ExID], r)
			performance[r.Data.ExID] = true
		case "Set":
			sets[r.Data.ExID] = true
			performance[r.Data.ExID] = true
		}
	}
	// Decide exercise and history cleanup together before validating individual
	// rows. History selected for deletion needs no invented target configuration.
	if opts.DeleteUnusedExercises {
		for id, ex := range exs {
			if ex.Owner != owner || len(placements[id]) != 0 || sets[id] {
				continue
			}
			if _, mapped := state.Mapped[ex.identity()]; mapped {
				continue
			}
			if len(history[id]) > 0 && !opts.DeleteUnusedHistory {
				continue
			}
			mappedHistory := false
			for _, r := range history[id] {
				if _, mapped := state.Mapped[r.identity()]; mapped {
					mappedHistory = true
					break
				}
			}
			if mappedHistory {
				continue
			}
			p.DeleteUnused[ex.identity()] = true
			for _, r := range history[id] {
				p.DeleteUnused[r.identity()] = true
			}
		}
	}
	add := func(r Row, code, severity, details string) {
		p.Issues = append(p.Issues, Issue{r.Table, r.Key, code, severity, details, r.Raw})
		if severity == "error" {
			p.Ready = false
		}
	}
	for _, table := range []string{"User", "Exercise", "Session", "SesExercise", "History", "Set"} {
		p.Counts[table] = &Count{}
	}
	for _, r := range all {
		// Both owners of a cross-account link must remain incomplete.
		if (r.Table == "SesExercise" || r.Table == "Set") && r.Owner != owner && sessions[r.Data.SesID].Owner == owner {
			add(r, "cross_owner_reference", "error", "Session contains a source record belonging to a different owner.")
		}
		if r.Owner != owner {
			continue
		}
		p.Rows = append(p.Rows, r)
		c := p.Counts[r.Table]
		c.Source++
		if r.Table == "User" && r.Data.ActiveSession != nil && *r.Data.ActiveSession != "" {
			add(r, "active_legacy_workout", "error", "Finish or explicitly resolve the active legacy workout before importing this account.")
		}
		if refs, ok := state.Mapped[r.identity()]; ok {
			c.Mapped++
			if (r.Table == "History" || r.Table == "Set") && !refs.DeletedUnused {
				c.Retained++
			}
			continue
		}
		c.Pending++
		if owner == "" || !users[owner] {
			add(r, "missing_owner", "error", "Source row has no existing account owner.")
		}
		if p.DeleteUnused[r.identity()] {
			c.DeletePlanned++
			if r.Table == "Exercise" {
				add(r, "delete_unused_exercise", "warning", fmt.Sprintf("Selected for deletion during import: exercise has no session placements. %d legacy history entries will also be deleted.", len(history[r.Data.ExID])))
			} else {
				add(r, "delete_unused_history", "warning", "Selected for deletion with its unused exercise during import.")
			}
			continue
		}
		if r.DecodeError != "" {
			add(r, "unsupported_source_values", "error", r.DecodeError)
		}
		switch r.Table {
		case "Exercise":
			if !validName(r.Data.Name) {
				add(r, "invalid_name", "error", "Exercise name is missing, empty, or exceeds 100 characters.")
				continue
			}
			if len(placements[r.Data.ExID]) == 0 {
				details := "Exercise has no placement from which to preserve its configuration."
				if opts.DeleteUnusedExercises && performance[r.Data.ExID] {
					if !opts.DeleteUnusedHistory {
						details += " Enable 'Also delete their history' to include its legacy history in cleanup."
					} else {
						details += " Existing imported history or retained Set records prevent automatic cleanup."
					}
				}
				add(r, "requires_setup", "error", details)
			}
			p.Groups[r.Data.NormalizedName] = append(p.Groups[r.Data.NormalizedName], r)
		case "Session":
			if r.Data.SessionLastRan != nil {
				if _, err := occurrence(*r.Data.SessionLastRan, opts.SourceTimezone); err != nil {
					add(r, "ambiguous_timestamp", "error", err.Error())
				}
			}
			if !validName(r.Data.SessionName) {
				add(r, "invalid_name", "error", "Session name is missing, empty, or exceeds 100 characters.")
			}
		case "SesExercise":
			ex, eok := exs[r.Data.ExID]
			se, sok := sessions[r.Data.SesID]
			if !eok || !sok {
				add(r, "missing_reference", "error", "Placement has a missing exercise or session.")
				continue
			}
			if ex.Owner != owner || se.Owner != owner {
				add(r, "cross_owner_reference", "error", "Placement exercise and session have different owners.")
			}
			if !number(r.Data.SetCount, 1, 20) || len(r.Data.Reps) != value(r.Data.SetCount) || len(r.Data.Weights) != value(r.Data.SetCount) {
				add(r, "invalid_arrays", "error", "Set count must be 1-20 and match both non-null target arrays.")
				continue
			}
			if !number(r.Data.Threshold, 1, 99) {
				add(r, "invalid_threshold", "error", "Repetition threshold must be 1-99.")
			}
			if r.Data.Rest == nil {
				add(r, "defaulted_rest", "warning", "Missing rest duration defaults to 0 seconds.")
			} else if !number(r.Data.Rest, 0, 32767) {
				add(r, "invalid_rest", "error", "Rest duration must be nonnegative.")
			}
			if _, ok := decimal(ex.Data.Increment, "0.25", "99"); !ok {
				add(r, "invalid_increment", "error", "Increment must be exact cents in 0.25-99; null/zero increments require an explicit policy.")
			}
			a := Assignment{SourceKey: r.Key, Group: ex.Data.NormalizedName, RestSeconds: restSeconds(r.Data.Rest)}
			bad := false
			for i := range r.Data.Reps {
				if !number(r.Data.Reps[i], 0, 32767) {
					bad = true
				} else {
					a.Reps = append(a.Reps, *r.Data.Reps[i])
				}
				weight, ok := decimal(r.Data.Weights[i], "0", "9999")
				if !ok {
					bad = true
				}
				a.Weights = append(a.Weights, weight)
			}
			if bad {
				add(r, "invalid_targets", "error", "Targets contain null/negative values, out-of-range weights, or weights requiring rounding.")
			}
			p.AssignmentsByKey[r.identity()] = a
		case "History", "Set":
			ex, ok := exs[r.Data.ExID]
			if !ok {
				add(r, "missing_reference", "error", "Legacy performance has no source exercise.")
			} else if ex.Owner != owner {
				add(r, "cross_owner_reference", "error", "Legacy performance has a different owner.")
			}
			if r.Table == "History" {
				if _, err := occurrence(r.Data.SaveDate, opts.SourceTimezone); err != nil {
					add(r, "ambiguous_timestamp", "error", err.Error())
				}
				if len(placements[r.Data.ExID]) != 1 {
					add(r, "unassigned_legacy_history", "warning", "Aggregate is retained at exercise level; its variant cannot be established.")
				}
			} else {
				se, ok := sessions[r.Data.SesID]
				if !ok || se.Owner != owner {
					add(r, "cross_owner_reference", "error", "Legacy Set row has a missing or differently owned session.")
				}
				add(r, "undated_legacy_set", "warning", "Retain the original Set row without inventing a workout or date.")
			}
		}
	}
	// Detect tombstones for pending dependents, even when the source parent is mapped.
	for _, r := range p.Rows {
		if p.DeleteUnused[r.identity()] {
			continue
		}
		if _, done := state.Mapped[r.identity()]; done {
			continue
		}
		if r.Table == "SesExercise" || r.Table == "History" || r.Table == "Set" {
			ex := exs[r.Data.ExID]
			if refs, ok := state.Mapped[ex.identity()]; ok {
				if _, alive := state.Exercises[refs.Exercise]; !alive {
					add(r, "deleted_destination", "error", "Mapped exercise was deleted; importing this dependent must not resurrect it.")
				}
			}
		}
		if r.Table == "SesExercise" {
			if refs, ok := state.Mapped[sessions[r.Data.SesID].identity()]; ok && !state.Sessions[refs.Session] {
				add(r, "deleted_destination", "error", "Mapped session was deleted.")
			}
		}
		if r.Table == "History" || r.Table == "Set" {
			for _, placement := range placements[r.Data.ExID] {
				if r.Table == "Set" && placement.Data.SesID != r.Data.SesID {
					continue
				}
				if r.Table == "History" && len(placements[r.Data.ExID]) != 1 {
					continue
				}
				if refs, done := state.Mapped[placement.identity()]; done {
					if parent, alive := state.Variants[refs.Variant]; !alive || parent != state.Mapped[exs[r.Data.ExID].identity()].Exercise {
						add(r, "deleted_destination", "error", "Mapped variant was deleted or moved; resolve its mapping redirect before importing dependents.")
					}
				}
			}
		}
	}
	for name, group := range p.Groups {
		if existing, ok := state.Groups[name]; ok {
			if _, alive := state.Exercises[existing.Exercise]; !alive {
				add(group[0], "deleted_destination", "error", "Source group points to a deleted exercise.")
			}
		} else {
			for _, current := range state.Exercises {
				if current == name {
					add(group[0], "destination_name_collision", "error", "An unrelated destination exercise already reserves this name.")
					break
				}
			}
		}
		if len(group) > 1 {
			add(group[0], "grouped_duplicate_names", "warning", fmt.Sprintf("Automatically group %d legacy exercise names under one exercise.", len(group)))
		}
	}
	// Include mapped source exercises so late placements use durable group identity.
	groupPlacements := map[string][]Row{}
	bySession := map[int][]Row{}
	for _, r := range p.Rows {
		if r.Table == "SesExercise" {
			groupPlacements[exs[r.Data.ExID].Data.NormalizedName] = append(groupPlacements[exs[r.Data.ExID].Data.NormalizedName], r)
			bySession[r.Data.SesID] = append(bySession[r.Data.SesID], r)
		}
	}
	for name, rows := range groupPlacements {
		sort.Slice(rows, func(i, j int) bool {
			if rows[i].Data.SesID != rows[j].Data.SesID {
				return rows[i].Data.SesID < rows[j].Data.SesID
			}
			return rows[i].Data.ExID < rows[j].Data.ExID
		})
		g, existing := state.Groups[name]
		next := g.Next
		if next < 1 {
			next = 1
		}
		used := map[string]bool{}
		for n := range state.VariantNames[g.Exercise] {
			used[n] = true
		}
		for _, r := range rows {
			if _, done := state.Mapped[r.identity()]; done {
				continue
			}
			a, ok := p.AssignmentsByKey[r.identity()]
			if !ok {
				continue
			}
			if !existing && len(rows) == 1 && !used["standard"] {
				a.Name = "Standard"
			} else {
				for {
					a.Name = fmt.Sprintf("Variant %d", next)
					next++
					if !used[strings.ToLower(a.Name)] {
						break
					}
				}
			}
			used[strings.ToLower(a.Name)] = true
			p.AssignmentsByKey[r.identity()] = a
		}
	}
	for _, rows := range bySession {
		sort.Slice(rows, func(i, j int) bool {
			a, b := rows[i].Data.Order, rows[j].Data.Order
			if a == nil && b != nil {
				return false
			}
			if a != nil && b == nil {
				return true
			}
			if a != nil && b != nil && *a != *b {
				return *a < *b
			}
			return rows[i].Data.ExID < rows[j].Data.ExID
		})
		position := state.LastPosition[state.Mapped[sessions[rows[0].Data.SesID].identity()].Session]
		seen := map[int]bool{}
		for _, r := range rows {
			if r.Data.Order == nil || seen[value(r.Data.Order)] {
				if _, done := state.Mapped[r.identity()]; !done {
					add(r, "reordered_placement", "warning", "Null/tied source order resolved by source exercise ID; positions are sequential.")
				}
			}
			if r.Data.Order != nil {
				seen[*r.Data.Order] = true
			}
			if _, done := state.Mapped[r.identity()]; done {
				continue
			}
			position++
			a, ok := p.AssignmentsByKey[r.identity()]
			if !ok {
				continue
			}
			a.Position = position
			p.AssignmentsByKey[r.identity()] = a
		}
	}
	for _, r := range p.Rows {
		if a, ok := p.AssignmentsByKey[r.identity()]; ok {
			p.Assignments = append(p.Assignments, a)
		}
	}
	sort.Slice(p.Issues, func(i, j int) bool {
		a, b := p.Issues[i], p.Issues[j]
		if a.Table != b.Table {
			return a.Table < b.Table
		}
		if string(a.Key) != string(b.Key) {
			return string(a.Key) < string(b.Key)
		}
		return a.Code < b.Code
	})
	return p
}

func value(n *int) int {
	if n == nil {
		return -1
	}
	return *n
}

// Resolve a naive timestamp only when its local wall-clock time has one instant.
// Reject DST gaps and folds rather than silently adopting time.Date's choice.
func occurrence(raw, zone string) (time.Time, error) {
	wall, err := time.Parse("2006-01-02T15:04:05.999999999", raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("Unsupported source timestamp %q; retain it pending a timestamp policy.", raw)
	}
	loc, err := time.LoadLocation(zone)
	if err != nil {
		return time.Time{}, err
	}
	guess := time.Date(wall.Year(), wall.Month(), wall.Day(), wall.Hour(), wall.Minute(), wall.Second(), wall.Nanosecond(), loc)
	offsets := map[int]bool{}
	for h := -48; h <= 48; h++ {
		_, off := guess.Add(time.Duration(h) * time.Hour).Zone()
		offsets[off] = true
	}
	var matches []time.Time
	for off := range offsets {
		candidate := wall.Add(-time.Duration(off) * time.Second)
		if candidate.In(loc).Format("2006-01-02T15:04:05.999999999") == raw || candidate.In(loc).Format("2006-01-02T15:04:05.999999999") == wall.Format("2006-01-02T15:04:05.999999999") {
			matches = append(matches, candidate)
		}
	}
	if len(matches) != 1 {
		return time.Time{}, fmt.Errorf("Source timestamp %s is ambiguous or nonexistent in %s.", raw, zone)
	}
	return matches[0], nil
}
