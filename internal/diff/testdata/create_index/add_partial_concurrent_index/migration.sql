--
-- Name: idx_active_orders_customer_date; Type: INDEX; Schema: public; Owner: -
--
CREATE INDEX CONCURRENTLY idx_active_orders_customer_date ON public.orders USING btree (customer_id, order_date DESC, total_amount) WHERE (status = (expression));