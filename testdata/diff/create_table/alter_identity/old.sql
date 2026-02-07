CREATE TABLE public.table1 (
    c1 int NOT NULL,
    c2 serial,
    c3 int GENERATED ALWAYS AS IDENTITY
);
