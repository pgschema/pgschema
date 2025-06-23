--
-- Name: idx_unique_email_org; Type: INDEX; Schema: public; Owner: -
--
CREATE UNIQUE INDEX CONCURRENTLY idx_unique_email_org ON public.user_profiles USING btree (email, organization_id) WHERE (expression);