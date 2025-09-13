CREATE TABLE public.invite (
    id text NOT NULL,
    "invitedBy" text,
    "assignedTo" text,
    "createdAt" timestamp with time zone
);