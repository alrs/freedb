--
-- PostgreSQL database dump
--

-- Dumped from database version 11.5 (Debian 11.5-1+deb10u1)
-- Dumped by pg_dump version 11.5 (Debian 11.5-1+deb10u1)

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

SET default_with_oids = false;

--
-- Name: discs; Type: TABLE; Schema: public; Owner: lars
--

CREATE TABLE public.discs (
    id bytea NOT NULL,
    title text NOT NULL
);


ALTER TABLE public.discs OWNER TO lars;

--
-- Name: tracks; Type: TABLE; Schema: public; Owner: lars
--

CREATE TABLE public.tracks (
    id integer NOT NULL,
    disc_id bytea NOT NULL,
    title text NOT NULL
);


ALTER TABLE public.tracks OWNER TO lars;

--
-- Name: tracks_id_seq; Type: SEQUENCE; Schema: public; Owner: lars
--

CREATE SEQUENCE public.tracks_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.tracks_id_seq OWNER TO lars;

--
-- Name: tracks_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: lars
--

ALTER SEQUENCE public.tracks_id_seq OWNED BY public.tracks.id;


--
-- Name: tracks id; Type: DEFAULT; Schema: public; Owner: lars
--

ALTER TABLE ONLY public.tracks ALTER COLUMN id SET DEFAULT nextval('public.tracks_id_seq'::regclass);


--
-- Name: discs discs_pkey; Type: CONSTRAINT; Schema: public; Owner: lars
--

ALTER TABLE ONLY public.discs
    ADD CONSTRAINT discs_pkey PRIMARY KEY (id);


--
-- Name: tracks tracks_pkey; Type: CONSTRAINT; Schema: public; Owner: lars
--

ALTER TABLE ONLY public.tracks
    ADD CONSTRAINT tracks_pkey PRIMARY KEY (id);


--
-- Name: tracks tracks_disc_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: lars
--

ALTER TABLE ONLY public.tracks
    ADD CONSTRAINT tracks_disc_id_fkey FOREIGN KEY (disc_id) REFERENCES public.discs(id);


--
-- PostgreSQL database dump complete
--

