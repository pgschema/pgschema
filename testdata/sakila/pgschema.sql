--
-- PostgreSQL database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 0.1.3


--
-- Name: bıgınt; Type: DOMAIN; Schema: -; Owner: -
--

CREATE DOMAIN bıgınt AS bigint;


--
-- Name: mpaa_rating; Type: TYPE; Schema: -; Owner: -
--

CREATE TYPE mpaa_rating AS ENUM (
    'G',
    'PG',
    'PG-13',
    'R',
    'NC-17'
);


--
-- Name: year; Type: DOMAIN; Schema: -; Owner: -
--

CREATE DOMAIN year AS integer
	CONSTRAINT year_check CHECK (((VALUE >= 1901) AND (VALUE <= 2155)));


--
-- Name: _group_concat(text, text); Type: FUNCTION; Schema: -; Owner: -
--

CREATE FUNCTION _group_concat(text, text) RETURNS text
    LANGUAGE sql IMMUTABLE
    AS $_$
SELECT CASE
  WHEN $2 IS NULL THEN $1
  WHEN $1 IS NULL THEN $2
  ELSE $1 || ', ' || $2
END
$_$;


--
-- Name: film_in_stock(integer, integer); Type: FUNCTION; Schema: -; Owner: -
--

CREATE FUNCTION film_in_stock(p_film_id integer, p_store_id integer, OUT p_film_count integer) RETURNS SETOF integer
    LANGUAGE sql
    AS $_$
     SELECT inventory_id
     FROM inventory
     WHERE film_id = $1
     AND store_id = $2
     AND inventory_in_stock(inventory_id);
$_$;


--
-- Name: film_not_in_stock(integer, integer); Type: FUNCTION; Schema: -; Owner: -
--

CREATE FUNCTION film_not_in_stock(p_film_id integer, p_store_id integer, OUT p_film_count integer) RETURNS SETOF integer
    LANGUAGE sql
    AS $_$
    SELECT inventory_id
    FROM inventory
    WHERE film_id = $1
    AND store_id = $2
    AND NOT inventory_in_stock(inventory_id);
$_$;


--
-- Name: get_customer_balance(integer, timestamp with time zone); Type: FUNCTION; Schema: -; Owner: -
--

CREATE FUNCTION get_customer_balance(p_customer_id integer, p_effective_date timestamp with time zone) RETURNS numeric
    LANGUAGE plpgsql
    AS $$
       --#OK, WE NEED TO CALCULATE THE CURRENT BALANCE GIVEN A CUSTOMER_ID AND A DATE
       --#THAT WE WANT THE BALANCE TO BE EFFECTIVE FOR. THE BALANCE IS:
       --#   1) RENTAL FEES FOR ALL PREVIOUS RENTALS
       --#   2) ONE DOLLAR FOR EVERY DAY THE PREVIOUS RENTALS ARE OVERDUE
       --#   3) IF A FILM IS MORE THAN RENTAL_DURATION * 2 OVERDUE, CHARGE THE REPLACEMENT_COST
       --#   4) SUBTRACT ALL PAYMENTS MADE BEFORE THE DATE SPECIFIED
DECLARE
    v_rentfees DECIMAL(5,2); --#FEES PAID TO RENT THE VIDEOS INITIALLY
    v_overfees INTEGER;      --#LATE FEES FOR PRIOR RENTALS
    v_payments DECIMAL(5,2); --#SUM OF PAYMENTS MADE PREVIOUSLY
BEGIN
    SELECT COALESCE(SUM(film.rental_rate),0) INTO v_rentfees
    FROM film, inventory, rental
    WHERE film.film_id = inventory.film_id
      AND inventory.inventory_id = rental.inventory_id
      AND rental.rental_date <= p_effective_date
      AND rental.customer_id = p_customer_id;

    SELECT COALESCE(SUM(IF((rental.return_date - rental.rental_date) > (film.rental_duration * '1 day'::interval),
        ((rental.return_date - rental.rental_date) - (film.rental_duration * '1 day'::interval)),0)),0) INTO v_overfees
    FROM rental, inventory, film
    WHERE film.film_id = inventory.film_id
      AND inventory.inventory_id = rental.inventory_id
      AND rental.rental_date <= p_effective_date
      AND rental.customer_id = p_customer_id;

    SELECT COALESCE(SUM(payment.amount),0) INTO v_payments
    FROM payment
    WHERE payment.payment_date <= p_effective_date
    AND payment.customer_id = p_customer_id;

    RETURN v_rentfees + v_overfees - v_payments;
END
$$;


--
-- Name: inventory_held_by_customer(integer); Type: FUNCTION; Schema: -; Owner: -
--

CREATE FUNCTION inventory_held_by_customer(p_inventory_id integer) RETURNS integer
    LANGUAGE plpgsql
    AS $$
DECLARE
    v_customer_id INTEGER;
BEGIN

  SELECT customer_id INTO v_customer_id
  FROM rental
  WHERE return_date IS NULL
  AND inventory_id = p_inventory_id;

  RETURN v_customer_id;
END $$;


--
-- Name: inventory_in_stock(integer); Type: FUNCTION; Schema: -; Owner: -
--

CREATE FUNCTION inventory_in_stock(p_inventory_id integer) RETURNS boolean
    LANGUAGE plpgsql
    AS $$
DECLARE
    v_rentals INTEGER;
    v_out     INTEGER;
BEGIN
    -- AN ITEM IS IN-STOCK IF THERE ARE EITHER NO ROWS IN THE rental TABLE
    -- FOR THE ITEM OR ALL ROWS HAVE return_date POPULATED

    SELECT count(*) INTO v_rentals
    FROM rental
    WHERE inventory_id = p_inventory_id;

    IF v_rentals = 0 THEN
      RETURN TRUE;
    END IF;

    SELECT COUNT(rental_id) INTO v_out
    FROM inventory LEFT JOIN rental USING(inventory_id)
    WHERE inventory.inventory_id = p_inventory_id
    AND rental.return_date IS NULL;

    IF v_out > 0 THEN
      RETURN FALSE;
    ELSE
      RETURN TRUE;
    END IF;
END $$;


--
-- Name: last_day(timestamp with time zone); Type: FUNCTION; Schema: -; Owner: -
--

CREATE FUNCTION last_day(timestamp with time zone) RETURNS date
    LANGUAGE sql IMMUTABLE STRICT
    AS $_$
  SELECT CASE
    WHEN EXTRACT(MONTH FROM $1) = 12 THEN
      (((EXTRACT(YEAR FROM $1) + 1) operator(pg_catalog.||) '-01-01')::date - INTERVAL '1 day')::date
    ELSE
      ((EXTRACT(YEAR FROM $1) operator(pg_catalog.||) '-' operator(pg_catalog.||) (EXTRACT(MONTH FROM $1) + 1) operator(pg_catalog.||) '-01')::date - INTERVAL '1 day')::date
    END
$_$;


--
-- Name: last_updated(); Type: FUNCTION; Schema: -; Owner: -
--

CREATE FUNCTION last_updated() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.last_update = CURRENT_TIMESTAMP;
    RETURN NEW;
END $$;


--
-- Name: rewards_report(integer, numeric); Type: FUNCTION; Schema: -; Owner: -
--

CREATE FUNCTION rewards_report(min_monthly_purchases integer, min_dollar_amount_purchased numeric) RETURNS SETOF customer
    LANGUAGE plpgsql SECURITY DEFINER
    AS $_$
DECLARE
    last_month_start DATE;
    last_month_end DATE;
rr RECORD;
tmpSQL TEXT;
BEGIN

    /* Some sanity checks... */
    IF min_monthly_purchases = 0 THEN
        RAISE EXCEPTION 'Minimum monthly purchases parameter must be > 0';
    END IF;
    IF min_dollar_amount_purchased = 0.00 THEN
        RAISE EXCEPTION 'Minimum monthly dollar amount purchased parameter must be > $0.00';
    END IF;

    last_month_start := CURRENT_DATE - '3 month'::interval;
    last_month_start := to_date((extract(YEAR FROM last_month_start) || '-' || extract(MONTH FROM last_month_start) || '-01'),'YYYY-MM-DD');
    last_month_end := LAST_DAY(last_month_start);

    /*
    Create a temporary storage area for Customer IDs.
    */
    CREATE TEMPORARY TABLE tmpCustomer (customer_id INTEGER NOT NULL PRIMARY KEY);

    /*
    Find all customers meeting the monthly purchase requirements
    */

    tmpSQL := 'INSERT INTO tmpCustomer (customer_id)
        SELECT p.customer_id
        FROM payment AS p
        WHERE DATE(p.payment_date) BETWEEN '||quote_literal(last_month_start) ||' AND '|| quote_literal(last_month_end) || '
        GROUP BY customer_id
        HAVING SUM(p.amount) > '|| min_dollar_amount_purchased || '
        AND COUNT(customer_id) > ' ||min_monthly_purchases ;

    EXECUTE tmpSQL;

    /*
    Output ALL customer information of matching rewardees.
    Customize output as needed.
    */
    FOR rr IN EXECUTE 'SELECT c.* FROM tmpCustomer AS t INNER JOIN customer AS c ON t.customer_id = c.customer_id' LOOP
        RETURN NEXT rr;
    END LOOP;

    /* Clean up */
    tmpSQL := 'DROP TABLE tmpCustomer';
    EXECUTE tmpSQL;

RETURN;
END
$_$;


--
-- Name: group_concat(text); Type: AGGREGATE; Schema: -; Owner: -
--

CREATE AGGREGATE group_concat(text) (
    SFUNC = _group_concat,
    STYPE = text,
    INITCOND = ''
);




--
-- Name: actor; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE actor (
    actor_id SERIAL PRIMARY KEY,
    first_name text NOT NULL,
    last_name text NOT NULL,
    last_update timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: actor_info; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW actor_info AS
 SELECT a.actor_id,
    a.first_name,
    a.last_name,
    group_concat(DISTINCT ((c.name || ': '::text) || ( SELECT group_concat(f.title) AS group_concat
           FROM ((film f
             JOIN film_category fc_1 ON ((f.film_id = fc_1.film_id)))
             JOIN film_actor fa_1 ON ((f.film_id = fa_1.film_id)))
          WHERE ((fc_1.category_id = c.category_id) AND (fa_1.actor_id = a.actor_id))
          GROUP BY fa_1.actor_id))) AS film_info
   FROM (((actor a
     LEFT JOIN film_actor fa ON ((a.actor_id = fa.actor_id)))
     LEFT JOIN film_category fc ON ((fa.film_id = fc.film_id)))
     LEFT JOIN category c ON ((fc.category_id = c.category_id)))
  GROUP BY a.actor_id, a.first_name, a.last_name;


--
-- Name: film_list; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW film_list AS
 SELECT film.film_id AS fid,
    film.title,
    film.description,
    category.name AS category,
    film.rental_rate AS price,
    film.length,
    film.rating,
    group_concat(((actor.first_name || ' '::text) || actor.last_name)) AS actors
   FROM ((((category
     LEFT JOIN film_category ON ((category.category_id = film_category.category_id)))
     LEFT JOIN film ON ((film_category.film_id = film.film_id)))
     JOIN film_actor ON ((film.film_id = film_actor.film_id)))
     JOIN actor ON ((film_actor.actor_id = actor.actor_id)))
  GROUP BY film.film_id, film.title, film.description, category.name, film.rental_rate, film.length, film.rating;


--
-- Name: nicer_but_slower_film_list; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW nicer_but_slower_film_list AS
 SELECT film.film_id AS fid,
    film.title,
    film.description,
    category.name AS category,
    film.rental_rate AS price,
    film.length,
    film.rating,
    group_concat((((upper("substring"(actor.first_name, 1, 1)) || lower("substring"(actor.first_name, 2))) || upper("substring"(actor.last_name, 1, 1))) || lower("substring"(actor.last_name, 2)))) AS actors
   FROM ((((category
     LEFT JOIN film_category ON ((category.category_id = film_category.category_id)))
     LEFT JOIN film ON ((film_category.film_id = film.film_id)))
     JOIN film_actor ON ((film.film_id = film_actor.film_id)))
     JOIN actor ON ((film_actor.actor_id = actor.actor_id)))
  GROUP BY film.film_id, film.title, film.description, category.name, film.rental_rate, film.length, film.rating;


--
-- Name: address; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE address (
    address_id SERIAL PRIMARY KEY,
    address text NOT NULL,
    address2 text,
    district text NOT NULL,
    city_id integer NOT NULL,
    postal_code text,
    phone text NOT NULL,
    last_update timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: customer_list; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW customer_list AS
 SELECT cu.customer_id AS id,
    ((cu.first_name || ' '::text) || cu.last_name) AS name,
    a.address,
    a.postal_code AS "zip code",
    a.phone,
    city.city,
    country.country,
        CASE
            WHEN cu.activebool THEN 'active'::text
            ELSE ''::text
        END AS notes,
    cu.store_id AS sid
   FROM (((customer cu
     JOIN address a ON ((cu.address_id = a.address_id)))
     JOIN city ON ((a.city_id = city.city_id)))
     JOIN country ON ((city.country_id = country.country_id)));


--
-- Name: sales_by_store; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW sales_by_store AS
 SELECT ((c.city || ','::text) || cy.country) AS store,
    ((m.first_name || ' '::text) || m.last_name) AS manager,
    sum(p.amount) AS total_sales
   FROM (((((((payment p
     JOIN rental r ON ((p.rental_id = r.rental_id)))
     JOIN inventory i ON ((r.inventory_id = i.inventory_id)))
     JOIN store s ON ((i.store_id = s.store_id)))
     JOIN address a ON ((s.address_id = a.address_id)))
     JOIN city c ON ((a.city_id = c.city_id)))
     JOIN country cy ON ((c.country_id = cy.country_id)))
     JOIN staff m ON ((s.manager_staff_id = m.staff_id)))
  GROUP BY cy.country, c.city, s.store_id, m.first_name, m.last_name
  ORDER BY cy.country, c.city;


--
-- Name: staff_list; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW staff_list AS
 SELECT s.staff_id AS id,
    ((s.first_name || ' '::text) || s.last_name) AS name,
    a.address,
    a.postal_code AS "zip code",
    a.phone,
    city.city,
    country.country,
    s.store_id AS sid
   FROM (((staff s
     JOIN address a ON ((s.address_id = a.address_id)))
     JOIN city ON ((a.city_id = city.city_id)))
     JOIN country ON ((city.country_id = country.country_id)));


--
-- Name: category; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE category (
    category_id SERIAL PRIMARY KEY,
    name text NOT NULL,
    last_update timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: actor_info; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW actor_info AS
 SELECT a.actor_id,
    a.first_name,
    a.last_name,
    group_concat(DISTINCT ((c.name || ': '::text) || ( SELECT group_concat(f.title) AS group_concat
           FROM ((film f
             JOIN film_category fc_1 ON ((f.film_id = fc_1.film_id)))
             JOIN film_actor fa_1 ON ((f.film_id = fa_1.film_id)))
          WHERE ((fc_1.category_id = c.category_id) AND (fa_1.actor_id = a.actor_id))
          GROUP BY fa_1.actor_id))) AS film_info
   FROM (((actor a
     LEFT JOIN film_actor fa ON ((a.actor_id = fa.actor_id)))
     LEFT JOIN film_category fc ON ((fa.film_id = fc.film_id)))
     LEFT JOIN category c ON ((fc.category_id = c.category_id)))
  GROUP BY a.actor_id, a.first_name, a.last_name;


--
-- Name: film_list; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW film_list AS
 SELECT film.film_id AS fid,
    film.title,
    film.description,
    category.name AS category,
    film.rental_rate AS price,
    film.length,
    film.rating,
    group_concat(((actor.first_name || ' '::text) || actor.last_name)) AS actors
   FROM ((((category
     LEFT JOIN film_category ON ((category.category_id = film_category.category_id)))
     LEFT JOIN film ON ((film_category.film_id = film.film_id)))
     JOIN film_actor ON ((film.film_id = film_actor.film_id)))
     JOIN actor ON ((film_actor.actor_id = actor.actor_id)))
  GROUP BY film.film_id, film.title, film.description, category.name, film.rental_rate, film.length, film.rating;


--
-- Name: nicer_but_slower_film_list; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW nicer_but_slower_film_list AS
 SELECT film.film_id AS fid,
    film.title,
    film.description,
    category.name AS category,
    film.rental_rate AS price,
    film.length,
    film.rating,
    group_concat((((upper("substring"(actor.first_name, 1, 1)) || lower("substring"(actor.first_name, 2))) || upper("substring"(actor.last_name, 1, 1))) || lower("substring"(actor.last_name, 2)))) AS actors
   FROM ((((category
     LEFT JOIN film_category ON ((category.category_id = film_category.category_id)))
     LEFT JOIN film ON ((film_category.film_id = film.film_id)))
     JOIN film_actor ON ((film.film_id = film_actor.film_id)))
     JOIN actor ON ((film_actor.actor_id = actor.actor_id)))
  GROUP BY film.film_id, film.title, film.description, category.name, film.rental_rate, film.length, film.rating;


--
-- Name: sales_by_film_category; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW sales_by_film_category AS
 SELECT c.name AS category,
    sum(p.amount) AS total_sales
   FROM (((((payment p
     JOIN rental r ON ((p.rental_id = r.rental_id)))
     JOIN inventory i ON ((r.inventory_id = i.inventory_id)))
     JOIN film f ON ((i.film_id = f.film_id)))
     JOIN film_category fc ON ((f.film_id = fc.film_id)))
     JOIN category c ON ((fc.category_id = c.category_id)))
  GROUP BY c.name
  ORDER BY (sum(p.amount)) DESC;


--
-- Name: city; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE city (
    city_id SERIAL PRIMARY KEY,
    city text NOT NULL,
    country_id integer NOT NULL,
    last_update timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: customer_list; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW customer_list AS
 SELECT cu.customer_id AS id,
    ((cu.first_name || ' '::text) || cu.last_name) AS name,
    a.address,
    a.postal_code AS "zip code",
    a.phone,
    city.city,
    country.country,
        CASE
            WHEN cu.activebool THEN 'active'::text
            ELSE ''::text
        END AS notes,
    cu.store_id AS sid
   FROM (((customer cu
     JOIN address a ON ((cu.address_id = a.address_id)))
     JOIN city ON ((a.city_id = city.city_id)))
     JOIN country ON ((city.country_id = country.country_id)));


--
-- Name: sales_by_store; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW sales_by_store AS
 SELECT ((c.city || ','::text) || cy.country) AS store,
    ((m.first_name || ' '::text) || m.last_name) AS manager,
    sum(p.amount) AS total_sales
   FROM (((((((payment p
     JOIN rental r ON ((p.rental_id = r.rental_id)))
     JOIN inventory i ON ((r.inventory_id = i.inventory_id)))
     JOIN store s ON ((i.store_id = s.store_id)))
     JOIN address a ON ((s.address_id = a.address_id)))
     JOIN city c ON ((a.city_id = c.city_id)))
     JOIN country cy ON ((c.country_id = cy.country_id)))
     JOIN staff m ON ((s.manager_staff_id = m.staff_id)))
  GROUP BY cy.country, c.city, s.store_id, m.first_name, m.last_name
  ORDER BY cy.country, c.city;


--
-- Name: staff_list; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW staff_list AS
 SELECT s.staff_id AS id,
    ((s.first_name || ' '::text) || s.last_name) AS name,
    a.address,
    a.postal_code AS "zip code",
    a.phone,
    city.city,
    country.country,
    s.store_id AS sid
   FROM (((staff s
     JOIN address a ON ((s.address_id = a.address_id)))
     JOIN city ON ((a.city_id = city.city_id)))
     JOIN country ON ((city.country_id = country.country_id)));


--
-- Name: country; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE country (
    country_id SERIAL PRIMARY KEY,
    country text NOT NULL,
    last_update timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: customer_list; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW customer_list AS
 SELECT cu.customer_id AS id,
    ((cu.first_name || ' '::text) || cu.last_name) AS name,
    a.address,
    a.postal_code AS "zip code",
    a.phone,
    city.city,
    country.country,
        CASE
            WHEN cu.activebool THEN 'active'::text
            ELSE ''::text
        END AS notes,
    cu.store_id AS sid
   FROM (((customer cu
     JOIN address a ON ((cu.address_id = a.address_id)))
     JOIN city ON ((a.city_id = city.city_id)))
     JOIN country ON ((city.country_id = country.country_id)));


--
-- Name: sales_by_store; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW sales_by_store AS
 SELECT ((c.city || ','::text) || cy.country) AS store,
    ((m.first_name || ' '::text) || m.last_name) AS manager,
    sum(p.amount) AS total_sales
   FROM (((((((payment p
     JOIN rental r ON ((p.rental_id = r.rental_id)))
     JOIN inventory i ON ((r.inventory_id = i.inventory_id)))
     JOIN store s ON ((i.store_id = s.store_id)))
     JOIN address a ON ((s.address_id = a.address_id)))
     JOIN city c ON ((a.city_id = c.city_id)))
     JOIN country cy ON ((c.country_id = cy.country_id)))
     JOIN staff m ON ((s.manager_staff_id = m.staff_id)))
  GROUP BY cy.country, c.city, s.store_id, m.first_name, m.last_name
  ORDER BY cy.country, c.city;


--
-- Name: staff_list; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW staff_list AS
 SELECT s.staff_id AS id,
    ((s.first_name || ' '::text) || s.last_name) AS name,
    a.address,
    a.postal_code AS "zip code",
    a.phone,
    city.city,
    country.country,
    s.store_id AS sid
   FROM (((staff s
     JOIN address a ON ((s.address_id = a.address_id)))
     JOIN city ON ((a.city_id = city.city_id)))
     JOIN country ON ((city.country_id = country.country_id)));


--
-- Name: customer; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE customer (
    customer_id SERIAL PRIMARY KEY,
    store_id integer NOT NULL,
    first_name text NOT NULL,
    last_name text NOT NULL,
    email text,
    address_id integer NOT NULL,
    activebool boolean DEFAULT true NOT NULL,
    create_date date DEFAULT CURRENT_DATE NOT NULL,
    last_update timestamp with time zone DEFAULT now(),
    active integer
);


--
-- Name: customer_list; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW customer_list AS
 SELECT cu.customer_id AS id,
    ((cu.first_name || ' '::text) || cu.last_name) AS name,
    a.address,
    a.postal_code AS "zip code",
    a.phone,
    city.city,
    country.country,
        CASE
            WHEN cu.activebool THEN 'active'::text
            ELSE ''::text
        END AS notes,
    cu.store_id AS sid
   FROM (((customer cu
     JOIN address a ON ((cu.address_id = a.address_id)))
     JOIN city ON ((a.city_id = city.city_id)))
     JOIN country ON ((city.country_id = country.country_id)));


--
-- Name: film; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE film (
    film_id SERIAL PRIMARY KEY,
    title text NOT NULL,
    description text,
    release_year year,
    language_id integer NOT NULL,
    original_language_id integer,
    rental_duration smallint DEFAULT 3 NOT NULL,
    rental_rate numeric(4,2) DEFAULT 4.99 NOT NULL,
    length smallint,
    replacement_cost numeric(5,2) DEFAULT 19.99 NOT NULL,
    rating mpaa_rating DEFAULT 'G'::mpaa_rating,
    last_update timestamp with time zone DEFAULT now() NOT NULL,
    special_features text[],
    fulltext tsvector NOT NULL
);


--
-- Name: actor_info; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW actor_info AS
 SELECT a.actor_id,
    a.first_name,
    a.last_name,
    group_concat(DISTINCT ((c.name || ': '::text) || ( SELECT group_concat(f.title) AS group_concat
           FROM ((film f
             JOIN film_category fc_1 ON ((f.film_id = fc_1.film_id)))
             JOIN film_actor fa_1 ON ((f.film_id = fa_1.film_id)))
          WHERE ((fc_1.category_id = c.category_id) AND (fa_1.actor_id = a.actor_id))
          GROUP BY fa_1.actor_id))) AS film_info
   FROM (((actor a
     LEFT JOIN film_actor fa ON ((a.actor_id = fa.actor_id)))
     LEFT JOIN film_category fc ON ((fa.film_id = fc.film_id)))
     LEFT JOIN category c ON ((fc.category_id = c.category_id)))
  GROUP BY a.actor_id, a.first_name, a.last_name;


--
-- Name: film_list; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW film_list AS
 SELECT film.film_id AS fid,
    film.title,
    film.description,
    category.name AS category,
    film.rental_rate AS price,
    film.length,
    film.rating,
    group_concat(((actor.first_name || ' '::text) || actor.last_name)) AS actors
   FROM ((((category
     LEFT JOIN film_category ON ((category.category_id = film_category.category_id)))
     LEFT JOIN film ON ((film_category.film_id = film.film_id)))
     JOIN film_actor ON ((film.film_id = film_actor.film_id)))
     JOIN actor ON ((film_actor.actor_id = actor.actor_id)))
  GROUP BY film.film_id, film.title, film.description, category.name, film.rental_rate, film.length, film.rating;


--
-- Name: nicer_but_slower_film_list; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW nicer_but_slower_film_list AS
 SELECT film.film_id AS fid,
    film.title,
    film.description,
    category.name AS category,
    film.rental_rate AS price,
    film.length,
    film.rating,
    group_concat((((upper("substring"(actor.first_name, 1, 1)) || lower("substring"(actor.first_name, 2))) || upper("substring"(actor.last_name, 1, 1))) || lower("substring"(actor.last_name, 2)))) AS actors
   FROM ((((category
     LEFT JOIN film_category ON ((category.category_id = film_category.category_id)))
     LEFT JOIN film ON ((film_category.film_id = film.film_id)))
     JOIN film_actor ON ((film.film_id = film_actor.film_id)))
     JOIN actor ON ((film_actor.actor_id = actor.actor_id)))
  GROUP BY film.film_id, film.title, film.description, category.name, film.rental_rate, film.length, film.rating;


--
-- Name: sales_by_film_category; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW sales_by_film_category AS
 SELECT c.name AS category,
    sum(p.amount) AS total_sales
   FROM (((((payment p
     JOIN rental r ON ((p.rental_id = r.rental_id)))
     JOIN inventory i ON ((r.inventory_id = i.inventory_id)))
     JOIN film f ON ((i.film_id = f.film_id)))
     JOIN film_category fc ON ((f.film_id = fc.film_id)))
     JOIN category c ON ((fc.category_id = c.category_id)))
  GROUP BY c.name
  ORDER BY (sum(p.amount)) DESC;


--
-- Name: film_actor; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE film_actor (
    actor_id integer NOT NULL,
    film_id integer NOT NULL,
    last_update timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: actor_info; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW actor_info AS
 SELECT a.actor_id,
    a.first_name,
    a.last_name,
    group_concat(DISTINCT ((c.name || ': '::text) || ( SELECT group_concat(f.title) AS group_concat
           FROM ((film f
             JOIN film_category fc_1 ON ((f.film_id = fc_1.film_id)))
             JOIN film_actor fa_1 ON ((f.film_id = fa_1.film_id)))
          WHERE ((fc_1.category_id = c.category_id) AND (fa_1.actor_id = a.actor_id))
          GROUP BY fa_1.actor_id))) AS film_info
   FROM (((actor a
     LEFT JOIN film_actor fa ON ((a.actor_id = fa.actor_id)))
     LEFT JOIN film_category fc ON ((fa.film_id = fc.film_id)))
     LEFT JOIN category c ON ((fc.category_id = c.category_id)))
  GROUP BY a.actor_id, a.first_name, a.last_name;


--
-- Name: film_list; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW film_list AS
 SELECT film.film_id AS fid,
    film.title,
    film.description,
    category.name AS category,
    film.rental_rate AS price,
    film.length,
    film.rating,
    group_concat(((actor.first_name || ' '::text) || actor.last_name)) AS actors
   FROM ((((category
     LEFT JOIN film_category ON ((category.category_id = film_category.category_id)))
     LEFT JOIN film ON ((film_category.film_id = film.film_id)))
     JOIN film_actor ON ((film.film_id = film_actor.film_id)))
     JOIN actor ON ((film_actor.actor_id = actor.actor_id)))
  GROUP BY film.film_id, film.title, film.description, category.name, film.rental_rate, film.length, film.rating;


--
-- Name: nicer_but_slower_film_list; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW nicer_but_slower_film_list AS
 SELECT film.film_id AS fid,
    film.title,
    film.description,
    category.name AS category,
    film.rental_rate AS price,
    film.length,
    film.rating,
    group_concat((((upper("substring"(actor.first_name, 1, 1)) || lower("substring"(actor.first_name, 2))) || upper("substring"(actor.last_name, 1, 1))) || lower("substring"(actor.last_name, 2)))) AS actors
   FROM ((((category
     LEFT JOIN film_category ON ((category.category_id = film_category.category_id)))
     LEFT JOIN film ON ((film_category.film_id = film.film_id)))
     JOIN film_actor ON ((film.film_id = film_actor.film_id)))
     JOIN actor ON ((film_actor.actor_id = actor.actor_id)))
  GROUP BY film.film_id, film.title, film.description, category.name, film.rental_rate, film.length, film.rating;


--
-- Name: film_category; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE film_category (
    film_id integer NOT NULL,
    category_id integer NOT NULL,
    last_update timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: actor_info; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW actor_info AS
 SELECT a.actor_id,
    a.first_name,
    a.last_name,
    group_concat(DISTINCT ((c.name || ': '::text) || ( SELECT group_concat(f.title) AS group_concat
           FROM ((film f
             JOIN film_category fc_1 ON ((f.film_id = fc_1.film_id)))
             JOIN film_actor fa_1 ON ((f.film_id = fa_1.film_id)))
          WHERE ((fc_1.category_id = c.category_id) AND (fa_1.actor_id = a.actor_id))
          GROUP BY fa_1.actor_id))) AS film_info
   FROM (((actor a
     LEFT JOIN film_actor fa ON ((a.actor_id = fa.actor_id)))
     LEFT JOIN film_category fc ON ((fa.film_id = fc.film_id)))
     LEFT JOIN category c ON ((fc.category_id = c.category_id)))
  GROUP BY a.actor_id, a.first_name, a.last_name;


--
-- Name: film_list; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW film_list AS
 SELECT film.film_id AS fid,
    film.title,
    film.description,
    category.name AS category,
    film.rental_rate AS price,
    film.length,
    film.rating,
    group_concat(((actor.first_name || ' '::text) || actor.last_name)) AS actors
   FROM ((((category
     LEFT JOIN film_category ON ((category.category_id = film_category.category_id)))
     LEFT JOIN film ON ((film_category.film_id = film.film_id)))
     JOIN film_actor ON ((film.film_id = film_actor.film_id)))
     JOIN actor ON ((film_actor.actor_id = actor.actor_id)))
  GROUP BY film.film_id, film.title, film.description, category.name, film.rental_rate, film.length, film.rating;


--
-- Name: nicer_but_slower_film_list; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW nicer_but_slower_film_list AS
 SELECT film.film_id AS fid,
    film.title,
    film.description,
    category.name AS category,
    film.rental_rate AS price,
    film.length,
    film.rating,
    group_concat((((upper("substring"(actor.first_name, 1, 1)) || lower("substring"(actor.first_name, 2))) || upper("substring"(actor.last_name, 1, 1))) || lower("substring"(actor.last_name, 2)))) AS actors
   FROM ((((category
     LEFT JOIN film_category ON ((category.category_id = film_category.category_id)))
     LEFT JOIN film ON ((film_category.film_id = film.film_id)))
     JOIN film_actor ON ((film.film_id = film_actor.film_id)))
     JOIN actor ON ((film_actor.actor_id = actor.actor_id)))
  GROUP BY film.film_id, film.title, film.description, category.name, film.rental_rate, film.length, film.rating;


--
-- Name: sales_by_film_category; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW sales_by_film_category AS
 SELECT c.name AS category,
    sum(p.amount) AS total_sales
   FROM (((((payment p
     JOIN rental r ON ((p.rental_id = r.rental_id)))
     JOIN inventory i ON ((r.inventory_id = i.inventory_id)))
     JOIN film f ON ((i.film_id = f.film_id)))
     JOIN film_category fc ON ((f.film_id = fc.film_id)))
     JOIN category c ON ((fc.category_id = c.category_id)))
  GROUP BY c.name
  ORDER BY (sum(p.amount)) DESC;


--
-- Name: inventory; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE inventory (
    inventory_id SERIAL PRIMARY KEY,
    film_id integer NOT NULL,
    store_id integer NOT NULL,
    last_update timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: sales_by_film_category; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW sales_by_film_category AS
 SELECT c.name AS category,
    sum(p.amount) AS total_sales
   FROM (((((payment p
     JOIN rental r ON ((p.rental_id = r.rental_id)))
     JOIN inventory i ON ((r.inventory_id = i.inventory_id)))
     JOIN film f ON ((i.film_id = f.film_id)))
     JOIN film_category fc ON ((f.film_id = fc.film_id)))
     JOIN category c ON ((fc.category_id = c.category_id)))
  GROUP BY c.name
  ORDER BY (sum(p.amount)) DESC;


--
-- Name: sales_by_store; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW sales_by_store AS
 SELECT ((c.city || ','::text) || cy.country) AS store,
    ((m.first_name || ' '::text) || m.last_name) AS manager,
    sum(p.amount) AS total_sales
   FROM (((((((payment p
     JOIN rental r ON ((p.rental_id = r.rental_id)))
     JOIN inventory i ON ((r.inventory_id = i.inventory_id)))
     JOIN store s ON ((i.store_id = s.store_id)))
     JOIN address a ON ((s.address_id = a.address_id)))
     JOIN city c ON ((a.city_id = c.city_id)))
     JOIN country cy ON ((c.country_id = cy.country_id)))
     JOIN staff m ON ((s.manager_staff_id = m.staff_id)))
  GROUP BY cy.country, c.city, s.store_id, m.first_name, m.last_name
  ORDER BY cy.country, c.city;


--
-- Name: language; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE language (
    language_id SERIAL PRIMARY KEY,
    name character(20) NOT NULL,
    last_update timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: payment; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE payment (
    payment_id SERIAL NOT NULL,
    customer_id integer NOT NULL,
    staff_id integer NOT NULL,
    rental_id integer NOT NULL,
    amount numeric(5,2) NOT NULL,
    payment_date timestamp with time zone NOT NULL
)
PARTITION BY RANGE (payment_date);


--
-- Name: sales_by_film_category; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW sales_by_film_category AS
 SELECT c.name AS category,
    sum(p.amount) AS total_sales
   FROM (((((payment p
     JOIN rental r ON ((p.rental_id = r.rental_id)))
     JOIN inventory i ON ((r.inventory_id = i.inventory_id)))
     JOIN film f ON ((i.film_id = f.film_id)))
     JOIN film_category fc ON ((f.film_id = fc.film_id)))
     JOIN category c ON ((fc.category_id = c.category_id)))
  GROUP BY c.name
  ORDER BY (sum(p.amount)) DESC;


--
-- Name: sales_by_store; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW sales_by_store AS
 SELECT ((c.city || ','::text) || cy.country) AS store,
    ((m.first_name || ' '::text) || m.last_name) AS manager,
    sum(p.amount) AS total_sales
   FROM (((((((payment p
     JOIN rental r ON ((p.rental_id = r.rental_id)))
     JOIN inventory i ON ((r.inventory_id = i.inventory_id)))
     JOIN store s ON ((i.store_id = s.store_id)))
     JOIN address a ON ((s.address_id = a.address_id)))
     JOIN city c ON ((a.city_id = c.city_id)))
     JOIN country cy ON ((c.country_id = cy.country_id)))
     JOIN staff m ON ((s.manager_staff_id = m.staff_id)))
  GROUP BY cy.country, c.city, s.store_id, m.first_name, m.last_name
  ORDER BY cy.country, c.city;


--
-- Name: payment_p2022_01; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE payment_p2022_01 (
    payment_id SERIAL NOT NULL,
    customer_id integer NOT NULL,
    staff_id integer NOT NULL,
    rental_id integer NOT NULL,
    amount numeric(5,2) NOT NULL,
    payment_date timestamp with time zone NOT NULL
);


--
-- Name: payment_p2022_02; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE payment_p2022_02 (
    payment_id SERIAL NOT NULL,
    customer_id integer NOT NULL,
    staff_id integer NOT NULL,
    rental_id integer NOT NULL,
    amount numeric(5,2) NOT NULL,
    payment_date timestamp with time zone NOT NULL
);


--
-- Name: payment_p2022_03; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE payment_p2022_03 (
    payment_id SERIAL NOT NULL,
    customer_id integer NOT NULL,
    staff_id integer NOT NULL,
    rental_id integer NOT NULL,
    amount numeric(5,2) NOT NULL,
    payment_date timestamp with time zone NOT NULL
);


--
-- Name: payment_p2022_04; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE payment_p2022_04 (
    payment_id SERIAL NOT NULL,
    customer_id integer NOT NULL,
    staff_id integer NOT NULL,
    rental_id integer NOT NULL,
    amount numeric(5,2) NOT NULL,
    payment_date timestamp with time zone NOT NULL
);


--
-- Name: payment_p2022_05; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE payment_p2022_05 (
    payment_id SERIAL NOT NULL,
    customer_id integer NOT NULL,
    staff_id integer NOT NULL,
    rental_id integer NOT NULL,
    amount numeric(5,2) NOT NULL,
    payment_date timestamp with time zone NOT NULL
);


--
-- Name: payment_p2022_06; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE payment_p2022_06 (
    payment_id SERIAL NOT NULL,
    customer_id integer NOT NULL,
    staff_id integer NOT NULL,
    rental_id integer NOT NULL,
    amount numeric(5,2) NOT NULL,
    payment_date timestamp with time zone NOT NULL
);


--
-- Name: payment_p2022_07; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE payment_p2022_07 (
    payment_id SERIAL NOT NULL,
    customer_id integer NOT NULL,
    staff_id integer NOT NULL,
    rental_id integer NOT NULL,
    amount numeric(5,2) NOT NULL,
    payment_date timestamp with time zone NOT NULL
);


--
-- Name: rental; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE rental (
    rental_id SERIAL PRIMARY KEY,
    rental_date timestamp with time zone NOT NULL,
    inventory_id integer NOT NULL,
    customer_id integer NOT NULL,
    return_date timestamp with time zone,
    staff_id integer NOT NULL,
    last_update timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: film_list; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW film_list AS
 SELECT film.film_id AS fid,
    film.title,
    film.description,
    category.name AS category,
    film.rental_rate AS price,
    film.length,
    film.rating,
    group_concat(((actor.first_name || ' '::text) || actor.last_name)) AS actors
   FROM ((((category
     LEFT JOIN film_category ON ((category.category_id = film_category.category_id)))
     LEFT JOIN film ON ((film_category.film_id = film.film_id)))
     JOIN film_actor ON ((film.film_id = film_actor.film_id)))
     JOIN actor ON ((film_actor.actor_id = actor.actor_id)))
  GROUP BY film.film_id, film.title, film.description, category.name, film.rental_rate, film.length, film.rating;


--
-- Name: nicer_but_slower_film_list; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW nicer_but_slower_film_list AS
 SELECT film.film_id AS fid,
    film.title,
    film.description,
    category.name AS category,
    film.rental_rate AS price,
    film.length,
    film.rating,
    group_concat((((upper("substring"(actor.first_name, 1, 1)) || lower("substring"(actor.first_name, 2))) || upper("substring"(actor.last_name, 1, 1))) || lower("substring"(actor.last_name, 2)))) AS actors
   FROM ((((category
     LEFT JOIN film_category ON ((category.category_id = film_category.category_id)))
     LEFT JOIN film ON ((film_category.film_id = film.film_id)))
     JOIN film_actor ON ((film.film_id = film_actor.film_id)))
     JOIN actor ON ((film_actor.actor_id = actor.actor_id)))
  GROUP BY film.film_id, film.title, film.description, category.name, film.rental_rate, film.length, film.rating;


--
-- Name: sales_by_film_category; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW sales_by_film_category AS
 SELECT c.name AS category,
    sum(p.amount) AS total_sales
   FROM (((((payment p
     JOIN rental r ON ((p.rental_id = r.rental_id)))
     JOIN inventory i ON ((r.inventory_id = i.inventory_id)))
     JOIN film f ON ((i.film_id = f.film_id)))
     JOIN film_category fc ON ((f.film_id = fc.film_id)))
     JOIN category c ON ((fc.category_id = c.category_id)))
  GROUP BY c.name
  ORDER BY (sum(p.amount)) DESC;


--
-- Name: sales_by_store; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW sales_by_store AS
 SELECT ((c.city || ','::text) || cy.country) AS store,
    ((m.first_name || ' '::text) || m.last_name) AS manager,
    sum(p.amount) AS total_sales
   FROM (((((((payment p
     JOIN rental r ON ((p.rental_id = r.rental_id)))
     JOIN inventory i ON ((r.inventory_id = i.inventory_id)))
     JOIN store s ON ((i.store_id = s.store_id)))
     JOIN address a ON ((s.address_id = a.address_id)))
     JOIN city c ON ((a.city_id = c.city_id)))
     JOIN country cy ON ((c.country_id = cy.country_id)))
     JOIN staff m ON ((s.manager_staff_id = m.staff_id)))
  GROUP BY cy.country, c.city, s.store_id, m.first_name, m.last_name
  ORDER BY cy.country, c.city;


--
-- Name: staff; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE staff (
    staff_id SERIAL PRIMARY KEY,
    first_name text NOT NULL,
    last_name text NOT NULL,
    address_id integer NOT NULL,
    email text,
    store_id integer NOT NULL,
    active boolean DEFAULT true NOT NULL,
    username text NOT NULL,
    password text,
    last_update timestamp with time zone DEFAULT now() NOT NULL,
    picture bytea
);


--
-- Name: sales_by_store; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW sales_by_store AS
 SELECT ((c.city || ','::text) || cy.country) AS store,
    ((m.first_name || ' '::text) || m.last_name) AS manager,
    sum(p.amount) AS total_sales
   FROM (((((((payment p
     JOIN rental r ON ((p.rental_id = r.rental_id)))
     JOIN inventory i ON ((r.inventory_id = i.inventory_id)))
     JOIN store s ON ((i.store_id = s.store_id)))
     JOIN address a ON ((s.address_id = a.address_id)))
     JOIN city c ON ((a.city_id = c.city_id)))
     JOIN country cy ON ((c.country_id = cy.country_id)))
     JOIN staff m ON ((s.manager_staff_id = m.staff_id)))
  GROUP BY cy.country, c.city, s.store_id, m.first_name, m.last_name
  ORDER BY cy.country, c.city;


--
-- Name: staff_list; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW staff_list AS
 SELECT s.staff_id AS id,
    ((s.first_name || ' '::text) || s.last_name) AS name,
    a.address,
    a.postal_code AS "zip code",
    a.phone,
    city.city,
    country.country,
    s.store_id AS sid
   FROM (((staff s
     JOIN address a ON ((s.address_id = a.address_id)))
     JOIN city ON ((a.city_id = city.city_id)))
     JOIN country ON ((city.country_id = country.country_id)));


--
-- Name: store; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE store (
    store_id SERIAL PRIMARY KEY,
    manager_staff_id integer NOT NULL,
    address_id integer NOT NULL,
    last_update timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: customer_list; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW customer_list AS
 SELECT cu.customer_id AS id,
    ((cu.first_name || ' '::text) || cu.last_name) AS name,
    a.address,
    a.postal_code AS "zip code",
    a.phone,
    city.city,
    country.country,
        CASE
            WHEN cu.activebool THEN 'active'::text
            ELSE ''::text
        END AS notes,
    cu.store_id AS sid
   FROM (((customer cu
     JOIN address a ON ((cu.address_id = a.address_id)))
     JOIN city ON ((a.city_id = city.city_id)))
     JOIN country ON ((city.country_id = country.country_id)));


--
-- Name: sales_by_store; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW sales_by_store AS
 SELECT ((c.city || ','::text) || cy.country) AS store,
    ((m.first_name || ' '::text) || m.last_name) AS manager,
    sum(p.amount) AS total_sales
   FROM (((((((payment p
     JOIN rental r ON ((p.rental_id = r.rental_id)))
     JOIN inventory i ON ((r.inventory_id = i.inventory_id)))
     JOIN store s ON ((i.store_id = s.store_id)))
     JOIN address a ON ((s.address_id = a.address_id)))
     JOIN city c ON ((a.city_id = c.city_id)))
     JOIN country cy ON ((c.country_id = cy.country_id)))
     JOIN staff m ON ((s.manager_staff_id = m.staff_id)))
  GROUP BY cy.country, c.city, s.store_id, m.first_name, m.last_name
  ORDER BY cy.country, c.city;


--
-- Name: staff_list; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW staff_list AS
 SELECT s.staff_id AS id,
    ((s.first_name || ' '::text) || s.last_name) AS name,
    a.address,
    a.postal_code AS "zip code",
    a.phone,
    city.city,
    country.country,
    s.store_id AS sid
   FROM (((staff s
     JOIN address a ON ((s.address_id = a.address_id)))
     JOIN city ON ((a.city_id = city.city_id)))
     JOIN country ON ((city.country_id = country.country_id)));


--
-- Name: payment_p2022_01; Type: TABLE ATTACH; Schema: -; Owner: -
--

ALTER TABLE ONLY payment ATTACH PARTITION payment_p2022_01 FOR VALUES FROM ('2022-01-01 00:00:00+00') TO ('2022-02-01 00:00:00+00');


--
-- Name: payment_p2022_02; Type: TABLE ATTACH; Schema: -; Owner: -
--

ALTER TABLE ONLY payment ATTACH PARTITION payment_p2022_02 FOR VALUES FROM ('2022-02-01 00:00:00+00') TO ('2022-03-01 00:00:00+00');


--
-- Name: payment_p2022_03; Type: TABLE ATTACH; Schema: -; Owner: -
--

ALTER TABLE ONLY payment ATTACH PARTITION payment_p2022_03 FOR VALUES FROM ('2022-03-01 00:00:00+00') TO ('2022-04-01 00:00:00+00');


--
-- Name: payment_p2022_04; Type: TABLE ATTACH; Schema: -; Owner: -
--

ALTER TABLE ONLY payment ATTACH PARTITION payment_p2022_04 FOR VALUES FROM ('2022-04-01 00:00:00+00') TO ('2022-05-01 00:00:00+00');


--
-- Name: payment_p2022_05; Type: TABLE ATTACH; Schema: -; Owner: -
--

ALTER TABLE ONLY payment ATTACH PARTITION payment_p2022_05 FOR VALUES FROM ('2022-05-01 00:00:00+00') TO ('2022-06-01 00:00:00+00');


--
-- Name: payment_p2022_06; Type: TABLE ATTACH; Schema: -; Owner: -
--

ALTER TABLE ONLY payment ATTACH PARTITION payment_p2022_06 FOR VALUES FROM ('2022-06-01 00:00:00+00') TO ('2022-07-01 00:00:00+00');


--
-- Name: payment_p2022_07; Type: TABLE ATTACH; Schema: -; Owner: -
--

ALTER TABLE ONLY payment ATTACH PARTITION payment_p2022_07 FOR VALUES FROM ('2022-07-01 00:00:00+00') TO ('2022-08-01 00:00:00+00');


--
-- Name: actor actor_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY actor
    ADD CONSTRAINT actor_pkey PRIMARY KEY (actor_id);


--
-- Name: address address_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY address
    ADD CONSTRAINT address_pkey PRIMARY KEY (address_id);


--
-- Name: category category_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY category
    ADD CONSTRAINT category_pkey PRIMARY KEY (category_id);


--
-- Name: city city_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY city
    ADD CONSTRAINT city_pkey PRIMARY KEY (city_id);


--
-- Name: country country_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY country
    ADD CONSTRAINT country_pkey PRIMARY KEY (country_id);


--
-- Name: customer customer_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY customer
    ADD CONSTRAINT customer_pkey PRIMARY KEY (customer_id);


--
-- Name: film film_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY film
    ADD CONSTRAINT film_pkey PRIMARY KEY (film_id);


--
-- Name: film_actor film_actor_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY film_actor
    ADD CONSTRAINT film_actor_pkey PRIMARY KEY (actor_id, film_id);


--
-- Name: film_category film_category_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY film_category
    ADD CONSTRAINT film_category_pkey PRIMARY KEY (film_id, category_id);


--
-- Name: inventory inventory_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY inventory
    ADD CONSTRAINT inventory_pkey PRIMARY KEY (inventory_id);


--
-- Name: language language_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY language
    ADD CONSTRAINT language_pkey PRIMARY KEY (language_id);


--
-- Name: payment payment_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY payment
    ADD CONSTRAINT payment_pkey PRIMARY KEY (payment_date, payment_id);


--
-- Name: payment_p2022_01 payment_p2022_01_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY payment_p2022_01
    ADD CONSTRAINT payment_p2022_01_pkey PRIMARY KEY (payment_date, payment_id);


--
-- Name: payment_p2022_02 payment_p2022_02_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY payment_p2022_02
    ADD CONSTRAINT payment_p2022_02_pkey PRIMARY KEY (payment_date, payment_id);


--
-- Name: payment_p2022_03 payment_p2022_03_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY payment_p2022_03
    ADD CONSTRAINT payment_p2022_03_pkey PRIMARY KEY (payment_date, payment_id);


--
-- Name: payment_p2022_04 payment_p2022_04_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY payment_p2022_04
    ADD CONSTRAINT payment_p2022_04_pkey PRIMARY KEY (payment_date, payment_id);


--
-- Name: payment_p2022_05 payment_p2022_05_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY payment_p2022_05
    ADD CONSTRAINT payment_p2022_05_pkey PRIMARY KEY (payment_date, payment_id);


--
-- Name: payment_p2022_06 payment_p2022_06_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY payment_p2022_06
    ADD CONSTRAINT payment_p2022_06_pkey PRIMARY KEY (payment_date, payment_id);


--
-- Name: payment_p2022_07 payment_p2022_07_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY payment_p2022_07
    ADD CONSTRAINT payment_p2022_07_pkey PRIMARY KEY (payment_date, payment_id);


--
-- Name: rental rental_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY rental
    ADD CONSTRAINT rental_pkey PRIMARY KEY (rental_id);


--
-- Name: staff staff_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY staff
    ADD CONSTRAINT staff_pkey PRIMARY KEY (staff_id);


--
-- Name: store store_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY store
    ADD CONSTRAINT store_pkey PRIMARY KEY (store_id);


--
-- Name: film_fulltext_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX film_fulltext_idx ON film USING gist (fulltext);


--
-- Name: idx_actor_last_name; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_actor_last_name ON actor USING btree (last_name);


--
-- Name: idx_fk_address_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_fk_address_id ON customer USING btree (address_id);


--
-- Name: idx_fk_city_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_fk_city_id ON address USING btree (city_id);


--
-- Name: idx_fk_country_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_fk_country_id ON city USING btree (country_id);


--
-- Name: idx_fk_film_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_fk_film_id ON film_actor USING btree (film_id);


--
-- Name: idx_fk_inventory_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_fk_inventory_id ON rental USING btree (inventory_id);


--
-- Name: idx_fk_language_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_fk_language_id ON film USING btree (language_id);


--
-- Name: idx_fk_original_language_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_fk_original_language_id ON film USING btree (original_language_id);


--
-- Name: idx_fk_payment_p2022_01_customer_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_fk_payment_p2022_01_customer_id ON payment_p2022_01 USING btree (customer_id);


--
-- Name: idx_fk_payment_p2022_01_staff_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_fk_payment_p2022_01_staff_id ON payment_p2022_01 USING btree (staff_id);


--
-- Name: idx_fk_payment_p2022_02_customer_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_fk_payment_p2022_02_customer_id ON payment_p2022_02 USING btree (customer_id);


--
-- Name: idx_fk_payment_p2022_02_staff_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_fk_payment_p2022_02_staff_id ON payment_p2022_02 USING btree (staff_id);


--
-- Name: idx_fk_payment_p2022_03_customer_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_fk_payment_p2022_03_customer_id ON payment_p2022_03 USING btree (customer_id);


--
-- Name: idx_fk_payment_p2022_03_staff_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_fk_payment_p2022_03_staff_id ON payment_p2022_03 USING btree (staff_id);


--
-- Name: idx_fk_payment_p2022_04_customer_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_fk_payment_p2022_04_customer_id ON payment_p2022_04 USING btree (customer_id);


--
-- Name: idx_fk_payment_p2022_04_staff_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_fk_payment_p2022_04_staff_id ON payment_p2022_04 USING btree (staff_id);


--
-- Name: idx_fk_payment_p2022_05_customer_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_fk_payment_p2022_05_customer_id ON payment_p2022_05 USING btree (customer_id);


--
-- Name: idx_fk_payment_p2022_05_staff_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_fk_payment_p2022_05_staff_id ON payment_p2022_05 USING btree (staff_id);


--
-- Name: idx_fk_payment_p2022_06_customer_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_fk_payment_p2022_06_customer_id ON payment_p2022_06 USING btree (customer_id);


--
-- Name: idx_fk_payment_p2022_06_staff_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_fk_payment_p2022_06_staff_id ON payment_p2022_06 USING btree (staff_id);


--
-- Name: idx_fk_store_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_fk_store_id ON customer USING btree (store_id);


--
-- Name: idx_last_name; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_last_name ON customer USING btree (last_name);


--
-- Name: idx_store_id_film_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_store_id_film_id ON inventory USING btree (store_id, film_id);


--
-- Name: idx_title; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_title ON film USING btree (title);


--
-- Name: idx_unq_manager_staff_id; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX idx_unq_manager_staff_id ON store USING btree (manager_staff_id);


--
-- Name: idx_unq_rental_rental_date_inventory_id_customer_id; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX idx_unq_rental_rental_date_inventory_id_customer_id ON rental USING btree (rental_date, inventory_id, customer_id);


--
-- Name: payment_p2022_01_customer_id_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX payment_p2022_01_customer_id_idx ON payment_p2022_01 USING btree (customer_id);


--
-- Name: payment_p2022_02_customer_id_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX payment_p2022_02_customer_id_idx ON payment_p2022_02 USING btree (customer_id);


--
-- Name: payment_p2022_03_customer_id_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX payment_p2022_03_customer_id_idx ON payment_p2022_03 USING btree (customer_id);


--
-- Name: payment_p2022_04_customer_id_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX payment_p2022_04_customer_id_idx ON payment_p2022_04 USING btree (customer_id);


--
-- Name: payment_p2022_05_customer_id_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX payment_p2022_05_customer_id_idx ON payment_p2022_05 USING btree (customer_id);


--
-- Name: payment_p2022_06_customer_id_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX payment_p2022_06_customer_id_idx ON payment_p2022_06 USING btree (customer_id);


--
-- Name: rental_category; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX rental_category ON rental_by_category USING btree (category);


--
-- Name: payment_p2022_01_pkey; Type: INDEX ATTACH; Schema: -; Owner: -
--

ALTER INDEX payment_pkey ATTACH PARTITION payment_p2022_01_pkey;


--
-- Name: payment_p2022_02_pkey; Type: INDEX ATTACH; Schema: -; Owner: -
--

ALTER INDEX payment_pkey ATTACH PARTITION payment_p2022_02_pkey;


--
-- Name: payment_p2022_03_pkey; Type: INDEX ATTACH; Schema: -; Owner: -
--

ALTER INDEX payment_pkey ATTACH PARTITION payment_p2022_03_pkey;


--
-- Name: payment_p2022_04_pkey; Type: INDEX ATTACH; Schema: -; Owner: -
--

ALTER INDEX payment_pkey ATTACH PARTITION payment_p2022_04_pkey;


--
-- Name: payment_p2022_05_pkey; Type: INDEX ATTACH; Schema: -; Owner: -
--

ALTER INDEX payment_pkey ATTACH PARTITION payment_p2022_05_pkey;


--
-- Name: payment_p2022_06_pkey; Type: INDEX ATTACH; Schema: -; Owner: -
--

ALTER INDEX payment_pkey ATTACH PARTITION payment_p2022_06_pkey;


--
-- Name: payment_p2022_07_pkey; Type: INDEX ATTACH; Schema: -; Owner: -
--

ALTER INDEX payment_pkey ATTACH PARTITION payment_p2022_07_pkey;


--
-- Name: actor last_updated; Type: TRIGGER; Schema: -; Owner: -
--

CREATE TRIGGER last_updated BEFORE UPDATE ON actor FOR EACH ROW EXECUTE FUNCTION last_updated();


--
-- Name: address last_updated; Type: TRIGGER; Schema: -; Owner: -
--

CREATE TRIGGER last_updated BEFORE UPDATE ON address FOR EACH ROW EXECUTE FUNCTION last_updated();


--
-- Name: category last_updated; Type: TRIGGER; Schema: -; Owner: -
--

CREATE TRIGGER last_updated BEFORE UPDATE ON category FOR EACH ROW EXECUTE FUNCTION last_updated();


--
-- Name: city last_updated; Type: TRIGGER; Schema: -; Owner: -
--

CREATE TRIGGER last_updated BEFORE UPDATE ON city FOR EACH ROW EXECUTE FUNCTION last_updated();


--
-- Name: country last_updated; Type: TRIGGER; Schema: -; Owner: -
--

CREATE TRIGGER last_updated BEFORE UPDATE ON country FOR EACH ROW EXECUTE FUNCTION last_updated();


--
-- Name: customer last_updated; Type: TRIGGER; Schema: -; Owner: -
--

CREATE TRIGGER last_updated BEFORE UPDATE ON customer FOR EACH ROW EXECUTE FUNCTION last_updated();


--
-- Name: film film_fulltext_trigger; Type: TRIGGER; Schema: -; Owner: -
--

CREATE TRIGGER film_fulltext_trigger BEFORE INSERT OR UPDATE ON film FOR EACH ROW EXECUTE FUNCTION tsvector_update_trigger('fulltext', 'pg_catalog.english', 'title', 'description');


--
-- Name: film last_updated; Type: TRIGGER; Schema: -; Owner: -
--

CREATE TRIGGER last_updated BEFORE UPDATE ON film FOR EACH ROW EXECUTE FUNCTION last_updated();


--
-- Name: film_actor last_updated; Type: TRIGGER; Schema: -; Owner: -
--

CREATE TRIGGER last_updated BEFORE UPDATE ON film_actor FOR EACH ROW EXECUTE FUNCTION last_updated();


--
-- Name: film_category last_updated; Type: TRIGGER; Schema: -; Owner: -
--

CREATE TRIGGER last_updated BEFORE UPDATE ON film_category FOR EACH ROW EXECUTE FUNCTION last_updated();


--
-- Name: inventory last_updated; Type: TRIGGER; Schema: -; Owner: -
--

CREATE TRIGGER last_updated BEFORE UPDATE ON inventory FOR EACH ROW EXECUTE FUNCTION last_updated();


--
-- Name: language last_updated; Type: TRIGGER; Schema: -; Owner: -
--

CREATE TRIGGER last_updated BEFORE UPDATE ON language FOR EACH ROW EXECUTE FUNCTION last_updated();


--
-- Name: rental last_updated; Type: TRIGGER; Schema: -; Owner: -
--

CREATE TRIGGER last_updated BEFORE UPDATE ON rental FOR EACH ROW EXECUTE FUNCTION last_updated();


--
-- Name: staff last_updated; Type: TRIGGER; Schema: -; Owner: -
--

CREATE TRIGGER last_updated BEFORE UPDATE ON staff FOR EACH ROW EXECUTE FUNCTION last_updated();


--
-- Name: store last_updated; Type: TRIGGER; Schema: -; Owner: -
--

CREATE TRIGGER last_updated BEFORE UPDATE ON store FOR EACH ROW EXECUTE FUNCTION last_updated();


--
-- Name: address address_city_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY address
    ADD CONSTRAINT address_city_id_fkey FOREIGN KEY (city_id) REFERENCES city(city_id) ON UPDATE CASCADE ON DELETE RESTRICT;


--
-- Name: city city_country_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY city
    ADD CONSTRAINT city_country_id_fkey FOREIGN KEY (country_id) REFERENCES country(country_id) ON UPDATE CASCADE ON DELETE RESTRICT;


--
-- Name: customer customer_address_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY customer
    ADD CONSTRAINT customer_address_id_fkey FOREIGN KEY (address_id) REFERENCES address(address_id) ON UPDATE CASCADE ON DELETE RESTRICT;


--
-- Name: customer customer_store_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY customer
    ADD CONSTRAINT customer_store_id_fkey FOREIGN KEY (store_id) REFERENCES store(store_id) ON UPDATE CASCADE ON DELETE RESTRICT;


--
-- Name: film film_language_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY film
    ADD CONSTRAINT film_language_id_fkey FOREIGN KEY (language_id) REFERENCES language(language_id) ON UPDATE CASCADE ON DELETE RESTRICT;


--
-- Name: film film_original_language_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY film
    ADD CONSTRAINT film_original_language_id_fkey FOREIGN KEY (original_language_id) REFERENCES language(language_id) ON UPDATE CASCADE ON DELETE RESTRICT;


--
-- Name: film_actor film_actor_actor_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY film_actor
    ADD CONSTRAINT film_actor_actor_id_fkey FOREIGN KEY (actor_id) REFERENCES actor(actor_id) ON UPDATE CASCADE ON DELETE RESTRICT;


--
-- Name: film_actor film_actor_film_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY film_actor
    ADD CONSTRAINT film_actor_film_id_fkey FOREIGN KEY (film_id) REFERENCES film(film_id) ON UPDATE CASCADE ON DELETE RESTRICT;


--
-- Name: film_category film_category_category_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY film_category
    ADD CONSTRAINT film_category_category_id_fkey FOREIGN KEY (category_id) REFERENCES category(category_id) ON UPDATE CASCADE ON DELETE RESTRICT;


--
-- Name: film_category film_category_film_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY film_category
    ADD CONSTRAINT film_category_film_id_fkey FOREIGN KEY (film_id) REFERENCES film(film_id) ON UPDATE CASCADE ON DELETE RESTRICT;


--
-- Name: inventory inventory_film_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY inventory
    ADD CONSTRAINT inventory_film_id_fkey FOREIGN KEY (film_id) REFERENCES film(film_id) ON UPDATE CASCADE ON DELETE RESTRICT;


--
-- Name: inventory inventory_store_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY inventory
    ADD CONSTRAINT inventory_store_id_fkey FOREIGN KEY (store_id) REFERENCES store(store_id) ON UPDATE CASCADE ON DELETE RESTRICT;


--
-- Name: payment_p2022_01 payment_p2022_01_customer_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY payment_p2022_01
    ADD CONSTRAINT payment_p2022_01_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES customer(customer_id);


--
-- Name: payment_p2022_01 payment_p2022_01_rental_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY payment_p2022_01
    ADD CONSTRAINT payment_p2022_01_rental_id_fkey FOREIGN KEY (rental_id) REFERENCES rental(rental_id);


--
-- Name: payment_p2022_01 payment_p2022_01_staff_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY payment_p2022_01
    ADD CONSTRAINT payment_p2022_01_staff_id_fkey FOREIGN KEY (staff_id) REFERENCES staff(staff_id);


--
-- Name: payment_p2022_02 payment_p2022_02_customer_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY payment_p2022_02
    ADD CONSTRAINT payment_p2022_02_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES customer(customer_id);


--
-- Name: payment_p2022_02 payment_p2022_02_rental_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY payment_p2022_02
    ADD CONSTRAINT payment_p2022_02_rental_id_fkey FOREIGN KEY (rental_id) REFERENCES rental(rental_id);


--
-- Name: payment_p2022_02 payment_p2022_02_staff_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY payment_p2022_02
    ADD CONSTRAINT payment_p2022_02_staff_id_fkey FOREIGN KEY (staff_id) REFERENCES staff(staff_id);


--
-- Name: payment_p2022_03 payment_p2022_03_customer_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY payment_p2022_03
    ADD CONSTRAINT payment_p2022_03_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES customer(customer_id);


--
-- Name: payment_p2022_03 payment_p2022_03_rental_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY payment_p2022_03
    ADD CONSTRAINT payment_p2022_03_rental_id_fkey FOREIGN KEY (rental_id) REFERENCES rental(rental_id);


--
-- Name: payment_p2022_03 payment_p2022_03_staff_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY payment_p2022_03
    ADD CONSTRAINT payment_p2022_03_staff_id_fkey FOREIGN KEY (staff_id) REFERENCES staff(staff_id);


--
-- Name: payment_p2022_04 payment_p2022_04_customer_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY payment_p2022_04
    ADD CONSTRAINT payment_p2022_04_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES customer(customer_id);


--
-- Name: payment_p2022_04 payment_p2022_04_rental_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY payment_p2022_04
    ADD CONSTRAINT payment_p2022_04_rental_id_fkey FOREIGN KEY (rental_id) REFERENCES rental(rental_id);


--
-- Name: payment_p2022_04 payment_p2022_04_staff_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY payment_p2022_04
    ADD CONSTRAINT payment_p2022_04_staff_id_fkey FOREIGN KEY (staff_id) REFERENCES staff(staff_id);


--
-- Name: payment_p2022_05 payment_p2022_05_customer_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY payment_p2022_05
    ADD CONSTRAINT payment_p2022_05_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES customer(customer_id);


--
-- Name: payment_p2022_05 payment_p2022_05_rental_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY payment_p2022_05
    ADD CONSTRAINT payment_p2022_05_rental_id_fkey FOREIGN KEY (rental_id) REFERENCES rental(rental_id);


--
-- Name: payment_p2022_05 payment_p2022_05_staff_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY payment_p2022_05
    ADD CONSTRAINT payment_p2022_05_staff_id_fkey FOREIGN KEY (staff_id) REFERENCES staff(staff_id);


--
-- Name: payment_p2022_06 payment_p2022_06_customer_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY payment_p2022_06
    ADD CONSTRAINT payment_p2022_06_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES customer(customer_id);


--
-- Name: payment_p2022_06 payment_p2022_06_rental_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY payment_p2022_06
    ADD CONSTRAINT payment_p2022_06_rental_id_fkey FOREIGN KEY (rental_id) REFERENCES rental(rental_id);


--
-- Name: payment_p2022_06 payment_p2022_06_staff_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY payment_p2022_06
    ADD CONSTRAINT payment_p2022_06_staff_id_fkey FOREIGN KEY (staff_id) REFERENCES staff(staff_id);


--
-- Name: rental rental_customer_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY rental
    ADD CONSTRAINT rental_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES customer(customer_id) ON UPDATE CASCADE ON DELETE RESTRICT;


--
-- Name: rental rental_inventory_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY rental
    ADD CONSTRAINT rental_inventory_id_fkey FOREIGN KEY (inventory_id) REFERENCES inventory(inventory_id) ON UPDATE CASCADE ON DELETE RESTRICT;


--
-- Name: rental rental_staff_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY rental
    ADD CONSTRAINT rental_staff_id_fkey FOREIGN KEY (staff_id) REFERENCES staff(staff_id) ON UPDATE CASCADE ON DELETE RESTRICT;


--
-- Name: staff staff_address_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY staff
    ADD CONSTRAINT staff_address_id_fkey FOREIGN KEY (address_id) REFERENCES address(address_id) ON UPDATE CASCADE ON DELETE RESTRICT;


--
-- Name: staff staff_store_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY staff
    ADD CONSTRAINT staff_store_id_fkey FOREIGN KEY (store_id) REFERENCES store(store_id);


--
-- Name: store store_address_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY store
    ADD CONSTRAINT store_address_id_fkey FOREIGN KEY (address_id) REFERENCES address(address_id) ON UPDATE CASCADE ON DELETE RESTRICT;


--
-- PostgreSQL database dump complete
--

