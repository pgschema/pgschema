CREATE TABLE public.events (
    id integer NOT NULL,
    name text,
    -- String literal default
    status text DEFAULT 'active' NOT NULL,
    -- Numeric defaults
    priority integer DEFAULT 0,
    score numeric DEFAULT 0.0,
    -- Boolean default
    is_active boolean DEFAULT true,
    -- Function call defaults
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT now(),
    -- Type cast default
    config jsonb DEFAULT '{}'::jsonb,
    tags text[] DEFAULT '{}'::text[],
    -- Complex expression with parentheses and AT TIME ZONE (issue #91)
    created_at_utc timestamp without time zone DEFAULT (now() AT TIME ZONE 'utc') NOT NULL
);
