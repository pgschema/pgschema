--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 0.1.5


--
-- Name: audit_log; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE audit_log (
    id BIGSERIAL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    payload jsonb DEFAULT '{}' NOT NULL
);

--
-- Name: idx_audit_log_created_at; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_audit_log_created_at ON audit_log (created_at);

--
-- Name: idx_audit_log_payload_method; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_audit_log_payload_method ON audit_log (((payload->>'method')));

--
-- Name: idx_audit_log_payload_parent; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_audit_log_payload_parent ON audit_log (((payload->>'parent')));

--
-- Name: idx_audit_log_payload_resource; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_audit_log_payload_resource ON audit_log (((payload->>'resource')));

--
-- Name: idx_audit_log_payload_user; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_audit_log_payload_user ON audit_log (((payload->>'user')));

--
-- Name: export_archive; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE export_archive (
    id SERIAL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    bytes bytea,
    payload jsonb DEFAULT '{}' NOT NULL
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
    config jsonb DEFAULT '{}' NOT NULL
);

--
-- Name: idx_idp_unique_resource_id; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX idx_idp_unique_resource_id ON idp (resource_id);

--
-- Name: instance; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE instance (
    id SERIAL PRIMARY KEY,
    deleted boolean DEFAULT false NOT NULL,
    environment text,
    resource_id text NOT NULL,
    metadata jsonb DEFAULT '{}' NOT NULL
);

--
-- Name: idx_instance_unique_resource_id; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX idx_instance_unique_resource_id ON instance (resource_id);

--
-- Name: data_source; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE data_source (
    id SERIAL PRIMARY KEY,
    instance text NOT NULL,
    options jsonb DEFAULT '{}' NOT NULL,
    FOREIGN KEY (instance) REFERENCES instance (resource_id)
);

--
-- Name: instance_change_history; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE instance_change_history (
    id BIGSERIAL PRIMARY KEY,
    version text NOT NULL
);

--
-- Name: idx_instance_change_history_unique_version; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX idx_instance_change_history_unique_version ON instance_change_history (version);

--
-- Name: policy; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE policy (
    id SERIAL PRIMARY KEY,
    enforce boolean DEFAULT true NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    resource_type text NOT NULL,
    resource text NOT NULL,
    type text NOT NULL,
    payload jsonb DEFAULT '{}' NOT NULL,
    inherit_from_parent boolean DEFAULT true NOT NULL
);

--
-- Name: idx_policy_unique_resource_type_resource_type; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX idx_policy_unique_resource_type_resource_type ON policy (resource_type, resource, type);

--
-- Name: principal; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE principal (
    id SERIAL PRIMARY KEY,
    deleted boolean DEFAULT false NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    type text NOT NULL CHECK (type IN('END_USER', 'SYSTEM_BOT', 'SERVICE_ACCOUNT')),
    name text NOT NULL,
    email text NOT NULL,
    password_hash text NOT NULL,
    phone text DEFAULT '' NOT NULL,
    mfa_config jsonb DEFAULT '{}' NOT NULL,
    profile jsonb DEFAULT '{}' NOT NULL
);

--
-- Name: project; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE project (
    id SERIAL PRIMARY KEY,
    deleted boolean DEFAULT false NOT NULL,
    name text NOT NULL,
    resource_id text NOT NULL,
    data_classification_config_id text DEFAULT '' NOT NULL,
    setting jsonb DEFAULT '{}' NOT NULL
);

--
-- Name: idx_project_unique_resource_id; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX idx_project_unique_resource_id ON project (resource_id);

--
-- Name: changelist; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE changelist (
    id SERIAL PRIMARY KEY,
    creator_id integer NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    project text NOT NULL,
    name text NOT NULL,
    payload jsonb DEFAULT '{}' NOT NULL,
    FOREIGN KEY (creator_id) REFERENCES principal (id),
    FOREIGN KEY (project) REFERENCES project (resource_id)
);

--
-- Name: idx_changelist_project_name; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX idx_changelist_project_name ON changelist (project, name);

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
    metadata jsonb DEFAULT '{}' NOT NULL,
    FOREIGN KEY (instance) REFERENCES instance (resource_id),
    FOREIGN KEY (project) REFERENCES project (resource_id)
);

--
-- Name: idx_db_project; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_db_project ON db (project);

--
-- Name: idx_db_unique_instance_name; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX idx_db_unique_instance_name ON db (instance, name);

--
-- Name: db_group; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE db_group (
    id BIGSERIAL PRIMARY KEY,
    project text NOT NULL,
    resource_id text NOT NULL,
    placeholder text DEFAULT '' NOT NULL,
    expression jsonb DEFAULT '{}' NOT NULL,
    payload jsonb DEFAULT '{}' NOT NULL,
    FOREIGN KEY (project) REFERENCES project (resource_id)
);

--
-- Name: idx_db_group_unique_project_placeholder; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX idx_db_group_unique_project_placeholder ON db_group (project, placeholder);

--
-- Name: idx_db_group_unique_project_resource_id; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX idx_db_group_unique_project_resource_id ON db_group (project, resource_id);

--
-- Name: db_schema; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE db_schema (
    id SERIAL PRIMARY KEY,
    instance text NOT NULL,
    db_name text NOT NULL,
    metadata json DEFAULT '{}' NOT NULL,
    raw_dump text DEFAULT '' NOT NULL,
    config jsonb DEFAULT '{}' NOT NULL,
    FOREIGN KEY (instance, db_name) REFERENCES db (instance, name)
);

--
-- Name: idx_db_schema_unique_instance_db_name; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX idx_db_schema_unique_instance_db_name ON db_schema (instance, db_name);

--
-- Name: pipeline; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE pipeline (
    id SERIAL PRIMARY KEY,
    creator_id integer NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    project text NOT NULL,
    name text NOT NULL,
    FOREIGN KEY (creator_id) REFERENCES principal (id),
    FOREIGN KEY (project) REFERENCES project (resource_id)
);

--
-- Name: plan; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE plan (
    id BIGSERIAL PRIMARY KEY,
    creator_id integer NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    project text NOT NULL,
    pipeline_id integer,
    name text NOT NULL,
    description text NOT NULL,
    config jsonb DEFAULT '{}' NOT NULL,
    FOREIGN KEY (creator_id) REFERENCES principal (id),
    FOREIGN KEY (pipeline_id) REFERENCES pipeline (id),
    FOREIGN KEY (project) REFERENCES project (resource_id)
);

--
-- Name: idx_plan_pipeline_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_plan_pipeline_id ON plan (pipeline_id);

--
-- Name: idx_plan_project; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_plan_project ON plan (project);

--
-- Name: issue; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE issue (
    id SERIAL PRIMARY KEY,
    creator_id integer NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    project text NOT NULL,
    plan_id bigint,
    pipeline_id integer,
    name text NOT NULL,
    status text NOT NULL CHECK (status IN('OPEN', 'DONE', 'CANCELED')),
    type text NOT NULL,
    description text DEFAULT '' NOT NULL,
    payload jsonb DEFAULT '{}' NOT NULL,
    ts_vector tsvector,
    FOREIGN KEY (creator_id) REFERENCES principal (id),
    FOREIGN KEY (pipeline_id) REFERENCES pipeline (id),
    FOREIGN KEY (plan_id) REFERENCES plan (id),
    FOREIGN KEY (project) REFERENCES project (resource_id)
);

--
-- Name: idx_issue_creator_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_issue_creator_id ON issue (creator_id);

--
-- Name: idx_issue_pipeline_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_issue_pipeline_id ON issue (pipeline_id);

--
-- Name: idx_issue_plan_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_issue_plan_id ON issue (plan_id);

--
-- Name: idx_issue_project; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_issue_project ON issue (project);

--
-- Name: idx_issue_ts_vector; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_issue_ts_vector ON issue USING gin (ts_vector);

--
-- Name: issue_comment; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE issue_comment (
    id BIGSERIAL PRIMARY KEY,
    creator_id integer NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    issue_id integer NOT NULL,
    payload jsonb DEFAULT '{}' NOT NULL,
    FOREIGN KEY (creator_id) REFERENCES principal (id),
    FOREIGN KEY (issue_id) REFERENCES issue (id)
);

--
-- Name: idx_issue_comment_issue_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_issue_comment_issue_id ON issue_comment (issue_id);

--
-- Name: issue_subscriber; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE issue_subscriber (
    issue_id integer,
    subscriber_id integer,
    PRIMARY KEY (issue_id, subscriber_id),
    FOREIGN KEY (issue_id) REFERENCES issue (id),
    FOREIGN KEY (subscriber_id) REFERENCES principal (id)
);

--
-- Name: idx_issue_subscriber_subscriber_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_issue_subscriber_subscriber_id ON issue_subscriber (subscriber_id);

--
-- Name: plan_check_run; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE plan_check_run (
    id SERIAL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    plan_id bigint NOT NULL,
    status text NOT NULL CHECK (status IN('RUNNING', 'DONE', 'FAILED', 'CANCELED')),
    type text NOT NULL CHECK (type LIKE 'bb.plan-check.%'),
    config jsonb DEFAULT '{}' NOT NULL,
    result jsonb DEFAULT '{}' NOT NULL,
    payload jsonb DEFAULT '{}' NOT NULL,
    FOREIGN KEY (plan_id) REFERENCES plan (id)
);

--
-- Name: idx_plan_check_run_plan_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_plan_check_run_plan_id ON plan_check_run (plan_id);

--
-- Name: project_webhook; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE project_webhook (
    id SERIAL PRIMARY KEY,
    project text NOT NULL,
    type text NOT NULL CHECK (type LIKE 'bb.plugin.webhook.%'),
    name text NOT NULL,
    url text NOT NULL,
    event_list text[] NOT NULL,
    payload jsonb DEFAULT '{}' NOT NULL,
    FOREIGN KEY (project) REFERENCES project (resource_id)
);

--
-- Name: idx_project_webhook_project; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_project_webhook_project ON project_webhook (project);

--
-- Name: query_history; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE query_history (
    id BIGSERIAL PRIMARY KEY,
    creator_id integer NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    project_id text NOT NULL,
    database text NOT NULL,
    statement text NOT NULL,
    type text NOT NULL,
    payload jsonb DEFAULT '{}' NOT NULL,
    FOREIGN KEY (creator_id) REFERENCES principal (id)
);

--
-- Name: idx_query_history_creator_id_created_at_project_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_query_history_creator_id_created_at_project_id ON query_history (creator_id, created_at, project_id DESC);

--
-- Name: release; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE release (
    id BIGSERIAL PRIMARY KEY,
    deleted boolean DEFAULT false NOT NULL,
    project text NOT NULL,
    creator_id integer NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    payload jsonb DEFAULT '{}' NOT NULL,
    FOREIGN KEY (creator_id) REFERENCES principal (id),
    FOREIGN KEY (project) REFERENCES project (resource_id)
);

--
-- Name: idx_release_project; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_release_project ON release (project);

--
-- Name: review_config; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE review_config (
    id text PRIMARY KEY,
    enabled boolean DEFAULT true NOT NULL,
    name text NOT NULL,
    payload jsonb DEFAULT '{}' NOT NULL
);

--
-- Name: revision; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE revision (
    id BIGSERIAL PRIMARY KEY,
    instance text NOT NULL,
    db_name text NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    deleter_id integer,
    deleted_at timestamptz,
    version text NOT NULL,
    payload jsonb DEFAULT '{}' NOT NULL,
    FOREIGN KEY (deleter_id) REFERENCES principal (id),
    FOREIGN KEY (instance, db_name) REFERENCES db (instance, name)
);

--
-- Name: idx_revision_instance_db_name_version; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_revision_instance_db_name_version ON revision (instance, db_name, version);

--
-- Name: idx_revision_unique_instance_db_name_version_deleted_at_null; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX idx_revision_unique_instance_db_name_version_deleted_at_null ON revision (instance, db_name, version) WHERE (deleted_at IS NULL);

--
-- Name: risk; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE risk (
    id BIGSERIAL PRIMARY KEY,
    source text NOT NULL CHECK (source LIKE 'bb.risk.%'),
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
    permissions jsonb DEFAULT '{}' NOT NULL,
    payload jsonb DEFAULT '{}' NOT NULL
);

--
-- Name: idx_role_unique_resource_id; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX idx_role_unique_resource_id ON role (resource_id);

--
-- Name: setting; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE setting (
    id SERIAL PRIMARY KEY,
    name text NOT NULL,
    value text NOT NULL
);

--
-- Name: idx_setting_unique_name; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX idx_setting_unique_name ON setting (name);

--
-- Name: sheet; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE sheet (
    id SERIAL PRIMARY KEY,
    creator_id integer NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    project text NOT NULL,
    name text NOT NULL,
    sha256 bytea NOT NULL,
    payload jsonb DEFAULT '{}' NOT NULL,
    FOREIGN KEY (creator_id) REFERENCES principal (id),
    FOREIGN KEY (project) REFERENCES project (resource_id)
);

--
-- Name: idx_sheet_project; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_sheet_project ON sheet (project);

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
    created_at timestamptz DEFAULT now() NOT NULL,
    instance text NOT NULL,
    db_name text NOT NULL,
    metadata json DEFAULT '{}' NOT NULL,
    raw_dump text DEFAULT '' NOT NULL,
    FOREIGN KEY (instance, db_name) REFERENCES db (instance, name)
);

--
-- Name: idx_sync_history_instance_db_name_created_at; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_sync_history_instance_db_name_created_at ON sync_history (instance, db_name, created_at);

--
-- Name: changelog; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE changelog (
    id BIGSERIAL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    instance text NOT NULL,
    db_name text NOT NULL,
    status text NOT NULL CHECK (status IN('PENDING', 'DONE', 'FAILED')),
    prev_sync_history_id bigint,
    sync_history_id bigint,
    payload jsonb DEFAULT '{}' NOT NULL,
    FOREIGN KEY (instance, db_name) REFERENCES db (instance, name),
    FOREIGN KEY (prev_sync_history_id) REFERENCES sync_history (id),
    FOREIGN KEY (sync_history_id) REFERENCES sync_history (id)
);

--
-- Name: idx_changelog_instance_db_name; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_changelog_instance_db_name ON changelog (instance, db_name);

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
    payload jsonb DEFAULT '{}' NOT NULL,
    FOREIGN KEY (instance) REFERENCES instance (resource_id),
    FOREIGN KEY (pipeline_id) REFERENCES pipeline (id)
);

--
-- Name: idx_task_pipeline_id_environment; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_task_pipeline_id_environment ON task (pipeline_id, environment);

--
-- Name: task_run; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE task_run (
    id SERIAL PRIMARY KEY,
    creator_id integer NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    task_id integer NOT NULL,
    sheet_id integer,
    attempt integer NOT NULL,
    status text NOT NULL CHECK (status IN('PENDING', 'RUNNING', 'DONE', 'FAILED', 'CANCELED')),
    started_at timestamptz,
    run_at timestamptz,
    code integer DEFAULT 0 NOT NULL,
    result jsonb DEFAULT '{}' NOT NULL,
    FOREIGN KEY (creator_id) REFERENCES principal (id),
    FOREIGN KEY (sheet_id) REFERENCES sheet (id),
    FOREIGN KEY (task_id) REFERENCES task (id)
);

--
-- Name: idx_task_run_task_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_task_run_task_id ON task_run (task_id);

--
-- Name: uk_task_run_task_id_attempt; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX uk_task_run_task_id_attempt ON task_run (task_id, attempt);

--
-- Name: task_run_log; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE task_run_log (
    id BIGSERIAL PRIMARY KEY,
    task_run_id integer NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    payload jsonb DEFAULT '{}' NOT NULL,
    FOREIGN KEY (task_run_id) REFERENCES task_run (id)
);

--
-- Name: idx_task_run_log_task_run_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_task_run_log_task_run_id ON task_run_log (task_run_id);

--
-- Name: user_group; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE user_group (
    email text PRIMARY KEY,
    name text NOT NULL,
    description text DEFAULT '' NOT NULL,
    payload jsonb DEFAULT '{}' NOT NULL
);

--
-- Name: worksheet; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE worksheet (
    id SERIAL PRIMARY KEY,
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
    FOREIGN KEY (creator_id) REFERENCES principal (id),
    FOREIGN KEY (project) REFERENCES project (resource_id)
);

--
-- Name: idx_worksheet_creator_id_project; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_worksheet_creator_id_project ON worksheet (creator_id, project);

--
-- Name: worksheet_organizer; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE worksheet_organizer (
    id SERIAL PRIMARY KEY,
    worksheet_id integer NOT NULL,
    principal_id integer NOT NULL,
    starred boolean DEFAULT false NOT NULL,
    FOREIGN KEY (principal_id) REFERENCES principal (id),
    FOREIGN KEY (worksheet_id) REFERENCES worksheet (id) ON DELETE CASCADE
);

--
-- Name: idx_worksheet_organizer_principal_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_worksheet_organizer_principal_id ON worksheet_organizer (principal_id);

--
-- Name: idx_worksheet_organizer_unique_sheet_id_principal_id; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX idx_worksheet_organizer_unique_sheet_id_principal_id ON worksheet_organizer (worksheet_id, principal_id);