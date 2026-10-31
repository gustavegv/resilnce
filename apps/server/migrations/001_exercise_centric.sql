-- Additive only: the quoted legacy tables and their data remain untouched.
CREATE TABLE public.exercise (
 id uuid PRIMARY KEY,
 owner_mail varchar(255) NOT NULL REFERENCES public."User"(mail),
 name varchar(100) NOT NULL CHECK (name = btrim(name) AND length(name) > 0),
 muscle_group_codes text[] NOT NULL DEFAULT '{}' CHECK (array_position(muscle_group_codes, NULL) IS NULL),
 archived_at timestamptz,
 revision bigint NOT NULL DEFAULT 1 CHECK (revision > 0),
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(owner_mail,id)
);
CREATE UNIQUE INDEX exercise_owner_name ON public.exercise(owner_mail,lower(btrim(name)));

CREATE TABLE public.exercise_variant (
 id uuid PRIMARY KEY, owner_mail varchar(255) NOT NULL,
 exercise_id uuid NOT NULL,
 name varchar(100) NOT NULL CHECK (name = btrim(name) AND length(name) > 0),
 progression_mode text NOT NULL CHECK (progression_mode IN ('average','all_sets','off')),
 rep_threshold integer NOT NULL CHECK (rep_threshold BETWEEN 1 AND 99),
 auto_increment numeric(8,2) NOT NULL CHECK (auto_increment BETWEEN 0.25 AND 99),
 rest_seconds integer NOT NULL CHECK (rest_seconds >= 0),
 archived_at timestamptz, revision bigint NOT NULL DEFAULT 1 CHECK (revision > 0),
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(owner_mail,id),
 FOREIGN KEY(owner_mail,exercise_id) REFERENCES public.exercise(owner_mail,id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX variant_exercise_name ON public.exercise_variant(exercise_id,lower(btrim(name)));
CREATE INDEX variant_owner_exercise ON public.exercise_variant(owner_mail,exercise_id);

CREATE TABLE public.variant_target_set (
 id uuid PRIMARY KEY, owner_mail varchar(255) NOT NULL, variant_id uuid NOT NULL,
 position integer NOT NULL CHECK (position BETWEEN 1 AND 20),
 target_reps integer NOT NULL CHECK (target_reps >= 0),
 target_weight numeric(8,2) NOT NULL CHECK (target_weight BETWEEN 0 AND 9999),
 UNIQUE(owner_mail,id), UNIQUE(variant_id,position) DEFERRABLE INITIALLY IMMEDIATE,
 FOREIGN KEY(owner_mail,variant_id) REFERENCES public.exercise_variant(owner_mail,id) ON DELETE CASCADE
);
CREATE INDEX target_owner_variant ON public.variant_target_set(owner_mail,variant_id);

-- A deferred invariant permits replacing a configuration within one transaction.
CREATE FUNCTION internal.check_variant_targets() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE vid uuid; n integer;
BEGIN
 IF TG_TABLE_NAME = 'exercise_variant' THEN vid := NEW.id;
 ELSIF TG_OP = 'DELETE' THEN vid := OLD.variant_id;
 ELSE vid := NEW.variant_id; END IF;
 IF EXISTS (SELECT 1 FROM public.exercise_variant WHERE id=vid) THEN
  SELECT count(*) INTO n FROM public.variant_target_set WHERE variant_id=vid;
  IF n < 1 OR n > 20 THEN RAISE EXCEPTION 'variant requires 1-20 target sets' USING ERRCODE='23514'; END IF;
 END IF;
 IF TG_TABLE_NAME = 'variant_target_set' THEN
  IF TG_OP = 'UPDATE' THEN
   IF OLD.variant_id <> NEW.variant_id
   AND EXISTS (SELECT 1 FROM public.exercise_variant WHERE id=OLD.variant_id)
   AND NOT EXISTS (SELECT 1 FROM public.variant_target_set WHERE variant_id=OLD.variant_id)
   THEN RAISE EXCEPTION 'variant requires target sets' USING ERRCODE='23514'; END IF;
  END IF;
 END IF;
 RETURN NULL;
END $$;
CREATE CONSTRAINT TRIGGER variant_requires_targets AFTER INSERT OR UPDATE ON public.exercise_variant
 DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION internal.check_variant_targets();
CREATE CONSTRAINT TRIGGER target_count AFTER INSERT OR UPDATE OR DELETE ON public.variant_target_set
 DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION internal.check_variant_targets();

CREATE TABLE public.training_session (
 id uuid PRIMARY KEY, owner_mail varchar(255) NOT NULL REFERENCES public."User"(mail),
 name varchar(100) NOT NULL CHECK (name=btrim(name) AND length(name)>0),
 revision bigint NOT NULL DEFAULT 1 CHECK (revision>0),
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(owner_mail,id)
);
CREATE TABLE public.session_variant (
 id uuid PRIMARY KEY, owner_mail varchar(255) NOT NULL, session_id uuid NOT NULL, variant_id uuid NOT NULL,
 position integer NOT NULL CHECK (position>0),
 UNIQUE(session_id,variant_id), UNIQUE(session_id,position) DEFERRABLE INITIALLY IMMEDIATE,
 FOREIGN KEY(owner_mail,session_id) REFERENCES public.training_session(owner_mail,id) ON DELETE CASCADE,
 FOREIGN KEY(owner_mail,variant_id) REFERENCES public.exercise_variant(owner_mail,id) ON DELETE RESTRICT
);
CREATE INDEX session_variant_owner_variant ON public.session_variant(owner_mail,variant_id);

CREATE TABLE public.workout (
 id uuid PRIMARY KEY, owner_mail varchar(255) NOT NULL REFERENCES public."User"(mail),
 source_session_id uuid, origin text NOT NULL CHECK (origin IN ('session','free')),
 status text NOT NULL CHECK (status IN ('active','ended')),
 revision bigint NOT NULL DEFAULT 1 CHECK (revision>0),
 started_at timestamptz NOT NULL DEFAULT now(), ended_at timestamptz,
 CHECK ((status='active' AND ended_at IS NULL) OR (status='ended' AND ended_at IS NOT NULL AND ended_at>=started_at)),
 UNIQUE(owner_mail,id),
 FOREIGN KEY(owner_mail,source_session_id) REFERENCES public.training_session(owner_mail,id) ON DELETE SET NULL (source_session_id)
);
CREATE UNIQUE INDEX single_active_workout ON public.workout(owner_mail) WHERE status='active';
CREATE INDEX workout_source_session ON public.workout(owner_mail,source_session_id);
CREATE TABLE public.workout_exercise (
 id uuid PRIMARY KEY, owner_mail varchar(255) NOT NULL, workout_id uuid NOT NULL, variant_id uuid NOT NULL,
 position integer NOT NULL CHECK (position>0), state text NOT NULL CHECK (state IN ('draft','completed')),
 origin text NOT NULL CHECK (origin IN ('planned','added')),
 progression_mode text NOT NULL CHECK (progression_mode IN ('average','all_sets','off')),
 rep_threshold integer NOT NULL CHECK (rep_threshold BETWEEN 1 AND 99),
 auto_increment numeric(8,2) NOT NULL CHECK (auto_increment BETWEEN 0.25 AND 99),
 config_revision bigint NOT NULL CHECK (config_revision>0), completed_at timestamptz,
 CHECK ((state='draft' AND completed_at IS NULL) OR (state='completed' AND completed_at IS NOT NULL)),
 UNIQUE(owner_mail,id), UNIQUE(workout_id,position) DEFERRABLE INITIALLY IMMEDIATE,
 FOREIGN KEY(owner_mail,workout_id) REFERENCES public.workout(owner_mail,id) ON DELETE CASCADE,
 FOREIGN KEY(owner_mail,variant_id) REFERENCES public.exercise_variant(owner_mail,id) ON DELETE CASCADE
);
CREATE INDEX attempt_variant_history ON public.workout_exercise(owner_mail,variant_id,completed_at);
CREATE TABLE public.workout_set (
 id uuid PRIMARY KEY, owner_mail varchar(255) NOT NULL, workout_exercise_id uuid NOT NULL,
 target_set_id uuid, position integer NOT NULL CHECK (position>0),
 kind text NOT NULL CHECK (kind IN ('prescribed','bonus')),
 state text NOT NULL CHECK (state IN ('pending','performed','skipped')),
 target_reps integer NOT NULL CHECK (target_reps>=0), target_weight numeric(8,2) NOT NULL CHECK (target_weight BETWEEN 0 AND 9999),
 recorded_reps integer CHECK (recorded_reps>=0), recorded_weight numeric(8,2) CHECK (recorded_weight BETWEEN 0 AND 9999),
 recorded_at timestamptz, late_weight_override boolean NOT NULL DEFAULT false,
 CHECK ((state='performed' AND recorded_reps IS NOT NULL AND recorded_weight IS NOT NULL AND recorded_at IS NOT NULL)
 OR (state IN ('pending','skipped') AND recorded_reps IS NULL AND recorded_weight IS NULL AND recorded_at IS NULL)),
 UNIQUE(workout_exercise_id,position) DEFERRABLE INITIALLY IMMEDIATE,
 FOREIGN KEY(owner_mail,workout_exercise_id) REFERENCES public.workout_exercise(owner_mail,id) ON DELETE CASCADE,
 FOREIGN KEY(owner_mail,target_set_id) REFERENCES public.variant_target_set(owner_mail,id) ON DELETE SET NULL (target_set_id)
);
CREATE INDEX workout_set_target ON public.workout_set(owner_mail,target_set_id);
CREATE TABLE public.mutation_receipt (
 owner_mail varchar(255) NOT NULL REFERENCES public."User"(mail), operation_id uuid NOT NULL,
 kind text NOT NULL, target_id uuid NOT NULL, payload_fingerprint text NOT NULL,
 accepted_revision bigint NOT NULL, committed_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(owner_mail,operation_id)
);

-- Aggregate history and undated Set rows never masquerade as new workouts.
CREATE TABLE public.legacy_history_entry (
 id uuid PRIMARY KEY, owner_mail varchar(255) NOT NULL, exercise_id uuid NOT NULL, variant_id uuid,
 source_ex_id integer NOT NULL, source_save_date timestamp without time zone NOT NULL,
 occurred_at timestamptz, source_timezone text,
 avg_rep integer, avg_weight numeric(6,2), no_of_sets integer,
 representation text NOT NULL DEFAULT 'legacy_aggregate' CHECK (representation='legacy_aggregate'),
 source_data jsonb NOT NULL, imported_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(source_ex_id,source_save_date),
 FOREIGN KEY(owner_mail,exercise_id) REFERENCES public.exercise(owner_mail,id) ON DELETE CASCADE,
 FOREIGN KEY(owner_mail,variant_id) REFERENCES public.exercise_variant(owner_mail,id) ON DELETE CASCADE
);
CREATE INDEX legacy_history_exercise ON public.legacy_history_entry(owner_mail,exercise_id,source_save_date);
CREATE INDEX legacy_history_variant ON public.legacy_history_entry(owner_mail,variant_id);
CREATE TABLE public.legacy_set_entry (
 id uuid PRIMARY KEY, owner_mail varchar(255) NOT NULL, exercise_id uuid NOT NULL, variant_id uuid,
 source_ses_id integer NOT NULL, source_ex_id integer NOT NULL, source_set_count integer NOT NULL,
 weight numeric(6,2), rep integer, source_data jsonb NOT NULL, imported_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(source_ses_id,source_ex_id,source_set_count),
 FOREIGN KEY(owner_mail,exercise_id) REFERENCES public.exercise(owner_mail,id) ON DELETE CASCADE,
 FOREIGN KEY(owner_mail,variant_id) REFERENCES public.exercise_variant(owner_mail,id) ON DELETE CASCADE
);
CREATE INDEX legacy_set_exercise ON public.legacy_set_entry(owner_mail,exercise_id);
CREATE INDEX legacy_set_variant ON public.legacy_set_entry(owner_mail,variant_id);
