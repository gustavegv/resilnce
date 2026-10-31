CREATE TABLE internal.porting_run (
 id uuid PRIMARY KEY, migration_version text NOT NULL, actor text NOT NULL,
 mode text NOT NULL CHECK(mode IN ('audit','apply')), owner_scope text,
 status text NOT NULL CHECK(status IN ('queued','running','completed','incomplete','failed')),
 options jsonb NOT NULL, result jsonb, error text,
 created_at timestamptz NOT NULL DEFAULT now(), started_at timestamptz, finished_at timestamptz,
 lease_token uuid, lease_until timestamptz, heartbeat_at timestamptz
);
CREATE INDEX porting_run_queue ON internal.porting_run(status,created_at);
CREATE TABLE internal.porting_owner (
 run_id uuid NOT NULL REFERENCES internal.porting_run(id), owner_key text NOT NULL,
 status text NOT NULL, result jsonb NOT NULL, updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(run_id,owner_key)
);
-- No domain FKs: edits, merges and deletions must never make a source new again.
CREATE TABLE internal.porting_group (
 migration_version text NOT NULL, owner_mail text NOT NULL, normalized_name text NOT NULL,
 exercise_id uuid NOT NULL, next_variant_number integer NOT NULL DEFAULT 1 CHECK(next_variant_number>0),
 PRIMARY KEY(migration_version,owner_mail,normalized_name)
);
CREATE TABLE internal.porting_map (
 migration_version text NOT NULL, source_table text NOT NULL, source_key jsonb NOT NULL,
 owner_key text NOT NULL, destinations jsonb NOT NULL, source_data jsonb NOT NULL,
 imported_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(migration_version,source_table,source_key)
);
CREATE INDEX porting_map_owner ON internal.porting_map(migration_version,owner_key);
CREATE TABLE internal.porting_issue (
 run_id uuid NOT NULL REFERENCES internal.porting_run(id), owner_key text NOT NULL,
 source_table text NOT NULL, source_key jsonb NOT NULL, code text NOT NULL,
 severity text NOT NULL CHECK(severity IN ('error','warning')),
 details text NOT NULL, source_data jsonb NOT NULL,
 resolution_status text NOT NULL DEFAULT 'open' CHECK(resolution_status IN ('open','resolved')),
 PRIMARY KEY(run_id,source_table,source_key,code)
);

-- The migration-owning backend/runner role retains access. Supabase browser roles
-- receive no grants. RLS is also enabled on the new public tables, with no browser policies.
DO $$
DECLARE t text; role_name text;
BEGIN
 FOREACH t IN ARRAY ARRAY['exercise','exercise_variant','variant_target_set','training_session','session_variant',
 'workout','workout_exercise','workout_set','mutation_receipt','legacy_history_entry','legacy_set_entry'] LOOP
  EXECUTE format('REVOKE ALL ON TABLE public.%I FROM PUBLIC',t);
  EXECUTE format('ALTER TABLE public.%I ENABLE ROW LEVEL SECURITY',t);
  FOREACH role_name IN ARRAY ARRAY['anon','authenticated'] LOOP
   IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname=role_name) THEN
    EXECUTE format('REVOKE ALL ON TABLE public.%I FROM %I',t,role_name);
   END IF;
  END LOOP;
 END LOOP;
 REVOKE ALL ON ALL TABLES IN SCHEMA internal FROM PUBLIC;
 REVOKE ALL ON ALL FUNCTIONS IN SCHEMA internal FROM PUBLIC;
 FOREACH role_name IN ARRAY ARRAY['anon','authenticated'] LOOP
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname=role_name) THEN
   EXECUTE format('REVOKE ALL ON SCHEMA internal FROM %I',role_name);
   EXECUTE format('REVOKE ALL ON ALL TABLES IN SCHEMA internal FROM %I',role_name);
   EXECUTE format('REVOKE ALL ON ALL FUNCTIONS IN SCHEMA internal FROM %I',role_name);
  END IF;
 END LOOP;
END $$;
