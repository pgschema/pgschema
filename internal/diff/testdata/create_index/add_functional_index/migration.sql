--
-- Name: idx_users_fullname_search; Type: INDEX; Schema: public; Owner: -
--
CREATE INDEX CONCURRENTLY idx_users_fullname_search ON public.users USING btree (lower(), lower(), lower());