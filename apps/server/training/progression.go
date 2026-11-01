package training

// NextTargets computes average-mode progression from explicitly resolved actual
// prescribed work and CURRENT targets. It never edits either input. Arithmetic
// uses integer reps/cents; comparing sums preserves the unrounded average.
func NextTargets(current Variant, sets []WorkoutSet) ([]TargetSet, error) {
	if current.ProgressionMode != "average" {
		return nil, invalid("progressionMode", "This completion requires average progression")
	}
	if current.RepThreshold < 1 || current.RepThreshold > 99 || len(current.TargetSets) < 1 || len(current.TargetSets) > 20 || current.PrescribedSets != len(current.TargetSets) {
		return nil, invalid("configuration", "Review the current prescribed sets and rep threshold before completing")
	}
	increment, err := parseCents(current.AutoIncrement, 25, 9900)
	if err != nil {
		return nil, invalid("autoIncrement", err.Error())
	}
	actual := map[string]WorkoutSet{}
	var sum int64
	eligible := true
	for _, set := range sets {
		if set.Kind != "prescribed" {
			continue
		}
		if set.TargetSetID == nil {
			return nil, conflict("stale-revision", "Prescribed targets changed. Review current configuration before completing")
		}
		if _, exists := actual[*set.TargetSetID]; exists {
			return nil, invalid("sets", "Each prescribed set must have one current target")
		}
		switch set.State {
		case "performed":
			if set.RecordedReps == nil || *set.RecordedReps < 0 || int64(*set.RecordedReps) > 2147483647 {
				return nil, invalid("recordedReps", "Record explicit whole reps before completing")
			}
			sum += int64(*set.RecordedReps)
		case "skipped":
			eligible = false
		default:
			return nil, invalid("sets", "Resolve every pending prescribed set as performed or skipped before finishing")
		}
		if set.LateWeightOverride {
			eligible = false
		}
		actual[*set.TargetSetID] = set
	}
	if len(actual) != len(current.TargetSets) {
		return nil, conflict("stale-revision", "Prescribed sets changed. Review current configuration before completing")
	}
	eligible = eligible && sum >= int64(current.RepThreshold)*int64(len(current.TargetSets))
	// floor(threshold / 1.7 + .5), exactly, for positive integer thresholds.
	reset := (current.RepThreshold*20 + 17) / 34
	next := append([]TargetSet(nil), current.TargetSets...)
	seen := map[string]bool{}
	for i, target := range next {
		set, ok := actual[target.ID]
		if !ok || seen[target.ID] {
			return nil, conflict("stale-revision", "Prescribed target identities changed. Review current configuration")
		}
		seen[target.ID] = true
		weight, e := parseCents(target.TargetWeight, 0, 999900)
		if e != nil {
			return nil, invalid("targetWeight", e.Error())
		}
		if eligible {
			weight += increment
			if weight > 999900 {
				return nil, invalid("targetWeight", "Automatic increase would exceed 9999 kg. Adjust the weight or increment, then finish again")
			}
			next[i].TargetReps = reset
		} else if set.State == "skipped" {
			next[i].TargetReps = reset
		} else {
			next[i].TargetReps = *set.RecordedReps
		}
		next[i].TargetWeight = formatCents(weight)
	}
	return next, nil
}
