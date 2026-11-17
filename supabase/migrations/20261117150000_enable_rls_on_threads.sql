-- public.threads was created without RLS, so PostgREST exposed it to the anon key.
-- No policies: the API uses the table owner over a direct Postgres connection.
-- anon and authenticated get no access.
ALTER TABLE public.threads ENABLE ROW LEVEL SECURITY;
