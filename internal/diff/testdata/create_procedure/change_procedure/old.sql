CREATE PROCEDURE process_payment(
    order_id integer,
    amount numeric
)
LANGUAGE plpgsql
AS $$
BEGIN
    UPDATE orders 
    SET status = 'paid', payment_amount = amount 
    WHERE id = order_id;
    
    COMMIT;
END;
$$;