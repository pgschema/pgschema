ALTER TABLE public.batch_spec_resolution_jobs 
ADD COLUMN initiator_id integer REFERENCES public.users(id) ON UPDATE CASCADE ON DELETE SET NULL DEFERRABLE;