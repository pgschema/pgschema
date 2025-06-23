CREATE TABLE public.batch_changes (
    id integer NOT NULL,
    name text NOT NULL,
    description text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT batch_changes_name_length CHECK ((char_length(name) >= 1) AND (char_length(name) <= 256)),
    CONSTRAINT batch_changes_name_valid CHECK ((name ~ '^[a-zA-Z0-9._-]+$'::text)),
    CONSTRAINT batch_changes_timestamps CHECK ((updated_at >= created_at))
);