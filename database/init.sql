-- Ensure the application role exists.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'echo') THEN
        CREATE ROLE echo LOGIN PASSWORD 'echo';
    END IF;
END
$$;

-- Allow the application role to connect to and use the echo database.
GRANT CONNECT ON DATABASE echo TO echo;
GRANT USAGE, CREATE ON SCHEMA public TO echo;

-- Ensure CRUD access for existing and future tables.
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO echo;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO echo;

-- Ensure sequence usage for existing and future sequences.
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO echo;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT USAGE, SELECT ON SEQUENCES TO echo;
