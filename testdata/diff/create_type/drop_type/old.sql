CREATE TYPE public.status AS ENUM (
   'active',
   'inactive',
   'pending'
);

CREATE TYPE public.priority AS ENUM (
   'low',
   'medium',
   'high',
   'urgent'
);