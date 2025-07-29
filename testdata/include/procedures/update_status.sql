--
-- Name: update_status; Type: PROCEDURE; Schema: -; Owner: -
--

CREATE OR REPLACE PROCEDURE update_status(
    user_id_param integer,
    new_status text
)
LANGUAGE sql
AS $$
    UPDATE orders SET status = new_status WHERE user_id = user_id_param;
$$;