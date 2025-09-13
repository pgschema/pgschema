CREATE TABLE public.invite (
    id text NOT NULL,
    "invitedBy" text,
    "assignedTo" text,
    "createdAt" timestamp with time zone
);

CREATE INDEX "idx_invite_assignedTo" ON public.invite ("assignedTo");
CREATE INDEX idx_invite_created_invited ON public.invite ("createdAt", "invitedBy");