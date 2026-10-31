-- Optional exercise-level effort ratings. Existing and imported data stay unrated.
ALTER TABLE public.workout_exercise
 ADD COLUMN recorded_rpe smallint,
 ADD CONSTRAINT workout_exercise_rpe_range CHECK (recorded_rpe BETWEEN 0 AND 10);

-- Maintained by the future completion transaction from the latest rated attempt.
ALTER TABLE public.exercise_variant
 ADD COLUMN last_rpe smallint,
 ADD COLUMN last_rpe_at timestamptz,
 ADD CONSTRAINT exercise_variant_rpe_range CHECK (last_rpe BETWEEN 0 AND 10),
 ADD CONSTRAINT exercise_variant_rpe_timestamp CHECK ((last_rpe IS NULL) = (last_rpe_at IS NULL));
