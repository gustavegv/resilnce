# Database schema

Application tables in the `public` schema. Refresh with `make db-schema` using `DATABASE_URL` from your environment or `apps/server/.env`. The integration test reads metadata only.

`User.active_session` holds a session ID as text; the application links it to `Session.ses_id`, but there is no foreign key.

## Tables

| table | primary_key |
| --- | --- |
| Exercise | ex_id |
| History | ex_id, save_date |
| SesExercise | ses_id, ex_id |
| Session | ses_id |
| Set | ses_id, ex_id, set_count |
| User | mail |

## Columns

| table | column | type | nullable |
| --- | --- | --- | --- |
| Exercise | ex_id | integer | No |
| Exercise | ex_name | character varying(255) | Yes |
| Exercise | auto_inc | numeric(6,2) | Yes |
| Exercise | mail | character varying(255) | Yes |
| History | save_date | timestamp without time zone | No |
| History | avg_rep | integer | Yes |
| History | avg_weight | numeric(6,2) | Yes |
| History | no_of_sets | integer | Yes |
| History | ex_id | integer | No |
| SesExercise | ses_id | integer | No |
| SesExercise | ex_id | integer | No |
| SesExercise | rep_threshold | smallint | Yes |
| SesExercise | set_count | smallint | Yes |
| SesExercise | rep_per_set | smallint[] | Yes |
| SesExercise | weight_per_set | real[] | Yes |
| SesExercise | rest_seconds | smallint | Yes |
| SesExercise | order | smallint | Yes |
| SesExercise | finished | boolean | No |
| Session | ses_id | integer | No |
| Session | ses_name | character varying(255) | Yes |
| Session | date_last_ran | timestamp without time zone | Yes |
| Session | mail | character varying(255) | Yes |
| Set | set_count | integer | No |
| Set | weight | numeric(6,2) | Yes |
| Set | rep | integer | Yes |
| Set | ses_id | integer | No |
| Set | ex_id | integer | No |
| User | mail | character varying(255) | No |
| User | signup_date | timestamp without time zone | Yes |
| User | name | character varying(255) | Yes |
| User | active_session | text | Yes |

## Relationships

| table | links_to |
| --- | --- |
| Exercise | FOREIGN KEY (mail) REFERENCES "User"(mail) ON DELETE CASCADE |
| History | FOREIGN KEY (ex_id) REFERENCES "Exercise"(ex_id) ON DELETE CASCADE |
| SesExercise | FOREIGN KEY (ex_id) REFERENCES "Exercise"(ex_id) |
| SesExercise | FOREIGN KEY (ses_id) REFERENCES "Session"(ses_id) ON DELETE CASCADE |
| Session | FOREIGN KEY (mail) REFERENCES "User"(mail) ON DELETE CASCADE |
| Set | FOREIGN KEY (ses_id, ex_id) REFERENCES "SesExercise"(ses_id, ex_id) ON DELETE CASCADE |

