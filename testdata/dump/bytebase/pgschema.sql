--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 1.3.0


--
-- Name: audit_log; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS audit_log (
    id BIGSERIAL,
    created_at timestamptz DEFAULT now() NOT NULL,
    payload jsonb DEFAULT '{}' NOT NULL,
    CONSTRAINT audit_log_pkey PRIMARY KEY (id)
);

--
-- Name: idx_audit_log_created_at; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_audit_log_created_at ON audit_log (created_at);

--
-- Name: idx_audit_log_payload_method; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_audit_log_payload_method ON audit_log (((payload->>'method')));

--
-- Name: idx_audit_log_payload_parent; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_audit_log_payload_parent ON audit_log (((payload->>'parent')));

--
-- Name: idx_audit_log_payload_resource; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_audit_log_payload_resource ON audit_log (((payload->>'resource')));

--
-- Name: idx_audit_log_payload_user; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_audit_log_payload_user ON audit_log (((payload->>'user')));

--
-- Name: export_archive; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS export_archive (
    id SERIAL,
    created_at timestamptz DEFAULT now() NOT NULL,
    bytes bytea,
    payload jsonb DEFAULT '{}' NOT NULL,
    CONSTRAINT export_archive_pkey PRIMARY KEY (id)
);

--
-- Name: idp; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS idp (
    id SERIAL,
    resource_id text NOT NULL,
    name text NOT NULL,
    domain text NOT NULL,
    type text NOT NULL,
    config jsonb DEFAULT '{}' NOT NULL,
    CONSTRAINT idp_pkey PRIMARY KEY (id),
    CONSTRAINT idp_type_check CHECK (type IN ('OAUTH2', 'OIDC', 'LDAP'))
);

--
-- Name: idx_idp_unique_resource_id; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX IF NOT EXISTS idx_idp_unique_resource_id ON idp (resource_id);

--
-- Name: instance; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS instance (
    id SERIAL,
    deleted boolean DEFAULT false NOT NULL,
    environment text,
    resource_id text NOT NULL,
    metadata jsonb DEFAULT '{}' NOT NULL,
    CONSTRAINT instance_pkey PRIMARY KEY (id)
);

--
-- Name: idx_instance_unique_resource_id; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX IF NOT EXISTS idx_instance_unique_resource_id ON instance (resource_id);

--
-- Name: data_source; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS data_source (
    id SERIAL,
    instance text NOT NULL,
    options jsonb DEFAULT '{}' NOT NULL,
    CONSTRAINT data_source_pkey PRIMARY KEY (id),
    CONSTRAINT data_source_instance_fkey FOREIGN KEY (instance) REFERENCES instance (resource_id)
);

--
-- Name: instance_change_history; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS instance_change_history (
    id BIGSERIAL,
    version text NOT NULL,
    CONSTRAINT instance_change_history_pkey PRIMARY KEY (id)
);

--
-- Name: idx_instance_change_history_unique_version; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX IF NOT EXISTS idx_instance_change_history_unique_version ON instance_change_history (version);

--
-- Name: policy; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS policy (
    id SERIAL,
    enforce boolean DEFAULT true NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    resource_type text NOT NULL,
    resource text NOT NULL,
    type text NOT NULL,
    payload jsonb DEFAULT '{}' NOT NULL,
    inherit_from_parent boolean DEFAULT true NOT NULL,
    CONSTRAINT policy_pkey PRIMARY KEY (id)
);

--
-- Name: idx_policy_unique_resource_type_resource_type; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX IF NOT EXISTS idx_policy_unique_resource_type_resource_type ON policy (resource_type, resource, type);

--
-- Name: principal; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS principal (
    id SERIAL,
    deleted boolean DEFAULT false NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    type text NOT NULL,
    name text NOT NULL,
    email text NOT NULL,
    password_hash text NOT NULL,
    phone text DEFAULT '' NOT NULL,
    mfa_config jsonb DEFAULT '{}' NOT NULL,
    profile jsonb DEFAULT '{}' NOT NULL,
    CONSTRAINT principal_pkey PRIMARY KEY (id),
    CONSTRAINT principal_type_check CHECK (type IN ('END_USER', 'SYSTEM_BOT', 'SERVICE_ACCOUNT'))
);

--
-- Name: project; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS project (
    id SERIAL,
    deleted boolean DEFAULT false NOT NULL,
    name text NOT NULL,
    resource_id text NOT NULL,
    data_classification_config_id text DEFAULT '' NOT NULL,
    setting jsonb DEFAULT '{}' NOT NULL,
    CONSTRAINT project_pkey PRIMARY KEY (id)
);

--
-- Name: idx_project_unique_resource_id; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX IF NOT EXISTS idx_project_unique_resource_id ON project (resource_id);

--
-- Name: changelist; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS changelist (
    id SERIAL,
    creator_id integer NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    project text NOT NULL,
    name text NOT NULL,
    payload jsonb DEFAULT '{}' NOT NULL,
    CONSTRAINT changelist_pkey PRIMARY KEY (id),
    CONSTRAINT changelist_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES principal (id),
    CONSTRAINT changelist_project_fkey FOREIGN KEY (project) REFERENCES project (resource_id)
);

--
-- Name: idx_changelist_project_name; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX IF NOT EXISTS idx_changelist_project_name ON changelist (project, name);

--
-- Name: db; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS db (
    id SERIAL,
    deleted boolean DEFAULT false NOT NULL,
    project text NOT NULL,
    instance text NOT NULL,
    name text NOT NULL,
    environment text,
    metadata jsonb DEFAULT '{}' NOT NULL,
    CONSTRAINT db_pkey PRIMARY KEY (id),
    CONSTRAINT db_instance_fkey FOREIGN KEY (instance) REFERENCES instance (resource_id),
    CONSTRAINT db_project_fkey FOREIGN KEY (project) REFERENCES project (resource_id)
);

--
-- Name: idx_db_project; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_db_project ON db (project);

--
-- Name: idx_db_unique_instance_name; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX IF NOT EXISTS idx_db_unique_instance_name ON db (instance, name);

--
-- Name: db_group; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS db_group (
    id BIGSERIAL,
    project text NOT NULL,
    resource_id text NOT NULL,
    placeholder text DEFAULT '' NOT NULL,
    expression jsonb DEFAULT '{}' NOT NULL,
    payload jsonb DEFAULT '{}' NOT NULL,
    CONSTRAINT db_group_pkey PRIMARY KEY (id),
    CONSTRAINT db_group_project_fkey FOREIGN KEY (project) REFERENCES project (resource_id)
);

--
-- Name: idx_db_group_unique_project_placeholder; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX IF NOT EXISTS idx_db_group_unique_project_placeholder ON db_group (project, placeholder);

--
-- Name: idx_db_group_unique_project_resource_id; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX IF NOT EXISTS idx_db_group_unique_project_resource_id ON db_group (project, resource_id);

--
-- Name: db_schema; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS db_schema (
    id SERIAL,
    instance text NOT NULL,
    db_name text NOT NULL,
    metadata json DEFAULT '{}' NOT NULL,
    raw_dump text DEFAULT '' NOT NULL,
    config jsonb DEFAULT '{}' NOT NULL,
    CONSTRAINT db_schema_pkey PRIMARY KEY (id),
    CONSTRAINT db_schema_instance_db_name_fkey FOREIGN KEY (instance, db_name) REFERENCES db (instance, name)
);

--
-- Name: idx_db_schema_unique_instance_db_name; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX IF NOT EXISTS idx_db_schema_unique_instance_db_name ON db_schema (instance, db_name);

--
-- Name: pipeline; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS pipeline (
    id SERIAL,
    creator_id integer NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    project text NOT NULL,
    name text NOT NULL,
    CONSTRAINT pipeline_pkey PRIMARY KEY (id),
    CONSTRAINT pipeline_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES principal (id),
    CONSTRAINT pipeline_project_fkey FOREIGN KEY (project) REFERENCES project (resource_id)
);

--
-- Name: plan; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS plan (
    id BIGSERIAL,
    creator_id integer NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    project text NOT NULL,
    pipeline_id integer,
    name text NOT NULL,
    description text NOT NULL,
    config jsonb DEFAULT '{}' NOT NULL,
    CONSTRAINT plan_pkey PRIMARY KEY (id),
    CONSTRAINT plan_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES principal (id),
    CONSTRAINT plan_pipeline_id_fkey FOREIGN KEY (pipeline_id) REFERENCES pipeline (id),
    CONSTRAINT plan_project_fkey FOREIGN KEY (project) REFERENCES project (resource_id)
);

--
-- Name: idx_plan_pipeline_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_plan_pipeline_id ON plan (pipeline_id);

--
-- Name: idx_plan_project; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_plan_project ON plan (project);

--
-- Name: issue; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS issue (
    id SERIAL,
    creator_id integer NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    project text NOT NULL,
    plan_id bigint,
    pipeline_id integer,
    name text NOT NULL,
    status text NOT NULL,
    type text NOT NULL,
    description text DEFAULT '' NOT NULL,
    payload jsonb DEFAULT '{}' NOT NULL,
    ts_vector tsvector,
    CONSTRAINT issue_pkey PRIMARY KEY (id),
    CONSTRAINT issue_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES principal (id),
    CONSTRAINT issue_pipeline_id_fkey FOREIGN KEY (pipeline_id) REFERENCES pipeline (id),
    CONSTRAINT issue_plan_id_fkey FOREIGN KEY (plan_id) REFERENCES plan (id),
    CONSTRAINT issue_project_fkey FOREIGN KEY (project) REFERENCES project (resource_id),
    CONSTRAINT issue_status_check CHECK (status IN ('OPEN', 'DONE', 'CANCELED'))
);

--
-- Name: idx_issue_creator_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_issue_creator_id ON issue (creator_id);

--
-- Name: idx_issue_pipeline_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_issue_pipeline_id ON issue (pipeline_id);

--
-- Name: idx_issue_plan_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_issue_plan_id ON issue (plan_id);

--
-- Name: idx_issue_project; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_issue_project ON issue (project);

--
-- Name: idx_issue_ts_vector; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_issue_ts_vector ON issue USING gin (ts_vector);

--
-- Name: issue_comment; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS issue_comment (
    id BIGSERIAL,
    creator_id integer NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    issue_id integer NOT NULL,
    payload jsonb DEFAULT '{}' NOT NULL,
    CONSTRAINT issue_comment_pkey PRIMARY KEY (id),
    CONSTRAINT issue_comment_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES principal (id),
    CONSTRAINT issue_comment_issue_id_fkey FOREIGN KEY (issue_id) REFERENCES issue (id)
);

--
-- Name: idx_issue_comment_issue_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_issue_comment_issue_id ON issue_comment (issue_id);

--
-- Name: issue_subscriber; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS issue_subscriber (
    issue_id integer,
    subscriber_id integer,
    CONSTRAINT issue_subscriber_pkey PRIMARY KEY (issue_id, subscriber_id),
    CONSTRAINT issue_subscriber_issue_id_fkey FOREIGN KEY (issue_id) REFERENCES issue (id),
    CONSTRAINT issue_subscriber_subscriber_id_fkey FOREIGN KEY (subscriber_id) REFERENCES principal (id)
);

--
-- Name: idx_issue_subscriber_subscriber_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_issue_subscriber_subscriber_id ON issue_subscriber (subscriber_id);

--
-- Name: plan_check_run; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS plan_check_run (
    id SERIAL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    plan_id bigint NOT NULL,
    status text NOT NULL,
    type text NOT NULL,
    config jsonb DEFAULT '{}' NOT NULL,
    result jsonb DEFAULT '{}' NOT NULL,
    payload jsonb DEFAULT '{}' NOT NULL,
    CONSTRAINT plan_check_run_pkey PRIMARY KEY (id),
    CONSTRAINT plan_check_run_plan_id_fkey FOREIGN KEY (plan_id) REFERENCES plan (id),
    CONSTRAINT plan_check_run_status_check CHECK (status IN ('RUNNING', 'DONE', 'FAILED', 'CANCELED')),
    CONSTRAINT plan_check_run_type_check CHECK (type LIKE 'bb.plan-check.%')
);

--
-- Name: idx_plan_check_run_plan_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_plan_check_run_plan_id ON plan_check_run (plan_id);

--
-- Name: project_webhook; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS project_webhook (
    id SERIAL,
    project text NOT NULL,
    type text NOT NULL,
    name text NOT NULL,
    url text NOT NULL,
    event_list text[] NOT NULL,
    payload jsonb DEFAULT '{}' NOT NULL,
    CONSTRAINT project_webhook_pkey PRIMARY KEY (id),
    CONSTRAINT project_webhook_project_fkey FOREIGN KEY (project) REFERENCES project (resource_id),
    CONSTRAINT project_webhook_type_check CHECK (type LIKE 'bb.plugin.webhook.%')
);

--
-- Name: idx_project_webhook_project; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_project_webhook_project ON project_webhook (project);

--
-- Name: query_history; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS query_history (
    id BIGSERIAL,
    creator_id integer NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    project_id text NOT NULL,
    database text NOT NULL,
    statement text NOT NULL,
    type text NOT NULL,
    payload jsonb DEFAULT '{}' NOT NULL,
    CONSTRAINT query_history_pkey PRIMARY KEY (id),
    CONSTRAINT query_history_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES principal (id)
);

--
-- Name: idx_query_history_creator_id_created_at_project_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_query_history_creator_id_created_at_project_id ON query_history (creator_id, created_at, project_id DESC);

--
-- Name: release; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS release (
    id BIGSERIAL,
    deleted boolean DEFAULT false NOT NULL,
    project text NOT NULL,
    creator_id integer NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    payload jsonb DEFAULT '{}' NOT NULL,
    CONSTRAINT release_pkey PRIMARY KEY (id),
    CONSTRAINT release_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES principal (id),
    CONSTRAINT release_project_fkey FOREIGN KEY (project) REFERENCES project (resource_id)
);

--
-- Name: idx_release_project; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_release_project ON release (project);

--
-- Name: review_config; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS review_config (
    id text,
    enabled boolean DEFAULT true NOT NULL,
    name text NOT NULL,
    payload jsonb DEFAULT '{}' NOT NULL,
    CONSTRAINT review_config_pkey PRIMARY KEY (id)
);

--
-- Name: revision; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS revision (
    id BIGSERIAL,
    instance text NOT NULL,
    db_name text NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    deleter_id integer,
    deleted_at timestamptz,
    version text NOT NULL,
    payload jsonb DEFAULT '{}' NOT NULL,
    CONSTRAINT revision_pkey PRIMARY KEY (id),
    CONSTRAINT revision_deleter_id_fkey FOREIGN KEY (deleter_id) REFERENCES principal (id),
    CONSTRAINT revision_instance_db_name_fkey FOREIGN KEY (instance, db_name) REFERENCES db (instance, name)
);

--
-- Name: idx_revision_instance_db_name_version; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_revision_instance_db_name_version ON revision (instance, db_name, version);

--
-- Name: idx_revision_unique_instance_db_name_version_deleted_at_null; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX IF NOT EXISTS idx_revision_unique_instance_db_name_version_deleted_at_null ON revision (instance, db_name, version) WHERE (deleted_at IS NULL);

--
-- Name: risk; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS risk (
    id BIGSERIAL,
    source text NOT NULL,
    level bigint NOT NULL,
    name text NOT NULL,
    active boolean NOT NULL,
    expression jsonb NOT NULL,
    CONSTRAINT risk_pkey PRIMARY KEY (id),
    CONSTRAINT risk_source_check CHECK (source LIKE 'bb.risk.%')
);

--
-- Name: role; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS role (
    id BIGSERIAL,
    resource_id text NOT NULL,
    name text NOT NULL,
    description text NOT NULL,
    permissions jsonb DEFAULT '{}' NOT NULL,
    payload jsonb DEFAULT '{}' NOT NULL,
    CONSTRAINT role_pkey PRIMARY KEY (id)
);

--
-- Name: idx_role_unique_resource_id; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX IF NOT EXISTS idx_role_unique_resource_id ON role (resource_id);

--
-- Name: setting; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS setting (
    id SERIAL,
    name text NOT NULL,
    value text NOT NULL,
    CONSTRAINT setting_pkey PRIMARY KEY (id)
);

--
-- Name: idx_setting_unique_name; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX IF NOT EXISTS idx_setting_unique_name ON setting (name);

--
-- Name: sheet; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS sheet (
    id SERIAL,
    creator_id integer NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    project text NOT NULL,
    name text NOT NULL,
    sha256 bytea NOT NULL,
    payload jsonb DEFAULT '{}' NOT NULL,
    CONSTRAINT sheet_pkey PRIMARY KEY (id),
    CONSTRAINT sheet_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES principal (id),
    CONSTRAINT sheet_project_fkey FOREIGN KEY (project) REFERENCES project (resource_id)
);

--
-- Name: idx_sheet_project; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_sheet_project ON sheet (project);

--
-- Name: sheet_blob; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS sheet_blob (
    sha256 bytea,
    content text NOT NULL,
    CONSTRAINT sheet_blob_pkey PRIMARY KEY (sha256)
);

--
-- Name: sync_history; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS sync_history (
    id BIGSERIAL,
    created_at timestamptz DEFAULT now() NOT NULL,
    instance text NOT NULL,
    db_name text NOT NULL,
    metadata json DEFAULT '{}' NOT NULL,
    raw_dump text DEFAULT '' NOT NULL,
    CONSTRAINT sync_history_pkey PRIMARY KEY (id),
    CONSTRAINT sync_history_instance_db_name_fkey FOREIGN KEY (instance, db_name) REFERENCES db (instance, name)
);

--
-- Name: idx_sync_history_instance_db_name_created_at; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_sync_history_instance_db_name_created_at ON sync_history (instance, db_name, created_at);

--
-- Name: changelog; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS changelog (
    id BIGSERIAL,
    created_at timestamptz DEFAULT now() NOT NULL,
    instance text NOT NULL,
    db_name text NOT NULL,
    status text NOT NULL,
    prev_sync_history_id bigint,
    sync_history_id bigint,
    payload jsonb DEFAULT '{}' NOT NULL,
    CONSTRAINT changelog_pkey PRIMARY KEY (id),
    CONSTRAINT changelog_instance_db_name_fkey FOREIGN KEY (instance, db_name) REFERENCES db (instance, name),
    CONSTRAINT changelog_prev_sync_history_id_fkey FOREIGN KEY (prev_sync_history_id) REFERENCES sync_history (id),
    CONSTRAINT changelog_sync_history_id_fkey FOREIGN KEY (sync_history_id) REFERENCES sync_history (id),
    CONSTRAINT changelog_status_check CHECK (status IN ('PENDING', 'DONE', 'FAILED'))
);

--
-- Name: idx_changelog_instance_db_name; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_changelog_instance_db_name ON changelog (instance, db_name);

--
-- Name: task; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS task (
    id SERIAL,
    pipeline_id integer NOT NULL,
    instance text NOT NULL,
    environment text,
    db_name text,
    type text NOT NULL,
    payload jsonb DEFAULT '{}' NOT NULL,
    CONSTRAINT task_pkey PRIMARY KEY (id),
    CONSTRAINT task_instance_fkey FOREIGN KEY (instance) REFERENCES instance (resource_id),
    CONSTRAINT task_pipeline_id_fkey FOREIGN KEY (pipeline_id) REFERENCES pipeline (id)
);

--
-- Name: idx_task_pipeline_id_environment; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_task_pipeline_id_environment ON task (pipeline_id, environment);

--
-- Name: task_run; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS task_run (
    id SERIAL,
    creator_id integer NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    task_id integer NOT NULL,
    sheet_id integer,
    attempt integer NOT NULL,
    status text NOT NULL,
    started_at timestamptz,
    run_at timestamptz,
    code integer DEFAULT 0 NOT NULL,
    result jsonb DEFAULT '{}' NOT NULL,
    CONSTRAINT task_run_pkey PRIMARY KEY (id),
    CONSTRAINT task_run_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES principal (id),
    CONSTRAINT task_run_sheet_id_fkey FOREIGN KEY (sheet_id) REFERENCES sheet (id),
    CONSTRAINT task_run_task_id_fkey FOREIGN KEY (task_id) REFERENCES task (id),
    CONSTRAINT task_run_status_check CHECK (status IN ('PENDING', 'RUNNING', 'DONE', 'FAILED', 'CANCELED'))
);

--
-- Name: idx_task_run_task_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_task_run_task_id ON task_run (task_id);

--
-- Name: uk_task_run_task_id_attempt; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX IF NOT EXISTS uk_task_run_task_id_attempt ON task_run (task_id, attempt);

--
-- Name: task_run_log; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS task_run_log (
    id BIGSERIAL,
    task_run_id integer NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    payload jsonb DEFAULT '{}' NOT NULL,
    CONSTRAINT task_run_log_pkey PRIMARY KEY (id),
    CONSTRAINT task_run_log_task_run_id_fkey FOREIGN KEY (task_run_id) REFERENCES task_run (id)
);

--
-- Name: idx_task_run_log_task_run_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_task_run_log_task_run_id ON task_run_log (task_run_id);

--
-- Name: user_group; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS user_group (
    email text,
    name text NOT NULL,
    description text DEFAULT '' NOT NULL,
    payload jsonb DEFAULT '{}' NOT NULL,
    CONSTRAINT user_group_pkey PRIMARY KEY (email)
);

--
-- Name: worksheet; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS worksheet (
    id SERIAL,
    creator_id integer NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    project text NOT NULL,
    instance text,
    db_name text,
    name text NOT NULL,
    statement text NOT NULL,
    visibility text NOT NULL,
    payload jsonb DEFAULT '{}' NOT NULL,
    CONSTRAINT worksheet_pkey PRIMARY KEY (id),
    CONSTRAINT worksheet_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES principal (id),
    CONSTRAINT worksheet_project_fkey FOREIGN KEY (project) REFERENCES project (resource_id)
);

--
-- Name: idx_worksheet_creator_id_project; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_worksheet_creator_id_project ON worksheet (creator_id, project);

--
-- Name: worksheet_organizer; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS worksheet_organizer (
    id SERIAL,
    worksheet_id integer NOT NULL,
    principal_id integer NOT NULL,
    starred boolean DEFAULT false NOT NULL,
    CONSTRAINT worksheet_organizer_pkey PRIMARY KEY (id),
    CONSTRAINT worksheet_organizer_principal_id_fkey FOREIGN KEY (principal_id) REFERENCES principal (id),
    CONSTRAINT worksheet_organizer_worksheet_id_fkey FOREIGN KEY (worksheet_id) REFERENCES worksheet (id) ON DELETE CASCADE
);

--
-- Name: idx_worksheet_organizer_principal_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_worksheet_organizer_principal_id ON worksheet_organizer (principal_id);

--
-- Name: idx_worksheet_organizer_unique_sheet_id_principal_id; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX IF NOT EXISTS idx_worksheet_organizer_unique_sheet_id_principal_id ON worksheet_organizer (worksheet_id, principal_id);

