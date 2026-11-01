-- Preserve policy for each new Session mapping. Old provenance stays unknown.
-- No FK: import mappings remain durable after their destination plan is deleted.
ALTER TABLE internal.porting_map
 ADD COLUMN source_timezone text,
 ADD COLUMN session_last_ran_at timestamptz;
