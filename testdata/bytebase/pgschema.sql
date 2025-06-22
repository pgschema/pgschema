--
-- PostgreSQL database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 0.0.1


--
-- Name: audit_log_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.audit_log_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: changelist_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.changelist_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: changelog_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.changelog_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: data_source_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.data_source_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: db_group_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.db_group_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: db_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.db_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: db_schema_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.db_schema_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: export_archive_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.export_archive_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: idp_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.idp_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: instance_change_history_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.instance_change_history_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: instance_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.instance_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: issue_comment_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.issue_comment_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: issue_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.issue_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: pipeline_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.pipeline_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: plan_check_run_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.plan_check_run_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: plan_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.plan_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: policy_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.policy_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: principal_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.principal_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: project_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.project_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: project_webhook_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.project_webhook_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: query_history_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.query_history_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: release_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.release_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: revision_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.revision_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: risk_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.risk_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: role_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.role_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: setting_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.setting_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: sheet_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.sheet_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: sync_history_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.sync_history_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: task_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.task_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: task_run_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.task_run_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: task_run_log_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.task_run_log_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: worksheet_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.worksheet_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: worksheet_organizer_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.worksheet_organizer_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: audit_log; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.audit_log (
    id bigint DEFAULT nextval('audit_log_id_seq'::regclass) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: changelist; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.changelist (
    id integer DEFAULT nextval('changelist_id_seq'::regclass) NOT NULL,
    creator_id integer NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    project text NOT NULL,
    name text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: changelog; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.changelog (
    id bigint DEFAULT nextval('changelog_id_seq'::regclass) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    instance text NOT NULL,
    db_name text NOT NULL,
    status text NOT NULL,
    prev_sync_history_id bigint,
    sync_history_id bigint,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    CONSTRAINT changelog_status_check CHECK ((status = ANY (ARRAY['PENDING'::text, 'DONE'::text, 'FAILED'::text])))
);


--
-- Name: data_source; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.data_source (
    id integer DEFAULT nextval('data_source_id_seq'::regclass) NOT NULL,
    instance text NOT NULL,
    options jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: db; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.db (
    id integer DEFAULT nextval('db_id_seq'::regclass) NOT NULL,
    deleted boolean DEFAULT false NOT NULL,
    project text NOT NULL,
    instance text NOT NULL,
    name text NOT NULL,
    environment text,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: db_group; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.db_group (
    id bigint DEFAULT nextval('db_group_id_seq'::regclass) NOT NULL,
    project text NOT NULL,
    resource_id text NOT NULL,
    placeholder text DEFAULT ''::text NOT NULL,
    expression jsonb DEFAULT '{}'::jsonb NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: db_schema; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.db_schema (
    id integer DEFAULT nextval('db_schema_id_seq'::regclass) NOT NULL,
    instance text NOT NULL,
    db_name text NOT NULL,
    metadata json DEFAULT '{}'::json NOT NULL,
    raw_dump text DEFAULT ''::text NOT NULL,
    config jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: export_archive; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.export_archive (
    id integer DEFAULT nextval('export_archive_id_seq'::regclass) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    bytes bytea,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: idp; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.idp (
    id integer DEFAULT nextval('idp_id_seq'::regclass) NOT NULL,
    resource_id text NOT NULL,
    name text NOT NULL,
    domain text NOT NULL,
    type text NOT NULL,
    config jsonb DEFAULT '{}'::jsonb NOT NULL,
    CONSTRAINT idp_type_check CHECK ((type = ANY (ARRAY['OAUTH2'::text, 'OIDC'::text, 'LDAP'::text])))
);


--
-- Name: instance; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.instance (
    id integer DEFAULT nextval('instance_id_seq'::regclass) NOT NULL,
    deleted boolean DEFAULT false NOT NULL,
    environment text,
    resource_id text NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: instance_change_history; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.instance_change_history (
    id bigint DEFAULT nextval('instance_change_history_id_seq'::regclass) NOT NULL,
    version text NOT NULL
);


--
-- Name: issue; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.issue (
    id integer DEFAULT nextval('issue_id_seq'::regclass) NOT NULL,
    creator_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    project text NOT NULL,
    plan_id bigint,
    pipeline_id integer,
    name text NOT NULL,
    status text NOT NULL,
    type text NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    ts_vector tsvector,
    CONSTRAINT issue_status_check CHECK ((status = ANY (ARRAY['OPEN'::text, 'DONE'::text, 'CANCELED'::text])))
);


--
-- Name: issue_comment; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.issue_comment (
    id bigint DEFAULT nextval('issue_comment_id_seq'::regclass) NOT NULL,
    creator_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    issue_id integer NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: issue_subscriber; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.issue_subscriber (
    issue_id integer NOT NULL,
    subscriber_id integer NOT NULL
);


--
-- Name: pipeline; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.pipeline (
    id integer DEFAULT nextval('pipeline_id_seq'::regclass) NOT NULL,
    creator_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    project text NOT NULL,
    name text NOT NULL
);


--
-- Name: plan; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.plan (
    id bigint DEFAULT nextval('plan_id_seq'::regclass) NOT NULL,
    creator_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    project text NOT NULL,
    pipeline_id integer,
    name text NOT NULL,
    description text NOT NULL,
    config jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: plan_check_run; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.plan_check_run (
    id integer DEFAULT nextval('plan_check_run_id_seq'::regclass) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    plan_id bigint NOT NULL,
    status text NOT NULL,
    type text NOT NULL,
    config jsonb DEFAULT '{}'::jsonb NOT NULL,
    result jsonb DEFAULT '{}'::jsonb NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    CONSTRAINT plan_check_run_status_check CHECK ((status = ANY (ARRAY['RUNNING'::text, 'DONE'::text, 'FAILED'::text, 'CANCELED'::text]))),
    CONSTRAINT plan_check_run_type_check CHECK ((type ~~ 'bb.plan-check.%'::text))
);


--
-- Name: policy; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.policy (
    id integer DEFAULT nextval('policy_id_seq'::regclass) NOT NULL,
    enforce boolean DEFAULT true NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    resource_type text NOT NULL,
    resource text NOT NULL,
    type text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    inherit_from_parent boolean DEFAULT true NOT NULL
);


--
-- Name: principal; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.principal (
    id integer DEFAULT nextval('principal_id_seq'::regclass) NOT NULL,
    deleted boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    type text NOT NULL,
    name text NOT NULL,
    email text NOT NULL,
    password_hash text NOT NULL,
    phone text DEFAULT ''::text NOT NULL,
    mfa_config jsonb DEFAULT '{}'::jsonb NOT NULL,
    profile jsonb DEFAULT '{}'::jsonb NOT NULL,
    CONSTRAINT principal_type_check CHECK ((type = ANY (ARRAY['END_USER'::text, 'SYSTEM_BOT'::text, 'SERVICE_ACCOUNT'::text])))
);


--
-- Name: project; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.project (
    id integer DEFAULT nextval('project_id_seq'::regclass) NOT NULL,
    deleted boolean DEFAULT false NOT NULL,
    name text NOT NULL,
    resource_id text NOT NULL,
    data_classification_config_id text DEFAULT ''::text NOT NULL,
    setting jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: project_webhook; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.project_webhook (
    id integer DEFAULT nextval('project_webhook_id_seq'::regclass) NOT NULL,
    project text NOT NULL,
    type text NOT NULL,
    name text NOT NULL,
    url text NOT NULL,
    event_list text[] NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    CONSTRAINT project_webhook_type_check CHECK ((type ~~ 'bb.plugin.webhook.%'::text))
);


--
-- Name: query_history; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.query_history (
    id bigint DEFAULT nextval('query_history_id_seq'::regclass) NOT NULL,
    creator_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    project_id text NOT NULL,
    database text NOT NULL,
    statement text NOT NULL,
    type text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: release; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.release (
    id bigint DEFAULT nextval('release_id_seq'::regclass) NOT NULL,
    deleted boolean DEFAULT false NOT NULL,
    project text NOT NULL,
    creator_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: review_config; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.review_config (
    id text NOT NULL,
    enabled boolean DEFAULT true NOT NULL,
    name text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: revision; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.revision (
    id bigint DEFAULT nextval('revision_id_seq'::regclass) NOT NULL,
    instance text NOT NULL,
    db_name text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    deleter_id integer,
    deleted_at timestamp with time zone,
    version text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: risk; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.risk (
    id bigint DEFAULT nextval('risk_id_seq'::regclass) NOT NULL,
    source text NOT NULL,
    level bigint NOT NULL,
    name text NOT NULL,
    active boolean NOT NULL,
    expression jsonb NOT NULL,
    CONSTRAINT risk_source_check CHECK ((source ~~ 'bb.risk.%'::text))
);


--
-- Name: role; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.role (
    id bigint DEFAULT nextval('role_id_seq'::regclass) NOT NULL,
    resource_id text NOT NULL,
    name text NOT NULL,
    description text NOT NULL,
    permissions jsonb DEFAULT '{}'::jsonb NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: setting; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.setting (
    id integer DEFAULT nextval('setting_id_seq'::regclass) NOT NULL,
    name text NOT NULL,
    value text NOT NULL
);


--
-- Name: sheet; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sheet (
    id integer DEFAULT nextval('sheet_id_seq'::regclass) NOT NULL,
    creator_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    project text NOT NULL,
    name text NOT NULL,
    sha256 bytea NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: sheet_blob; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sheet_blob (
    sha256 bytea NOT NULL,
    content text NOT NULL
);


--
-- Name: sync_history; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sync_history (
    id bigint DEFAULT nextval('sync_history_id_seq'::regclass) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    instance text NOT NULL,
    db_name text NOT NULL,
    metadata json DEFAULT '{}'::json NOT NULL,
    raw_dump text DEFAULT ''::text NOT NULL
);


--
-- Name: task; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.task (
    id integer DEFAULT nextval('task_id_seq'::regclass) NOT NULL,
    pipeline_id integer NOT NULL,
    instance text NOT NULL,
    environment text,
    db_name text,
    type text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: task_run; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.task_run (
    id integer DEFAULT nextval('task_run_id_seq'::regclass) NOT NULL,
    creator_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    task_id integer NOT NULL,
    sheet_id integer,
    attempt integer NOT NULL,
    status text NOT NULL,
    started_at timestamp with time zone,
    run_at timestamp with time zone,
    code integer DEFAULT 0 NOT NULL,
    result jsonb DEFAULT '{}'::jsonb NOT NULL,
    CONSTRAINT task_run_status_check CHECK ((status = ANY (ARRAY['PENDING'::text, 'RUNNING'::text, 'DONE'::text, 'FAILED'::text, 'CANCELED'::text])))
);


--
-- Name: task_run_log; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.task_run_log (
    id bigint DEFAULT nextval('task_run_log_id_seq'::regclass) NOT NULL,
    task_run_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: user_group; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_group (
    email text NOT NULL,
    name text NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: worksheet; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.worksheet (
    id integer DEFAULT nextval('worksheet_id_seq'::regclass) NOT NULL,
    creator_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    project text NOT NULL,
    instance text,
    db_name text,
    name text NOT NULL,
    statement text NOT NULL,
    visibility text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: worksheet_organizer; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.worksheet_organizer (
    id integer DEFAULT nextval('worksheet_organizer_id_seq'::regclass) NOT NULL,
    worksheet_id integer NOT NULL,
    principal_id integer NOT NULL,
    starred boolean DEFAULT false NOT NULL
);


--
-- Name: audit_log audit_log_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.audit_log
    ADD CONSTRAINT audit_log_pkey PRIMARY KEY (id);


--
-- Name: changelist changelist_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changelist
    ADD CONSTRAINT changelist_pkey PRIMARY KEY (id);


--
-- Name: changelog changelog_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changelog
    ADD CONSTRAINT changelog_pkey PRIMARY KEY (id);


--
-- Name: data_source data_source_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.data_source
    ADD CONSTRAINT data_source_pkey PRIMARY KEY (id);


--
-- Name: db db_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.db
    ADD CONSTRAINT db_pkey PRIMARY KEY (id);


--
-- Name: db_group db_group_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.db_group
    ADD CONSTRAINT db_group_pkey PRIMARY KEY (id);


--
-- Name: db_schema db_schema_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.db_schema
    ADD CONSTRAINT db_schema_pkey PRIMARY KEY (id);


--
-- Name: export_archive export_archive_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.export_archive
    ADD CONSTRAINT export_archive_pkey PRIMARY KEY (id);


--
-- Name: idp idp_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.idp
    ADD CONSTRAINT idp_pkey PRIMARY KEY (id);


--
-- Name: instance instance_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.instance
    ADD CONSTRAINT instance_pkey PRIMARY KEY (id);


--
-- Name: instance_change_history instance_change_history_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.instance_change_history
    ADD CONSTRAINT instance_change_history_pkey PRIMARY KEY (id);


--
-- Name: issue issue_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue
    ADD CONSTRAINT issue_pkey PRIMARY KEY (id);


--
-- Name: issue_comment issue_comment_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_comment
    ADD CONSTRAINT issue_comment_pkey PRIMARY KEY (id);


--
-- Name: issue_subscriber issue_subscriber_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_subscriber
    ADD CONSTRAINT issue_subscriber_pkey PRIMARY KEY (issue_id, subscriber_id);


--
-- Name: pipeline pipeline_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pipeline
    ADD CONSTRAINT pipeline_pkey PRIMARY KEY (id);


--
-- Name: plan plan_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan
    ADD CONSTRAINT plan_pkey PRIMARY KEY (id);


--
-- Name: plan_check_run plan_check_run_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan_check_run
    ADD CONSTRAINT plan_check_run_pkey PRIMARY KEY (id);


--
-- Name: policy policy_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.policy
    ADD CONSTRAINT policy_pkey PRIMARY KEY (id);


--
-- Name: principal principal_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.principal
    ADD CONSTRAINT principal_pkey PRIMARY KEY (id);


--
-- Name: project project_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project
    ADD CONSTRAINT project_pkey PRIMARY KEY (id);


--
-- Name: project_webhook project_webhook_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_webhook
    ADD CONSTRAINT project_webhook_pkey PRIMARY KEY (id);


--
-- Name: query_history query_history_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.query_history
    ADD CONSTRAINT query_history_pkey PRIMARY KEY (id);


--
-- Name: release release_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.release
    ADD CONSTRAINT release_pkey PRIMARY KEY (id);


--
-- Name: review_config review_config_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.review_config
    ADD CONSTRAINT review_config_pkey PRIMARY KEY (id);


--
-- Name: revision revision_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.revision
    ADD CONSTRAINT revision_pkey PRIMARY KEY (id);


--
-- Name: risk risk_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.risk
    ADD CONSTRAINT risk_pkey PRIMARY KEY (id);


--
-- Name: role role_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role
    ADD CONSTRAINT role_pkey PRIMARY KEY (id);


--
-- Name: setting setting_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.setting
    ADD CONSTRAINT setting_pkey PRIMARY KEY (id);


--
-- Name: sheet sheet_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sheet
    ADD CONSTRAINT sheet_pkey PRIMARY KEY (id);


--
-- Name: sheet_blob sheet_blob_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sheet_blob
    ADD CONSTRAINT sheet_blob_pkey PRIMARY KEY (sha256);


--
-- Name: sync_history sync_history_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sync_history
    ADD CONSTRAINT sync_history_pkey PRIMARY KEY (id);


--
-- Name: task task_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task
    ADD CONSTRAINT task_pkey PRIMARY KEY (id);


--
-- Name: task_run task_run_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_run
    ADD CONSTRAINT task_run_pkey PRIMARY KEY (id);


--
-- Name: task_run_log task_run_log_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_run_log
    ADD CONSTRAINT task_run_log_pkey PRIMARY KEY (id);


--
-- Name: user_group user_group_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_group
    ADD CONSTRAINT user_group_pkey PRIMARY KEY (email);


--
-- Name: worksheet worksheet_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.worksheet
    ADD CONSTRAINT worksheet_pkey PRIMARY KEY (id);


--
-- Name: worksheet_organizer worksheet_organizer_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.worksheet_organizer
    ADD CONSTRAINT worksheet_organizer_pkey PRIMARY KEY (id);


--
-- Name: idx_audit_log_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_log_created_at ON public.audit_log USING btree (created_at);


--
-- Name: idx_audit_log_payload_method; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_log_payload_method ON public.audit_log USING btree (((payload ->> 'method'::text)));


--
-- Name: idx_audit_log_payload_parent; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_log_payload_parent ON public.audit_log USING btree (((payload ->> 'parent'::text)));


--
-- Name: idx_audit_log_payload_resource; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_log_payload_resource ON public.audit_log USING btree (((payload ->> 'resource'::text)));


--
-- Name: idx_audit_log_payload_user; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_log_payload_user ON public.audit_log USING btree (((payload ->> 'user'::text)));


--
-- Name: idx_changelist_project_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_changelist_project_name ON public.changelist USING btree (project, name);


--
-- Name: idx_changelog_instance_db_name; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_changelog_instance_db_name ON public.changelog USING btree (instance, db_name);


--
-- Name: idx_db_group_unique_project_placeholder; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_db_group_unique_project_placeholder ON public.db_group USING btree (project, placeholder);


--
-- Name: idx_db_group_unique_project_resource_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_db_group_unique_project_resource_id ON public.db_group USING btree (project, resource_id);


--
-- Name: idx_db_project; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_db_project ON public.db USING btree (project);


--
-- Name: idx_db_schema_unique_instance_db_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_db_schema_unique_instance_db_name ON public.db_schema USING btree (instance, db_name);


--
-- Name: idx_db_unique_instance_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_db_unique_instance_name ON public.db USING btree (instance, name);


--
-- Name: idx_idp_unique_resource_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_idp_unique_resource_id ON public.idp USING btree (resource_id);


--
-- Name: idx_instance_change_history_unique_version; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_instance_change_history_unique_version ON public.instance_change_history USING btree (version);


--
-- Name: idx_instance_unique_resource_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_instance_unique_resource_id ON public.instance USING btree (resource_id);


--
-- Name: idx_issue_comment_issue_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_issue_comment_issue_id ON public.issue_comment USING btree (issue_id);


--
-- Name: idx_issue_creator_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_issue_creator_id ON public.issue USING btree (creator_id);


--
-- Name: idx_issue_pipeline_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_issue_pipeline_id ON public.issue USING btree (pipeline_id);


--
-- Name: idx_issue_plan_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_issue_plan_id ON public.issue USING btree (plan_id);


--
-- Name: idx_issue_project; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_issue_project ON public.issue USING btree (project);


--
-- Name: idx_issue_subscriber_subscriber_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_issue_subscriber_subscriber_id ON public.issue_subscriber USING btree (subscriber_id);


--
-- Name: idx_issue_ts_vector; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_issue_ts_vector ON public.issue USING gin (ts_vector);


--
-- Name: idx_plan_check_run_plan_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_plan_check_run_plan_id ON public.plan_check_run USING btree (plan_id);


--
-- Name: idx_plan_pipeline_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_plan_pipeline_id ON public.plan USING btree (pipeline_id);


--
-- Name: idx_plan_project; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_plan_project ON public.plan USING btree (project);


--
-- Name: idx_policy_unique_resource_type_resource_type; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_policy_unique_resource_type_resource_type ON public.policy USING btree (resource_type, resource, type);


--
-- Name: idx_project_unique_resource_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_project_unique_resource_id ON public.project USING btree (resource_id);


--
-- Name: idx_project_webhook_project; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_project_webhook_project ON public.project_webhook USING btree (project);


--
-- Name: idx_query_history_creator_id_created_at_project_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_query_history_creator_id_created_at_project_id ON public.query_history USING btree (creator_id, created_at, project_id DESC);


--
-- Name: idx_release_project; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_release_project ON public.release USING btree (project);


--
-- Name: idx_revision_instance_db_name_version; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_revision_instance_db_name_version ON public.revision USING btree (instance, db_name, version);


--
-- Name: idx_revision_unique_instance_db_name_version_deleted_at_null; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_revision_unique_instance_db_name_version_deleted_at_null ON public.revision USING btree (instance, db_name, version) WHERE (deleted_at IS NULL);


--
-- Name: idx_role_unique_resource_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_role_unique_resource_id ON public.role USING btree (resource_id);


--
-- Name: idx_setting_unique_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_setting_unique_name ON public.setting USING btree (name);


--
-- Name: idx_sheet_project; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sheet_project ON public.sheet USING btree (project);


--
-- Name: idx_sync_history_instance_db_name_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sync_history_instance_db_name_created_at ON public.sync_history USING btree (instance, db_name, created_at);


--
-- Name: idx_task_pipeline_id_environment; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_task_pipeline_id_environment ON public.task USING btree (pipeline_id, environment);


--
-- Name: idx_task_run_log_task_run_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_task_run_log_task_run_id ON public.task_run_log USING btree (task_run_id);


--
-- Name: idx_task_run_task_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_task_run_task_id ON public.task_run USING btree (task_id);


--
-- Name: idx_worksheet_creator_id_project; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_worksheet_creator_id_project ON public.worksheet USING btree (creator_id, project);


--
-- Name: idx_worksheet_organizer_principal_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_worksheet_organizer_principal_id ON public.worksheet_organizer USING btree (principal_id);


--
-- Name: idx_worksheet_organizer_unique_sheet_id_principal_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_worksheet_organizer_unique_sheet_id_principal_id ON public.worksheet_organizer USING btree (worksheet_id, principal_id);


--
-- Name: uk_task_run_task_id_attempt; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uk_task_run_task_id_attempt ON public.task_run USING btree (task_id, attempt);


--
-- Name: changelist changelist_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changelist
    ADD CONSTRAINT changelist_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: changelist changelist_project_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changelist
    ADD CONSTRAINT changelist_project_fkey FOREIGN KEY (project) REFERENCES public.project(resource_id);


--
-- Name: changelog changelog_instance_db_name_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changelog
    ADD CONSTRAINT changelog_instance_db_name_fkey FOREIGN KEY (instance, db_name) REFERENCES public.db(instance, name);


--
-- Name: changelog changelog_prev_sync_history_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changelog
    ADD CONSTRAINT changelog_prev_sync_history_id_fkey FOREIGN KEY (prev_sync_history_id) REFERENCES public.sync_history(id);


--
-- Name: changelog changelog_sync_history_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changelog
    ADD CONSTRAINT changelog_sync_history_id_fkey FOREIGN KEY (sync_history_id) REFERENCES public.sync_history(id);


--
-- Name: data_source data_source_instance_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.data_source
    ADD CONSTRAINT data_source_instance_fkey FOREIGN KEY (instance) REFERENCES public.instance(resource_id);


--
-- Name: db db_instance_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.db
    ADD CONSTRAINT db_instance_fkey FOREIGN KEY (instance) REFERENCES public.instance(resource_id);


--
-- Name: db db_project_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.db
    ADD CONSTRAINT db_project_fkey FOREIGN KEY (project) REFERENCES public.project(resource_id);


--
-- Name: db_group db_group_project_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.db_group
    ADD CONSTRAINT db_group_project_fkey FOREIGN KEY (project) REFERENCES public.project(resource_id);


--
-- Name: db_schema db_schema_instance_db_name_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.db_schema
    ADD CONSTRAINT db_schema_instance_db_name_fkey FOREIGN KEY (instance, db_name) REFERENCES public.db(instance, name);


--
-- Name: issue issue_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue
    ADD CONSTRAINT issue_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: issue issue_pipeline_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue
    ADD CONSTRAINT issue_pipeline_id_fkey FOREIGN KEY (pipeline_id) REFERENCES public.pipeline(id);


--
-- Name: issue issue_plan_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue
    ADD CONSTRAINT issue_plan_id_fkey FOREIGN KEY (plan_id) REFERENCES public.plan(id);


--
-- Name: issue issue_project_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue
    ADD CONSTRAINT issue_project_fkey FOREIGN KEY (project) REFERENCES public.project(resource_id);


--
-- Name: issue_comment issue_comment_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_comment
    ADD CONSTRAINT issue_comment_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: issue_comment issue_comment_issue_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_comment
    ADD CONSTRAINT issue_comment_issue_id_fkey FOREIGN KEY (issue_id) REFERENCES public.issue(id);


--
-- Name: issue_subscriber issue_subscriber_issue_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_subscriber
    ADD CONSTRAINT issue_subscriber_issue_id_fkey FOREIGN KEY (issue_id) REFERENCES public.issue(id);


--
-- Name: issue_subscriber issue_subscriber_subscriber_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_subscriber
    ADD CONSTRAINT issue_subscriber_subscriber_id_fkey FOREIGN KEY (subscriber_id) REFERENCES public.principal(id);


--
-- Name: pipeline pipeline_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pipeline
    ADD CONSTRAINT pipeline_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: pipeline pipeline_project_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pipeline
    ADD CONSTRAINT pipeline_project_fkey FOREIGN KEY (project) REFERENCES public.project(resource_id);


--
-- Name: plan plan_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan
    ADD CONSTRAINT plan_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: plan plan_pipeline_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan
    ADD CONSTRAINT plan_pipeline_id_fkey FOREIGN KEY (pipeline_id) REFERENCES public.pipeline(id);


--
-- Name: plan plan_project_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan
    ADD CONSTRAINT plan_project_fkey FOREIGN KEY (project) REFERENCES public.project(resource_id);


--
-- Name: plan_check_run plan_check_run_plan_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan_check_run
    ADD CONSTRAINT plan_check_run_plan_id_fkey FOREIGN KEY (plan_id) REFERENCES public.plan(id);


--
-- Name: project_webhook project_webhook_project_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_webhook
    ADD CONSTRAINT project_webhook_project_fkey FOREIGN KEY (project) REFERENCES public.project(resource_id);


--
-- Name: query_history query_history_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.query_history
    ADD CONSTRAINT query_history_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: release release_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.release
    ADD CONSTRAINT release_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: release release_project_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.release
    ADD CONSTRAINT release_project_fkey FOREIGN KEY (project) REFERENCES public.project(resource_id);


--
-- Name: revision revision_deleter_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.revision
    ADD CONSTRAINT revision_deleter_id_fkey FOREIGN KEY (deleter_id) REFERENCES public.principal(id);


--
-- Name: revision revision_instance_db_name_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.revision
    ADD CONSTRAINT revision_instance_db_name_fkey FOREIGN KEY (instance, db_name) REFERENCES public.db(instance, name);


--
-- Name: sheet sheet_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sheet
    ADD CONSTRAINT sheet_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: sheet sheet_project_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sheet
    ADD CONSTRAINT sheet_project_fkey FOREIGN KEY (project) REFERENCES public.project(resource_id);


--
-- Name: sync_history sync_history_instance_db_name_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sync_history
    ADD CONSTRAINT sync_history_instance_db_name_fkey FOREIGN KEY (instance, db_name) REFERENCES public.db(instance, name);


--
-- Name: task task_instance_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task
    ADD CONSTRAINT task_instance_fkey FOREIGN KEY (instance) REFERENCES public.instance(resource_id);


--
-- Name: task task_pipeline_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task
    ADD CONSTRAINT task_pipeline_id_fkey FOREIGN KEY (pipeline_id) REFERENCES public.pipeline(id);


--
-- Name: task_run task_run_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_run
    ADD CONSTRAINT task_run_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: task_run task_run_sheet_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_run
    ADD CONSTRAINT task_run_sheet_id_fkey FOREIGN KEY (sheet_id) REFERENCES public.sheet(id);


--
-- Name: task_run task_run_task_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_run
    ADD CONSTRAINT task_run_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.task(id);


--
-- Name: task_run_log task_run_log_task_run_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_run_log
    ADD CONSTRAINT task_run_log_task_run_id_fkey FOREIGN KEY (task_run_id) REFERENCES public.task_run(id);


--
-- Name: worksheet worksheet_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.worksheet
    ADD CONSTRAINT worksheet_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.principal(id);


--
-- Name: worksheet worksheet_project_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.worksheet
    ADD CONSTRAINT worksheet_project_fkey FOREIGN KEY (project) REFERENCES public.project(resource_id);


--
-- Name: worksheet_organizer worksheet_organizer_principal_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.worksheet_organizer
    ADD CONSTRAINT worksheet_organizer_principal_id_fkey FOREIGN KEY (principal_id) REFERENCES public.principal(id);


--
-- Name: worksheet_organizer worksheet_organizer_worksheet_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.worksheet_organizer
    ADD CONSTRAINT worksheet_organizer_worksheet_id_fkey FOREIGN KEY (worksheet_id) REFERENCES public.worksheet(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

