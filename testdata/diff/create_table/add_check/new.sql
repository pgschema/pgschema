CREATE TABLE public.code (
    code integer PRIMARY KEY CONSTRAINT code_check CHECK (code > 0 AND code < 255)
);
