--
-- PostgreSQL database dump
--

-- Dumped from database version 12.6
-- Dumped by pg_dump version 12.6

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: potential_customers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.potential_customers (
    id bigint NOT NULL,
    email text NOT NULL,
    token text NOT NULL,
    token_verified boolean DEFAULT false,
    registration_date timestamp with time zone DEFAULT now()
);


--
-- Name: potential_customers_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.potential_customers_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: potential_customers_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.potential_customers_id_seq OWNED BY public.potential_customers.id;


--
-- Name: potential_customers id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.potential_customers ALTER COLUMN id SET DEFAULT nextval('public.potential_customers_id_seq'::regclass);


--
-- Name: potential_customers potential_customers_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.potential_customers
    ADD CONSTRAINT potential_customers_pk PRIMARY KEY (email);


--
-- Name: potential_customers_email_uindex; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX potential_customers_email_uindex ON public.potential_customers USING btree (email);


--
-- PostgreSQL database dump complete
--

