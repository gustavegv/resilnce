-- Add notes without replacing existing tables or inferring legacy content.
ALTER TABLE public.exercise_variant
 ADD COLUMN notes text,
 ADD CONSTRAINT exercise_variant_notes_length CHECK (char_length(notes) <= 4000);

ALTER TABLE public.workout
 ADD COLUMN notes text,
 ADD CONSTRAINT workout_notes_length CHECK (char_length(notes) <= 4000);

ALTER TABLE public.workout_exercise
 ADD COLUMN notes text,
 ADD CONSTRAINT workout_exercise_notes_length CHECK (char_length(notes) <= 4000);
