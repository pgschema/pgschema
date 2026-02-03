CREATE TABLE public.table1 (
    c1 serial NOT NULL,
    c2 int GENERATED ALWAYS AS IDENTITY,
    c3 int NOT NULL
);
