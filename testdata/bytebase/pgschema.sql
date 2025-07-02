--
-- PostgreSQL database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 0.1.2




--
-- Name: audit_log; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE audit_log (
    id BIGSERIAL PRIMARY KEY,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: changelist; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE changelist (
    id SERIAL PRIMARY KEY,
    creator_id integer NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    project text NOT NULL,
    name text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: changelog; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE changelog (
    id BIGSERIAL PRIMARY KEY,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    instance text NOT NULL,
    db_name text NOT NULL,
    status text NOT NULL CHECK (status IN('PENDING', 'DONE', 'FAILED')),
    prev_sync_history_id bigint,
    sync_history_id bigint,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: data_source; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE data_source (
    id SERIAL PRIMARY KEY,
    instance text NOT NULL,
    options jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: db; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE db (
    id SERIAL PRIMARY KEY,
    deleted boolean DEFAULT false NOT NULL,
    project text NOT NULL,
    instance text NOT NULL,
    name text NOT NULL,
    environment text,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: db_group; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE db_group (
    id BIGSERIAL PRIMARY KEY,
    project text NOT NULL,
    resource_id text NOT NULL,
    placeholder text DEFAULT ''::text NOT NULL,
    expression jsonb DEFAULT '{}'::jsonb NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: db_schema; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE db_schema (
    id SERIAL PRIMARY KEY,
    instance text NOT NULL,
    db_name text NOT NULL,
    metadata json DEFAULT '{}'::json NOT NULL,
    raw_dump text DEFAULT ''::text NOT NULL,
    config jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: export_archive; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE export_archive (
    id SERIAL PRIMARY KEY,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    bytes bytea,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: idp; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE idp (
    id SERIAL PRIMARY KEY,
    resource_id text NOT NULL,
    name text NOT NULL,
    domain text NOT NULL,
    type text NOT NULL CHECK (type IN('OAUTH2', 'OIDC', 'LDAP')),
    config jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: instance; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE instance (
    id SERIAL PRIMARY KEY,
    deleted boolean DEFAULT false NOT NULL,
    environment text,
    resource_id text NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: instance_change_history; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE instance_change_history (
    id BIGSERIAL PRIMARY KEY,
    version text NOT NULL
);


--
-- Name: issue; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE issue (
    id SERIAL PRIMARY KEY,
    creator_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    project text NOT NULL,
    plan_id bigint,
    pipeline_id integer,
    name text NOT NULL,
    status text NOT NULL CHECK (status IN('OPEN', 'DONE', 'CANCELED')),
    type text NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    ts_vector tsvector
);


--
-- Name: issue_comment; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE issue_comment (
    id BIGSERIAL PRIMARY KEY,
    creator_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    issue_id integer NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: issue_subscriber; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE issue_subscriber (
    issue_id integer NOT NULL,
    subscriber_id integer NOT NULL
);


--
-- Name: pipeline; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE pipeline (
    id SERIAL PRIMARY KEY,
    creator_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    project text NOT NULL,
    name text NOT NULL
);


--
-- Name: plan; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE plan (
    id BIGSERIAL PRIMARY KEY,
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
-- Name: plan_check_run; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE plan_check_run (
    id SERIAL PRIMARY KEY,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    plan_id bigint NOT NULL,
    status text NOT NULL CHECK (status IN('RUNNING', 'DONE', 'FAILED', 'CANCELED')),
    type text NOT NULL CHECK (type ~~ 'bb.plan-check.%'::text),
    config jsonb DEFAULT '{}'::jsonb NOT NULL,
    result jsonb DEFAULT '{}'::jsonb NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: policy; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE policy (
    id SERIAL PRIMARY KEY,
    enforce boolean DEFAULT true NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    resource_type text NOT NULL,
    resource text NOT NULL,
    type text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    inherit_from_parent boolean DEFAULT true NOT NULL
);


--
-- Name: principal; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE principal (
    id SERIAL PRIMARY KEY,
    deleted boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    type text NOT NULL CHECK (type IN('END_USER', 'SYSTEM_BOT', 'SERVICE_ACCOUNT')),
    name text NOT NULL,
    email text NOT NULL,
    password_hash text NOT NULL,
    phone text DEFAULT ''::text NOT NULL,
    mfa_config jsonb DEFAULT '{}'::jsonb NOT NULL,
    profile jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: project; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE project (
    id SERIAL PRIMARY KEY,
    deleted boolean DEFAULT false NOT NULL,
    name text NOT NULL,
    resource_id text NOT NULL,
    data_classification_config_id text DEFAULT ''::text NOT NULL,
    setting jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: project_webhook; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE project_webhook (
    id SERIAL PRIMARY KEY,
    project text NOT NULL,
    type text NOT NULL CHECK (type ~~ 'bb.plugin.webhook.%'::text),
    name text NOT NULL,
    url text NOT NULL,
    event_list text[] NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: query_history; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE query_history (
    id BIGSERIAL PRIMARY KEY,
    creator_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    project_id text NOT NULL,
    database text NOT NULL,
    statement text NOT NULL,
    type text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: release; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE release (
    id BIGSERIAL PRIMARY KEY,
    deleted boolean DEFAULT false NOT NULL,
    project text NOT NULL,
    creator_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: review_config; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE review_config (
    id text PRIMARY KEY,
    enabled boolean DEFAULT true NOT NULL,
    name text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: revision; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE revision (
    id BIGSERIAL PRIMARY KEY,
    instance text NOT NULL,
    db_name text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    deleter_id integer,
    deleted_at timestamp with time zone,
    version text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: risk; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE risk (
    id BIGSERIAL PRIMARY KEY,
    source text NOT NULL CHECK (source ~~ 'bb.risk.%'::text),
    level bigint NOT NULL,
    name text NOT NULL,
    active boolean NOT NULL,
    expression jsonb NOT NULL
);


--
-- Name: role; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE role (
    id BIGSERIAL PRIMARY KEY,
    resource_id text NOT NULL,
    name text NOT NULL,
    description text NOT NULL,
    permissions jsonb DEFAULT '{}'::jsonb NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: setting; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE setting (
    id SERIAL PRIMARY KEY,
    name text NOT NULL,
    value text NOT NULL
);


--
-- Name: sheet; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE sheet (
    id SERIAL PRIMARY KEY,
    creator_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    project text NOT NULL,
    name text NOT NULL,
    sha256 bytea NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: sheet_blob; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE sheet_blob (
    sha256 bytea PRIMARY KEY,
    content text NOT NULL
);


--
-- Name: sync_history; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE sync_history (
    id BIGSERIAL PRIMARY KEY,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    instance text NOT NULL,
    db_name text NOT NULL,
    metadata json DEFAULT '{}'::json NOT NULL,
    raw_dump text DEFAULT ''::text NOT NULL
);


--
-- Name: task; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE task (
    id SERIAL PRIMARY KEY,
    pipeline_id integer NOT NULL,
    instance text NOT NULL,
    environment text,
    db_name text,
    type text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: task_run; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE task_run (
    id SERIAL PRIMARY KEY,
    creator_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    task_id integer NOT NULL,
    sheet_id integer,
    attempt integer NOT NULL,
    status text NOT NULL CHECK (status IN('PENDING', 'RUNNING', 'DONE', 'FAILED', 'CANCELED')),
    started_at timestamp with time zone,
    run_at timestamp with time zone,
    code integer DEFAULT 0 NOT NULL,
    result jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: task_run_log; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE task_run_log (
    id BIGSERIAL PRIMARY KEY,
    task_run_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: user_group; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE user_group (
    email text PRIMARY KEY,
    name text NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: worksheet; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE worksheet (
    id SERIAL PRIMARY KEY,
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
-- Name: worksheet_organizer; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE worksheet_organizer (
    id SERIAL PRIMARY KEY,
    worksheet_id integer NOT NULL,
    principal_id integer NOT NULL,
    starred boolean DEFAULT false NOT NULL
);


--
-- Name: audit_log audit_log_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY audit_log
    ADD CONSTRAINT audit_log_pkey PRIMARY KEY (id);


--
-- Name: changelist changelist_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY changelist
    ADD CONSTRAINT changelist_pkey PRIMARY KEY (id);


--
-- Name: changelog changelog_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY changelog
    ADD CONSTRAINT changelog_pkey PRIMARY KEY (id);


--
-- Name: data_source data_source_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY data_source
    ADD CONSTRAINT data_source_pkey PRIMARY KEY (id);


--
-- Name: db db_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY db
    ADD CONSTRAINT db_pkey PRIMARY KEY (id);


--
-- Name: db_group db_group_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY db_group
    ADD CONSTRAINT db_group_pkey PRIMARY KEY (id);


--
-- Name: db_schema db_schema_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY db_schema
    ADD CONSTRAINT db_schema_pkey PRIMARY KEY (id);


--
-- Name: export_archive export_archive_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY export_archive
    ADD CONSTRAINT export_archive_pkey PRIMARY KEY (id);


--
-- Name: idp idp_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY idp
    ADD CONSTRAINT idp_pkey PRIMARY KEY (id);


--
-- Name: instance instance_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY instance
    ADD CONSTRAINT instance_pkey PRIMARY KEY (id);


--
-- Name: instance_change_history instance_change_history_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY instance_change_history
    ADD CONSTRAINT instance_change_history_pkey PRIMARY KEY (id);


--
-- Name: issue issue_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY issue
    ADD CONSTRAINT issue_pkey PRIMARY KEY (id);


--
-- Name: issue_comment issue_comment_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY issue_comment
    ADD CONSTRAINT issue_comment_pkey PRIMARY KEY (id);


--
-- Name: issue_subscriber issue_subscriber_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY issue_subscriber
    ADD CONSTRAINT issue_subscriber_pkey PRIMARY KEY (issue_id, subscriber_id);


--
-- Name: pipeline pipeline_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY pipeline
    ADD CONSTRAINT pipeline_pkey PRIMARY KEY (id);


--
-- Name: plan plan_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY plan
    ADD CONSTRAINT plan_pkey PRIMARY KEY (id);


--
-- Name: plan_check_run plan_check_run_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY plan_check_run
    ADD CONSTRAINT plan_check_run_pkey PRIMARY KEY (id);


--
-- Name: policy policy_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY policy
    ADD CONSTRAINT policy_pkey PRIMARY KEY (id);


--
-- Name: principal principal_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY principal
    ADD CONSTRAINT principal_pkey PRIMARY KEY (id);


--
-- Name: project project_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY project
    ADD CONSTRAINT project_pkey PRIMARY KEY (id);


--
-- Name: project_webhook project_webhook_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY project_webhook
    ADD CONSTRAINT project_webhook_pkey PRIMARY KEY (id);


--
-- Name: query_history query_history_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY query_history
    ADD CONSTRAINT query_history_pkey PRIMARY KEY (id);


--
-- Name: release release_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY release
    ADD CONSTRAINT release_pkey PRIMARY KEY (id);


--
-- Name: review_config review_config_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY review_config
    ADD CONSTRAINT review_config_pkey PRIMARY KEY (id);


--
-- Name: revision revision_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY revision
    ADD CONSTRAINT revision_pkey PRIMARY KEY (id);


--
-- Name: risk risk_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY risk
    ADD CONSTRAINT risk_pkey PRIMARY KEY (id);


--
-- Name: role role_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY role
    ADD CONSTRAINT role_pkey PRIMARY KEY (id);


--
-- Name: setting setting_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY setting
    ADD CONSTRAINT setting_pkey PRIMARY KEY (id);


--
-- Name: sheet sheet_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY sheet
    ADD CONSTRAINT sheet_pkey PRIMARY KEY (id);


--
-- Name: sheet_blob sheet_blob_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY sheet_blob
    ADD CONSTRAINT sheet_blob_pkey PRIMARY KEY (sha256);


--
-- Name: sync_history sync_history_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY sync_history
    ADD CONSTRAINT sync_history_pkey PRIMARY KEY (id);


--
-- Name: task task_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY task
    ADD CONSTRAINT task_pkey PRIMARY KEY (id);


--
-- Name: task_run task_run_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY task_run
    ADD CONSTRAINT task_run_pkey PRIMARY KEY (id);


--
-- Name: task_run_log task_run_log_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY task_run_log
    ADD CONSTRAINT task_run_log_pkey PRIMARY KEY (id);


--
-- Name: user_group user_group_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY user_group
    ADD CONSTRAINT user_group_pkey PRIMARY KEY (email);


--
-- Name: worksheet worksheet_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY worksheet
    ADD CONSTRAINT worksheet_pkey PRIMARY KEY (id);


--
-- Name: worksheet_organizer worksheet_organizer_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY worksheet_organizer
    ADD CONSTRAINT worksheet_organizer_pkey PRIMARY KEY (id);


--
-- Name: idx_audit_log_created_at; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_audit_log_created_at ON audit_log USING btree (created_at);


--
-- Name: idx_audit_log_payload_method; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_audit_log_payload_method ON audit_log USING btree (((payload ->> 'method'::text)));


--
-- Name: idx_audit_log_payload_parent; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_audit_log_payload_parent ON audit_log USING btree (((payload ->> 'parent'::text)));


--
-- Name: idx_audit_log_payload_resource; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_audit_log_payload_resource ON audit_log USING btree (((payload ->> 'resource'::text)));


--
-- Name: idx_audit_log_payload_user; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_audit_log_payload_user ON audit_log USING btree (((payload ->> 'user'::text)));


--
-- Name: idx_changelist_project_name; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX idx_changelist_project_name ON changelist USING btree (project, name);


--
-- Name: idx_changelog_instance_db_name; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_changelog_instance_db_name ON changelog USING btree (instance, db_name);


--
-- Name: idx_db_group_unique_project_placeholder; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX idx_db_group_unique_project_placeholder ON db_group USING btree (project, placeholder);


--
-- Name: idx_db_group_unique_project_resource_id; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX idx_db_group_unique_project_resource_id ON db_group USING btree (project, resource_id);


--
-- Name: idx_db_project; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_db_project ON db USING btree (project);


--
-- Name: idx_db_schema_unique_instance_db_name; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX idx_db_schema_unique_instance_db_name ON db_schema USING btree (instance, db_name);


--
-- Name: idx_db_unique_instance_name; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX idx_db_unique_instance_name ON db USING btree (instance, name);


--
-- Name: idx_idp_unique_resource_id; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX idx_idp_unique_resource_id ON idp USING btree (resource_id);


--
-- Name: idx_instance_change_history_unique_version; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX idx_instance_change_history_unique_version ON instance_change_history USING btree (version);


--
-- Name: idx_instance_unique_resource_id; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX idx_instance_unique_resource_id ON instance USING btree (resource_id);


--
-- Name: idx_issue_comment_issue_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_issue_comment_issue_id ON issue_comment USING btree (issue_id);


--
-- Name: idx_issue_creator_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_issue_creator_id ON issue USING btree (creator_id);


--
-- Name: idx_issue_pipeline_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_issue_pipeline_id ON issue USING btree (pipeline_id);


--
-- Name: idx_issue_plan_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_issue_plan_id ON issue USING btree (plan_id);


--
-- Name: idx_issue_project; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_issue_project ON issue USING btree (project);


--
-- Name: idx_issue_subscriber_subscriber_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_issue_subscriber_subscriber_id ON issue_subscriber USING btree (subscriber_id);


--
-- Name: idx_issue_ts_vector; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_issue_ts_vector ON issue USING gin (ts_vector);


--
-- Name: idx_plan_check_run_plan_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_plan_check_run_plan_id ON plan_check_run USING btree (plan_id);


--
-- Name: idx_plan_pipeline_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_plan_pipeline_id ON plan USING btree (pipeline_id);


--
-- Name: idx_plan_project; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_plan_project ON plan USING btree (project);


--
-- Name: idx_policy_unique_resource_type_resource_type; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX idx_policy_unique_resource_type_resource_type ON policy USING btree (resource_type, resource, type);


--
-- Name: idx_project_unique_resource_id; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX idx_project_unique_resource_id ON project USING btree (resource_id);


--
-- Name: idx_project_webhook_project; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_project_webhook_project ON project_webhook USING btree (project);


--
-- Name: idx_query_history_creator_id_created_at_project_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_query_history_creator_id_created_at_project_id ON query_history USING btree (creator_id, created_at, project_id DESC);


--
-- Name: idx_release_project; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_release_project ON release USING btree (project);


--
-- Name: idx_revision_instance_db_name_version; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_revision_instance_db_name_version ON revision USING btree (instance, db_name, version);


--
-- Name: idx_revision_unique_instance_db_name_version_deleted_at_null; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX idx_revision_unique_instance_db_name_version_deleted_at_null ON revision USING btree (instance, db_name, version) WHERE (deleted_at IS NULL);


--
-- Name: idx_role_unique_resource_id; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX idx_role_unique_resource_id ON role USING btree (resource_id);


--
-- Name: idx_setting_unique_name; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX idx_setting_unique_name ON setting USING btree (name);


--
-- Name: idx_sheet_project; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_sheet_project ON sheet USING btree (project);


--
-- Name: idx_sync_history_instance_db_name_created_at; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_sync_history_instance_db_name_created_at ON sync_history USING btree (instance, db_name, created_at);


--
-- Name: idx_task_pipeline_id_environment; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_task_pipeline_id_environment ON task USING btree (pipeline_id, environment);


--
-- Name: idx_task_run_log_task_run_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_task_run_log_task_run_id ON task_run_log USING btree (task_run_id);


--
-- Name: idx_task_run_task_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_task_run_task_id ON task_run USING btree (task_id);


--
-- Name: idx_worksheet_creator_id_project; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_worksheet_creator_id_project ON worksheet USING btree (creator_id, project);


--
-- Name: idx_worksheet_organizer_principal_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_worksheet_organizer_principal_id ON worksheet_organizer USING btree (principal_id);


--
-- Name: idx_worksheet_organizer_unique_sheet_id_principal_id; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX idx_worksheet_organizer_unique_sheet_id_principal_id ON worksheet_organizer USING btree (worksheet_id, principal_id);


--
-- Name: uk_task_run_task_id_attempt; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX uk_task_run_task_id_attempt ON task_run USING btree (task_id, attempt);


--
-- Name: changelist changelist_creator_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY changelist
    ADD CONSTRAINT changelist_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES principal(id);


--
-- Name: changelist changelist_project_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY changelist
    ADD CONSTRAINT changelist_project_fkey FOREIGN KEY (project) REFERENCES project(resource_id);


--
-- Name: changelog changelog_instance_db_name_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY changelog
    ADD CONSTRAINT changelog_instance_db_name_fkey FOREIGN KEY (instance, db_name) REFERENCES db(instance, name);


--
-- Name: changelog changelog_prev_sync_history_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY changelog
    ADD CONSTRAINT changelog_prev_sync_history_id_fkey FOREIGN KEY (prev_sync_history_id) REFERENCES sync_history(id);


--
-- Name: changelog changelog_sync_history_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY changelog
    ADD CONSTRAINT changelog_sync_history_id_fkey FOREIGN KEY (sync_history_id) REFERENCES sync_history(id);


--
-- Name: data_source data_source_instance_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY data_source
    ADD CONSTRAINT data_source_instance_fkey FOREIGN KEY (instance) REFERENCES instance(resource_id);


--
-- Name: db db_instance_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY db
    ADD CONSTRAINT db_instance_fkey FOREIGN KEY (instance) REFERENCES instance(resource_id);


--
-- Name: db db_project_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY db
    ADD CONSTRAINT db_project_fkey FOREIGN KEY (project) REFERENCES project(resource_id);


--
-- Name: db_group db_group_project_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY db_group
    ADD CONSTRAINT db_group_project_fkey FOREIGN KEY (project) REFERENCES project(resource_id);


--
-- Name: db_schema db_schema_instance_db_name_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY db_schema
    ADD CONSTRAINT db_schema_instance_db_name_fkey FOREIGN KEY (instance, db_name) REFERENCES db(instance, name);


--
-- Name: issue issue_creator_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY issue
    ADD CONSTRAINT issue_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES principal(id);


--
-- Name: issue issue_pipeline_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY issue
    ADD CONSTRAINT issue_pipeline_id_fkey FOREIGN KEY (pipeline_id) REFERENCES pipeline(id);


--
-- Name: issue issue_plan_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY issue
    ADD CONSTRAINT issue_plan_id_fkey FOREIGN KEY (plan_id) REFERENCES plan(id);


--
-- Name: issue issue_project_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY issue
    ADD CONSTRAINT issue_project_fkey FOREIGN KEY (project) REFERENCES project(resource_id);


--
-- Name: issue_comment issue_comment_creator_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY issue_comment
    ADD CONSTRAINT issue_comment_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES principal(id);


--
-- Name: issue_comment issue_comment_issue_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY issue_comment
    ADD CONSTRAINT issue_comment_issue_id_fkey FOREIGN KEY (issue_id) REFERENCES issue(id);


--
-- Name: issue_subscriber issue_subscriber_issue_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY issue_subscriber
    ADD CONSTRAINT issue_subscriber_issue_id_fkey FOREIGN KEY (issue_id) REFERENCES issue(id);


--
-- Name: issue_subscriber issue_subscriber_subscriber_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY issue_subscriber
    ADD CONSTRAINT issue_subscriber_subscriber_id_fkey FOREIGN KEY (subscriber_id) REFERENCES principal(id);


--
-- Name: pipeline pipeline_creator_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY pipeline
    ADD CONSTRAINT pipeline_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES principal(id);


--
-- Name: pipeline pipeline_project_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY pipeline
    ADD CONSTRAINT pipeline_project_fkey FOREIGN KEY (project) REFERENCES project(resource_id);


--
-- Name: plan plan_creator_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY plan
    ADD CONSTRAINT plan_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES principal(id);


--
-- Name: plan plan_pipeline_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY plan
    ADD CONSTRAINT plan_pipeline_id_fkey FOREIGN KEY (pipeline_id) REFERENCES pipeline(id);


--
-- Name: plan plan_project_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY plan
    ADD CONSTRAINT plan_project_fkey FOREIGN KEY (project) REFERENCES project(resource_id);


--
-- Name: plan_check_run plan_check_run_plan_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY plan_check_run
    ADD CONSTRAINT plan_check_run_plan_id_fkey FOREIGN KEY (plan_id) REFERENCES plan(id);


--
-- Name: project_webhook project_webhook_project_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY project_webhook
    ADD CONSTRAINT project_webhook_project_fkey FOREIGN KEY (project) REFERENCES project(resource_id);


--
-- Name: query_history query_history_creator_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY query_history
    ADD CONSTRAINT query_history_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES principal(id);


--
-- Name: release release_creator_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY release
    ADD CONSTRAINT release_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES principal(id);


--
-- Name: release release_project_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY release
    ADD CONSTRAINT release_project_fkey FOREIGN KEY (project) REFERENCES project(resource_id);


--
-- Name: revision revision_deleter_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY revision
    ADD CONSTRAINT revision_deleter_id_fkey FOREIGN KEY (deleter_id) REFERENCES principal(id);


--
-- Name: revision revision_instance_db_name_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY revision
    ADD CONSTRAINT revision_instance_db_name_fkey FOREIGN KEY (instance, db_name) REFERENCES db(instance, name);


--
-- Name: sheet sheet_creator_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY sheet
    ADD CONSTRAINT sheet_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES principal(id);


--
-- Name: sheet sheet_project_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY sheet
    ADD CONSTRAINT sheet_project_fkey FOREIGN KEY (project) REFERENCES project(resource_id);


--
-- Name: sync_history sync_history_instance_db_name_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY sync_history
    ADD CONSTRAINT sync_history_instance_db_name_fkey FOREIGN KEY (instance, db_name) REFERENCES db(instance, name);


--
-- Name: task task_instance_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY task
    ADD CONSTRAINT task_instance_fkey FOREIGN KEY (instance) REFERENCES instance(resource_id);


--
-- Name: task task_pipeline_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY task
    ADD CONSTRAINT task_pipeline_id_fkey FOREIGN KEY (pipeline_id) REFERENCES pipeline(id);


--
-- Name: task_run task_run_creator_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY task_run
    ADD CONSTRAINT task_run_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES principal(id);


--
-- Name: task_run task_run_sheet_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY task_run
    ADD CONSTRAINT task_run_sheet_id_fkey FOREIGN KEY (sheet_id) REFERENCES sheet(id);


--
-- Name: task_run task_run_task_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY task_run
    ADD CONSTRAINT task_run_task_id_fkey FOREIGN KEY (task_id) REFERENCES task(id);


--
-- Name: task_run_log task_run_log_task_run_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY task_run_log
    ADD CONSTRAINT task_run_log_task_run_id_fkey FOREIGN KEY (task_run_id) REFERENCES task_run(id);


--
-- Name: worksheet worksheet_creator_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY worksheet
    ADD CONSTRAINT worksheet_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES principal(id);


--
-- Name: worksheet worksheet_project_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY worksheet
    ADD CONSTRAINT worksheet_project_fkey FOREIGN KEY (project) REFERENCES project(resource_id);


--
-- Name: worksheet_organizer worksheet_organizer_principal_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY worksheet_organizer
    ADD CONSTRAINT worksheet_organizer_principal_id_fkey FOREIGN KEY (principal_id) REFERENCES principal(id);


--
-- Name: worksheet_organizer worksheet_organizer_worksheet_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY worksheet_organizer
    ADD CONSTRAINT worksheet_organizer_worksheet_id_fkey FOREIGN KEY (worksheet_id) REFERENCES worksheet(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

