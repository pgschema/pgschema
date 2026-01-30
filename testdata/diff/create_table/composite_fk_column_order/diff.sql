ALTER TABLE order_items
ADD CONSTRAINT fk_order_items_order FOREIGN KEY (customer_id, order_id) REFERENCES orders (customer_id, order_id);
