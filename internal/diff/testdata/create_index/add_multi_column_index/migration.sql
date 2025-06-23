--
-- Name: idx_dept_salary_hire; Type: INDEX; Schema: public; Owner: -
--
CREATE INDEX idx_dept_salary_hire ON public.employees USING btree (department_id, salary DESC, hire_date);