-- Capture rest with each attempt's progression settings. Existing attempts use
-- the current configuration because no prior rest snapshot exists.
ALTER TABLE public.workout_exercise ADD COLUMN rest_seconds integer NOT NULL DEFAULT 0;
UPDATE public.workout_exercise a SET rest_seconds=v.rest_seconds
FROM public.exercise_variant v WHERE (v.owner_mail,v.id)=(a.owner_mail,a.variant_id);
ALTER TABLE public.workout_exercise ADD CONSTRAINT workout_exercise_rest_nonnegative CHECK (rest_seconds >= 0);
