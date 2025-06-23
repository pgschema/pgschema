--
-- Name: idx_users_email_status; Type: INDEX; Schema: public; Owner: -
--
CREATE INDEX CONCURRENTLY idx_users_email_status ON public.users USING btree (email, status) WHERE (status = 'active');