--
-- PostgreSQL database dump
--

-- Dumped from database version 17.5 (Debian 17.5-1.pgdg120+1)
-- Dumped by pg_dump version 17.2

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: citext; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS citext WITH SCHEMA public;


--
-- Name: EXTENSION citext; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION citext IS 'data type for case-insensitive character strings';


--
-- Name: hstore; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS hstore WITH SCHEMA public;


--
-- Name: EXTENSION hstore; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION hstore IS 'data type for storing sets of (key, value) pairs';


--
-- Name: intarray; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS intarray WITH SCHEMA public;


--
-- Name: EXTENSION intarray; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION intarray IS 'functions, operators, and index support for 1-D arrays of integers';


--
-- Name: pg_stat_statements; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pg_stat_statements WITH SCHEMA public;


--
-- Name: EXTENSION pg_stat_statements; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION pg_stat_statements IS 'track execution statistics of all SQL statements executed';


--
-- Name: pg_trgm; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pg_trgm WITH SCHEMA public;


--
-- Name: EXTENSION pg_trgm; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION pg_trgm IS 'text similarity measurement and index searching based on trigrams';


--
-- Name: pgcrypto; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;


--
-- Name: EXTENSION pgcrypto; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION pgcrypto IS 'cryptographic functions';


--
-- Name: audit_log_operation; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.audit_log_operation AS ENUM (
    'create',
    'modify',
    'delete'
);


--
-- Name: batch_changes_changeset_ui_publication_state; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.batch_changes_changeset_ui_publication_state AS ENUM (
    'UNPUBLISHED',
    'DRAFT',
    'PUBLISHED'
);


--
-- Name: cm_email_priority; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.cm_email_priority AS ENUM (
    'NORMAL',
    'CRITICAL'
);


--
-- Name: configuration_policies_transition_columns; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.configuration_policies_transition_columns AS (
	name text,
	type text,
	pattern text,
	retention_enabled boolean,
	retention_duration_hours integer,
	retain_intermediate_commits boolean,
	indexing_enabled boolean,
	index_commit_max_age_hours integer,
	index_intermediate_commits boolean,
	protected boolean,
	repository_patterns text[]
);


--
-- Name: TYPE configuration_policies_transition_columns; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TYPE public.configuration_policies_transition_columns IS 'A type containing the columns that make-up the set of tracked transition columns. Primarily used to create a nulled record due to `OLD` being unset in INSERT queries, and creating a nulled record with a subquery is not allowed.';


--
-- Name: critical_or_site; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.critical_or_site AS ENUM (
    'critical',
    'site'
);


--
-- Name: feature_flag_type; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.feature_flag_type AS ENUM (
    'bool',
    'rollout'
);


--
-- Name: github_app_kind; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.github_app_kind AS ENUM (
    'COMMIT_SIGNING',
    'REPO_SYNC',
    'USER_CREDENTIAL',
    'SITE_CREDENTIAL'
);


--
-- Name: lsif_uploads_transition_columns; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.lsif_uploads_transition_columns AS (
	state text,
	expired boolean,
	num_resets integer,
	num_failures integer,
	worker_hostname text,
	committed_at timestamp with time zone
);


--
-- Name: TYPE lsif_uploads_transition_columns; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TYPE public.lsif_uploads_transition_columns IS 'A type containing the columns that make-up the set of tracked transition columns. Primarily used to create a nulled record due to `OLD` being unset in INSERT queries, and creating a nulled record with a subquery is not allowed.';


--
-- Name: pattern_type; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.pattern_type AS ENUM (
    'keyword',
    'literal',
    'regexp',
    'standard',
    'structural'
);


--
-- Name: persistmode; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.persistmode AS ENUM (
    'record',
    'snapshot'
);


--
-- Name: batch_spec_workspace_execution_last_dequeues_upsert(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.batch_spec_workspace_execution_last_dequeues_upsert() RETURNS trigger
    LANGUAGE plpgsql
    AS $$ BEGIN
    INSERT INTO
        batch_spec_workspace_execution_last_dequeues
    SELECT
        user_id,
        MAX(started_at) as latest_dequeue
    FROM
        newtab
    GROUP BY
        user_id
    ON CONFLICT (user_id) DO UPDATE SET
        latest_dequeue = GREATEST(batch_spec_workspace_execution_last_dequeues.latest_dequeue, EXCLUDED.latest_dequeue);

    RETURN NULL;
END $$;


--
-- Name: changesets_computed_state_ensure(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.changesets_computed_state_ensure() RETURNS trigger
    LANGUAGE plpgsql
    AS $$ BEGIN

    NEW.computed_state = CASE
        WHEN NEW.reconciler_state = 'errored' THEN 'RETRYING'
        WHEN NEW.reconciler_state = 'failed' THEN 'FAILED'
        WHEN NEW.reconciler_state = 'scheduled' THEN 'SCHEDULED'
        WHEN NEW.reconciler_state != 'completed' THEN 'PROCESSING'
        WHEN NEW.publication_state = 'UNPUBLISHED' THEN 'UNPUBLISHED'
        ELSE NEW.external_state
    END AS computed_state;

    RETURN NEW;
END $$;


--
-- Name: delete_batch_change_reference_on_changesets(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.delete_batch_change_reference_on_changesets() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
    BEGIN
        UPDATE
          changesets
        SET
          batch_change_ids = changesets.batch_change_ids - OLD.id::text
        WHERE
          changesets.batch_change_ids ? OLD.id::text;

        RETURN OLD;
    END;
$$;


--
-- Name: delete_repo_ref_on_external_service_repos(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.delete_repo_ref_on_external_service_repos() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
    BEGIN
        -- if a repo is soft-deleted, delete every row that references that repo
        IF (OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL) THEN
        DELETE FROM
            external_service_repos
        WHERE
            repo_id = OLD.id;
        END IF;

        RETURN OLD;
    END;
$$;


--
-- Name: delete_user_repo_permissions_on_external_account_soft_delete(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.delete_user_repo_permissions_on_external_account_soft_delete() RETURNS trigger
    LANGUAGE plpgsql
    AS $$ BEGIN
    IF NEW.deleted_at IS NOT NULL AND OLD.deleted_at IS NULL THEN
    	DELETE FROM user_repo_permissions WHERE user_id = OLD.user_id AND user_external_account_id = OLD.id;
    END IF;
    RETURN NULL;
  END
$$;


--
-- Name: delete_user_repo_permissions_on_repo_soft_delete(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.delete_user_repo_permissions_on_repo_soft_delete() RETURNS trigger
    LANGUAGE plpgsql
    AS $$ BEGIN
    IF NEW.deleted_at IS NOT NULL AND OLD.deleted_at IS NULL THEN
    	DELETE FROM user_repo_permissions WHERE repo_id = NEW.id;
    END IF;
    RETURN NULL;
  END
$$;


--
-- Name: delete_user_repo_permissions_on_user_soft_delete(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.delete_user_repo_permissions_on_user_soft_delete() RETURNS trigger
    LANGUAGE plpgsql
    AS $$ BEGIN
    IF NEW.deleted_at IS NOT NULL AND OLD.deleted_at IS NULL THEN
    	DELETE FROM user_repo_permissions WHERE user_id = OLD.id;
    END IF;
    RETURN NULL;
  END
$$;


--
-- Name: extract_topics_from_metadata(text, jsonb); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.extract_topics_from_metadata(external_service_type text, metadata jsonb) RETURNS text[]
    LANGUAGE plpgsql IMMUTABLE
    AS $_$
BEGIN
    RETURN CASE external_service_type
    WHEN 'github' THEN
        ARRAY(SELECT * FROM jsonb_array_elements_text(jsonb_path_query_array(metadata, '$.RepositoryTopics.Nodes[*].Topic.Name')))
    WHEN 'gitlab' THEN
        ARRAY(SELECT * FROM jsonb_array_elements_text(metadata->'topics'))
    ELSE
        '{}'::text[]
    END;
EXCEPTION WHEN others THEN
    -- Catch exceptions in the case that metadata is not shaped like we expect
    RETURN '{}'::text[];
END;
$_$;


--
-- Name: func_configuration_policies_delete(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.func_configuration_policies_delete() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
    BEGIN
        UPDATE configuration_policies_audit_logs
        SET record_deleted_at = NOW()
        WHERE policy_id IN (
            SELECT id FROM OLD
        );

        RETURN NULL;
    END;
$$;


--
-- Name: func_configuration_policies_insert(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.func_configuration_policies_insert() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
    BEGIN
        INSERT INTO configuration_policies_audit_logs
        (policy_id, operation, transition_columns)
        VALUES (
            NEW.id, 'create',
            func_configuration_policies_transition_columns_diff(
                (NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL),
                func_row_to_configuration_policies_transition_columns(NEW)
            )
        );
        RETURN NULL;
    END;
$$;


--
-- Name: FUNCTION func_configuration_policies_insert(); Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON FUNCTION public.func_configuration_policies_insert() IS 'Transforms a record from the configuration_policies table into an `configuration_policies_transition_columns` type variable.';


--
-- Name: func_configuration_policies_transition_columns_diff(public.configuration_policies_transition_columns, public.configuration_policies_transition_columns); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.func_configuration_policies_transition_columns_diff(old public.configuration_policies_transition_columns, new public.configuration_policies_transition_columns) RETURNS public.hstore[]
    LANGUAGE plpgsql
    AS $$
    BEGIN
        -- array || NULL should be a noop, but that doesn't seem to be happening
        -- hence array_remove here
        RETURN array_remove(
            ARRAY[]::hstore[] ||
            CASE WHEN old.name IS DISTINCT FROM new.name THEN
                hstore(ARRAY['column', 'name', 'old', old.name, 'new', new.name])
                ELSE NULL
            END ||
            CASE WHEN old.type IS DISTINCT FROM new.type THEN
                hstore(ARRAY['column', 'type', 'old', old.type, 'new', new.type])
                ELSE NULL
            END ||
            CASE WHEN old.pattern IS DISTINCT FROM new.pattern THEN
                hstore(ARRAY['column', 'pattern', 'old', old.pattern, 'new', new.pattern])
                ELSE NULL
            END ||
            CASE WHEN old.retention_enabled IS DISTINCT FROM new.retention_enabled THEN
                hstore(ARRAY['column', 'retention_enabled', 'old', old.retention_enabled::text, 'new', new.retention_enabled::text])
                ELSE NULL
            END ||
            CASE WHEN old.retention_duration_hours IS DISTINCT FROM new.retention_duration_hours THEN
                hstore(ARRAY['column', 'retention_duration_hours', 'old', old.retention_duration_hours::text, 'new', new.retention_duration_hours::text])
                ELSE NULL
            END ||
            CASE WHEN old.indexing_enabled IS DISTINCT FROM new.indexing_enabled THEN
                hstore(ARRAY['column', 'indexing_enabled', 'old', old.indexing_enabled::text, 'new', new.indexing_enabled::text])
                ELSE NULL
            END ||
            CASE WHEN old.index_commit_max_age_hours IS DISTINCT FROM new.index_commit_max_age_hours THEN
                hstore(ARRAY['column', 'index_commit_max_age_hours', 'old', old.index_commit_max_age_hours::text, 'new', new.index_commit_max_age_hours::text])
                ELSE NULL
            END ||
            CASE WHEN old.index_intermediate_commits IS DISTINCT FROM new.index_intermediate_commits THEN
                hstore(ARRAY['column', 'index_intermediate_commits', 'old', old.index_intermediate_commits::text, 'new', new.index_intermediate_commits::text])
                ELSE NULL
            END ||
            CASE WHEN old.protected IS DISTINCT FROM new.protected THEN
                hstore(ARRAY['column', 'protected', 'old', old.protected::text, 'new', new.protected::text])
                ELSE NULL
            END ||
            CASE WHEN old.repository_patterns IS DISTINCT FROM new.repository_patterns THEN
                hstore(ARRAY['column', 'repository_patterns', 'old', old.repository_patterns::text, 'new', new.repository_patterns::text])
                ELSE NULL
            END,
        NULL);
    END;
$$;


--
-- Name: FUNCTION func_configuration_policies_transition_columns_diff(old public.configuration_policies_transition_columns, new public.configuration_policies_transition_columns); Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON FUNCTION public.func_configuration_policies_transition_columns_diff(old public.configuration_policies_transition_columns, new public.configuration_policies_transition_columns) IS 'Diffs two `configuration_policies_transition_columns` values into an array of hstores, where each hstore is in the format {"column"=>"<column name>", "old"=>"<previous value>", "new"=>"<new value>"}.';


--
-- Name: func_configuration_policies_update(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.func_configuration_policies_update() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
    DECLARE
        diff hstore[];
    BEGIN
        diff = func_configuration_policies_transition_columns_diff(
            func_row_to_configuration_policies_transition_columns(OLD),
            func_row_to_configuration_policies_transition_columns(NEW)
        );

        IF (array_length(diff, 1) > 0) THEN
            INSERT INTO configuration_policies_audit_logs
            (policy_id, operation, transition_columns)
            VALUES (NEW.id, 'modify', diff);
        END IF;

        RETURN NEW;
    END;
$$;


--
-- Name: func_insert_gitserver_repo(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.func_insert_gitserver_repo() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
INSERT INTO gitserver_repos
(repo_id, shard_id)
VALUES (NEW.id, '');
RETURN NULL;
END;
$$;


--
-- Name: func_insert_zoekt_repo(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.func_insert_zoekt_repo() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
  INSERT INTO zoekt_repos (repo_id) VALUES (NEW.id);

  RETURN NULL;
END;
$$;


--
-- Name: func_lsif_uploads_delete(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.func_lsif_uploads_delete() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
    BEGIN
        UPDATE lsif_uploads_audit_logs
        SET record_deleted_at = NOW()
        WHERE upload_id IN (
            SELECT id FROM OLD
        );

        RETURN NULL;
    END;
$$;


--
-- Name: func_lsif_uploads_insert(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.func_lsif_uploads_insert() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
    BEGIN
        INSERT INTO lsif_uploads_audit_logs
        (upload_id, commit, root, repository_id, uploaded_at,
        indexer, indexer_version, upload_size, associated_index_id,
        content_type,
        operation, transition_columns)
        VALUES (
            NEW.id, NEW.commit, NEW.root, NEW.repository_id, NEW.uploaded_at,
            NEW.indexer, NEW.indexer_version, NEW.upload_size, NEW.associated_index_id,
            NEW.content_type,
            'create', func_lsif_uploads_transition_columns_diff(
                (NULL, NULL, NULL, NULL, NULL, NULL),
                func_row_to_lsif_uploads_transition_columns(NEW)
            )
        );
        RETURN NULL;
    END;
$$;


--
-- Name: FUNCTION func_lsif_uploads_insert(); Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON FUNCTION public.func_lsif_uploads_insert() IS 'Transforms a record from the lsif_uploads table into an `lsif_uploads_transition_columns` type variable.';


--
-- Name: func_lsif_uploads_transition_columns_diff(public.lsif_uploads_transition_columns, public.lsif_uploads_transition_columns); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.func_lsif_uploads_transition_columns_diff(old public.lsif_uploads_transition_columns, new public.lsif_uploads_transition_columns) RETURNS public.hstore[]
    LANGUAGE plpgsql
    AS $$
    BEGIN
        -- array || NULL should be a noop, but that doesn't seem to be happening
        -- hence array_remove here
        RETURN array_remove(
            ARRAY[]::hstore[] ||
            CASE WHEN old.state IS DISTINCT FROM new.state THEN
                hstore(ARRAY['column', 'state', 'old', old.state, 'new', new.state])
                ELSE NULL
            END ||
            CASE WHEN old.expired IS DISTINCT FROM new.expired THEN
                hstore(ARRAY['column', 'expired', 'old', old.expired::text, 'new', new.expired::text])
                ELSE NULL
            END ||
            CASE WHEN old.num_resets IS DISTINCT FROM new.num_resets THEN
                hstore(ARRAY['column', 'num_resets', 'old', old.num_resets::text, 'new', new.num_resets::text])
                ELSE NULL
            END ||
            CASE WHEN old.num_failures IS DISTINCT FROM new.num_failures THEN
                hstore(ARRAY['column', 'num_failures', 'old', old.num_failures::text, 'new', new.num_failures::text])
                ELSE NULL
            END ||
            CASE WHEN old.worker_hostname IS DISTINCT FROM new.worker_hostname THEN
                hstore(ARRAY['column', 'worker_hostname', 'old', old.worker_hostname, 'new', new.worker_hostname])
                ELSE NULL
            END ||
            CASE WHEN old.committed_at IS DISTINCT FROM new.committed_at THEN
                hstore(ARRAY['column', 'committed_at', 'old', old.committed_at::text, 'new', new.committed_at::text])
                ELSE NULL
            END,
        NULL);
    END;
$$;


--
-- Name: FUNCTION func_lsif_uploads_transition_columns_diff(old public.lsif_uploads_transition_columns, new public.lsif_uploads_transition_columns); Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON FUNCTION public.func_lsif_uploads_transition_columns_diff(old public.lsif_uploads_transition_columns, new public.lsif_uploads_transition_columns) IS 'Diffs two `lsif_uploads_transition_columns` values into an array of hstores, where each hstore is in the format {"column"=>"<column name>", "old"=>"<previous value>", "new"=>"<new value>"}.';


--
-- Name: func_lsif_uploads_update(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.func_lsif_uploads_update() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
    DECLARE
        diff hstore[];
    BEGIN
        diff = func_lsif_uploads_transition_columns_diff(
            func_row_to_lsif_uploads_transition_columns(OLD),
            func_row_to_lsif_uploads_transition_columns(NEW)
        );

        IF (array_length(diff, 1) > 0) THEN
            INSERT INTO lsif_uploads_audit_logs
            (reason, upload_id, commit, root, repository_id, uploaded_at,
            indexer, indexer_version, upload_size, associated_index_id,
            content_type,
            operation, transition_columns)
            VALUES (
                COALESCE(current_setting('codeintel.lsif_uploads_audit.reason', true), ''),
                NEW.id, NEW.commit, NEW.root, NEW.repository_id, NEW.uploaded_at,
                NEW.indexer, NEW.indexer_version, NEW.upload_size, NEW.associated_index_id,
                NEW.content_type,
                'modify', diff
            );
        END IF;

        RETURN NEW;
    END;
$$;


--
-- Name: func_package_repo_filters_updated_at(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.func_package_repo_filters_updated_at() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = statement_timestamp();
    RETURN NEW;
END $$;


--
-- Name: func_row_to_configuration_policies_transition_columns(record); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.func_row_to_configuration_policies_transition_columns(rec record) RETURNS public.configuration_policies_transition_columns
    LANGUAGE plpgsql
    AS $$
    BEGIN
        RETURN (
            rec.name, rec.type, rec.pattern,
            rec.retention_enabled, rec.retention_duration_hours, rec.retain_intermediate_commits,
            rec.indexing_enabled, rec.index_commit_max_age_hours, rec.index_intermediate_commits,
            rec.protected, rec.repository_patterns);
    END;
$$;


--
-- Name: func_row_to_lsif_uploads_transition_columns(record); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.func_row_to_lsif_uploads_transition_columns(rec record) RETURNS public.lsif_uploads_transition_columns
    LANGUAGE plpgsql
    AS $$
    BEGIN
        RETURN (rec.state, rec.expired, rec.num_resets, rec.num_failures, rec.worker_hostname, rec.committed_at);
    END;
$$;


--
-- Name: invalidate_session_for_userid_on_password_change(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.invalidate_session_for_userid_on_password_change() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
    BEGIN
        IF OLD.passwd != NEW.passwd THEN
            NEW.invalidated_sessions_at = now() + (1 * interval '1 second');
            RETURN NEW;
        END IF;
    RETURN NEW;
    END;
$$;


--
-- Name: merge_audit_log_transitions(public.hstore, public.hstore[]); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.merge_audit_log_transitions(internal public.hstore, arrayhstore public.hstore[]) RETURNS public.hstore
    LANGUAGE plpgsql IMMUTABLE
    AS $$
    DECLARE
        trans hstore;
    BEGIN
      FOREACH trans IN ARRAY arrayhstore
      LOOP
          internal := internal || hstore(trans->'column', trans->'new');
      END LOOP;

      RETURN internal;
    END;
$$;


--
-- Name: recalc_gitserver_repos_statistics_on_delete(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.recalc_gitserver_repos_statistics_on_delete() RETURNS trigger
    LANGUAGE plpgsql
    AS $$ BEGIN
      INSERT INTO gitserver_repos_statistics AS grs (shard_id, total, not_cloned, cloning, cloned, failed_fetch, corrupted)
      SELECT
        oldtab.shard_id,
        (-COUNT(*)),
        (-COUNT(*) FILTER(WHERE clone_status = 'not_cloned')),
        (-COUNT(*) FILTER(WHERE clone_status = 'cloning')),
        (-COUNT(*) FILTER(WHERE clone_status = 'cloned')),
        (-COUNT(*) FILTER(WHERE last_error IS NOT NULL)),
        (-COUNT(*) FILTER(WHERE corrupted_at IS NOT NULL))
      FROM oldtab
      GROUP BY oldtab.shard_id;

      RETURN NULL;
  END
$$;


--
-- Name: recalc_gitserver_repos_statistics_on_insert(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.recalc_gitserver_repos_statistics_on_insert() RETURNS trigger
    LANGUAGE plpgsql
    AS $$ BEGIN
      -------------------------------------------------
      -- THIS IS CHANGED TO APPEND
      -------------------------------------------------
      INSERT INTO gitserver_repos_statistics AS grs (shard_id, total, not_cloned, cloning, cloned, failed_fetch, corrupted)
      SELECT
        shard_id,
        COUNT(*) AS total,
        COUNT(*) FILTER(WHERE clone_status = 'not_cloned') AS not_cloned,
        COUNT(*) FILTER(WHERE clone_status = 'cloning') AS cloning,
        COUNT(*) FILTER(WHERE clone_status = 'cloned') AS cloned,
        COUNT(*) FILTER(WHERE last_error IS NOT NULL) AS failed_fetch,
        COUNT(*) FILTER(WHERE corrupted_at IS NOT NULL) AS corrupted
      FROM
        newtab
      GROUP BY shard_id
      ;

      RETURN NULL;
  END
$$;


--
-- Name: recalc_gitserver_repos_statistics_on_update(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.recalc_gitserver_repos_statistics_on_update() RETURNS trigger
    LANGUAGE plpgsql
    AS $$ BEGIN

      -------------------------------------------------
      -- THIS IS CHANGED TO APPEND
      -------------------------------------------------
      WITH diff(shard_id, total, not_cloned, cloning, cloned, failed_fetch, corrupted) AS (
        SELECT
            COALESCE(newtab.shard_id, oldtab.shard_id) AS shard_id,
            COUNT(newtab.repo_id) - COUNT(oldtab.repo_id) AS total,
            COUNT(newtab.repo_id) FILTER (WHERE newtab.clone_status = 'not_cloned') - COUNT(oldtab.repo_id) FILTER (WHERE oldtab.clone_status = 'not_cloned') AS not_cloned,
            COUNT(newtab.repo_id) FILTER (WHERE newtab.clone_status = 'cloning')    - COUNT(oldtab.repo_id) FILTER (WHERE oldtab.clone_status = 'cloning') AS cloning,
            COUNT(newtab.repo_id) FILTER (WHERE newtab.clone_status = 'cloned')     - COUNT(oldtab.repo_id) FILTER (WHERE oldtab.clone_status = 'cloned') AS cloned,
            COUNT(newtab.repo_id) FILTER (WHERE newtab.last_error IS NOT NULL)      - COUNT(oldtab.repo_id) FILTER (WHERE oldtab.last_error IS NOT NULL) AS failed_fetch,
            COUNT(newtab.repo_id) FILTER (WHERE newtab.corrupted_at IS NOT NULL)    - COUNT(oldtab.repo_id) FILTER (WHERE oldtab.corrupted_at IS NOT NULL) AS corrupted
        FROM
            newtab
        FULL OUTER JOIN
            oldtab ON newtab.repo_id = oldtab.repo_id AND newtab.shard_id = oldtab.shard_id
        GROUP BY
            COALESCE(newtab.shard_id, oldtab.shard_id)
      )
      INSERT INTO gitserver_repos_statistics AS grs (shard_id, total, not_cloned, cloning, cloned, failed_fetch, corrupted)
      SELECT shard_id, total, not_cloned, cloning, cloned, failed_fetch, corrupted
      FROM diff
      WHERE
            total != 0
        OR not_cloned != 0
        OR cloning != 0
        OR cloned != 0
        OR failed_fetch != 0
        OR corrupted != 0
      ;

      -------------------------------------------------
      -- UNCHANGED
      -------------------------------------------------
      WITH diff(not_cloned, cloning, cloned, failed_fetch, corrupted) AS (
        VALUES (
          (
            (SELECT COUNT(*) FROM newtab JOIN repo r ON newtab.repo_id = r.id WHERE r.deleted_at is NULL AND r.blocked IS NULL AND newtab.clone_status = 'not_cloned')
            -
            (SELECT COUNT(*) FROM oldtab JOIN repo r ON oldtab.repo_id = r.id WHERE r.deleted_at is NULL AND r.blocked IS NULL AND oldtab.clone_status = 'not_cloned')
          ),
          (
            (SELECT COUNT(*) FROM newtab JOIN repo r ON newtab.repo_id = r.id WHERE r.deleted_at is NULL AND r.blocked IS NULL AND newtab.clone_status = 'cloning')
            -
            (SELECT COUNT(*) FROM oldtab JOIN repo r ON oldtab.repo_id = r.id WHERE r.deleted_at is NULL AND r.blocked IS NULL AND oldtab.clone_status = 'cloning')
          ),
          (
            (SELECT COUNT(*) FROM newtab JOIN repo r ON newtab.repo_id = r.id WHERE r.deleted_at is NULL AND r.blocked IS NULL AND newtab.clone_status = 'cloned')
            -
            (SELECT COUNT(*) FROM oldtab JOIN repo r ON oldtab.repo_id = r.id WHERE r.deleted_at is NULL AND r.blocked IS NULL AND oldtab.clone_status = 'cloned')
          ),
          (
            (SELECT COUNT(*) FROM newtab JOIN repo r ON newtab.repo_id = r.id WHERE r.deleted_at is NULL AND r.blocked IS NULL AND newtab.last_error IS NOT NULL)
            -
            (SELECT COUNT(*) FROM oldtab JOIN repo r ON oldtab.repo_id = r.id WHERE r.deleted_at is NULL AND r.blocked IS NULL AND oldtab.last_error IS NOT NULL)
          ),
          (
            (SELECT COUNT(*) FROM newtab JOIN repo r ON newtab.repo_id = r.id WHERE r.deleted_at is NULL AND r.blocked IS NULL AND newtab.corrupted_at IS NOT NULL)
            -
            (SELECT COUNT(*) FROM oldtab JOIN repo r ON oldtab.repo_id = r.id WHERE r.deleted_at is NULL AND r.blocked IS NULL AND oldtab.corrupted_at IS NOT NULL)
          )

        )
      )
      INSERT INTO repo_statistics (not_cloned, cloning, cloned, failed_fetch, corrupted)
      SELECT not_cloned, cloning, cloned, failed_fetch, corrupted
      FROM diff
      WHERE
           not_cloned != 0
        OR cloning != 0
        OR cloned != 0
        OR failed_fetch != 0
        OR corrupted != 0
      ;

      RETURN NULL;
  END
$$;


--
-- Name: recalc_repo_statistics_on_repo_delete(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.recalc_repo_statistics_on_repo_delete() RETURNS trigger
    LANGUAGE plpgsql
    AS $$ BEGIN
      INSERT INTO
        repo_statistics (total, soft_deleted, not_cloned, cloning, cloned, failed_fetch)
      VALUES (
        -- Insert negative counts
        (SELECT -COUNT(*) FROM oldtab WHERE deleted_at IS NULL     AND blocked IS NULL),
        (SELECT -COUNT(*) FROM oldtab WHERE deleted_at IS NOT NULL AND blocked IS NULL),
        (SELECT -COUNT(*) FROM oldtab JOIN gitserver_repos gr ON gr.repo_id = oldtab.id WHERE oldtab.deleted_at is NULL AND oldtab.blocked IS NULL AND gr.clone_status = 'not_cloned'),
        (SELECT -COUNT(*) FROM oldtab JOIN gitserver_repos gr ON gr.repo_id = oldtab.id WHERE oldtab.deleted_at is NULL AND oldtab.blocked IS NULL AND gr.clone_status = 'cloning'),
        (SELECT -COUNT(*) FROM oldtab JOIN gitserver_repos gr ON gr.repo_id = oldtab.id WHERE oldtab.deleted_at is NULL AND oldtab.blocked IS NULL AND gr.clone_status = 'cloned'),
        (SELECT -COUNT(*) FROM oldtab JOIN gitserver_repos gr ON gr.repo_id = oldtab.id WHERE oldtab.deleted_at is NULL AND oldtab.blocked IS NULL AND gr.last_error IS NOT NULL)
      );
      RETURN NULL;
  END
$$;


--
-- Name: recalc_repo_statistics_on_repo_insert(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.recalc_repo_statistics_on_repo_insert() RETURNS trigger
    LANGUAGE plpgsql
    AS $$ BEGIN
      INSERT INTO
        repo_statistics (total, soft_deleted, not_cloned)
      VALUES (
        (SELECT COUNT(*) FROM newtab WHERE deleted_at IS NULL     AND blocked IS NULL),
        (SELECT COUNT(*) FROM newtab WHERE deleted_at IS NOT NULL AND blocked IS NULL),
        -- New repositories are always not_cloned by default, so we can count them as not cloned here
        (SELECT COUNT(*) FROM newtab WHERE deleted_at IS NULL     AND blocked IS NULL)
        -- New repositories never have last_error set, so we can also ignore those here
      );
      RETURN NULL;
  END
$$;


--
-- Name: recalc_repo_statistics_on_repo_update(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.recalc_repo_statistics_on_repo_update() RETURNS trigger
    LANGUAGE plpgsql
    AS $$ BEGIN
      -- Insert diff of changes
      WITH diff(total, soft_deleted, not_cloned, cloning, cloned, failed_fetch, corrupted) AS (
        VALUES (
          (SELECT COUNT(*) FROM newtab WHERE deleted_at IS NULL     AND blocked IS NULL) - (SELECT COUNT(*) FROM oldtab WHERE deleted_at IS NULL     AND blocked IS NULL),
          (SELECT COUNT(*) FROM newtab WHERE deleted_at IS NOT NULL AND blocked IS NULL) - (SELECT COUNT(*) FROM oldtab WHERE deleted_at IS NOT NULL AND blocked IS NULL),
          (
            (SELECT COUNT(*) FROM newtab JOIN gitserver_repos gr ON gr.repo_id = newtab.id WHERE newtab.deleted_at is NULL AND newtab.blocked IS NULL AND gr.clone_status = 'not_cloned')
            -
            (SELECT COUNT(*) FROM oldtab JOIN gitserver_repos gr ON gr.repo_id = oldtab.id WHERE oldtab.deleted_at is NULL AND oldtab.blocked IS NULL AND gr.clone_status = 'not_cloned')
          ),
          (
            (SELECT COUNT(*) FROM newtab JOIN gitserver_repos gr ON gr.repo_id = newtab.id WHERE newtab.deleted_at is NULL AND newtab.blocked IS NULL AND gr.clone_status = 'cloning')
            -
            (SELECT COUNT(*) FROM oldtab JOIN gitserver_repos gr ON gr.repo_id = oldtab.id WHERE oldtab.deleted_at is NULL AND oldtab.blocked IS NULL AND gr.clone_status = 'cloning')
          ),
          (
            (SELECT COUNT(*) FROM newtab JOIN gitserver_repos gr ON gr.repo_id = newtab.id WHERE newtab.deleted_at is NULL AND newtab.blocked IS NULL AND gr.clone_status = 'cloned')
            -
            (SELECT COUNT(*) FROM oldtab JOIN gitserver_repos gr ON gr.repo_id = oldtab.id WHERE oldtab.deleted_at is NULL AND oldtab.blocked IS NULL AND gr.clone_status = 'cloned')
          ),
          (
            (SELECT COUNT(*) FROM newtab JOIN gitserver_repos gr ON gr.repo_id = newtab.id WHERE newtab.deleted_at is NULL AND newtab.blocked IS NULL AND gr.last_error IS NOT NULL)
            -
            (SELECT COUNT(*) FROM oldtab JOIN gitserver_repos gr ON gr.repo_id = oldtab.id WHERE oldtab.deleted_at is NULL AND oldtab.blocked IS NULL AND gr.last_error IS NOT NULL)
          ),
          (
            (SELECT COUNT(*) FROM newtab JOIN gitserver_repos gr ON gr.repo_id = newtab.id WHERE newtab.deleted_at is NULL AND newtab.blocked IS NULL AND gr.corrupted_at IS NOT NULL)
            -
            (SELECT COUNT(*) FROM oldtab JOIN gitserver_repos gr ON gr.repo_id = oldtab.id WHERE oldtab.deleted_at is NULL AND oldtab.blocked IS NULL AND gr.corrupted_at IS NOT NULL)
          )
        )
      )
      INSERT INTO
        repo_statistics (total, soft_deleted, not_cloned, cloning, cloned, failed_fetch, corrupted)
      SELECT total, soft_deleted, not_cloned, cloning, cloned, failed_fetch, corrupted
      FROM diff
      WHERE
           total != 0
        OR soft_deleted != 0
        OR not_cloned != 0
        OR cloning != 0
        OR cloned != 0
        OR failed_fetch != 0
        OR corrupted != 0
      ;
      RETURN NULL;
  END
$$;


--
-- Name: repo_block(text, timestamp with time zone); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.repo_block(reason text, at timestamp with time zone) RETURNS jsonb
    LANGUAGE sql IMMUTABLE STRICT
    AS $$
SELECT jsonb_build_object(
    'reason', reason,
    'at', extract(epoch from timezone('utc', at))::bigint
);
$$;


--
-- Name: set_repo_stars_null_to_zero(); Type: PROCEDURE; Schema: public; Owner: -
--

CREATE PROCEDURE public.set_repo_stars_null_to_zero()
    LANGUAGE plpgsql
    AS $$
DECLARE
  done boolean;
  total integer = 0;
  updated integer = 0;

BEGIN
  SELECT COUNT(*) INTO total FROM repo WHERE stars IS NULL;

  RAISE NOTICE 'repo_stars_null_to_zero: updating % rows', total;

  done := total = 0;

  WHILE NOT done LOOP
    UPDATE repo SET stars = 0
    FROM (
      SELECT id FROM repo
      WHERE stars IS NULL
      LIMIT 10000
      FOR UPDATE SKIP LOCKED
    ) s
    WHERE repo.id = s.id;

    COMMIT;

    SELECT COUNT(*) = 0 INTO done FROM repo WHERE stars IS NULL LIMIT 1;

    updated := updated + 10000;

    RAISE NOTICE 'repo_stars_null_to_zero: updated % of % rows', updated, total;
  END LOOP;
END
$$;


--
-- Name: soft_delete_orphan_repo_by_external_service_repos(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.soft_delete_orphan_repo_by_external_service_repos() RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
    -- When an external service is soft or hard-deleted,
    -- performs a clean up to soft-delete orphan repositories.
    UPDATE
        repo
    SET
        name = soft_deleted_repository_name(name),
        deleted_at = transaction_timestamp()
    WHERE
      deleted_at IS NULL
      AND NOT EXISTS (
        SELECT FROM external_service_repos WHERE repo_id = repo.id
      );
END;
$$;


--
-- Name: soft_deleted_repository_name(text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.soft_deleted_repository_name(name text) RETURNS text
    LANGUAGE plpgsql
    AS $$
BEGIN
    IF name LIKE 'DELETED-%' THEN
        RETURN name;
    ELSE
        RETURN 'DELETED-' || extract(epoch from transaction_timestamp()) || '-' || name;
    END IF;
END;
$$;


--
-- Name: update_codeintel_path_ranks_statistics_columns(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.update_codeintel_path_ranks_statistics_columns() RETURNS trigger
    LANGUAGE plpgsql
    AS $$ BEGIN
    SELECT
        COUNT(r.v) AS num_paths,
        SUM(LOG(2, r.v::int + 1)) AS refcount_logsum
    INTO
        NEW.num_paths,
        NEW.refcount_logsum
    FROM jsonb_each(
        CASE WHEN NEW.payload::text = 'null'
            THEN '{}'::jsonb
            ELSE COALESCE(NEW.payload, '{}'::jsonb)
        END
    ) r(k, v);

    RETURN NEW;
END;
$$;


--
-- Name: update_codeintel_path_ranks_updated_at_column(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.update_codeintel_path_ranks_updated_at_column() RETURNS trigger
    LANGUAGE plpgsql
    AS $$ BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;


--
-- Name: update_own_aggregate_recent_contribution(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.update_own_aggregate_recent_contribution() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    WITH RECURSIVE ancestors AS (
        SELECT id, parent_id, 1 AS level
        FROM repo_paths
        WHERE id = NEW.changed_file_path_id
        UNION ALL
        SELECT p.id, p.parent_id, a.level + 1
        FROM repo_paths p
        JOIN ancestors a ON p.id = a.parent_id
    )
    UPDATE own_aggregate_recent_contribution
    SET contributions_count = contributions_count + 1
    WHERE commit_author_id = NEW.commit_author_id AND changed_file_path_id IN (
        SELECT id FROM ancestors
    );

    WITH RECURSIVE ancestors AS (
        SELECT id, parent_id, 1 AS level
        FROM repo_paths
        WHERE id = NEW.changed_file_path_id
        UNION ALL
        SELECT p.id, p.parent_id, a.level + 1
        FROM repo_paths p
        JOIN ancestors a ON p.id = a.parent_id
    )
    INSERT INTO own_aggregate_recent_contribution (commit_author_id, changed_file_path_id, contributions_count)
    SELECT NEW.commit_author_id, id, 1
    FROM ancestors
    WHERE NOT EXISTS (
        SELECT 1 FROM own_aggregate_recent_contribution
        WHERE commit_author_id = NEW.commit_author_id AND changed_file_path_id = ancestors.id
    );

    RETURN NEW;
END;
$$;


--
-- Name: versions_insert_row_trigger(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.versions_insert_row_trigger() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.first_version = NEW.version;
    RETURN NEW;
END $$;


--
-- Name: snapshot_transition_columns(public.hstore[]); Type: AGGREGATE; Schema: public; Owner: -
--

CREATE AGGREGATE public.snapshot_transition_columns(public.hstore[]) (
    SFUNC = public.merge_audit_log_transitions,
    STYPE = public.hstore,
    INITCOND = ''
);


SET default_table_access_method = heap;

--
-- Name: access_requests; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.access_requests (
    id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    name text NOT NULL,
    email text NOT NULL,
    additional_info text,
    status text NOT NULL,
    decision_by_user_id integer,
    tenant_id integer
);


--
-- Name: access_requests_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.access_requests_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: access_requests_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.access_requests_id_seq OWNED BY public.access_requests.id;


--
-- Name: access_tokens; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.access_tokens (
    id bigint NOT NULL,
    subject_user_id integer NOT NULL,
    value_sha256 bytea NOT NULL,
    note text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    last_used_at timestamp with time zone,
    deleted_at timestamp with time zone,
    creator_user_id integer NOT NULL,
    scopes text[] NOT NULL,
    internal boolean DEFAULT false,
    expires_at timestamp with time zone,
    tenant_id integer
);


--
-- Name: access_tokens_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.access_tokens_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: access_tokens_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.access_tokens_id_seq OWNED BY public.access_tokens.id;


--
-- Name: aggregated_user_statistics; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.aggregated_user_statistics (
    user_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    user_last_active_at timestamp with time zone,
    user_events_count bigint,
    tenant_id integer
);


--
-- Name: assigned_owners; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.assigned_owners (
    id integer NOT NULL,
    owner_user_id integer NOT NULL,
    file_path_id integer NOT NULL,
    who_assigned_user_id integer,
    assigned_at timestamp without time zone DEFAULT now() NOT NULL,
    tenant_id integer
);


--
-- Name: TABLE assigned_owners; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.assigned_owners IS 'Table for ownership assignments, one entry contains an assigned user ID, which repo_path is assigned and the date and user who assigned the owner.';


--
-- Name: assigned_owners_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.assigned_owners_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: assigned_owners_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.assigned_owners_id_seq OWNED BY public.assigned_owners.id;


--
-- Name: assigned_teams; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.assigned_teams (
    id integer NOT NULL,
    owner_team_id integer NOT NULL,
    file_path_id integer NOT NULL,
    who_assigned_team_id integer,
    assigned_at timestamp without time zone DEFAULT now() NOT NULL,
    tenant_id integer
);


--
-- Name: TABLE assigned_teams; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.assigned_teams IS 'Table for team ownership assignments, one entry contains an assigned team ID, which repo_path is assigned and the date and user who assigned the owner team.';


--
-- Name: assigned_teams_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.assigned_teams_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: assigned_teams_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.assigned_teams_id_seq OWNED BY public.assigned_teams.id;


--
-- Name: batch_changes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.batch_changes (
    id bigint NOT NULL,
    name text NOT NULL,
    description text,
    creator_id integer,
    namespace_user_id integer,
    namespace_org_id integer,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    closed_at timestamp with time zone,
    batch_spec_id bigint NOT NULL,
    last_applier_id bigint,
    last_applied_at timestamp with time zone,
    tenant_id integer,
    CONSTRAINT batch_change_name_is_valid CHECK ((name ~ '^[\w.-]+$'::text)),
    CONSTRAINT batch_changes_has_1_namespace CHECK (((namespace_user_id IS NULL) <> (namespace_org_id IS NULL))),
    CONSTRAINT batch_changes_name_not_blank CHECK ((name <> ''::text))
);


--
-- Name: batch_changes_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.batch_changes_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: batch_changes_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.batch_changes_id_seq OWNED BY public.batch_changes.id;


--
-- Name: batch_changes_site_credentials; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.batch_changes_site_credentials (
    id bigint NOT NULL,
    external_service_type text NOT NULL,
    external_service_id text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    credential bytea NOT NULL,
    encryption_key_id text DEFAULT ''::text NOT NULL,
    github_app_id integer,
    tenant_id integer,
    CONSTRAINT check_github_app_id_and_external_service_type_site_credentials CHECK (((github_app_id IS NULL) OR (external_service_type = 'github'::text)))
);


--
-- Name: batch_changes_site_credentials_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.batch_changes_site_credentials_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: batch_changes_site_credentials_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.batch_changes_site_credentials_id_seq OWNED BY public.batch_changes_site_credentials.id;


--
-- Name: batch_spec_execution_cache_entries; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.batch_spec_execution_cache_entries (
    id bigint NOT NULL,
    key text NOT NULL,
    value text NOT NULL,
    version integer NOT NULL,
    last_used_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    user_id integer NOT NULL,
    tenant_id integer
);


--
-- Name: batch_spec_execution_cache_entries_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.batch_spec_execution_cache_entries_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: batch_spec_execution_cache_entries_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.batch_spec_execution_cache_entries_id_seq OWNED BY public.batch_spec_execution_cache_entries.id;


--
-- Name: batch_spec_resolution_jobs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.batch_spec_resolution_jobs (
    id bigint NOT NULL,
    batch_spec_id integer NOT NULL,
    state text DEFAULT 'queued'::text NOT NULL,
    failure_message text,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    process_after timestamp with time zone,
    num_resets integer DEFAULT 0 NOT NULL,
    num_failures integer DEFAULT 0 NOT NULL,
    execution_logs json[],
    worker_hostname text DEFAULT ''::text NOT NULL,
    last_heartbeat_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    queued_at timestamp with time zone DEFAULT now(),
    initiator_id integer NOT NULL,
    cancel boolean DEFAULT false NOT NULL,
    tenant_id integer
);


--
-- Name: batch_spec_resolution_jobs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.batch_spec_resolution_jobs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: batch_spec_resolution_jobs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.batch_spec_resolution_jobs_id_seq OWNED BY public.batch_spec_resolution_jobs.id;


--
-- Name: batch_spec_workspace_execution_jobs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.batch_spec_workspace_execution_jobs (
    id bigint NOT NULL,
    batch_spec_workspace_id integer NOT NULL,
    state text DEFAULT 'queued'::text NOT NULL,
    failure_message text,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    process_after timestamp with time zone,
    num_resets integer DEFAULT 0 NOT NULL,
    num_failures integer DEFAULT 0 NOT NULL,
    execution_logs json[],
    worker_hostname text DEFAULT ''::text NOT NULL,
    last_heartbeat_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    cancel boolean DEFAULT false NOT NULL,
    queued_at timestamp with time zone DEFAULT now(),
    user_id integer NOT NULL,
    version integer DEFAULT 1 NOT NULL,
    tenant_id integer
);


--
-- Name: batch_spec_workspace_execution_jobs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.batch_spec_workspace_execution_jobs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: batch_spec_workspace_execution_jobs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.batch_spec_workspace_execution_jobs_id_seq OWNED BY public.batch_spec_workspace_execution_jobs.id;


--
-- Name: batch_spec_workspace_execution_last_dequeues; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.batch_spec_workspace_execution_last_dequeues (
    user_id integer NOT NULL,
    latest_dequeue timestamp with time zone,
    tenant_id integer
);


--
-- Name: batch_spec_workspace_execution_queue; Type: VIEW; Schema: public; Owner: -
--

CREATE VIEW public.batch_spec_workspace_execution_queue AS
 WITH queue_candidates AS (
         SELECT exec.id,
            rank() OVER (PARTITION BY queue.user_id ORDER BY exec.created_at, exec.id) AS place_in_user_queue
           FROM (public.batch_spec_workspace_execution_jobs exec
             JOIN public.batch_spec_workspace_execution_last_dequeues queue ON ((queue.user_id = exec.user_id)))
          WHERE (exec.state = 'queued'::text)
          ORDER BY (rank() OVER (PARTITION BY queue.user_id ORDER BY exec.created_at, exec.id)), queue.latest_dequeue NULLS FIRST
        )
 SELECT id,
    row_number() OVER () AS place_in_global_queue,
    place_in_user_queue
   FROM queue_candidates;


--
-- Name: batch_spec_workspace_execution_jobs_with_rank; Type: VIEW; Schema: public; Owner: -
--

CREATE VIEW public.batch_spec_workspace_execution_jobs_with_rank AS
 SELECT j.id,
    j.batch_spec_workspace_id,
    j.state,
    j.failure_message,
    j.started_at,
    j.finished_at,
    j.process_after,
    j.num_resets,
    j.num_failures,
    j.execution_logs,
    j.worker_hostname,
    j.last_heartbeat_at,
    j.created_at,
    j.updated_at,
    j.cancel,
    j.queued_at,
    j.user_id,
    j.version,
    q.place_in_global_queue,
    q.place_in_user_queue
   FROM (public.batch_spec_workspace_execution_jobs j
     LEFT JOIN public.batch_spec_workspace_execution_queue q ON ((j.id = q.id)));


--
-- Name: batch_spec_workspace_files; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.batch_spec_workspace_files (
    id integer NOT NULL,
    rand_id text NOT NULL,
    batch_spec_id bigint NOT NULL,
    filename text NOT NULL,
    path text NOT NULL,
    size bigint NOT NULL,
    content bytea NOT NULL,
    modified_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    tenant_id integer
);


--
-- Name: batch_spec_workspace_files_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.batch_spec_workspace_files_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: batch_spec_workspace_files_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.batch_spec_workspace_files_id_seq OWNED BY public.batch_spec_workspace_files.id;


--
-- Name: batch_spec_workspaces; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.batch_spec_workspaces (
    id bigint NOT NULL,
    batch_spec_id integer NOT NULL,
    changeset_spec_ids jsonb DEFAULT '{}'::jsonb NOT NULL,
    repo_id integer NOT NULL,
    branch text NOT NULL,
    commit text NOT NULL,
    path text NOT NULL,
    file_matches text[] NOT NULL,
    only_fetch_workspace boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    ignored boolean DEFAULT false NOT NULL,
    unsupported boolean DEFAULT false NOT NULL,
    skipped boolean DEFAULT false NOT NULL,
    cached_result_found boolean DEFAULT false NOT NULL,
    step_cache_results jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id integer
);


--
-- Name: batch_spec_workspaces_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.batch_spec_workspaces_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: batch_spec_workspaces_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.batch_spec_workspaces_id_seq OWNED BY public.batch_spec_workspaces.id;


--
-- Name: batch_specs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.batch_specs (
    id bigint NOT NULL,
    rand_id text NOT NULL,
    raw_spec text NOT NULL,
    spec jsonb DEFAULT '{}'::jsonb NOT NULL,
    namespace_user_id integer,
    namespace_org_id integer,
    user_id integer,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_from_raw boolean DEFAULT false NOT NULL,
    allow_unsupported boolean DEFAULT false NOT NULL,
    allow_ignored boolean DEFAULT false NOT NULL,
    no_cache boolean DEFAULT false NOT NULL,
    batch_change_id bigint,
    tenant_id integer,
    CONSTRAINT batch_specs_has_1_namespace CHECK (((namespace_user_id IS NULL) <> (namespace_org_id IS NULL)))
);


--
-- Name: batch_specs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.batch_specs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: batch_specs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.batch_specs_id_seq OWNED BY public.batch_specs.id;


--
-- Name: changeset_specs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.changeset_specs (
    id bigint NOT NULL,
    rand_id text NOT NULL,
    spec jsonb DEFAULT '{}'::jsonb,
    batch_spec_id bigint,
    repo_id integer NOT NULL,
    user_id integer,
    diff_stat_added integer,
    diff_stat_deleted integer,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    head_ref text,
    title text,
    external_id text,
    fork_namespace public.citext,
    diff bytea,
    base_rev text,
    base_ref text,
    body text,
    published text,
    commit_message text,
    commit_author_name text,
    commit_author_email text,
    type text NOT NULL,
    tenant_id integer,
    CONSTRAINT changeset_specs_published_valid_values CHECK (((published = 'true'::text) OR (published = 'false'::text) OR (published = '"draft"'::text) OR (published IS NULL)))
);


--
-- Name: changesets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.changesets (
    id bigint NOT NULL,
    batch_change_ids jsonb DEFAULT '{}'::jsonb NOT NULL,
    repo_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb,
    external_id text,
    external_service_type text NOT NULL,
    external_deleted_at timestamp with time zone,
    external_branch text,
    external_updated_at timestamp with time zone,
    external_state text,
    external_review_state text,
    external_check_state text,
    diff_stat_added integer,
    diff_stat_deleted integer,
    sync_state jsonb DEFAULT '{}'::jsonb NOT NULL,
    current_spec_id bigint,
    previous_spec_id bigint,
    publication_state text DEFAULT 'UNPUBLISHED'::text,
    owned_by_batch_change_id bigint,
    reconciler_state text DEFAULT 'queued'::text,
    failure_message text,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    process_after timestamp with time zone,
    num_resets integer DEFAULT 0 NOT NULL,
    closing boolean DEFAULT false NOT NULL,
    num_failures integer DEFAULT 0 NOT NULL,
    log_contents text,
    execution_logs json[],
    syncer_error text,
    external_title text,
    worker_hostname text DEFAULT ''::text NOT NULL,
    ui_publication_state public.batch_changes_changeset_ui_publication_state,
    last_heartbeat_at timestamp with time zone,
    external_fork_namespace public.citext,
    queued_at timestamp with time zone DEFAULT now(),
    cancel boolean DEFAULT false NOT NULL,
    detached_at timestamp with time zone,
    computed_state text NOT NULL,
    external_fork_name public.citext,
    previous_failure_message text,
    commit_verification jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id integer,
    CONSTRAINT changesets_batch_change_ids_check CHECK ((jsonb_typeof(batch_change_ids) = 'object'::text)),
    CONSTRAINT changesets_external_id_check CHECK ((external_id <> ''::text)),
    CONSTRAINT changesets_external_service_type_not_blank CHECK ((external_service_type <> ''::text)),
    CONSTRAINT changesets_metadata_check CHECK ((jsonb_typeof(metadata) = 'object'::text)),
    CONSTRAINT external_branch_ref_prefix CHECK ((external_branch ~~ 'refs/heads/%'::text))
);


--
-- Name: COLUMN changesets.external_title; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.changesets.external_title IS 'Normalized property generated on save using Changeset.Title()';


--
-- Name: repo; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.repo (
    id integer NOT NULL,
    name public.citext NOT NULL,
    description text,
    fork boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone,
    external_id text,
    external_service_type text,
    external_service_id text,
    archived boolean DEFAULT false NOT NULL,
    uri public.citext,
    deleted_at timestamp with time zone,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    private boolean DEFAULT false NOT NULL,
    stars integer DEFAULT 0 NOT NULL,
    blocked jsonb,
    topics text[] GENERATED ALWAYS AS (public.extract_topics_from_metadata(external_service_type, metadata)) STORED,
    tenant_id integer,
    CONSTRAINT check_name_nonempty CHECK ((name OPERATOR(public.<>) ''::public.citext)),
    CONSTRAINT repo_metadata_check CHECK ((jsonb_typeof(metadata) = 'object'::text))
);


--
-- Name: branch_changeset_specs_and_changesets; Type: VIEW; Schema: public; Owner: -
--

CREATE VIEW public.branch_changeset_specs_and_changesets AS
 SELECT changeset_specs.id AS changeset_spec_id,
    COALESCE(changesets.id, (0)::bigint) AS changeset_id,
    changeset_specs.repo_id,
    changeset_specs.batch_spec_id,
    changesets.owned_by_batch_change_id AS owner_batch_change_id,
    repo.name AS repo_name,
    changeset_specs.title AS changeset_name,
    changesets.external_state,
    changesets.publication_state,
    changesets.reconciler_state,
    changesets.computed_state
   FROM ((public.changeset_specs
     LEFT JOIN public.changesets ON (((changesets.repo_id = changeset_specs.repo_id) AND (changesets.current_spec_id IS NOT NULL) AND (EXISTS ( SELECT 1
           FROM public.changeset_specs changeset_specs_1
          WHERE ((changeset_specs_1.id = changesets.current_spec_id) AND (changeset_specs_1.head_ref = changeset_specs.head_ref)))))))
     JOIN public.repo ON ((changeset_specs.repo_id = repo.id)))
  WHERE ((changeset_specs.external_id IS NULL) AND (repo.deleted_at IS NULL));


--
-- Name: cached_available_indexers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.cached_available_indexers (
    id integer NOT NULL,
    repository_id integer NOT NULL,
    num_events integer NOT NULL,
    available_indexers jsonb NOT NULL,
    tenant_id integer
);


--
-- Name: cached_available_indexers_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.cached_available_indexers_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: cached_available_indexers_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.cached_available_indexers_id_seq OWNED BY public.cached_available_indexers.id;


--
-- Name: changeset_events; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.changeset_events (
    id bigint NOT NULL,
    changeset_id bigint NOT NULL,
    kind text NOT NULL,
    key text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    tenant_id integer,
    CONSTRAINT changeset_events_key_check CHECK ((key <> ''::text)),
    CONSTRAINT changeset_events_kind_check CHECK ((kind <> ''::text)),
    CONSTRAINT changeset_events_metadata_check CHECK ((jsonb_typeof(metadata) = 'object'::text))
);


--
-- Name: changeset_events_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.changeset_events_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: changeset_events_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.changeset_events_id_seq OWNED BY public.changeset_events.id;


--
-- Name: changeset_jobs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.changeset_jobs (
    id bigint NOT NULL,
    bulk_group text NOT NULL,
    user_id integer NOT NULL,
    batch_change_id integer NOT NULL,
    changeset_id integer NOT NULL,
    job_type text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb,
    state text DEFAULT 'queued'::text NOT NULL,
    failure_message text,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    process_after timestamp with time zone,
    num_resets integer DEFAULT 0 NOT NULL,
    num_failures integer DEFAULT 0 NOT NULL,
    execution_logs json[],
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    worker_hostname text DEFAULT ''::text NOT NULL,
    last_heartbeat_at timestamp with time zone,
    queued_at timestamp with time zone DEFAULT now(),
    cancel boolean DEFAULT false NOT NULL,
    tenant_id integer,
    CONSTRAINT changeset_jobs_payload_check CHECK ((jsonb_typeof(payload) = 'object'::text))
);


--
-- Name: changeset_jobs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.changeset_jobs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: changeset_jobs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.changeset_jobs_id_seq OWNED BY public.changeset_jobs.id;


--
-- Name: changeset_specs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.changeset_specs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: changeset_specs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.changeset_specs_id_seq OWNED BY public.changeset_specs.id;


--
-- Name: changesets_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.changesets_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: changesets_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.changesets_id_seq OWNED BY public.changesets.id;


--
-- Name: cm_action_jobs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.cm_action_jobs (
    id integer NOT NULL,
    email bigint,
    state text DEFAULT 'queued'::text,
    failure_message text,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    process_after timestamp with time zone,
    num_resets integer DEFAULT 0 NOT NULL,
    num_failures integer DEFAULT 0 NOT NULL,
    log_contents text,
    trigger_event integer,
    worker_hostname text DEFAULT ''::text NOT NULL,
    last_heartbeat_at timestamp with time zone,
    execution_logs json[],
    webhook bigint,
    slack_webhook bigint,
    queued_at timestamp with time zone DEFAULT now(),
    cancel boolean DEFAULT false NOT NULL,
    tenant_id integer,
    CONSTRAINT cm_action_jobs_only_one_action_type CHECK ((((
CASE
    WHEN (email IS NULL) THEN 0
    ELSE 1
END +
CASE
    WHEN (webhook IS NULL) THEN 0
    ELSE 1
END) +
CASE
    WHEN (slack_webhook IS NULL) THEN 0
    ELSE 1
END) = 1))
);


--
-- Name: COLUMN cm_action_jobs.email; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.cm_action_jobs.email IS 'The ID of the cm_emails action to execute if this is an email job. Mutually exclusive with webhook and slack_webhook';


--
-- Name: COLUMN cm_action_jobs.webhook; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.cm_action_jobs.webhook IS 'The ID of the cm_webhooks action to execute if this is a webhook job. Mutually exclusive with email and slack_webhook';


--
-- Name: COLUMN cm_action_jobs.slack_webhook; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.cm_action_jobs.slack_webhook IS 'The ID of the cm_slack_webhook action to execute if this is a slack webhook job. Mutually exclusive with email and webhook';


--
-- Name: CONSTRAINT cm_action_jobs_only_one_action_type ON cm_action_jobs; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON CONSTRAINT cm_action_jobs_only_one_action_type ON public.cm_action_jobs IS 'Constrains that each queued code monitor action has exactly one action type';


--
-- Name: cm_action_jobs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.cm_action_jobs_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: cm_action_jobs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.cm_action_jobs_id_seq OWNED BY public.cm_action_jobs.id;


--
-- Name: cm_emails; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.cm_emails (
    id bigint NOT NULL,
    monitor bigint NOT NULL,
    enabled boolean NOT NULL,
    priority public.cm_email_priority NOT NULL,
    header text NOT NULL,
    created_by integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    changed_by integer NOT NULL,
    changed_at timestamp with time zone DEFAULT now() NOT NULL,
    include_results boolean DEFAULT false NOT NULL,
    tenant_id integer
);


--
-- Name: cm_emails_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.cm_emails_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: cm_emails_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.cm_emails_id_seq OWNED BY public.cm_emails.id;


--
-- Name: cm_last_searched; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.cm_last_searched (
    monitor_id bigint NOT NULL,
    commit_oids text[] NOT NULL,
    repo_id integer NOT NULL,
    tenant_id integer
);


--
-- Name: TABLE cm_last_searched; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.cm_last_searched IS 'The last searched commit hashes for the given code monitor and unique set of search arguments';


--
-- Name: COLUMN cm_last_searched.commit_oids; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.cm_last_searched.commit_oids IS 'The set of commit OIDs that was previously successfully searched and should be excluded on the next run';


--
-- Name: cm_monitors; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.cm_monitors (
    id bigint NOT NULL,
    created_by integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    description text NOT NULL,
    changed_at timestamp with time zone DEFAULT now() NOT NULL,
    changed_by integer NOT NULL,
    enabled boolean DEFAULT true NOT NULL,
    namespace_user_id integer NOT NULL,
    namespace_org_id integer,
    tenant_id integer
);


--
-- Name: COLUMN cm_monitors.namespace_org_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.cm_monitors.namespace_org_id IS 'DEPRECATED: code monitors cannot be owned by an org';


--
-- Name: cm_monitors_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.cm_monitors_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: cm_monitors_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.cm_monitors_id_seq OWNED BY public.cm_monitors.id;


--
-- Name: cm_queries; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.cm_queries (
    id bigint NOT NULL,
    monitor bigint NOT NULL,
    query text NOT NULL,
    created_by integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    changed_by integer NOT NULL,
    changed_at timestamp with time zone DEFAULT now() NOT NULL,
    next_run timestamp with time zone DEFAULT now(),
    latest_result timestamp with time zone,
    tenant_id integer
);


--
-- Name: cm_queries_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.cm_queries_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: cm_queries_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.cm_queries_id_seq OWNED BY public.cm_queries.id;


--
-- Name: cm_recipients; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.cm_recipients (
    id bigint NOT NULL,
    email bigint NOT NULL,
    namespace_user_id integer,
    namespace_org_id integer,
    tenant_id integer
);


--
-- Name: cm_recipients_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.cm_recipients_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: cm_recipients_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.cm_recipients_id_seq OWNED BY public.cm_recipients.id;


--
-- Name: cm_slack_webhooks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.cm_slack_webhooks (
    id bigint NOT NULL,
    monitor bigint NOT NULL,
    url text NOT NULL,
    enabled boolean NOT NULL,
    created_by integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    changed_by integer NOT NULL,
    changed_at timestamp with time zone DEFAULT now() NOT NULL,
    include_results boolean DEFAULT false NOT NULL,
    tenant_id integer
);


--
-- Name: TABLE cm_slack_webhooks; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.cm_slack_webhooks IS 'Slack webhook actions configured on code monitors';


--
-- Name: COLUMN cm_slack_webhooks.monitor; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.cm_slack_webhooks.monitor IS 'The code monitor that the action is defined on';


--
-- Name: COLUMN cm_slack_webhooks.url; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.cm_slack_webhooks.url IS 'The Slack webhook URL we send the code monitor event to';


--
-- Name: cm_slack_webhooks_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.cm_slack_webhooks_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: cm_slack_webhooks_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.cm_slack_webhooks_id_seq OWNED BY public.cm_slack_webhooks.id;


--
-- Name: cm_trigger_jobs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.cm_trigger_jobs (
    id integer NOT NULL,
    query bigint NOT NULL,
    state text DEFAULT 'queued'::text,
    failure_message text,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    process_after timestamp with time zone,
    num_resets integer DEFAULT 0 NOT NULL,
    num_failures integer DEFAULT 0 NOT NULL,
    log_contents text,
    query_string text,
    worker_hostname text DEFAULT ''::text NOT NULL,
    last_heartbeat_at timestamp with time zone,
    execution_logs json[],
    search_results jsonb,
    queued_at timestamp with time zone DEFAULT now(),
    cancel boolean DEFAULT false NOT NULL,
    logs json[],
    tenant_id integer,
    CONSTRAINT search_results_is_array CHECK ((jsonb_typeof(search_results) = 'array'::text))
);


--
-- Name: cm_trigger_jobs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.cm_trigger_jobs_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: cm_trigger_jobs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.cm_trigger_jobs_id_seq OWNED BY public.cm_trigger_jobs.id;


--
-- Name: cm_webhooks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.cm_webhooks (
    id bigint NOT NULL,
    monitor bigint NOT NULL,
    url text NOT NULL,
    enabled boolean NOT NULL,
    created_by integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    changed_by integer NOT NULL,
    changed_at timestamp with time zone DEFAULT now() NOT NULL,
    include_results boolean DEFAULT false NOT NULL,
    tenant_id integer
);


--
-- Name: TABLE cm_webhooks; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.cm_webhooks IS 'Webhook actions configured on code monitors';


--
-- Name: COLUMN cm_webhooks.monitor; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.cm_webhooks.monitor IS 'The code monitor that the action is defined on';


--
-- Name: COLUMN cm_webhooks.url; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.cm_webhooks.url IS 'The webhook URL we send the code monitor event to';


--
-- Name: COLUMN cm_webhooks.enabled; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.cm_webhooks.enabled IS 'Whether this Slack webhook action is enabled. When not enabled, the action will not be run when its code monitor generates events';


--
-- Name: cm_webhooks_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.cm_webhooks_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: cm_webhooks_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.cm_webhooks_id_seq OWNED BY public.cm_webhooks.id;


--
-- Name: code_hosts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.code_hosts (
    id integer NOT NULL,
    kind text NOT NULL,
    url text NOT NULL,
    api_rate_limit_quota integer,
    api_rate_limit_interval_seconds integer,
    git_rate_limit_quota integer,
    git_rate_limit_interval_seconds integer,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    tenant_id integer
);


--
-- Name: code_hosts_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.code_hosts_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: code_hosts_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.code_hosts_id_seq OWNED BY public.code_hosts.id;


--
-- Name: codeintel_autoindex_queue; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.codeintel_autoindex_queue (
    id integer NOT NULL,
    repository_id integer NOT NULL,
    rev text NOT NULL,
    queued_at timestamp with time zone DEFAULT now() NOT NULL,
    processed_at timestamp with time zone,
    tenant_id integer
);


--
-- Name: codeintel_autoindex_queue_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.codeintel_autoindex_queue_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: codeintel_autoindex_queue_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.codeintel_autoindex_queue_id_seq OWNED BY public.codeintel_autoindex_queue.id;


--
-- Name: codeintel_autoindexing_exceptions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.codeintel_autoindexing_exceptions (
    id integer NOT NULL,
    repository_id integer NOT NULL,
    disable_scheduling boolean DEFAULT false NOT NULL,
    disable_inference boolean DEFAULT false NOT NULL,
    tenant_id integer
);


--
-- Name: codeintel_autoindexing_exceptions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.codeintel_autoindexing_exceptions_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: codeintel_autoindexing_exceptions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.codeintel_autoindexing_exceptions_id_seq OWNED BY public.codeintel_autoindexing_exceptions.id;


--
-- Name: codeintel_commit_dates; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.codeintel_commit_dates (
    repository_id integer NOT NULL,
    commit_bytea bytea NOT NULL,
    committed_at timestamp with time zone,
    tenant_id integer
);


--
-- Name: TABLE codeintel_commit_dates; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.codeintel_commit_dates IS 'Maps commits within a repository to the commit date as reported by gitserver.';


--
-- Name: COLUMN codeintel_commit_dates.repository_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.codeintel_commit_dates.repository_id IS 'Identifies a row in the `repo` table.';


--
-- Name: COLUMN codeintel_commit_dates.commit_bytea; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.codeintel_commit_dates.commit_bytea IS 'Identifies the 40-character commit hash.';


--
-- Name: COLUMN codeintel_commit_dates.committed_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.codeintel_commit_dates.committed_at IS 'The commit date (may be -infinity if unresolvable).';


--
-- Name: lsif_configuration_policies; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.lsif_configuration_policies (
    id integer NOT NULL,
    repository_id integer,
    name text,
    type text NOT NULL,
    pattern text NOT NULL,
    retention_enabled boolean NOT NULL,
    retention_duration_hours integer,
    retain_intermediate_commits boolean NOT NULL,
    indexing_enabled boolean NOT NULL,
    index_commit_max_age_hours integer,
    index_intermediate_commits boolean NOT NULL,
    protected boolean DEFAULT false NOT NULL,
    repository_patterns text[],
    last_resolved_at timestamp with time zone,
    embeddings_enabled boolean DEFAULT false NOT NULL,
    syntactic_indexing_enabled boolean DEFAULT false NOT NULL
);


--
-- Name: COLUMN lsif_configuration_policies.repository_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_configuration_policies.repository_id IS 'The identifier of the repository to which this configuration policy applies. If absent, this policy is applied globally.';


--
-- Name: COLUMN lsif_configuration_policies.type; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_configuration_policies.type IS 'The type of Git object (e.g., COMMIT, BRANCH, TAG).';


--
-- Name: COLUMN lsif_configuration_policies.pattern; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_configuration_policies.pattern IS 'A pattern used to match` names of the associated Git object type.';


--
-- Name: COLUMN lsif_configuration_policies.retention_enabled; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_configuration_policies.retention_enabled IS 'Whether or not this configuration policy affects data retention rules.';


--
-- Name: COLUMN lsif_configuration_policies.retention_duration_hours; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_configuration_policies.retention_duration_hours IS 'The max age of data retained by this configuration policy. If null, the age is unbounded.';


--
-- Name: COLUMN lsif_configuration_policies.retain_intermediate_commits; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_configuration_policies.retain_intermediate_commits IS 'If the matching Git object is a branch, setting this value to true will also retain all data used to resolve queries for any commit on the matching branches. Setting this value to false will only consider the tip of the branch.';


--
-- Name: COLUMN lsif_configuration_policies.indexing_enabled; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_configuration_policies.indexing_enabled IS 'Whether or not this configuration policy affects auto-indexing schedules.';


--
-- Name: COLUMN lsif_configuration_policies.index_commit_max_age_hours; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_configuration_policies.index_commit_max_age_hours IS 'The max age of commits indexed by this configuration policy. If null, the age is unbounded.';


--
-- Name: COLUMN lsif_configuration_policies.index_intermediate_commits; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_configuration_policies.index_intermediate_commits IS 'If the matching Git object is a branch, setting this value to true will also index all commits on the matching branches. Setting this value to false will only consider the tip of the branch.';


--
-- Name: COLUMN lsif_configuration_policies.protected; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_configuration_policies.protected IS 'Whether or not this configuration policy is protected from modification of its data retention behavior (except for duration).';


--
-- Name: COLUMN lsif_configuration_policies.repository_patterns; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_configuration_policies.repository_patterns IS 'The name pattern matching repositories to which this configuration policy applies. If absent, all repositories are matched.';


--
-- Name: codeintel_configuration_policies; Type: VIEW; Schema: public; Owner: -
--

CREATE VIEW public.codeintel_configuration_policies AS
 SELECT id,
    repository_id,
    name,
    type,
    pattern,
    retention_enabled,
    retention_duration_hours,
    retain_intermediate_commits,
    indexing_enabled,
    index_commit_max_age_hours,
    index_intermediate_commits,
    protected,
    repository_patterns,
    last_resolved_at,
    embeddings_enabled
   FROM public.lsif_configuration_policies;


--
-- Name: lsif_configuration_policies_repository_pattern_lookup; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.lsif_configuration_policies_repository_pattern_lookup (
    policy_id integer NOT NULL,
    repo_id integer NOT NULL
);


--
-- Name: TABLE lsif_configuration_policies_repository_pattern_lookup; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.lsif_configuration_policies_repository_pattern_lookup IS 'A lookup table to get all the repository patterns by repository id that apply to a configuration policy.';


--
-- Name: COLUMN lsif_configuration_policies_repository_pattern_lookup.policy_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_configuration_policies_repository_pattern_lookup.policy_id IS 'The policy identifier associated with the repository.';


--
-- Name: COLUMN lsif_configuration_policies_repository_pattern_lookup.repo_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_configuration_policies_repository_pattern_lookup.repo_id IS 'The repository identifier associated with the policy.';


--
-- Name: codeintel_configuration_policies_repository_pattern_lookup; Type: VIEW; Schema: public; Owner: -
--

CREATE VIEW public.codeintel_configuration_policies_repository_pattern_lookup AS
 SELECT policy_id,
    repo_id
   FROM public.lsif_configuration_policies_repository_pattern_lookup;


--
-- Name: codeintel_inference_scripts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.codeintel_inference_scripts (
    insert_timestamp timestamp with time zone DEFAULT now() NOT NULL,
    script text NOT NULL,
    tenant_id integer
);


--
-- Name: TABLE codeintel_inference_scripts; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.codeintel_inference_scripts IS 'Contains auto-index job inference Lua scripts as an alternative to setting via environment variables.';


--
-- Name: codeintel_initial_path_ranks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.codeintel_initial_path_ranks (
    id bigint NOT NULL,
    document_path text DEFAULT ''::text NOT NULL,
    graph_key text NOT NULL,
    document_paths text[] DEFAULT '{}'::text[] NOT NULL,
    exported_upload_id integer NOT NULL,
    tenant_id integer
);


--
-- Name: codeintel_initial_path_ranks_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.codeintel_initial_path_ranks_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: codeintel_initial_path_ranks_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.codeintel_initial_path_ranks_id_seq OWNED BY public.codeintel_initial_path_ranks.id;


--
-- Name: codeintel_initial_path_ranks_processed; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.codeintel_initial_path_ranks_processed (
    id bigint NOT NULL,
    graph_key text NOT NULL,
    codeintel_initial_path_ranks_id bigint NOT NULL,
    tenant_id integer
);


--
-- Name: codeintel_initial_path_ranks_processed_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.codeintel_initial_path_ranks_processed_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: codeintel_initial_path_ranks_processed_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.codeintel_initial_path_ranks_processed_id_seq OWNED BY public.codeintel_initial_path_ranks_processed.id;


--
-- Name: codeintel_langugage_support_requests; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.codeintel_langugage_support_requests (
    id integer NOT NULL,
    user_id integer NOT NULL,
    language_id text NOT NULL,
    tenant_id integer
);


--
-- Name: codeintel_langugage_support_requests_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.codeintel_langugage_support_requests_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: codeintel_langugage_support_requests_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.codeintel_langugage_support_requests_id_seq OWNED BY public.codeintel_langugage_support_requests.id;


--
-- Name: codeintel_path_ranks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.codeintel_path_ranks (
    repository_id integer NOT NULL,
    payload jsonb NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    graph_key text NOT NULL,
    num_paths integer,
    refcount_logsum double precision,
    id bigint NOT NULL,
    tenant_id integer
);


--
-- Name: codeintel_path_ranks_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.codeintel_path_ranks_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: codeintel_path_ranks_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.codeintel_path_ranks_id_seq OWNED BY public.codeintel_path_ranks.id;


--
-- Name: codeintel_ranking_definitions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.codeintel_ranking_definitions (
    id bigint NOT NULL,
    symbol_name text NOT NULL,
    document_path text NOT NULL,
    graph_key text NOT NULL,
    exported_upload_id integer NOT NULL,
    symbol_checksum bytea DEFAULT '\x'::bytea NOT NULL,
    tenant_id integer
);


--
-- Name: codeintel_ranking_definitions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.codeintel_ranking_definitions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: codeintel_ranking_definitions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.codeintel_ranking_definitions_id_seq OWNED BY public.codeintel_ranking_definitions.id;


--
-- Name: codeintel_ranking_exports; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.codeintel_ranking_exports (
    upload_id integer,
    graph_key text NOT NULL,
    locked_at timestamp with time zone DEFAULT now() NOT NULL,
    id integer NOT NULL,
    last_scanned_at timestamp with time zone,
    deleted_at timestamp with time zone,
    upload_key text,
    tenant_id integer
);


--
-- Name: codeintel_ranking_exports_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.codeintel_ranking_exports_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: codeintel_ranking_exports_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.codeintel_ranking_exports_id_seq OWNED BY public.codeintel_ranking_exports.id;


--
-- Name: codeintel_ranking_graph_keys; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.codeintel_ranking_graph_keys (
    id integer NOT NULL,
    graph_key text NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    tenant_id integer
);


--
-- Name: codeintel_ranking_graph_keys_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.codeintel_ranking_graph_keys_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: codeintel_ranking_graph_keys_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.codeintel_ranking_graph_keys_id_seq OWNED BY public.codeintel_ranking_graph_keys.id;


--
-- Name: codeintel_ranking_path_counts_inputs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.codeintel_ranking_path_counts_inputs (
    id bigint NOT NULL,
    count integer NOT NULL,
    graph_key text NOT NULL,
    processed boolean DEFAULT false NOT NULL,
    definition_id bigint,
    tenant_id integer
);


--
-- Name: codeintel_ranking_path_counts_inputs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.codeintel_ranking_path_counts_inputs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: codeintel_ranking_path_counts_inputs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.codeintel_ranking_path_counts_inputs_id_seq OWNED BY public.codeintel_ranking_path_counts_inputs.id;


--
-- Name: codeintel_ranking_progress; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.codeintel_ranking_progress (
    id bigint NOT NULL,
    graph_key text NOT NULL,
    mappers_started_at timestamp with time zone NOT NULL,
    mapper_completed_at timestamp with time zone,
    seed_mapper_completed_at timestamp with time zone,
    reducer_started_at timestamp with time zone,
    reducer_completed_at timestamp with time zone,
    num_path_records_total integer,
    num_reference_records_total integer,
    num_count_records_total integer,
    num_path_records_processed integer,
    num_reference_records_processed integer,
    num_count_records_processed integer,
    max_export_id bigint NOT NULL,
    reference_cursor_export_deleted_at timestamp with time zone,
    reference_cursor_export_id integer,
    path_cursor_deleted_export_at timestamp with time zone,
    path_cursor_export_id integer,
    tenant_id integer
);


--
-- Name: codeintel_ranking_progress_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.codeintel_ranking_progress_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: codeintel_ranking_progress_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.codeintel_ranking_progress_id_seq OWNED BY public.codeintel_ranking_progress.id;


--
-- Name: codeintel_ranking_references; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.codeintel_ranking_references (
    id bigint NOT NULL,
    symbol_names text[] NOT NULL,
    graph_key text NOT NULL,
    exported_upload_id integer NOT NULL,
    symbol_checksums bytea[] DEFAULT '{}'::bytea[] NOT NULL,
    tenant_id integer
);


--
-- Name: TABLE codeintel_ranking_references; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.codeintel_ranking_references IS 'References for a given upload proceduced by background job consuming SCIP indexes.';


--
-- Name: codeintel_ranking_references_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.codeintel_ranking_references_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: codeintel_ranking_references_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.codeintel_ranking_references_id_seq OWNED BY public.codeintel_ranking_references.id;


--
-- Name: codeintel_ranking_references_processed; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.codeintel_ranking_references_processed (
    graph_key text NOT NULL,
    codeintel_ranking_reference_id integer NOT NULL,
    id bigint NOT NULL,
    tenant_id integer
);


--
-- Name: codeintel_ranking_references_processed_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.codeintel_ranking_references_processed_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: codeintel_ranking_references_processed_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.codeintel_ranking_references_processed_id_seq OWNED BY public.codeintel_ranking_references_processed.id;


--
-- Name: codeowners; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.codeowners (
    id integer NOT NULL,
    contents text NOT NULL,
    contents_proto bytea NOT NULL,
    repo_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    tenant_id integer
);


--
-- Name: codeowners_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.codeowners_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: codeowners_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.codeowners_id_seq OWNED BY public.codeowners.id;


--
-- Name: codeowners_individual_stats; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.codeowners_individual_stats (
    file_path_id integer NOT NULL,
    owner_id integer NOT NULL,
    tree_owned_files_count integer NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    tenant_id integer
);


--
-- Name: TABLE codeowners_individual_stats; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.codeowners_individual_stats IS 'Data on how many files in given tree are owned by given owner.

As opposed to ownership-general `ownership_path_stats` table, the individual <path x owner> stats
are stored in CODEOWNERS-specific table `codeowners_individual_stats`. The reason for that is that
we are also indexing on owner_id which is CODEOWNERS-specific.';


--
-- Name: COLUMN codeowners_individual_stats.tree_owned_files_count; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.codeowners_individual_stats.tree_owned_files_count IS 'Total owned file count by given owner at given file tree.';


--
-- Name: COLUMN codeowners_individual_stats.updated_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.codeowners_individual_stats.updated_at IS 'When the last background job updating counts run.';


--
-- Name: codeowners_owners; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.codeowners_owners (
    id integer NOT NULL,
    reference text NOT NULL,
    tenant_id integer
);


--
-- Name: TABLE codeowners_owners; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.codeowners_owners IS 'Text reference in CODEOWNERS entry to use in codeowners_individual_stats. Reference is either email or handle without @ in front.';


--
-- Name: COLUMN codeowners_owners.reference; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.codeowners_owners.reference IS 'We just keep the reference as opposed to splitting it to handle or email
since the distinction is not relevant for query, and this makes indexing way easier.';


--
-- Name: codeowners_owners_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.codeowners_owners_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: codeowners_owners_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.codeowners_owners_id_seq OWNED BY public.codeowners_owners.id;


--
-- Name: commit_authors; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.commit_authors (
    id integer NOT NULL,
    email text NOT NULL,
    name text NOT NULL,
    tenant_id integer
);


--
-- Name: commit_authors_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.commit_authors_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: commit_authors_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.commit_authors_id_seq OWNED BY public.commit_authors.id;


--
-- Name: configuration_policies_audit_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.configuration_policies_audit_logs (
    log_timestamp timestamp with time zone DEFAULT clock_timestamp(),
    record_deleted_at timestamp with time zone,
    policy_id integer NOT NULL,
    transition_columns public.hstore[],
    sequence bigint NOT NULL,
    operation public.audit_log_operation NOT NULL,
    tenant_id integer
);


--
-- Name: COLUMN configuration_policies_audit_logs.log_timestamp; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.configuration_policies_audit_logs.log_timestamp IS 'Timestamp for this log entry.';


--
-- Name: COLUMN configuration_policies_audit_logs.record_deleted_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.configuration_policies_audit_logs.record_deleted_at IS 'Set once the upload this entry is associated with is deleted. Once NOW() - record_deleted_at is above a certain threshold, this log entry will be deleted.';


--
-- Name: COLUMN configuration_policies_audit_logs.transition_columns; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.configuration_policies_audit_logs.transition_columns IS 'Array of changes that occurred to the upload for this entry, in the form of {"column"=>"<column name>", "old"=>"<previous value>", "new"=>"<new value>"}.';


--
-- Name: configuration_policies_audit_logs_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.configuration_policies_audit_logs_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: configuration_policies_audit_logs_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.configuration_policies_audit_logs_seq OWNED BY public.configuration_policies_audit_logs.sequence;


--
-- Name: context_detection_embedding_jobs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.context_detection_embedding_jobs (
    id integer NOT NULL,
    state text DEFAULT 'queued'::text,
    failure_message text,
    queued_at timestamp with time zone DEFAULT now(),
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    process_after timestamp with time zone,
    num_resets integer DEFAULT 0 NOT NULL,
    num_failures integer DEFAULT 0 NOT NULL,
    last_heartbeat_at timestamp with time zone,
    execution_logs json[],
    worker_hostname text DEFAULT ''::text NOT NULL,
    cancel boolean DEFAULT false NOT NULL,
    tenant_id integer
);


--
-- Name: context_detection_embedding_jobs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.context_detection_embedding_jobs_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: context_detection_embedding_jobs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.context_detection_embedding_jobs_id_seq OWNED BY public.context_detection_embedding_jobs.id;


--
-- Name: critical_and_site_config; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.critical_and_site_config (
    id integer NOT NULL,
    type public.critical_or_site NOT NULL,
    contents text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    author_user_id integer,
    redacted_contents text
);


--
-- Name: COLUMN critical_and_site_config.author_user_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.critical_and_site_config.author_user_id IS 'A null value indicates that this config was most likely added by code on the start-up path, for example from the SITE_CONFIG_FILE unless the config itself was added before this column existed in which case it could also have been a user.';


--
-- Name: COLUMN critical_and_site_config.redacted_contents; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.critical_and_site_config.redacted_contents IS 'This column stores the contents but redacts all secrets. The redacted form is a sha256 hash of the secret appended to the REDACTED string. This is used to generate diffs between two subsequent changes in a way that allows us to detect changes to any secrets while also ensuring that we do not leak it in the diff. A null value indicates that this config was added before this column was added or redacting the secrets during write failed so we skipped writing to this column instead of a hard failure.';


--
-- Name: critical_and_site_config_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.critical_and_site_config_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: critical_and_site_config_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.critical_and_site_config_id_seq OWNED BY public.critical_and_site_config.id;


--
-- Name: discussion_comments; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.discussion_comments (
    id bigint NOT NULL,
    thread_id bigint NOT NULL,
    author_user_id integer NOT NULL,
    contents text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    reports text[] DEFAULT '{}'::text[] NOT NULL,
    tenant_id integer
);


--
-- Name: discussion_comments_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.discussion_comments_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: discussion_comments_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.discussion_comments_id_seq OWNED BY public.discussion_comments.id;


--
-- Name: discussion_mail_reply_tokens; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.discussion_mail_reply_tokens (
    token text NOT NULL,
    user_id integer NOT NULL,
    thread_id bigint NOT NULL,
    deleted_at timestamp with time zone,
    tenant_id integer
);


--
-- Name: discussion_threads; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.discussion_threads (
    id bigint NOT NULL,
    author_user_id integer NOT NULL,
    title text,
    target_repo_id bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    archived_at timestamp with time zone,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    tenant_id integer
);


--
-- Name: discussion_threads_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.discussion_threads_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: discussion_threads_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.discussion_threads_id_seq OWNED BY public.discussion_threads.id;


--
-- Name: discussion_threads_target_repo; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.discussion_threads_target_repo (
    id bigint NOT NULL,
    thread_id bigint NOT NULL,
    repo_id integer NOT NULL,
    path text,
    branch text,
    revision text,
    start_line integer,
    end_line integer,
    start_character integer,
    end_character integer,
    lines_before text,
    lines text,
    lines_after text,
    tenant_id integer
);


--
-- Name: discussion_threads_target_repo_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.discussion_threads_target_repo_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: discussion_threads_target_repo_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.discussion_threads_target_repo_id_seq OWNED BY public.discussion_threads_target_repo.id;


--
-- Name: event_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.event_logs (
    id bigint NOT NULL,
    name text NOT NULL,
    url text NOT NULL,
    user_id integer NOT NULL,
    anonymous_user_id text NOT NULL,
    source text NOT NULL,
    argument jsonb NOT NULL,
    version text NOT NULL,
    "timestamp" timestamp with time zone NOT NULL,
    feature_flags jsonb,
    cohort_id date,
    public_argument jsonb DEFAULT '{}'::jsonb NOT NULL,
    first_source_url text,
    last_source_url text,
    referrer text,
    device_id text,
    insert_id text,
    billing_product_category text,
    billing_event_id text,
    client text,
    tenant_id integer,
    CONSTRAINT event_logs_check_has_user CHECK ((((user_id = 0) AND (anonymous_user_id <> ''::text)) OR ((user_id <> 0) AND (anonymous_user_id = ''::text)) OR ((user_id <> 0) AND (anonymous_user_id <> ''::text)))),
    CONSTRAINT event_logs_check_name_not_empty CHECK ((name <> ''::text)),
    CONSTRAINT event_logs_check_source_not_empty CHECK ((source <> ''::text)),
    CONSTRAINT event_logs_check_version_not_empty CHECK ((version <> ''::text))
);


--
-- Name: event_logs_export_allowlist; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.event_logs_export_allowlist (
    id integer NOT NULL,
    event_name text NOT NULL,
    tenant_id integer
);


--
-- Name: TABLE event_logs_export_allowlist; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.event_logs_export_allowlist IS 'An allowlist of events that are approved for export if the scraping job is enabled';


--
-- Name: COLUMN event_logs_export_allowlist.event_name; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.event_logs_export_allowlist.event_name IS 'Name of the event that corresponds to event_logs.name';


--
-- Name: event_logs_export_allowlist_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.event_logs_export_allowlist_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: event_logs_export_allowlist_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.event_logs_export_allowlist_id_seq OWNED BY public.event_logs_export_allowlist.id;


--
-- Name: event_logs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.event_logs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: event_logs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.event_logs_id_seq OWNED BY public.event_logs.id;


--
-- Name: event_logs_scrape_state; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.event_logs_scrape_state (
    id integer NOT NULL,
    bookmark_id integer NOT NULL,
    tenant_id integer
);


--
-- Name: TABLE event_logs_scrape_state; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.event_logs_scrape_state IS 'Contains state for the periodic telemetry job that scrapes events if enabled.';


--
-- Name: COLUMN event_logs_scrape_state.bookmark_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.event_logs_scrape_state.bookmark_id IS 'Bookmarks the maximum most recent successful event_logs.id that was scraped';


--
-- Name: event_logs_scrape_state_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.event_logs_scrape_state_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: event_logs_scrape_state_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.event_logs_scrape_state_id_seq OWNED BY public.event_logs_scrape_state.id;


--
-- Name: event_logs_scrape_state_own; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.event_logs_scrape_state_own (
    id integer NOT NULL,
    bookmark_id integer NOT NULL,
    job_type integer NOT NULL,
    tenant_id integer
);


--
-- Name: TABLE event_logs_scrape_state_own; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.event_logs_scrape_state_own IS 'Contains state for own jobs that scrape events if enabled.';


--
-- Name: COLUMN event_logs_scrape_state_own.bookmark_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.event_logs_scrape_state_own.bookmark_id IS 'Bookmarks the maximum most recent successful event_logs.id that was scraped';


--
-- Name: event_logs_scrape_state_own_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.event_logs_scrape_state_own_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: event_logs_scrape_state_own_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.event_logs_scrape_state_own_id_seq OWNED BY public.event_logs_scrape_state_own.id;


--
-- Name: executor_heartbeats; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.executor_heartbeats (
    id integer NOT NULL,
    hostname text NOT NULL,
    queue_name text,
    os text NOT NULL,
    architecture text NOT NULL,
    docker_version text NOT NULL,
    executor_version text NOT NULL,
    git_version text NOT NULL,
    ignite_version text NOT NULL,
    src_cli_version text NOT NULL,
    first_seen_at timestamp with time zone DEFAULT now() NOT NULL,
    last_seen_at timestamp with time zone DEFAULT now() NOT NULL,
    queue_names text[],
    tenant_id integer,
    CONSTRAINT one_of_queue_name_queue_names CHECK ((((queue_name IS NOT NULL) AND (queue_names IS NULL)) OR ((queue_names IS NOT NULL) AND (queue_name IS NULL))))
);


--
-- Name: TABLE executor_heartbeats; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.executor_heartbeats IS 'Tracks the most recent activity of executors attached to this Sourcegraph instance.';


--
-- Name: COLUMN executor_heartbeats.hostname; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.executor_heartbeats.hostname IS 'The uniquely identifying name of the executor.';


--
-- Name: COLUMN executor_heartbeats.queue_name; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.executor_heartbeats.queue_name IS 'The queue name that the executor polls for work.';


--
-- Name: COLUMN executor_heartbeats.os; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.executor_heartbeats.os IS 'The operating system running the executor.';


--
-- Name: COLUMN executor_heartbeats.architecture; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.executor_heartbeats.architecture IS 'The machine architure running the executor.';


--
-- Name: COLUMN executor_heartbeats.docker_version; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.executor_heartbeats.docker_version IS 'The version of Docker used by the executor.';


--
-- Name: COLUMN executor_heartbeats.executor_version; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.executor_heartbeats.executor_version IS 'The version of the executor.';


--
-- Name: COLUMN executor_heartbeats.git_version; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.executor_heartbeats.git_version IS 'The version of Git used by the executor.';


--
-- Name: COLUMN executor_heartbeats.ignite_version; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.executor_heartbeats.ignite_version IS 'The version of Ignite used by the executor.';


--
-- Name: COLUMN executor_heartbeats.src_cli_version; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.executor_heartbeats.src_cli_version IS 'The version of src-cli used by the executor.';


--
-- Name: COLUMN executor_heartbeats.first_seen_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.executor_heartbeats.first_seen_at IS 'The first time a heartbeat from the executor was received.';


--
-- Name: COLUMN executor_heartbeats.last_seen_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.executor_heartbeats.last_seen_at IS 'The last time a heartbeat from the executor was received.';


--
-- Name: COLUMN executor_heartbeats.queue_names; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.executor_heartbeats.queue_names IS 'The list of queue names that the executor polls for work.';


--
-- Name: executor_heartbeats_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.executor_heartbeats_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: executor_heartbeats_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.executor_heartbeats_id_seq OWNED BY public.executor_heartbeats.id;


--
-- Name: executor_job_tokens; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.executor_job_tokens (
    id integer NOT NULL,
    value_sha256 bytea NOT NULL,
    job_id bigint NOT NULL,
    queue text NOT NULL,
    repo_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    tenant_id integer
);


--
-- Name: executor_job_tokens_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.executor_job_tokens_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: executor_job_tokens_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.executor_job_tokens_id_seq OWNED BY public.executor_job_tokens.id;


--
-- Name: executor_secret_access_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.executor_secret_access_logs (
    id integer NOT NULL,
    executor_secret_id integer NOT NULL,
    user_id integer,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    machine_user text DEFAULT ''::text NOT NULL,
    tenant_id integer,
    CONSTRAINT user_id_or_machine_user CHECK ((((user_id IS NULL) AND (machine_user <> ''::text)) OR ((user_id IS NOT NULL) AND (machine_user = ''::text))))
);


--
-- Name: executor_secret_access_logs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.executor_secret_access_logs_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: executor_secret_access_logs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.executor_secret_access_logs_id_seq OWNED BY public.executor_secret_access_logs.id;


--
-- Name: executor_secrets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.executor_secrets (
    id integer NOT NULL,
    key text NOT NULL,
    value bytea NOT NULL,
    scope text NOT NULL,
    encryption_key_id text,
    namespace_user_id integer,
    namespace_org_id integer,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    creator_id integer,
    tenant_id integer
);


--
-- Name: COLUMN executor_secrets.creator_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.executor_secrets.creator_id IS 'NULL, if the user has been deleted.';


--
-- Name: executor_secrets_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.executor_secrets_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: executor_secrets_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.executor_secrets_id_seq OWNED BY public.executor_secrets.id;


--
-- Name: exhaustive_search_jobs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.exhaustive_search_jobs (
    id integer NOT NULL,
    state text DEFAULT 'queued'::text,
    initiator_id integer NOT NULL,
    query text NOT NULL,
    failure_message text,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    process_after timestamp with time zone,
    num_resets integer DEFAULT 0 NOT NULL,
    num_failures integer DEFAULT 0 NOT NULL,
    last_heartbeat_at timestamp with time zone,
    execution_logs json[],
    worker_hostname text DEFAULT ''::text NOT NULL,
    cancel boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    queued_at timestamp with time zone DEFAULT now(),
    is_aggregated boolean DEFAULT false NOT NULL,
    tenant_id integer
);


--
-- Name: exhaustive_search_jobs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.exhaustive_search_jobs_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: exhaustive_search_jobs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.exhaustive_search_jobs_id_seq OWNED BY public.exhaustive_search_jobs.id;


--
-- Name: exhaustive_search_repo_jobs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.exhaustive_search_repo_jobs (
    id integer NOT NULL,
    state text DEFAULT 'queued'::text,
    repo_id integer NOT NULL,
    ref_spec text NOT NULL,
    search_job_id integer NOT NULL,
    failure_message text,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    process_after timestamp with time zone,
    num_resets integer DEFAULT 0 NOT NULL,
    num_failures integer DEFAULT 0 NOT NULL,
    last_heartbeat_at timestamp with time zone,
    execution_logs json[],
    worker_hostname text DEFAULT ''::text NOT NULL,
    cancel boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    queued_at timestamp with time zone DEFAULT now(),
    tenant_id integer
);


--
-- Name: exhaustive_search_repo_jobs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.exhaustive_search_repo_jobs_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: exhaustive_search_repo_jobs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.exhaustive_search_repo_jobs_id_seq OWNED BY public.exhaustive_search_repo_jobs.id;


--
-- Name: exhaustive_search_repo_revision_jobs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.exhaustive_search_repo_revision_jobs (
    id integer NOT NULL,
    state text DEFAULT 'queued'::text,
    search_repo_job_id integer NOT NULL,
    revision text NOT NULL,
    failure_message text,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    process_after timestamp with time zone,
    num_resets integer DEFAULT 0 NOT NULL,
    num_failures integer DEFAULT 0 NOT NULL,
    last_heartbeat_at timestamp with time zone,
    execution_logs json[],
    worker_hostname text DEFAULT ''::text NOT NULL,
    cancel boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    queued_at timestamp with time zone DEFAULT now(),
    tenant_id integer
);


--
-- Name: exhaustive_search_repo_revision_jobs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.exhaustive_search_repo_revision_jobs_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: exhaustive_search_repo_revision_jobs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.exhaustive_search_repo_revision_jobs_id_seq OWNED BY public.exhaustive_search_repo_revision_jobs.id;


--
-- Name: explicit_permissions_bitbucket_projects_jobs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.explicit_permissions_bitbucket_projects_jobs (
    id integer NOT NULL,
    state text DEFAULT 'queued'::text,
    failure_message text,
    queued_at timestamp with time zone DEFAULT now(),
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    process_after timestamp with time zone,
    num_resets integer DEFAULT 0 NOT NULL,
    num_failures integer DEFAULT 0 NOT NULL,
    last_heartbeat_at timestamp with time zone,
    execution_logs json[],
    worker_hostname text DEFAULT ''::text NOT NULL,
    project_key text NOT NULL,
    external_service_id integer NOT NULL,
    permissions json[],
    unrestricted boolean DEFAULT false NOT NULL,
    cancel boolean DEFAULT false NOT NULL,
    tenant_id integer,
    CONSTRAINT explicit_permissions_bitbucket_projects_jobs_check CHECK ((((permissions IS NOT NULL) AND (unrestricted IS FALSE)) OR ((permissions IS NULL) AND (unrestricted IS TRUE))))
);


--
-- Name: explicit_permissions_bitbucket_projects_jobs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.explicit_permissions_bitbucket_projects_jobs_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: explicit_permissions_bitbucket_projects_jobs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.explicit_permissions_bitbucket_projects_jobs_id_seq OWNED BY public.explicit_permissions_bitbucket_projects_jobs.id;


--
-- Name: external_service_repos; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.external_service_repos (
    external_service_id bigint NOT NULL,
    repo_id integer NOT NULL,
    clone_url text NOT NULL,
    created_at timestamp with time zone DEFAULT transaction_timestamp() NOT NULL,
    tenant_id integer
);


--
-- Name: external_service_sync_jobs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.external_service_sync_jobs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: external_service_sync_jobs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.external_service_sync_jobs (
    id integer DEFAULT nextval('public.external_service_sync_jobs_id_seq'::regclass) NOT NULL,
    state text DEFAULT 'queued'::text NOT NULL,
    failure_message text,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    process_after timestamp with time zone,
    num_resets integer DEFAULT 0 NOT NULL,
    external_service_id bigint NOT NULL,
    num_failures integer DEFAULT 0 NOT NULL,
    log_contents text,
    execution_logs json[],
    worker_hostname text DEFAULT ''::text NOT NULL,
    last_heartbeat_at timestamp with time zone,
    queued_at timestamp with time zone DEFAULT now(),
    cancel boolean DEFAULT false NOT NULL,
    repos_synced integer DEFAULT 0 NOT NULL,
    repo_sync_errors integer DEFAULT 0 NOT NULL,
    repos_added integer DEFAULT 0 NOT NULL,
    repos_deleted integer DEFAULT 0 NOT NULL,
    repos_modified integer DEFAULT 0 NOT NULL,
    repos_unmodified integer DEFAULT 0 NOT NULL,
    tenant_id integer
);


--
-- Name: COLUMN external_service_sync_jobs.repos_synced; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.external_service_sync_jobs.repos_synced IS 'The number of repos synced during this sync job.';


--
-- Name: COLUMN external_service_sync_jobs.repo_sync_errors; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.external_service_sync_jobs.repo_sync_errors IS 'The number of times an error occurred syncing a repo during this sync job.';


--
-- Name: COLUMN external_service_sync_jobs.repos_added; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.external_service_sync_jobs.repos_added IS 'The number of new repos discovered during this sync job.';


--
-- Name: COLUMN external_service_sync_jobs.repos_deleted; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.external_service_sync_jobs.repos_deleted IS 'The number of repos deleted as a result of this sync job.';


--
-- Name: COLUMN external_service_sync_jobs.repos_modified; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.external_service_sync_jobs.repos_modified IS 'The number of existing repos whose metadata has changed during this sync job.';


--
-- Name: COLUMN external_service_sync_jobs.repos_unmodified; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.external_service_sync_jobs.repos_unmodified IS 'The number of existing repos whose metadata did not change during this sync job.';


--
-- Name: external_services; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.external_services (
    id bigint NOT NULL,
    kind text NOT NULL,
    display_name text NOT NULL,
    config text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    last_sync_at timestamp with time zone,
    next_sync_at timestamp with time zone,
    unrestricted boolean DEFAULT false NOT NULL,
    cloud_default boolean DEFAULT false NOT NULL,
    encryption_key_id text DEFAULT ''::text NOT NULL,
    has_webhooks boolean,
    token_expires_at timestamp with time zone,
    code_host_id integer,
    creator_id integer,
    last_updater_id integer,
    tenant_id integer,
    CONSTRAINT check_non_empty_config CHECK ((btrim(config) <> ''::text))
);


--
-- Name: external_service_sync_jobs_with_next_sync_at; Type: VIEW; Schema: public; Owner: -
--

CREATE VIEW public.external_service_sync_jobs_with_next_sync_at AS
 SELECT j.id,
    j.state,
    j.failure_message,
    j.queued_at,
    j.started_at,
    j.finished_at,
    j.process_after,
    j.num_resets,
    j.num_failures,
    j.execution_logs,
    j.external_service_id,
    e.next_sync_at
   FROM (public.external_services e
     JOIN public.external_service_sync_jobs j ON ((e.id = j.external_service_id)));


--
-- Name: external_services_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.external_services_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: external_services_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.external_services_id_seq OWNED BY public.external_services.id;


--
-- Name: feature_flag_overrides; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.feature_flag_overrides (
    namespace_org_id integer,
    namespace_user_id integer,
    flag_name text NOT NULL,
    flag_value boolean NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    tenant_id integer,
    CONSTRAINT feature_flag_overrides_has_org_or_user_id CHECK (((namespace_org_id IS NOT NULL) OR (namespace_user_id IS NOT NULL)))
);


--
-- Name: feature_flags; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.feature_flags (
    flag_name text NOT NULL,
    flag_type public.feature_flag_type NOT NULL,
    bool_value boolean,
    rollout integer,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    tenant_id integer,
    CONSTRAINT feature_flags_rollout_check CHECK (((rollout >= 0) AND (rollout <= 10000))),
    CONSTRAINT required_bool_fields CHECK ((1 =
CASE
    WHEN ((flag_type = 'bool'::public.feature_flag_type) AND (bool_value IS NULL)) THEN 0
    WHEN ((flag_type <> 'bool'::public.feature_flag_type) AND (bool_value IS NOT NULL)) THEN 0
    ELSE 1
END)),
    CONSTRAINT required_rollout_fields CHECK ((1 =
CASE
    WHEN ((flag_type = 'rollout'::public.feature_flag_type) AND (rollout IS NULL)) THEN 0
    WHEN ((flag_type <> 'rollout'::public.feature_flag_type) AND (rollout IS NOT NULL)) THEN 0
    ELSE 1
END))
);


--
-- Name: COLUMN feature_flags.bool_value; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.feature_flags.bool_value IS 'Bool value only defined when flag_type is bool';


--
-- Name: COLUMN feature_flags.rollout; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.feature_flags.rollout IS 'Rollout only defined when flag_type is rollout. Increments of 0.01%';


--
-- Name: CONSTRAINT required_bool_fields ON feature_flags; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON CONSTRAINT required_bool_fields ON public.feature_flags IS 'Checks that bool_value is set IFF flag_type = bool';


--
-- Name: CONSTRAINT required_rollout_fields ON feature_flags; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON CONSTRAINT required_rollout_fields ON public.feature_flags IS 'Checks that rollout is set IFF flag_type = rollout';


--
-- Name: github_app_installs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.github_app_installs (
    id integer NOT NULL,
    app_id integer NOT NULL,
    installation_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    url text,
    account_login text,
    account_avatar_url text,
    account_url text,
    account_type text,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    tenant_id integer
);


--
-- Name: github_app_installs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.github_app_installs_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: github_app_installs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.github_app_installs_id_seq OWNED BY public.github_app_installs.id;


--
-- Name: github_apps; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.github_apps (
    id integer NOT NULL,
    app_id integer NOT NULL,
    name text NOT NULL,
    slug text NOT NULL,
    base_url text NOT NULL,
    client_id text NOT NULL,
    client_secret text NOT NULL,
    private_key text NOT NULL,
    encryption_key_id text NOT NULL,
    logo text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    app_url text DEFAULT ''::text NOT NULL,
    webhook_id integer,
    domain text DEFAULT 'repos'::text NOT NULL,
    kind public.github_app_kind DEFAULT 'REPO_SYNC'::public.github_app_kind NOT NULL,
    creator_id bigint DEFAULT 0 NOT NULL,
    tenant_id integer
);


--
-- Name: github_apps_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.github_apps_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: github_apps_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.github_apps_id_seq OWNED BY public.github_apps.id;


--
-- Name: gitserver_relocator_jobs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.gitserver_relocator_jobs (
    id integer NOT NULL,
    state text DEFAULT 'queued'::text,
    queued_at timestamp with time zone DEFAULT now(),
    failure_message text,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    process_after timestamp with time zone,
    num_resets integer DEFAULT 0 NOT NULL,
    num_failures integer DEFAULT 0 NOT NULL,
    last_heartbeat_at timestamp with time zone,
    execution_logs json[],
    worker_hostname text DEFAULT ''::text NOT NULL,
    repo_id integer NOT NULL,
    source_hostname text NOT NULL,
    dest_hostname text NOT NULL,
    delete_source boolean DEFAULT false NOT NULL,
    cancel boolean DEFAULT false NOT NULL,
    tenant_id integer
);


--
-- Name: gitserver_relocator_jobs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.gitserver_relocator_jobs_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: gitserver_relocator_jobs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.gitserver_relocator_jobs_id_seq OWNED BY public.gitserver_relocator_jobs.id;


--
-- Name: gitserver_relocator_jobs_with_repo_name; Type: VIEW; Schema: public; Owner: -
--

CREATE VIEW public.gitserver_relocator_jobs_with_repo_name AS
 SELECT glj.id,
    glj.state,
    glj.queued_at,
    glj.failure_message,
    glj.started_at,
    glj.finished_at,
    glj.process_after,
    glj.num_resets,
    glj.num_failures,
    glj.last_heartbeat_at,
    glj.execution_logs,
    glj.worker_hostname,
    glj.repo_id,
    glj.source_hostname,
    glj.dest_hostname,
    glj.delete_source,
    r.name AS repo_name
   FROM (public.gitserver_relocator_jobs glj
     JOIN public.repo r ON ((r.id = glj.repo_id)));


--
-- Name: gitserver_repos; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.gitserver_repos (
    repo_id integer NOT NULL,
    clone_status text DEFAULT 'not_cloned'::text NOT NULL,
    shard_id text NOT NULL,
    last_error text,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    last_fetched timestamp with time zone DEFAULT now() NOT NULL,
    last_changed timestamp with time zone DEFAULT now() NOT NULL,
    repo_size_bytes bigint,
    corrupted_at timestamp with time zone,
    corruption_logs jsonb DEFAULT '[]'::jsonb NOT NULL,
    cloning_progress text DEFAULT ''::text,
    tenant_id integer
);


--
-- Name: COLUMN gitserver_repos.corrupted_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.gitserver_repos.corrupted_at IS 'Timestamp of when repo corruption was detected';


--
-- Name: COLUMN gitserver_repos.corruption_logs; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.gitserver_repos.corruption_logs IS 'Log output of repo corruptions that have been detected - encoded as json';


--
-- Name: gitserver_repos_statistics; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.gitserver_repos_statistics (
    shard_id text,
    total bigint DEFAULT 0 NOT NULL,
    not_cloned bigint DEFAULT 0 NOT NULL,
    cloning bigint DEFAULT 0 NOT NULL,
    cloned bigint DEFAULT 0 NOT NULL,
    failed_fetch bigint DEFAULT 0 NOT NULL,
    corrupted bigint DEFAULT 0 NOT NULL,
    tenant_id integer
);


--
-- Name: COLUMN gitserver_repos_statistics.shard_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.gitserver_repos_statistics.shard_id IS 'ID of this gitserver shard. If an empty string then the repositories havent been assigned a shard.';


--
-- Name: COLUMN gitserver_repos_statistics.total; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.gitserver_repos_statistics.total IS 'Number of repositories in gitserver_repos table on this shard';


--
-- Name: COLUMN gitserver_repos_statistics.not_cloned; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.gitserver_repos_statistics.not_cloned IS 'Number of repositories in gitserver_repos table on this shard that are not cloned yet';


--
-- Name: COLUMN gitserver_repos_statistics.cloning; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.gitserver_repos_statistics.cloning IS 'Number of repositories in gitserver_repos table on this shard that cloning';


--
-- Name: COLUMN gitserver_repos_statistics.cloned; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.gitserver_repos_statistics.cloned IS 'Number of repositories in gitserver_repos table on this shard that are cloned';


--
-- Name: COLUMN gitserver_repos_statistics.failed_fetch; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.gitserver_repos_statistics.failed_fetch IS 'Number of repositories in gitserver_repos table on this shard where last_error is set';


--
-- Name: COLUMN gitserver_repos_statistics.corrupted; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.gitserver_repos_statistics.corrupted IS 'Number of repositories that are NOT soft-deleted and not blocked and have corrupted_at set in gitserver_repos table';


--
-- Name: gitserver_repos_sync_output; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.gitserver_repos_sync_output (
    repo_id integer NOT NULL,
    last_output text DEFAULT ''::text NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    tenant_id integer
);


--
-- Name: TABLE gitserver_repos_sync_output; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.gitserver_repos_sync_output IS 'Contains the most recent output from gitserver repository sync jobs.';


--
-- Name: global_state; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.global_state (
    site_id uuid NOT NULL,
    initialized boolean DEFAULT false NOT NULL,
    tenant_id integer
);


--
-- Name: insights_query_runner_jobs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.insights_query_runner_jobs (
    id integer NOT NULL,
    series_id text NOT NULL,
    search_query text NOT NULL,
    state text DEFAULT 'queued'::text,
    failure_message text,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    process_after timestamp with time zone,
    num_resets integer DEFAULT 0 NOT NULL,
    num_failures integer DEFAULT 0 NOT NULL,
    execution_logs json[],
    record_time timestamp with time zone,
    worker_hostname text DEFAULT ''::text NOT NULL,
    last_heartbeat_at timestamp with time zone,
    priority integer DEFAULT 1 NOT NULL,
    cost integer DEFAULT 500 NOT NULL,
    persist_mode public.persistmode DEFAULT 'record'::public.persistmode NOT NULL,
    queued_at timestamp with time zone DEFAULT now(),
    cancel boolean DEFAULT false NOT NULL,
    trace_id text,
    tenant_id integer
);


--
-- Name: TABLE insights_query_runner_jobs; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.insights_query_runner_jobs IS 'See [internal/insights/background/queryrunner/worker.go:Job](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/sourcegraph/sourcegraph%24+file:internal/insights/background/queryrunner/worker.go+type+Job&patternType=literal)';


--
-- Name: COLUMN insights_query_runner_jobs.priority; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.insights_query_runner_jobs.priority IS 'Integer representing a category of priority for this query. Priority in this context is ambiguously defined for consumers to decide an interpretation.';


--
-- Name: COLUMN insights_query_runner_jobs.cost; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.insights_query_runner_jobs.cost IS 'Integer representing a cost approximation of executing this search query.';


--
-- Name: COLUMN insights_query_runner_jobs.persist_mode; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.insights_query_runner_jobs.persist_mode IS 'The persistence level for this query. This value will determine the lifecycle of the resulting value.';


--
-- Name: insights_query_runner_jobs_dependencies; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.insights_query_runner_jobs_dependencies (
    id integer NOT NULL,
    job_id integer NOT NULL,
    recording_time timestamp without time zone NOT NULL,
    tenant_id integer
);


--
-- Name: TABLE insights_query_runner_jobs_dependencies; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.insights_query_runner_jobs_dependencies IS 'Stores data points for a code insight that do not need to be queried directly, but depend on the result of a query at a different point';


--
-- Name: COLUMN insights_query_runner_jobs_dependencies.job_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.insights_query_runner_jobs_dependencies.job_id IS 'Foreign key to the job that owns this record.';


--
-- Name: COLUMN insights_query_runner_jobs_dependencies.recording_time; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.insights_query_runner_jobs_dependencies.recording_time IS 'The time for which this dependency should be recorded at using the parents value.';


--
-- Name: insights_query_runner_jobs_dependencies_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.insights_query_runner_jobs_dependencies_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: insights_query_runner_jobs_dependencies_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.insights_query_runner_jobs_dependencies_id_seq OWNED BY public.insights_query_runner_jobs_dependencies.id;


--
-- Name: insights_query_runner_jobs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.insights_query_runner_jobs_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: insights_query_runner_jobs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.insights_query_runner_jobs_id_seq OWNED BY public.insights_query_runner_jobs.id;


--
-- Name: insights_settings_migration_jobs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.insights_settings_migration_jobs (
    id integer NOT NULL,
    user_id integer,
    org_id integer,
    global boolean,
    settings_id integer NOT NULL,
    total_insights integer DEFAULT 0 NOT NULL,
    migrated_insights integer DEFAULT 0 NOT NULL,
    total_dashboards integer DEFAULT 0 NOT NULL,
    migrated_dashboards integer DEFAULT 0 NOT NULL,
    runs integer DEFAULT 0 NOT NULL,
    completed_at timestamp without time zone,
    tenant_id integer
);


--
-- Name: insights_settings_migration_jobs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.insights_settings_migration_jobs_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: insights_settings_migration_jobs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.insights_settings_migration_jobs_id_seq OWNED BY public.insights_settings_migration_jobs.id;


--
-- Name: lsif_configuration_policies_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.lsif_configuration_policies_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: lsif_configuration_policies_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.lsif_configuration_policies_id_seq OWNED BY public.lsif_configuration_policies.id;


--
-- Name: lsif_dependency_indexing_jobs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.lsif_dependency_indexing_jobs (
    id integer NOT NULL,
    state text DEFAULT 'queued'::text NOT NULL,
    failure_message text,
    queued_at timestamp with time zone DEFAULT now() NOT NULL,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    process_after timestamp with time zone,
    num_resets integer DEFAULT 0 NOT NULL,
    num_failures integer DEFAULT 0 NOT NULL,
    execution_logs json[],
    last_heartbeat_at timestamp with time zone,
    worker_hostname text DEFAULT ''::text NOT NULL,
    upload_id integer,
    external_service_kind text DEFAULT ''::text NOT NULL,
    external_service_sync timestamp with time zone,
    cancel boolean DEFAULT false NOT NULL
);


--
-- Name: COLUMN lsif_dependency_indexing_jobs.external_service_kind; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_dependency_indexing_jobs.external_service_kind IS 'Filter the external services for this kind to wait to have synced. If empty, external_service_sync is ignored and no external services are polled for their last sync time.';


--
-- Name: COLUMN lsif_dependency_indexing_jobs.external_service_sync; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_dependency_indexing_jobs.external_service_sync IS 'The sync time after which external services of the given kind will have synced/created any repositories referenced by the LSIF upload that are resolvable.';


--
-- Name: lsif_dependency_syncing_jobs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.lsif_dependency_syncing_jobs (
    id integer NOT NULL,
    state text DEFAULT 'queued'::text NOT NULL,
    failure_message text,
    queued_at timestamp with time zone DEFAULT now() NOT NULL,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    process_after timestamp with time zone,
    num_resets integer DEFAULT 0 NOT NULL,
    num_failures integer DEFAULT 0 NOT NULL,
    execution_logs json[],
    upload_id integer,
    worker_hostname text DEFAULT ''::text NOT NULL,
    last_heartbeat_at timestamp with time zone,
    cancel boolean DEFAULT false NOT NULL
);


--
-- Name: TABLE lsif_dependency_syncing_jobs; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.lsif_dependency_syncing_jobs IS 'Tracks jobs that scan imports of indexes to schedule auto-index jobs.';


--
-- Name: COLUMN lsif_dependency_syncing_jobs.upload_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_dependency_syncing_jobs.upload_id IS 'The identifier of the triggering upload record.';


--
-- Name: lsif_dependency_indexing_jobs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.lsif_dependency_indexing_jobs_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: lsif_dependency_indexing_jobs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.lsif_dependency_indexing_jobs_id_seq OWNED BY public.lsif_dependency_syncing_jobs.id;


--
-- Name: lsif_dependency_indexing_jobs_id_seq1; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.lsif_dependency_indexing_jobs_id_seq1
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: lsif_dependency_indexing_jobs_id_seq1; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.lsif_dependency_indexing_jobs_id_seq1 OWNED BY public.lsif_dependency_indexing_jobs.id;


--
-- Name: lsif_dependency_repos; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.lsif_dependency_repos (
    id bigint NOT NULL,
    name text NOT NULL,
    scheme text NOT NULL,
    blocked boolean DEFAULT false NOT NULL,
    last_checked_at timestamp with time zone
);


--
-- Name: lsif_dependency_repos_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.lsif_dependency_repos_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: lsif_dependency_repos_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.lsif_dependency_repos_id_seq OWNED BY public.lsif_dependency_repos.id;


--
-- Name: lsif_dirty_repositories; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.lsif_dirty_repositories (
    repository_id integer NOT NULL,
    dirty_token integer NOT NULL,
    update_token integer NOT NULL,
    updated_at timestamp with time zone,
    set_dirty_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE lsif_dirty_repositories; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.lsif_dirty_repositories IS 'Stores whether or not the nearest upload data for a repository is out of date (when update_token > dirty_token).';


--
-- Name: COLUMN lsif_dirty_repositories.dirty_token; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_dirty_repositories.dirty_token IS 'Set to the value of update_token visible to the transaction that updates the commit graph. Updates of dirty_token during this time will cause a second update.';


--
-- Name: COLUMN lsif_dirty_repositories.update_token; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_dirty_repositories.update_token IS 'This value is incremented on each request to update the commit graph for the repository.';


--
-- Name: COLUMN lsif_dirty_repositories.updated_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_dirty_repositories.updated_at IS 'The time the update_token value was last updated.';


--
-- Name: lsif_uploads; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.lsif_uploads (
    id integer NOT NULL,
    commit text NOT NULL,
    root text DEFAULT ''::text NOT NULL,
    uploaded_at timestamp with time zone DEFAULT now() NOT NULL,
    state text DEFAULT 'queued'::text NOT NULL,
    failure_message text,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    repository_id integer NOT NULL,
    indexer text NOT NULL,
    num_parts integer NOT NULL,
    uploaded_parts integer[] NOT NULL,
    process_after timestamp with time zone,
    num_resets integer DEFAULT 0 NOT NULL,
    upload_size bigint,
    num_failures integer DEFAULT 0 NOT NULL,
    associated_index_id bigint,
    committed_at timestamp with time zone,
    commit_last_checked_at timestamp with time zone,
    worker_hostname text DEFAULT ''::text NOT NULL,
    last_heartbeat_at timestamp with time zone,
    execution_logs json[],
    num_references integer,
    expired boolean DEFAULT false NOT NULL,
    last_retention_scan_at timestamp with time zone,
    reference_count integer,
    indexer_version text,
    queued_at timestamp with time zone,
    cancel boolean DEFAULT false NOT NULL,
    uncompressed_size bigint,
    last_referenced_scan_at timestamp with time zone,
    last_traversal_scan_at timestamp with time zone,
    last_reconcile_at timestamp with time zone,
    content_type text DEFAULT 'application/x-ndjson+lsif'::text NOT NULL,
    should_reindex boolean DEFAULT false NOT NULL,
    CONSTRAINT lsif_uploads_commit_valid_chars CHECK ((commit ~ '^[a-z0-9]{40}$'::text))
);


--
-- Name: TABLE lsif_uploads; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.lsif_uploads IS 'Stores metadata about an LSIF index uploaded by a user.';


--
-- Name: COLUMN lsif_uploads.id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_uploads.id IS 'Used as a logical foreign key with the (disjoint) codeintel database.';


--
-- Name: COLUMN lsif_uploads.commit; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_uploads.commit IS 'A 40-char revhash. Note that this commit may not be resolvable in the future.';


--
-- Name: COLUMN lsif_uploads.root; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_uploads.root IS 'The path for which the index can resolve code intelligence relative to the repository root.';


--
-- Name: COLUMN lsif_uploads.indexer; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_uploads.indexer IS 'The name of the indexer that produced the index file. If not supplied by the user it will be pulled from the index metadata.';


--
-- Name: COLUMN lsif_uploads.num_parts; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_uploads.num_parts IS 'The number of parts src-cli split the upload file into.';


--
-- Name: COLUMN lsif_uploads.uploaded_parts; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_uploads.uploaded_parts IS 'The index of parts that have been successfully uploaded.';


--
-- Name: COLUMN lsif_uploads.upload_size; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_uploads.upload_size IS 'The size of the index file (in bytes).';


--
-- Name: COLUMN lsif_uploads.num_references; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_uploads.num_references IS 'Deprecated in favor of reference_count.';


--
-- Name: COLUMN lsif_uploads.expired; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_uploads.expired IS 'Whether or not this upload data is no longer protected by any data retention policy.';


--
-- Name: COLUMN lsif_uploads.last_retention_scan_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_uploads.last_retention_scan_at IS 'The last time this upload was checked against data retention policies.';


--
-- Name: COLUMN lsif_uploads.reference_count; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_uploads.reference_count IS 'The number of references to this upload data from other upload records (via lsif_references).';


--
-- Name: COLUMN lsif_uploads.indexer_version; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_uploads.indexer_version IS 'The version of the indexer that produced the index file. If not supplied by the user it will be pulled from the index metadata.';


--
-- Name: COLUMN lsif_uploads.last_referenced_scan_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_uploads.last_referenced_scan_at IS 'The last time this upload was known to be referenced by another (possibly expired) index.';


--
-- Name: COLUMN lsif_uploads.last_traversal_scan_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_uploads.last_traversal_scan_at IS 'The last time this upload was known to be reachable by a non-expired index.';


--
-- Name: COLUMN lsif_uploads.content_type; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_uploads.content_type IS 'The content type of the upload record. For now, the default value is `application/x-ndjson+lsif` to backfill existing records. This will change as we remove LSIF support.';


--
-- Name: lsif_dumps; Type: VIEW; Schema: public; Owner: -
--

CREATE VIEW public.lsif_dumps AS
 SELECT id,
    commit,
    root,
    queued_at,
    uploaded_at,
    state,
    failure_message,
    started_at,
    finished_at,
    repository_id,
    indexer,
    indexer_version,
    num_parts,
    uploaded_parts,
    process_after,
    num_resets,
    upload_size,
    num_failures,
    associated_index_id,
    expired,
    last_retention_scan_at,
    finished_at AS processed_at
   FROM public.lsif_uploads u
  WHERE ((state = 'completed'::text) OR (state = 'deleting'::text));


--
-- Name: lsif_dumps_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.lsif_dumps_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: lsif_dumps_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.lsif_dumps_id_seq OWNED BY public.lsif_uploads.id;


--
-- Name: lsif_dumps_with_repository_name; Type: VIEW; Schema: public; Owner: -
--

CREATE VIEW public.lsif_dumps_with_repository_name AS
 SELECT u.id,
    u.commit,
    u.root,
    u.queued_at,
    u.uploaded_at,
    u.state,
    u.failure_message,
    u.started_at,
    u.finished_at,
    u.repository_id,
    u.indexer,
    u.indexer_version,
    u.num_parts,
    u.uploaded_parts,
    u.process_after,
    u.num_resets,
    u.upload_size,
    u.num_failures,
    u.associated_index_id,
    u.expired,
    u.last_retention_scan_at,
    u.processed_at,
    r.name AS repository_name
   FROM (public.lsif_dumps u
     JOIN public.repo r ON ((r.id = u.repository_id)))
  WHERE (r.deleted_at IS NULL);


--
-- Name: lsif_index_configuration; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.lsif_index_configuration (
    id bigint NOT NULL,
    repository_id integer NOT NULL,
    data bytea NOT NULL,
    autoindex_enabled boolean DEFAULT true NOT NULL
);


--
-- Name: TABLE lsif_index_configuration; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.lsif_index_configuration IS 'Stores the configuration used for code intel index jobs for a repository.';


--
-- Name: COLUMN lsif_index_configuration.data; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_index_configuration.data IS 'The raw user-supplied [configuration](https://sourcegraph.com/github.com/sourcegraph/sourcegraph@3.23/-/blob/enterprise/internal/codeintel/autoindex/config/types.go#L3:6) (encoded in JSONC).';


--
-- Name: COLUMN lsif_index_configuration.autoindex_enabled; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_index_configuration.autoindex_enabled IS 'Whether or not auto-indexing should be attempted on this repo. Index jobs may be inferred from the repository contents if data is empty.';


--
-- Name: lsif_index_configuration_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.lsif_index_configuration_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: lsif_index_configuration_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.lsif_index_configuration_id_seq OWNED BY public.lsif_index_configuration.id;


--
-- Name: lsif_indexes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.lsif_indexes (
    id bigint NOT NULL,
    commit text NOT NULL,
    queued_at timestamp with time zone DEFAULT now() NOT NULL,
    state text DEFAULT 'queued'::text NOT NULL,
    failure_message text,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    repository_id integer NOT NULL,
    process_after timestamp with time zone,
    num_resets integer DEFAULT 0 NOT NULL,
    num_failures integer DEFAULT 0 NOT NULL,
    docker_steps jsonb[] NOT NULL,
    root text NOT NULL,
    indexer text NOT NULL,
    indexer_args text[] NOT NULL,
    outfile text NOT NULL,
    log_contents text,
    execution_logs json[],
    local_steps text[] NOT NULL,
    commit_last_checked_at timestamp with time zone,
    worker_hostname text DEFAULT ''::text NOT NULL,
    last_heartbeat_at timestamp with time zone,
    cancel boolean DEFAULT false NOT NULL,
    should_reindex boolean DEFAULT false NOT NULL,
    requested_envvars text[],
    enqueuer_user_id integer DEFAULT 0 NOT NULL,
    CONSTRAINT lsif_uploads_commit_valid_chars CHECK ((commit ~ '^[a-z0-9]{40}$'::text))
);


--
-- Name: TABLE lsif_indexes; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.lsif_indexes IS 'Stores metadata about a code intel index job.';


--
-- Name: COLUMN lsif_indexes.commit; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_indexes.commit IS 'A 40-char revhash. Note that this commit may not be resolvable in the future.';


--
-- Name: COLUMN lsif_indexes.docker_steps; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_indexes.docker_steps IS 'An array of pre-index [steps](https://sourcegraph.com/github.com/sourcegraph/sourcegraph@3.23/-/blob/enterprise/internal/codeintel/stores/dbstore/docker_step.go#L9:6) to run.';


--
-- Name: COLUMN lsif_indexes.root; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_indexes.root IS 'The working directory of the indexer image relative to the repository root.';


--
-- Name: COLUMN lsif_indexes.indexer; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_indexes.indexer IS 'The docker image used to run the index command (e.g. sourcegraph/lsif-go).';


--
-- Name: COLUMN lsif_indexes.indexer_args; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_indexes.indexer_args IS 'The command run inside the indexer image to produce the index file (e.g. [''lsif-node'', ''-p'', ''.''])';


--
-- Name: COLUMN lsif_indexes.outfile; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_indexes.outfile IS 'The path to the index file produced by the index command relative to the working directory.';


--
-- Name: COLUMN lsif_indexes.log_contents; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_indexes.log_contents IS '**Column deprecated in favor of execution_logs.**';


--
-- Name: COLUMN lsif_indexes.execution_logs; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_indexes.execution_logs IS 'An array of [log entries](https://sourcegraph.com/github.com/sourcegraph/sourcegraph@3.23/-/blob/internal/workerutil/store.go#L48:6) (encoded as JSON) from the most recent execution.';


--
-- Name: COLUMN lsif_indexes.local_steps; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_indexes.local_steps IS 'A list of commands to run inside the indexer image prior to running the indexer command.';


--
-- Name: lsif_indexes_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.lsif_indexes_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: lsif_indexes_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.lsif_indexes_id_seq OWNED BY public.lsif_indexes.id;


--
-- Name: lsif_indexes_with_repository_name; Type: VIEW; Schema: public; Owner: -
--

CREATE VIEW public.lsif_indexes_with_repository_name AS
 SELECT u.id,
    u.commit,
    u.queued_at,
    u.state,
    u.failure_message,
    u.started_at,
    u.finished_at,
    u.repository_id,
    u.process_after,
    u.num_resets,
    u.num_failures,
    u.docker_steps,
    u.root,
    u.indexer,
    u.indexer_args,
    u.outfile,
    u.log_contents,
    u.execution_logs,
    u.local_steps,
    u.should_reindex,
    u.requested_envvars,
    r.name AS repository_name,
    u.enqueuer_user_id
   FROM (public.lsif_indexes u
     JOIN public.repo r ON ((r.id = u.repository_id)))
  WHERE (r.deleted_at IS NULL);


--
-- Name: lsif_last_index_scan; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.lsif_last_index_scan (
    repository_id integer NOT NULL,
    last_index_scan_at timestamp with time zone NOT NULL
);


--
-- Name: TABLE lsif_last_index_scan; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.lsif_last_index_scan IS 'Tracks the last time repository was checked for auto-indexing job scheduling.';


--
-- Name: COLUMN lsif_last_index_scan.last_index_scan_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_last_index_scan.last_index_scan_at IS 'The last time uploads of this repository were considered for auto-indexing job scheduling.';


--
-- Name: lsif_last_retention_scan; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.lsif_last_retention_scan (
    repository_id integer NOT NULL,
    last_retention_scan_at timestamp with time zone NOT NULL
);


--
-- Name: TABLE lsif_last_retention_scan; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.lsif_last_retention_scan IS 'Tracks the last time uploads a repository were checked against data retention policies.';


--
-- Name: COLUMN lsif_last_retention_scan.last_retention_scan_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_last_retention_scan.last_retention_scan_at IS 'The last time uploads of this repository were checked against data retention policies.';


--
-- Name: lsif_nearest_uploads; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.lsif_nearest_uploads (
    repository_id integer NOT NULL,
    commit_bytea bytea NOT NULL,
    uploads jsonb NOT NULL
);


--
-- Name: TABLE lsif_nearest_uploads; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.lsif_nearest_uploads IS 'Associates commits with the complete set of uploads visible from that commit. Every commit with upload data is present in this table.';


--
-- Name: COLUMN lsif_nearest_uploads.commit_bytea; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_nearest_uploads.commit_bytea IS 'A 40-char revhash. Note that this commit may not be resolvable in the future.';


--
-- Name: COLUMN lsif_nearest_uploads.uploads; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_nearest_uploads.uploads IS 'Encodes an {upload_id => distance} map that includes an entry for every upload visible from the commit. There is always at least one entry with a distance of zero.';


--
-- Name: lsif_nearest_uploads_links; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.lsif_nearest_uploads_links (
    repository_id integer NOT NULL,
    commit_bytea bytea NOT NULL,
    ancestor_commit_bytea bytea NOT NULL,
    distance integer NOT NULL
);


--
-- Name: TABLE lsif_nearest_uploads_links; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.lsif_nearest_uploads_links IS 'Associates commits with the closest ancestor commit with usable upload data. Together, this table and lsif_nearest_uploads cover all commits with resolvable code intelligence.';


--
-- Name: COLUMN lsif_nearest_uploads_links.commit_bytea; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_nearest_uploads_links.commit_bytea IS 'A 40-char revhash. Note that this commit may not be resolvable in the future.';


--
-- Name: COLUMN lsif_nearest_uploads_links.ancestor_commit_bytea; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_nearest_uploads_links.ancestor_commit_bytea IS 'The 40-char revhash of the ancestor. Note that this commit may not be resolvable in the future.';


--
-- Name: COLUMN lsif_nearest_uploads_links.distance; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_nearest_uploads_links.distance IS 'The distance bewteen the commits. Parent = 1, Grandparent = 2, etc.';


--
-- Name: lsif_packages; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.lsif_packages (
    id integer NOT NULL,
    scheme text NOT NULL,
    name text NOT NULL,
    version text,
    dump_id integer NOT NULL,
    manager text DEFAULT ''::text NOT NULL
);


--
-- Name: TABLE lsif_packages; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.lsif_packages IS 'Associates an upload with the set of packages they provide within a given packages management scheme.';


--
-- Name: COLUMN lsif_packages.scheme; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_packages.scheme IS 'The (export) moniker scheme.';


--
-- Name: COLUMN lsif_packages.name; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_packages.name IS 'The package name.';


--
-- Name: COLUMN lsif_packages.version; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_packages.version IS 'The package version.';


--
-- Name: COLUMN lsif_packages.dump_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_packages.dump_id IS 'The identifier of the upload that provides the package.';


--
-- Name: COLUMN lsif_packages.manager; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_packages.manager IS 'The package manager name.';


--
-- Name: lsif_packages_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.lsif_packages_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: lsif_packages_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.lsif_packages_id_seq OWNED BY public.lsif_packages.id;


--
-- Name: lsif_references; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.lsif_references (
    id integer NOT NULL,
    scheme text NOT NULL,
    name text NOT NULL,
    version text,
    dump_id integer NOT NULL,
    manager text DEFAULT ''::text NOT NULL
);


--
-- Name: TABLE lsif_references; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.lsif_references IS 'Associates an upload with the set of packages they require within a given packages management scheme.';


--
-- Name: COLUMN lsif_references.scheme; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_references.scheme IS 'The (import) moniker scheme.';


--
-- Name: COLUMN lsif_references.name; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_references.name IS 'The package name.';


--
-- Name: COLUMN lsif_references.version; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_references.version IS 'The package version.';


--
-- Name: COLUMN lsif_references.dump_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_references.dump_id IS 'The identifier of the upload that references the package.';


--
-- Name: COLUMN lsif_references.manager; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_references.manager IS 'The package manager name.';


--
-- Name: lsif_references_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.lsif_references_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: lsif_references_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.lsif_references_id_seq OWNED BY public.lsif_references.id;


--
-- Name: lsif_retention_configuration; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.lsif_retention_configuration (
    id integer NOT NULL,
    repository_id integer NOT NULL,
    max_age_for_non_stale_branches_seconds integer NOT NULL,
    max_age_for_non_stale_tags_seconds integer NOT NULL
);


--
-- Name: TABLE lsif_retention_configuration; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.lsif_retention_configuration IS 'Stores the retention policy of code intellience data for a repository.';


--
-- Name: COLUMN lsif_retention_configuration.max_age_for_non_stale_branches_seconds; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_retention_configuration.max_age_for_non_stale_branches_seconds IS 'The number of seconds since the last modification of a branch until it is considered stale.';


--
-- Name: COLUMN lsif_retention_configuration.max_age_for_non_stale_tags_seconds; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_retention_configuration.max_age_for_non_stale_tags_seconds IS 'The nujmber of seconds since the commit date of a tagged commit until it is considered stale.';


--
-- Name: lsif_retention_configuration_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.lsif_retention_configuration_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: lsif_retention_configuration_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.lsif_retention_configuration_id_seq OWNED BY public.lsif_retention_configuration.id;


--
-- Name: lsif_uploads_audit_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.lsif_uploads_audit_logs (
    log_timestamp timestamp with time zone DEFAULT now(),
    record_deleted_at timestamp with time zone,
    upload_id integer NOT NULL,
    commit text NOT NULL,
    root text NOT NULL,
    repository_id integer NOT NULL,
    uploaded_at timestamp with time zone NOT NULL,
    indexer text NOT NULL,
    indexer_version text,
    upload_size bigint,
    associated_index_id integer,
    transition_columns public.hstore[],
    reason text DEFAULT ''::text,
    sequence bigint NOT NULL,
    operation public.audit_log_operation NOT NULL,
    content_type text DEFAULT 'application/x-ndjson+lsif'::text NOT NULL
);


--
-- Name: COLUMN lsif_uploads_audit_logs.log_timestamp; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_uploads_audit_logs.log_timestamp IS 'Timestamp for this log entry.';


--
-- Name: COLUMN lsif_uploads_audit_logs.record_deleted_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_uploads_audit_logs.record_deleted_at IS 'Set once the upload this entry is associated with is deleted. Once NOW() - record_deleted_at is above a certain threshold, this log entry will be deleted.';


--
-- Name: COLUMN lsif_uploads_audit_logs.transition_columns; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_uploads_audit_logs.transition_columns IS 'Array of changes that occurred to the upload for this entry, in the form of {"column"=>"<column name>", "old"=>"<previous value>", "new"=>"<new value>"}.';


--
-- Name: COLUMN lsif_uploads_audit_logs.reason; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_uploads_audit_logs.reason IS 'The reason/source for this entry.';


--
-- Name: lsif_uploads_audit_logs_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.lsif_uploads_audit_logs_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: lsif_uploads_audit_logs_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.lsif_uploads_audit_logs_seq OWNED BY public.lsif_uploads_audit_logs.sequence;


--
-- Name: lsif_uploads_reference_counts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.lsif_uploads_reference_counts (
    upload_id integer NOT NULL,
    reference_count integer NOT NULL
);


--
-- Name: TABLE lsif_uploads_reference_counts; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.lsif_uploads_reference_counts IS 'A less hot-path reference count for upload records.';


--
-- Name: COLUMN lsif_uploads_reference_counts.upload_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_uploads_reference_counts.upload_id IS 'The identifier of the referenced upload.';


--
-- Name: COLUMN lsif_uploads_reference_counts.reference_count; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_uploads_reference_counts.reference_count IS 'The number of references to the associated upload from other records (via lsif_references).';


--
-- Name: lsif_uploads_visible_at_tip; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.lsif_uploads_visible_at_tip (
    repository_id integer NOT NULL,
    upload_id integer NOT NULL,
    branch_or_tag_name text DEFAULT ''::text NOT NULL,
    is_default_branch boolean DEFAULT false NOT NULL
);


--
-- Name: TABLE lsif_uploads_visible_at_tip; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.lsif_uploads_visible_at_tip IS 'Associates a repository with the set of LSIF upload identifiers that can serve intelligence for the tip of the default branch.';


--
-- Name: COLUMN lsif_uploads_visible_at_tip.upload_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_uploads_visible_at_tip.upload_id IS 'The identifier of the upload visible from the tip of the specified branch or tag.';


--
-- Name: COLUMN lsif_uploads_visible_at_tip.branch_or_tag_name; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_uploads_visible_at_tip.branch_or_tag_name IS 'The name of the branch or tag.';


--
-- Name: COLUMN lsif_uploads_visible_at_tip.is_default_branch; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.lsif_uploads_visible_at_tip.is_default_branch IS 'Whether the specified branch is the default of the repository. Always false for tags.';


--
-- Name: lsif_uploads_vulnerability_scan; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.lsif_uploads_vulnerability_scan (
    id bigint NOT NULL,
    upload_id integer NOT NULL,
    last_scanned_at timestamp without time zone DEFAULT now() NOT NULL
);


--
-- Name: lsif_uploads_vulnerability_scan_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.lsif_uploads_vulnerability_scan_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: lsif_uploads_vulnerability_scan_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.lsif_uploads_vulnerability_scan_id_seq OWNED BY public.lsif_uploads_vulnerability_scan.id;


--
-- Name: lsif_uploads_with_repository_name; Type: VIEW; Schema: public; Owner: -
--

CREATE VIEW public.lsif_uploads_with_repository_name AS
 SELECT u.id,
    u.commit,
    u.root,
    u.queued_at,
    u.uploaded_at,
    u.state,
    u.failure_message,
    u.started_at,
    u.finished_at,
    u.repository_id,
    u.indexer,
    u.indexer_version,
    u.num_parts,
    u.uploaded_parts,
    u.process_after,
    u.num_resets,
    u.upload_size,
    u.num_failures,
    u.associated_index_id,
    u.content_type,
    u.should_reindex,
    u.expired,
    u.last_retention_scan_at,
    r.name AS repository_name,
    u.uncompressed_size
   FROM (public.lsif_uploads u
     JOIN public.repo r ON ((r.id = u.repository_id)))
  WHERE (r.deleted_at IS NULL);


--
-- Name: names; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.names (
    name public.citext NOT NULL,
    user_id integer,
    org_id integer,
    team_id integer,
    tenant_id integer,
    CONSTRAINT names_check CHECK (((user_id IS NOT NULL) OR (org_id IS NOT NULL) OR (team_id IS NOT NULL)))
);


--
-- Name: namespace_permissions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.namespace_permissions (
    id integer NOT NULL,
    namespace text NOT NULL,
    resource_id integer NOT NULL,
    user_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    tenant_id integer,
    CONSTRAINT namespace_not_blank CHECK ((namespace <> ''::text))
);


--
-- Name: namespace_permissions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.namespace_permissions_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: namespace_permissions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.namespace_permissions_id_seq OWNED BY public.namespace_permissions.id;


--
-- Name: notebook_stars; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.notebook_stars (
    notebook_id integer NOT NULL,
    user_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    tenant_id integer
);


--
-- Name: notebooks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.notebooks (
    id bigint NOT NULL,
    title text NOT NULL,
    blocks jsonb DEFAULT '[]'::jsonb NOT NULL,
    public boolean NOT NULL,
    creator_user_id integer,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    blocks_tsvector tsvector GENERATED ALWAYS AS (jsonb_to_tsvector('english'::regconfig, blocks, '["string"]'::jsonb)) STORED,
    namespace_user_id integer,
    namespace_org_id integer,
    updater_user_id integer,
    pattern_type public.pattern_type DEFAULT 'keyword'::public.pattern_type NOT NULL,
    tenant_id integer,
    CONSTRAINT blocks_is_array CHECK ((jsonb_typeof(blocks) = 'array'::text)),
    CONSTRAINT notebooks_has_max_1_namespace CHECK ((((namespace_user_id IS NULL) AND (namespace_org_id IS NULL)) OR ((namespace_user_id IS NULL) <> (namespace_org_id IS NULL))))
);


--
-- Name: notebooks_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.notebooks_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: notebooks_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.notebooks_id_seq OWNED BY public.notebooks.id;


--
-- Name: org_invitations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.org_invitations (
    id bigint NOT NULL,
    org_id integer NOT NULL,
    sender_user_id integer NOT NULL,
    recipient_user_id integer,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    notified_at timestamp with time zone,
    responded_at timestamp with time zone,
    response_type boolean,
    revoked_at timestamp with time zone,
    deleted_at timestamp with time zone,
    recipient_email public.citext,
    expires_at timestamp with time zone,
    tenant_id integer,
    CONSTRAINT check_atomic_response CHECK (((responded_at IS NULL) = (response_type IS NULL))),
    CONSTRAINT check_single_use CHECK ((((responded_at IS NULL) AND (response_type IS NULL)) OR (revoked_at IS NULL))),
    CONSTRAINT either_user_id_or_email_defined CHECK (((recipient_user_id IS NULL) <> (recipient_email IS NULL)))
);


--
-- Name: org_invitations_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.org_invitations_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: org_invitations_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.org_invitations_id_seq OWNED BY public.org_invitations.id;


--
-- Name: org_members; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.org_members (
    id integer NOT NULL,
    org_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    user_id integer NOT NULL,
    tenant_id integer
);


--
-- Name: org_members_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.org_members_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: org_members_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.org_members_id_seq OWNED BY public.org_members.id;


--
-- Name: org_stats; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.org_stats (
    org_id integer NOT NULL,
    code_host_repo_count integer DEFAULT 0,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    tenant_id integer
);


--
-- Name: TABLE org_stats; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.org_stats IS 'Business statistics for organizations';


--
-- Name: COLUMN org_stats.org_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.org_stats.org_id IS 'Org ID that the stats relate to.';


--
-- Name: COLUMN org_stats.code_host_repo_count; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.org_stats.code_host_repo_count IS 'Count of repositories accessible on all code hosts for this organization.';


--
-- Name: orgs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.orgs (
    id integer NOT NULL,
    name public.citext NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    display_name text,
    slack_webhook_url text,
    deleted_at timestamp with time zone,
    tenant_id integer,
    CONSTRAINT orgs_display_name_max_length CHECK ((char_length(display_name) <= 255)),
    CONSTRAINT orgs_name_max_length CHECK ((char_length((name)::text) <= 255)),
    CONSTRAINT orgs_name_valid_chars CHECK ((name OPERATOR(public.~) '^[a-zA-Z0-9](?:[a-zA-Z0-9]|[-.](?=[a-zA-Z0-9]))*-?$'::public.citext))
);


--
-- Name: orgs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.orgs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: orgs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.orgs_id_seq OWNED BY public.orgs.id;


--
-- Name: orgs_open_beta_stats; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.orgs_open_beta_stats (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id integer,
    org_id integer,
    created_at timestamp with time zone DEFAULT now(),
    data jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id integer
);


--
-- Name: out_of_band_migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.out_of_band_migrations (
    id integer NOT NULL,
    team text NOT NULL,
    component text NOT NULL,
    description text NOT NULL,
    progress double precision DEFAULT 0 NOT NULL,
    created timestamp with time zone NOT NULL,
    last_updated timestamp with time zone,
    non_destructive boolean NOT NULL,
    apply_reverse boolean DEFAULT false NOT NULL,
    is_enterprise boolean DEFAULT false NOT NULL,
    introduced_version_major integer NOT NULL,
    introduced_version_minor integer NOT NULL,
    deprecated_version_major integer,
    deprecated_version_minor integer,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id integer,
    CONSTRAINT out_of_band_migrations_component_nonempty CHECK ((component <> ''::text)),
    CONSTRAINT out_of_band_migrations_description_nonempty CHECK ((description <> ''::text)),
    CONSTRAINT out_of_band_migrations_progress_range CHECK (((progress >= (0)::double precision) AND (progress <= (1)::double precision))),
    CONSTRAINT out_of_band_migrations_team_nonempty CHECK ((team <> ''::text))
);


--
-- Name: TABLE out_of_band_migrations; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.out_of_band_migrations IS 'Stores metadata and progress about an out-of-band migration routine.';


--
-- Name: COLUMN out_of_band_migrations.id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.out_of_band_migrations.id IS 'A globally unique primary key for this migration. The same key is used consistently across all Sourcegraph instances for the same migration.';


--
-- Name: COLUMN out_of_band_migrations.team; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.out_of_band_migrations.team IS 'The name of the engineering team responsible for the migration.';


--
-- Name: COLUMN out_of_band_migrations.component; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.out_of_band_migrations.component IS 'The name of the component undergoing a migration.';


--
-- Name: COLUMN out_of_band_migrations.description; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.out_of_band_migrations.description IS 'A brief description about the migration.';


--
-- Name: COLUMN out_of_band_migrations.progress; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.out_of_band_migrations.progress IS 'The percentage progress in the up direction (0=0%, 1=100%).';


--
-- Name: COLUMN out_of_band_migrations.created; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.out_of_band_migrations.created IS 'The date and time the migration was inserted into the database (via an upgrade).';


--
-- Name: COLUMN out_of_band_migrations.last_updated; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.out_of_band_migrations.last_updated IS 'The date and time the migration was last updated.';


--
-- Name: COLUMN out_of_band_migrations.non_destructive; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.out_of_band_migrations.non_destructive IS 'Whether or not this migration alters data so it can no longer be read by the previous Sourcegraph instance.';


--
-- Name: COLUMN out_of_band_migrations.apply_reverse; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.out_of_band_migrations.apply_reverse IS 'Whether this migration should run in the opposite direction (to support an upcoming downgrade).';


--
-- Name: COLUMN out_of_band_migrations.is_enterprise; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.out_of_band_migrations.is_enterprise IS 'When true, these migrations are invisible to OSS mode.';


--
-- Name: COLUMN out_of_band_migrations.introduced_version_major; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.out_of_band_migrations.introduced_version_major IS 'The Sourcegraph version (major component) in which this migration was first introduced.';


--
-- Name: COLUMN out_of_band_migrations.introduced_version_minor; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.out_of_band_migrations.introduced_version_minor IS 'The Sourcegraph version (minor component) in which this migration was first introduced.';


--
-- Name: COLUMN out_of_band_migrations.deprecated_version_major; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.out_of_band_migrations.deprecated_version_major IS 'The lowest Sourcegraph version (major component) that assumes the migration has completed.';


--
-- Name: COLUMN out_of_band_migrations.deprecated_version_minor; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.out_of_band_migrations.deprecated_version_minor IS 'The lowest Sourcegraph version (minor component) that assumes the migration has completed.';


--
-- Name: out_of_band_migrations_errors; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.out_of_band_migrations_errors (
    id integer NOT NULL,
    migration_id integer NOT NULL,
    message text NOT NULL,
    created timestamp with time zone DEFAULT now() NOT NULL,
    tenant_id integer,
    CONSTRAINT out_of_band_migrations_errors_message_nonempty CHECK ((message <> ''::text))
);


--
-- Name: TABLE out_of_band_migrations_errors; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.out_of_band_migrations_errors IS 'Stores errors that occurred while performing an out-of-band migration.';


--
-- Name: COLUMN out_of_band_migrations_errors.id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.out_of_band_migrations_errors.id IS 'A unique identifer.';


--
-- Name: COLUMN out_of_band_migrations_errors.migration_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.out_of_band_migrations_errors.migration_id IS 'The identifier of the migration.';


--
-- Name: COLUMN out_of_band_migrations_errors.message; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.out_of_band_migrations_errors.message IS 'The error message.';


--
-- Name: COLUMN out_of_band_migrations_errors.created; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.out_of_band_migrations_errors.created IS 'The date and time the error occurred.';


--
-- Name: out_of_band_migrations_errors_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.out_of_band_migrations_errors_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: out_of_band_migrations_errors_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.out_of_band_migrations_errors_id_seq OWNED BY public.out_of_band_migrations_errors.id;


--
-- Name: out_of_band_migrations_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.out_of_band_migrations_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: out_of_band_migrations_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.out_of_band_migrations_id_seq OWNED BY public.out_of_band_migrations.id;


--
-- Name: outbound_webhook_event_types; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.outbound_webhook_event_types (
    id bigint NOT NULL,
    outbound_webhook_id bigint NOT NULL,
    event_type text NOT NULL,
    scope text,
    tenant_id integer
);


--
-- Name: outbound_webhook_event_types_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.outbound_webhook_event_types_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: outbound_webhook_event_types_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.outbound_webhook_event_types_id_seq OWNED BY public.outbound_webhook_event_types.id;


--
-- Name: outbound_webhook_jobs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.outbound_webhook_jobs (
    id bigint NOT NULL,
    event_type text NOT NULL,
    scope text,
    encryption_key_id text,
    payload bytea NOT NULL,
    state text DEFAULT 'queued'::text NOT NULL,
    failure_message text,
    queued_at timestamp with time zone DEFAULT now() NOT NULL,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    process_after timestamp with time zone,
    num_resets integer DEFAULT 0 NOT NULL,
    num_failures integer DEFAULT 0 NOT NULL,
    last_heartbeat_at timestamp with time zone,
    execution_logs json[],
    worker_hostname text DEFAULT ''::text NOT NULL,
    cancel boolean DEFAULT false NOT NULL,
    tenant_id integer
);


--
-- Name: outbound_webhook_jobs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.outbound_webhook_jobs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: outbound_webhook_jobs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.outbound_webhook_jobs_id_seq OWNED BY public.outbound_webhook_jobs.id;


--
-- Name: outbound_webhook_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.outbound_webhook_logs (
    id bigint NOT NULL,
    job_id bigint NOT NULL,
    outbound_webhook_id bigint NOT NULL,
    sent_at timestamp with time zone DEFAULT now() NOT NULL,
    status_code integer NOT NULL,
    encryption_key_id text,
    request bytea NOT NULL,
    response bytea NOT NULL,
    error bytea NOT NULL,
    tenant_id integer
);


--
-- Name: outbound_webhook_logs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.outbound_webhook_logs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: outbound_webhook_logs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.outbound_webhook_logs_id_seq OWNED BY public.outbound_webhook_logs.id;


--
-- Name: outbound_webhooks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.outbound_webhooks (
    id bigint NOT NULL,
    created_by integer,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_by integer,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    encryption_key_id text,
    url bytea NOT NULL,
    secret bytea NOT NULL,
    tenant_id integer
);


--
-- Name: outbound_webhooks_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.outbound_webhooks_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: outbound_webhooks_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.outbound_webhooks_id_seq OWNED BY public.outbound_webhooks.id;


--
-- Name: outbound_webhooks_with_event_types; Type: VIEW; Schema: public; Owner: -
--

CREATE VIEW public.outbound_webhooks_with_event_types AS
 SELECT id,
    created_by,
    created_at,
    updated_by,
    updated_at,
    encryption_key_id,
    url,
    secret,
    array_to_json(ARRAY( SELECT json_build_object('id', outbound_webhook_event_types.id, 'outbound_webhook_id', outbound_webhook_event_types.outbound_webhook_id, 'event_type', outbound_webhook_event_types.event_type, 'scope', outbound_webhook_event_types.scope) AS json_build_object
           FROM public.outbound_webhook_event_types
          WHERE (outbound_webhook_event_types.outbound_webhook_id = outbound_webhooks.id))) AS event_types
   FROM public.outbound_webhooks;


--
-- Name: own_aggregate_recent_contribution; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.own_aggregate_recent_contribution (
    id integer NOT NULL,
    commit_author_id integer NOT NULL,
    changed_file_path_id integer NOT NULL,
    contributions_count integer DEFAULT 0,
    tenant_id integer
);


--
-- Name: own_aggregate_recent_contribution_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.own_aggregate_recent_contribution_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: own_aggregate_recent_contribution_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.own_aggregate_recent_contribution_id_seq OWNED BY public.own_aggregate_recent_contribution.id;


--
-- Name: own_aggregate_recent_view; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.own_aggregate_recent_view (
    id integer NOT NULL,
    viewer_id integer NOT NULL,
    viewed_file_path_id integer NOT NULL,
    views_count integer DEFAULT 0,
    tenant_id integer
);


--
-- Name: TABLE own_aggregate_recent_view; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.own_aggregate_recent_view IS 'One entry contains a number of views of a single file by a given viewer.';


--
-- Name: own_aggregate_recent_view_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.own_aggregate_recent_view_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: own_aggregate_recent_view_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.own_aggregate_recent_view_id_seq OWNED BY public.own_aggregate_recent_view.id;


--
-- Name: own_background_jobs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.own_background_jobs (
    id integer NOT NULL,
    state text DEFAULT 'queued'::text,
    failure_message text,
    queued_at timestamp with time zone DEFAULT now(),
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    process_after timestamp with time zone,
    num_resets integer DEFAULT 0 NOT NULL,
    num_failures integer DEFAULT 0 NOT NULL,
    last_heartbeat_at timestamp with time zone,
    execution_logs json[],
    worker_hostname text DEFAULT ''::text NOT NULL,
    cancel boolean DEFAULT false NOT NULL,
    repo_id integer NOT NULL,
    job_type integer NOT NULL,
    tenant_id integer
);


--
-- Name: own_signal_configurations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.own_signal_configurations (
    id integer NOT NULL,
    name text NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    excluded_repo_patterns text[],
    enabled boolean DEFAULT false NOT NULL,
    tenant_id integer
);


--
-- Name: own_background_jobs_config_aware; Type: VIEW; Schema: public; Owner: -
--

CREATE VIEW public.own_background_jobs_config_aware AS
 SELECT obj.id,
    obj.state,
    obj.failure_message,
    obj.queued_at,
    obj.started_at,
    obj.finished_at,
    obj.process_after,
    obj.num_resets,
    obj.num_failures,
    obj.last_heartbeat_at,
    obj.execution_logs,
    obj.worker_hostname,
    obj.cancel,
    obj.repo_id,
    obj.job_type,
    osc.name AS config_name
   FROM (public.own_background_jobs obj
     JOIN public.own_signal_configurations osc ON ((obj.job_type = osc.id)))
  WHERE (osc.enabled IS TRUE);


--
-- Name: own_background_jobs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.own_background_jobs_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: own_background_jobs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.own_background_jobs_id_seq OWNED BY public.own_background_jobs.id;


--
-- Name: own_signal_configurations_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.own_signal_configurations_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: own_signal_configurations_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.own_signal_configurations_id_seq OWNED BY public.own_signal_configurations.id;


--
-- Name: own_signal_recent_contribution; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.own_signal_recent_contribution (
    id integer NOT NULL,
    commit_author_id integer NOT NULL,
    changed_file_path_id integer NOT NULL,
    commit_timestamp timestamp without time zone NOT NULL,
    commit_id bytea NOT NULL,
    tenant_id integer
);


--
-- Name: TABLE own_signal_recent_contribution; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.own_signal_recent_contribution IS 'One entry per file changed in every commit that classifies as a contribution signal.';


--
-- Name: own_signal_recent_contribution_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.own_signal_recent_contribution_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: own_signal_recent_contribution_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.own_signal_recent_contribution_id_seq OWNED BY public.own_signal_recent_contribution.id;


--
-- Name: ownership_path_stats; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.ownership_path_stats (
    file_path_id integer NOT NULL,
    tree_codeowned_files_count integer,
    last_updated_at timestamp without time zone NOT NULL,
    tree_assigned_ownership_files_count integer,
    tree_any_ownership_files_count integer,
    tenant_id integer
);


--
-- Name: TABLE ownership_path_stats; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.ownership_path_stats IS 'Data on how many files in given tree are owned by anyone.

We choose to have a table for `ownership_path_stats` - more general than for CODEOWNERS,
with a specific tree_codeowned_files_count CODEOWNERS column. The reason for that
is that we aim at expanding path stats by including total owned files (via CODEOWNERS
or assigned ownership), and perhaps files count by assigned ownership only.';


--
-- Name: COLUMN ownership_path_stats.last_updated_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.ownership_path_stats.last_updated_at IS 'When the last background job updating counts run.';


--
-- Name: package_repo_filters; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.package_repo_filters (
    id integer NOT NULL,
    behaviour text NOT NULL,
    scheme text NOT NULL,
    matcher jsonb NOT NULL,
    deleted_at timestamp with time zone,
    updated_at timestamp with time zone DEFAULT statement_timestamp() NOT NULL,
    tenant_id integer,
    CONSTRAINT package_repo_filters_behaviour_is_allow_or_block CHECK ((behaviour = ANY ('{BLOCK,ALLOW}'::text[]))),
    CONSTRAINT package_repo_filters_is_pkgrepo_scheme CHECK ((scheme = ANY ('{semanticdb,npm,go,python,rust-analyzer,scip-ruby}'::text[]))),
    CONSTRAINT package_repo_filters_valid_oneof_glob CHECK ((((matcher ? 'VersionGlob'::text) AND ((matcher ->> 'VersionGlob'::text) <> ''::text) AND ((matcher ->> 'PackageName'::text) <> ''::text) AND (NOT (matcher ? 'PackageGlob'::text))) OR ((matcher ? 'PackageGlob'::text) AND ((matcher ->> 'PackageGlob'::text) <> ''::text) AND (NOT (matcher ? 'VersionGlob'::text)))))
);


--
-- Name: package_repo_filters_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.package_repo_filters_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: package_repo_filters_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.package_repo_filters_id_seq OWNED BY public.package_repo_filters.id;


--
-- Name: package_repo_versions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.package_repo_versions (
    id bigint NOT NULL,
    package_id bigint NOT NULL,
    version text NOT NULL,
    blocked boolean DEFAULT false NOT NULL,
    last_checked_at timestamp with time zone,
    tenant_id integer
);


--
-- Name: package_repo_versions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.package_repo_versions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: package_repo_versions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.package_repo_versions_id_seq OWNED BY public.package_repo_versions.id;


--
-- Name: permission_sync_jobs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.permission_sync_jobs (
    id integer NOT NULL,
    state text DEFAULT 'queued'::text,
    reason text NOT NULL,
    failure_message text,
    queued_at timestamp with time zone DEFAULT now(),
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    process_after timestamp with time zone,
    num_resets integer DEFAULT 0 NOT NULL,
    num_failures integer DEFAULT 0 NOT NULL,
    last_heartbeat_at timestamp with time zone,
    execution_logs json[],
    worker_hostname text DEFAULT ''::text NOT NULL,
    cancel boolean DEFAULT false NOT NULL,
    repository_id integer,
    user_id integer,
    triggered_by_user_id integer,
    priority integer DEFAULT 0 NOT NULL,
    invalidate_caches boolean DEFAULT false NOT NULL,
    cancellation_reason text,
    no_perms boolean DEFAULT false NOT NULL,
    permissions_added integer DEFAULT 0 NOT NULL,
    permissions_removed integer DEFAULT 0 NOT NULL,
    permissions_found integer DEFAULT 0 NOT NULL,
    code_host_states json[],
    is_partial_success boolean DEFAULT false,
    tenant_id integer,
    CONSTRAINT permission_sync_jobs_for_repo_or_user CHECK (((user_id IS NULL) <> (repository_id IS NULL)))
);


--
-- Name: COLUMN permission_sync_jobs.reason; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.permission_sync_jobs.reason IS 'Specifies why permissions sync job was triggered.';


--
-- Name: COLUMN permission_sync_jobs.triggered_by_user_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.permission_sync_jobs.triggered_by_user_id IS 'Specifies an ID of a user who triggered a sync.';


--
-- Name: COLUMN permission_sync_jobs.priority; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.permission_sync_jobs.priority IS 'Specifies numeric priority for the permissions sync job.';


--
-- Name: COLUMN permission_sync_jobs.cancellation_reason; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.permission_sync_jobs.cancellation_reason IS 'Specifies why permissions sync job was cancelled.';


--
-- Name: permission_sync_jobs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.permission_sync_jobs_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: permission_sync_jobs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.permission_sync_jobs_id_seq OWNED BY public.permission_sync_jobs.id;


--
-- Name: permissions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.permissions (
    id integer NOT NULL,
    namespace text NOT NULL,
    action text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    tenant_id integer,
    CONSTRAINT action_not_blank CHECK ((action <> ''::text)),
    CONSTRAINT namespace_not_blank CHECK ((namespace <> ''::text))
);


--
-- Name: permissions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.permissions_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: permissions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.permissions_id_seq OWNED BY public.permissions.id;


--
-- Name: phabricator_repos; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.phabricator_repos (
    id integer NOT NULL,
    callsign public.citext NOT NULL,
    repo_name public.citext NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    url text DEFAULT ''::text NOT NULL,
    tenant_id integer
);


--
-- Name: phabricator_repos_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.phabricator_repos_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: phabricator_repos_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.phabricator_repos_id_seq OWNED BY public.phabricator_repos.id;


--
-- Name: product_licenses; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.product_licenses (
    id uuid NOT NULL,
    product_subscription_id uuid NOT NULL,
    license_key text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    license_version integer,
    license_tags text[],
    license_user_count integer,
    license_expires_at timestamp with time zone,
    access_token_enabled boolean DEFAULT true NOT NULL,
    site_id uuid,
    license_check_token bytea,
    revoked_at timestamp with time zone,
    salesforce_sub_id text,
    salesforce_opp_id text,
    revoke_reason text,
    tenant_id integer
);


--
-- Name: COLUMN product_licenses.access_token_enabled; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.product_licenses.access_token_enabled IS 'Whether this license key can be used as an access token to authenticate API requests';


--
-- Name: product_subscriptions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.product_subscriptions (
    id uuid NOT NULL,
    user_id integer NOT NULL,
    billing_subscription_id text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    archived_at timestamp with time zone,
    account_number text,
    cody_gateway_enabled boolean DEFAULT false NOT NULL,
    cody_gateway_chat_rate_limit bigint,
    cody_gateway_chat_rate_interval_seconds integer,
    cody_gateway_embeddings_api_rate_limit bigint,
    cody_gateway_embeddings_api_rate_interval_seconds integer,
    cody_gateway_embeddings_api_allowed_models text[],
    cody_gateway_chat_rate_limit_allowed_models text[],
    cody_gateway_code_rate_limit bigint,
    cody_gateway_code_rate_interval_seconds integer,
    cody_gateway_code_rate_limit_allowed_models text[],
    tenant_id integer
);


--
-- Name: COLUMN product_subscriptions.cody_gateway_embeddings_api_rate_limit; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.product_subscriptions.cody_gateway_embeddings_api_rate_limit IS 'Custom requests per time interval allowed for embeddings';


--
-- Name: COLUMN product_subscriptions.cody_gateway_embeddings_api_rate_interval_seconds; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.product_subscriptions.cody_gateway_embeddings_api_rate_interval_seconds IS 'Custom time interval over which the embeddings rate limit is applied';


--
-- Name: COLUMN product_subscriptions.cody_gateway_embeddings_api_allowed_models; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.product_subscriptions.cody_gateway_embeddings_api_allowed_models IS 'Custom override for the set of models allowed for embedding';


--
-- Name: prompts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.prompts (
    id integer NOT NULL,
    name public.citext NOT NULL,
    description text NOT NULL,
    definition_text text NOT NULL,
    draft boolean DEFAULT false NOT NULL,
    visibility_secret boolean DEFAULT true NOT NULL,
    owner_user_id integer,
    owner_org_id integer,
    created_by integer,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_by integer,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    tenant_id integer,
    CONSTRAINT prompts_definition_text_max_length CHECK ((char_length(definition_text) <= (1024 * 100))),
    CONSTRAINT prompts_description_max_length CHECK ((char_length(description) <= (1024 * 50))),
    CONSTRAINT prompts_has_valid_owner CHECK ((((owner_user_id IS NOT NULL) AND (owner_org_id IS NULL)) OR ((owner_org_id IS NOT NULL) AND (owner_user_id IS NULL)))),
    CONSTRAINT prompts_name_max_length CHECK ((char_length((name)::text) <= 255)),
    CONSTRAINT prompts_name_valid_chars CHECK ((name OPERATOR(public.~) '^[a-zA-Z0-9](?:[a-zA-Z0-9]|[-.](?=[a-zA-Z0-9]))*-?$'::public.citext))
);


--
-- Name: prompts_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.prompts_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: prompts_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.prompts_id_seq OWNED BY public.prompts.id;


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id integer NOT NULL,
    username public.citext NOT NULL,
    display_name text,
    avatar_url text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    invite_quota integer DEFAULT 100 NOT NULL,
    passwd text,
    passwd_reset_code text,
    passwd_reset_time timestamp with time zone,
    site_admin boolean DEFAULT false NOT NULL,
    page_views integer DEFAULT 0 NOT NULL,
    search_queries integer DEFAULT 0 NOT NULL,
    billing_customer_id text,
    invalidated_sessions_at timestamp with time zone DEFAULT now() NOT NULL,
    tos_accepted boolean DEFAULT false NOT NULL,
    completions_quota integer,
    code_completions_quota integer,
    completed_post_signup boolean DEFAULT false NOT NULL,
    cody_pro_enabled_at timestamp with time zone,
    tenant_id integer,
    CONSTRAINT users_display_name_max_length CHECK ((char_length(display_name) <= 255)),
    CONSTRAINT users_username_max_length CHECK ((char_length((username)::text) <= 255)),
    CONSTRAINT users_username_valid_chars CHECK ((username OPERATOR(public.~) '^\w(?:\w|[-.](?=\w))*-?$'::public.citext))
);


--
-- Name: prompts_view; Type: VIEW; Schema: public; Owner: -
--

CREATE VIEW public.prompts_view AS
 SELECT prompts.id,
    prompts.name,
    prompts.description,
    prompts.definition_text,
    prompts.draft,
    prompts.visibility_secret,
    prompts.owner_user_id,
    prompts.owner_org_id,
    prompts.created_by,
    prompts.created_at,
    prompts.updated_by,
    prompts.updated_at,
    (((COALESCE(users.username, orgs.name))::text || '/'::text) || (prompts.name)::text) AS name_with_owner
   FROM ((public.prompts
     LEFT JOIN public.users ON ((users.id = prompts.owner_user_id)))
     LEFT JOIN public.orgs ON ((orgs.id = prompts.owner_org_id)));


--
-- Name: query_runner_state; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.query_runner_state (
    query text,
    last_executed timestamp with time zone,
    latest_result timestamp with time zone,
    exec_duration_ns bigint,
    tenant_id integer
);


--
-- Name: reconciler_changesets; Type: VIEW; Schema: public; Owner: -
--

CREATE VIEW public.reconciler_changesets AS
 SELECT c.id,
    c.batch_change_ids,
    c.repo_id,
    c.queued_at,
    c.created_at,
    c.updated_at,
    c.metadata,
    c.external_id,
    c.external_service_type,
    c.external_deleted_at,
    c.external_branch,
    c.external_updated_at,
    c.external_state,
    c.external_review_state,
    c.external_check_state,
    c.commit_verification,
    c.diff_stat_added,
    c.diff_stat_deleted,
    c.sync_state,
    c.current_spec_id,
    c.previous_spec_id,
    c.publication_state,
    c.owned_by_batch_change_id,
    c.reconciler_state,
    c.computed_state,
    c.failure_message,
    c.started_at,
    c.finished_at,
    c.process_after,
    c.num_resets,
    c.closing,
    c.num_failures,
    c.log_contents,
    c.execution_logs,
    c.syncer_error,
    c.external_title,
    c.worker_hostname,
    c.ui_publication_state,
    c.last_heartbeat_at,
    c.external_fork_name,
    c.external_fork_namespace,
    c.detached_at,
    c.previous_failure_message
   FROM (public.changesets c
     JOIN public.repo r ON ((r.id = c.repo_id)))
  WHERE ((r.deleted_at IS NULL) AND (EXISTS ( SELECT 1
           FROM ((public.batch_changes
             LEFT JOIN public.users namespace_user ON ((batch_changes.namespace_user_id = namespace_user.id)))
             LEFT JOIN public.orgs namespace_org ON ((batch_changes.namespace_org_id = namespace_org.id)))
          WHERE ((c.batch_change_ids ? (batch_changes.id)::text) AND (namespace_user.deleted_at IS NULL) AND (namespace_org.deleted_at IS NULL)))));


--
-- Name: redis_key_value; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.redis_key_value (
    namespace text NOT NULL,
    key text NOT NULL,
    value bytea NOT NULL,
    tenant_id integer
);


--
-- Name: registry_extension_releases; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.registry_extension_releases (
    id bigint NOT NULL,
    registry_extension_id integer NOT NULL,
    creator_user_id integer NOT NULL,
    release_version public.citext,
    release_tag public.citext NOT NULL,
    manifest jsonb NOT NULL,
    bundle text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    source_map text,
    tenant_id integer
);


--
-- Name: registry_extension_releases_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.registry_extension_releases_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: registry_extension_releases_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.registry_extension_releases_id_seq OWNED BY public.registry_extension_releases.id;


--
-- Name: registry_extensions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.registry_extensions (
    id integer NOT NULL,
    uuid uuid NOT NULL,
    publisher_user_id integer,
    publisher_org_id integer,
    name public.citext NOT NULL,
    manifest text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    tenant_id integer,
    CONSTRAINT registry_extensions_name_length CHECK (((char_length((name)::text) > 0) AND (char_length((name)::text) <= 128))),
    CONSTRAINT registry_extensions_name_valid_chars CHECK ((name OPERATOR(public.~) '^[a-zA-Z0-9](?:[a-zA-Z0-9]|[_.-](?=[a-zA-Z0-9]))*$'::public.citext)),
    CONSTRAINT registry_extensions_single_publisher CHECK (((publisher_user_id IS NULL) <> (publisher_org_id IS NULL)))
);


--
-- Name: registry_extensions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.registry_extensions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: registry_extensions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.registry_extensions_id_seq OWNED BY public.registry_extensions.id;


--
-- Name: repo_commits_changelists; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.repo_commits_changelists (
    id integer NOT NULL,
    repo_id integer NOT NULL,
    commit_sha bytea NOT NULL,
    perforce_changelist_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    tenant_id integer
);


--
-- Name: repo_commits_changelists_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.repo_commits_changelists_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: repo_commits_changelists_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.repo_commits_changelists_id_seq OWNED BY public.repo_commits_changelists.id;


--
-- Name: repo_embedding_job_stats; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.repo_embedding_job_stats (
    job_id integer NOT NULL,
    is_incremental boolean DEFAULT false NOT NULL,
    code_files_total integer DEFAULT 0 NOT NULL,
    code_files_embedded integer DEFAULT 0 NOT NULL,
    code_chunks_embedded integer DEFAULT 0 NOT NULL,
    code_files_skipped jsonb DEFAULT '{}'::jsonb NOT NULL,
    code_bytes_embedded bigint DEFAULT 0 NOT NULL,
    text_files_total integer DEFAULT 0 NOT NULL,
    text_files_embedded integer DEFAULT 0 NOT NULL,
    text_chunks_embedded integer DEFAULT 0 NOT NULL,
    text_files_skipped jsonb DEFAULT '{}'::jsonb NOT NULL,
    text_bytes_embedded bigint DEFAULT 0 NOT NULL,
    code_chunks_excluded integer DEFAULT 0 NOT NULL,
    text_chunks_excluded integer DEFAULT 0 NOT NULL,
    tenant_id integer
);


--
-- Name: repo_embedding_jobs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.repo_embedding_jobs (
    id integer NOT NULL,
    state text DEFAULT 'queued'::text,
    failure_message text,
    queued_at timestamp with time zone DEFAULT now(),
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    process_after timestamp with time zone,
    num_resets integer DEFAULT 0 NOT NULL,
    num_failures integer DEFAULT 0 NOT NULL,
    last_heartbeat_at timestamp with time zone,
    execution_logs json[],
    worker_hostname text DEFAULT ''::text NOT NULL,
    cancel boolean DEFAULT false NOT NULL,
    repo_id integer NOT NULL,
    revision text NOT NULL,
    tenant_id integer
);


--
-- Name: repo_embedding_jobs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.repo_embedding_jobs_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: repo_embedding_jobs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.repo_embedding_jobs_id_seq OWNED BY public.repo_embedding_jobs.id;


--
-- Name: repo_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.repo_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: repo_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.repo_id_seq OWNED BY public.repo.id;


--
-- Name: repo_kvps; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.repo_kvps (
    repo_id integer NOT NULL,
    key text NOT NULL,
    value text,
    tenant_id integer
);


--
-- Name: repo_paths; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.repo_paths (
    id integer NOT NULL,
    repo_id integer NOT NULL,
    absolute_path text NOT NULL,
    parent_id integer,
    tree_files_count integer,
    tree_files_counts_updated_at timestamp without time zone,
    tenant_id integer
);


--
-- Name: COLUMN repo_paths.absolute_path; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.repo_paths.absolute_path IS 'Absolute path does not start or end with forward slash. Example: "a/b/c". Root directory is empty path "".';


--
-- Name: COLUMN repo_paths.tree_files_count; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.repo_paths.tree_files_count IS 'Total count of files in the file tree rooted at the path. 1 for files.';


--
-- Name: COLUMN repo_paths.tree_files_counts_updated_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.repo_paths.tree_files_counts_updated_at IS 'Timestamp of the job that updated the file counts';


--
-- Name: repo_paths_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.repo_paths_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: repo_paths_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.repo_paths_id_seq OWNED BY public.repo_paths.id;


--
-- Name: repo_pending_permissions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.repo_pending_permissions (
    repo_id integer NOT NULL,
    permission text NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    user_ids_ints bigint[] DEFAULT '{}'::integer[] NOT NULL,
    tenant_id integer
);


--
-- Name: repo_permissions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.repo_permissions (
    repo_id integer NOT NULL,
    permission text NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    synced_at timestamp with time zone,
    user_ids_ints integer[] DEFAULT '{}'::integer[] NOT NULL,
    unrestricted boolean DEFAULT false NOT NULL,
    tenant_id integer
);


--
-- Name: repo_statistics; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.repo_statistics (
    total bigint DEFAULT 0 NOT NULL,
    soft_deleted bigint DEFAULT 0 NOT NULL,
    not_cloned bigint DEFAULT 0 NOT NULL,
    cloning bigint DEFAULT 0 NOT NULL,
    cloned bigint DEFAULT 0 NOT NULL,
    failed_fetch bigint DEFAULT 0 NOT NULL,
    corrupted bigint DEFAULT 0 NOT NULL,
    tenant_id integer
);


--
-- Name: COLUMN repo_statistics.total; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.repo_statistics.total IS 'Number of repositories that are not soft-deleted and not blocked';


--
-- Name: COLUMN repo_statistics.soft_deleted; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.repo_statistics.soft_deleted IS 'Number of repositories that are soft-deleted and not blocked';


--
-- Name: COLUMN repo_statistics.not_cloned; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.repo_statistics.not_cloned IS 'Number of repositories that are NOT soft-deleted and not blocked and not cloned by gitserver';


--
-- Name: COLUMN repo_statistics.cloning; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.repo_statistics.cloning IS 'Number of repositories that are NOT soft-deleted and not blocked and currently being cloned by gitserver';


--
-- Name: COLUMN repo_statistics.cloned; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.repo_statistics.cloned IS 'Number of repositories that are NOT soft-deleted and not blocked and cloned by gitserver';


--
-- Name: COLUMN repo_statistics.failed_fetch; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.repo_statistics.failed_fetch IS 'Number of repositories that are NOT soft-deleted and not blocked and have last_error set in gitserver_repos table';


--
-- Name: COLUMN repo_statistics.corrupted; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.repo_statistics.corrupted IS 'Number of repositories that are NOT soft-deleted and not blocked and have corrupted_at set in gitserver_repos table';


--
-- Name: role_permissions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.role_permissions (
    role_id integer NOT NULL,
    permission_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    tenant_id integer
);


--
-- Name: roles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.roles (
    id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    system boolean DEFAULT false NOT NULL,
    name public.citext NOT NULL,
    tenant_id integer
);


--
-- Name: COLUMN roles.system; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.roles.system IS 'This is used to indicate whether a role is read-only or can be modified.';


--
-- Name: roles_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.roles_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: roles_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.roles_id_seq OWNED BY public.roles.id;


--
-- Name: saved_searches; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.saved_searches (
    id integer NOT NULL,
    description text NOT NULL,
    query text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    notify_owner boolean DEFAULT false NOT NULL,
    notify_slack boolean DEFAULT false NOT NULL,
    user_id integer,
    org_id integer,
    slack_webhook_url text,
    created_by integer,
    updated_by integer,
    draft boolean DEFAULT false NOT NULL,
    visibility_secret boolean DEFAULT true NOT NULL,
    tenant_id integer,
    CONSTRAINT saved_searches_notifications_disabled CHECK (((notify_owner = false) AND (notify_slack = false))),
    CONSTRAINT user_or_org_id_not_null CHECK ((((user_id IS NOT NULL) AND (org_id IS NULL)) OR ((org_id IS NOT NULL) AND (user_id IS NULL))))
);


--
-- Name: saved_searches_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.saved_searches_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: saved_searches_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.saved_searches_id_seq OWNED BY public.saved_searches.id;


--
-- Name: search_context_default; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.search_context_default (
    user_id integer NOT NULL,
    search_context_id bigint NOT NULL,
    tenant_id integer
);


--
-- Name: TABLE search_context_default; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.search_context_default IS 'When a user sets a search context as default, a row is inserted into this table. A user can only have one default search context. If the user has not set their default search context, it will fall back to `global`.';


--
-- Name: search_context_repos; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.search_context_repos (
    search_context_id bigint NOT NULL,
    repo_id integer NOT NULL,
    revision text NOT NULL,
    tenant_id integer
);


--
-- Name: search_context_stars; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.search_context_stars (
    search_context_id bigint NOT NULL,
    user_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    tenant_id integer
);


--
-- Name: TABLE search_context_stars; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.search_context_stars IS 'When a user stars a search context, a row is inserted into this table. If the user unstars the search context, the row is deleted. The global context is not in the database, and therefore cannot be starred.';


--
-- Name: search_contexts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.search_contexts (
    id bigint NOT NULL,
    name public.citext NOT NULL,
    description text NOT NULL,
    public boolean NOT NULL,
    namespace_user_id integer,
    namespace_org_id integer,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    query text,
    tenant_id integer,
    CONSTRAINT search_contexts_has_one_or_no_namespace CHECK (((namespace_user_id IS NULL) OR (namespace_org_id IS NULL)))
);


--
-- Name: COLUMN search_contexts.deleted_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.search_contexts.deleted_at IS 'This column is unused as of Sourcegraph 3.34. Do not refer to it anymore. It will be dropped in a future version.';


--
-- Name: search_contexts_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.search_contexts_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: search_contexts_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.search_contexts_id_seq OWNED BY public.search_contexts.id;


--
-- Name: security_event_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.security_event_logs (
    id bigint NOT NULL,
    name text NOT NULL,
    url text NOT NULL,
    user_id integer NOT NULL,
    anonymous_user_id text NOT NULL,
    source text NOT NULL,
    argument jsonb NOT NULL,
    version text NOT NULL,
    "timestamp" timestamp with time zone NOT NULL,
    tenant_id integer,
    CONSTRAINT security_event_logs_check_has_user CHECK ((((user_id = 0) AND (anonymous_user_id <> ''::text)) OR ((user_id <> 0) AND (anonymous_user_id = ''::text)) OR ((user_id <> 0) AND (anonymous_user_id <> ''::text)))),
    CONSTRAINT security_event_logs_check_name_not_empty CHECK ((name <> ''::text)),
    CONSTRAINT security_event_logs_check_source_not_empty CHECK ((source <> ''::text)),
    CONSTRAINT security_event_logs_check_version_not_empty CHECK ((version <> ''::text))
);


--
-- Name: TABLE security_event_logs; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.security_event_logs IS 'Contains security-relevant events with a long time horizon for storage.';


--
-- Name: COLUMN security_event_logs.name; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.security_event_logs.name IS 'The event name as a CAPITALIZED_SNAKE_CASE string.';


--
-- Name: COLUMN security_event_logs.url; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.security_event_logs.url IS 'The URL within the Sourcegraph app which generated the event.';


--
-- Name: COLUMN security_event_logs.user_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.security_event_logs.user_id IS 'The ID of the actor associated with the event.';


--
-- Name: COLUMN security_event_logs.anonymous_user_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.security_event_logs.anonymous_user_id IS 'The UUID of the actor associated with the event.';


--
-- Name: COLUMN security_event_logs.source; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.security_event_logs.source IS 'The site section (WEB, BACKEND, etc.) that generated the event.';


--
-- Name: COLUMN security_event_logs.argument; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.security_event_logs.argument IS 'An arbitrary JSON blob containing event data.';


--
-- Name: COLUMN security_event_logs.version; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.security_event_logs.version IS 'The version of Sourcegraph which generated the event.';


--
-- Name: security_event_logs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.security_event_logs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: security_event_logs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.security_event_logs_id_seq OWNED BY public.security_event_logs.id;


--
-- Name: settings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.settings (
    id integer NOT NULL,
    org_id integer,
    contents text DEFAULT '{}'::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    user_id integer,
    author_user_id integer,
    tenant_id integer,
    CONSTRAINT settings_no_empty_contents CHECK ((contents <> ''::text))
);


--
-- Name: settings_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.settings_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: settings_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.settings_id_seq OWNED BY public.settings.id;


--
-- Name: site_config; Type: VIEW; Schema: public; Owner: -
--

CREATE VIEW public.site_config AS
 SELECT site_id,
    initialized
   FROM public.global_state;


--
-- Name: sub_repo_permissions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sub_repo_permissions (
    repo_id integer NOT NULL,
    user_id integer NOT NULL,
    version integer DEFAULT 1 NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    paths text[],
    ips text[],
    tenant_id integer,
    CONSTRAINT ips_paths_length_check CHECK (((ips IS NULL) OR ((array_length(ips, 1) = array_length(paths, 1)) AND (NOT (''::text = ANY (ips))))))
);


--
-- Name: TABLE sub_repo_permissions; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.sub_repo_permissions IS 'Responsible for storing permissions at a finer granularity than repo';


--
-- Name: COLUMN sub_repo_permissions.paths; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.sub_repo_permissions.paths IS 'Paths that begin with a minus sign (-) are exclusion paths.';


--
-- Name: COLUMN sub_repo_permissions.ips; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.sub_repo_permissions.ips IS 'IP addresses corresponding to each path. IP in slot 0 in the array corresponds to path the in slot 0 of the path array, etc. NULL if not yet migrated, empty array for no IP restrictions.';


--
-- Name: survey_responses; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.survey_responses (
    id bigint NOT NULL,
    user_id integer,
    email text,
    score integer NOT NULL,
    reason text,
    better text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    use_cases text[],
    other_use_case text,
    tenant_id integer
);


--
-- Name: survey_responses_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.survey_responses_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: survey_responses_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.survey_responses_id_seq OWNED BY public.survey_responses.id;


--
-- Name: syntactic_scip_indexing_jobs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.syntactic_scip_indexing_jobs (
    id bigint NOT NULL,
    commit text NOT NULL,
    queued_at timestamp with time zone DEFAULT now() NOT NULL,
    state text DEFAULT 'queued'::text NOT NULL,
    failure_message text,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    repository_id integer NOT NULL,
    process_after timestamp with time zone,
    num_resets integer DEFAULT 0 NOT NULL,
    num_failures integer DEFAULT 0 NOT NULL,
    execution_logs json[],
    commit_last_checked_at timestamp with time zone,
    worker_hostname text DEFAULT ''::text NOT NULL,
    last_heartbeat_at timestamp with time zone,
    cancel boolean DEFAULT false NOT NULL,
    should_reindex boolean DEFAULT false NOT NULL,
    enqueuer_user_id integer DEFAULT 0 NOT NULL,
    tenant_id integer,
    CONSTRAINT syntactic_scip_indexing_jobs_commit_valid_chars CHECK ((commit ~ '^[a-f0-9]{40}$'::text))
);


--
-- Name: TABLE syntactic_scip_indexing_jobs; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.syntactic_scip_indexing_jobs IS 'Stores metadata about a code intel syntactic index job.';


--
-- Name: COLUMN syntactic_scip_indexing_jobs.commit; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.syntactic_scip_indexing_jobs.commit IS 'A 40-char revhash. Note that this commit may not be resolvable in the future.';


--
-- Name: COLUMN syntactic_scip_indexing_jobs.execution_logs; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.syntactic_scip_indexing_jobs.execution_logs IS 'An array of [log entries](https://sourcegraph.com/github.com/sourcegraph/sourcegraph@3.23/-/blob/internal/workerutil/store.go#L48:6) (encoded as JSON) from the most recent execution.';


--
-- Name: COLUMN syntactic_scip_indexing_jobs.enqueuer_user_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.syntactic_scip_indexing_jobs.enqueuer_user_id IS 'ID of the user who scheduled this index. Records with a non-NULL user ID are prioritised over the rest';


--
-- Name: syntactic_scip_indexing_jobs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.syntactic_scip_indexing_jobs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: syntactic_scip_indexing_jobs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.syntactic_scip_indexing_jobs_id_seq OWNED BY public.syntactic_scip_indexing_jobs.id;


--
-- Name: syntactic_scip_indexing_jobs_with_repository_name; Type: VIEW; Schema: public; Owner: -
--

CREATE VIEW public.syntactic_scip_indexing_jobs_with_repository_name AS
 SELECT u.id,
    u.commit,
    u.queued_at,
    u.state,
    u.failure_message,
    u.started_at,
    u.finished_at,
    u.repository_id,
    u.process_after,
    u.num_resets,
    u.num_failures,
    u.execution_logs,
    u.should_reindex,
    u.enqueuer_user_id,
    r.name AS repository_name
   FROM (public.syntactic_scip_indexing_jobs u
     JOIN public.repo r ON ((r.id = u.repository_id)))
  WHERE (r.deleted_at IS NULL);


--
-- Name: syntactic_scip_last_index_scan; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.syntactic_scip_last_index_scan (
    repository_id integer NOT NULL,
    last_index_scan_at timestamp with time zone NOT NULL,
    tenant_id integer
);


--
-- Name: TABLE syntactic_scip_last_index_scan; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.syntactic_scip_last_index_scan IS 'Tracks the last time repository was checked for syntactic indexing job scheduling.';


--
-- Name: COLUMN syntactic_scip_last_index_scan.last_index_scan_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.syntactic_scip_last_index_scan.last_index_scan_at IS 'The last time uploads of this repository were considered for syntactic indexing job scheduling.';


--
-- Name: team_members; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.team_members (
    team_id integer NOT NULL,
    user_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    tenant_id integer
);


--
-- Name: teams; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.teams (
    id integer NOT NULL,
    name public.citext NOT NULL,
    display_name text,
    readonly boolean DEFAULT false NOT NULL,
    parent_team_id integer,
    creator_id integer,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    tenant_id integer,
    CONSTRAINT teams_display_name_max_length CHECK ((char_length(display_name) <= 255)),
    CONSTRAINT teams_name_max_length CHECK ((char_length((name)::text) <= 255)),
    CONSTRAINT teams_name_valid_chars CHECK ((name OPERATOR(public.~) '^[a-zA-Z0-9](?:[a-zA-Z0-9]|[-.](?=[a-zA-Z0-9]))*-?$'::public.citext))
);


--
-- Name: teams_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.teams_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: teams_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.teams_id_seq OWNED BY public.teams.id;


--
-- Name: telemetry_events_export_queue; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.telemetry_events_export_queue (
    id text NOT NULL,
    "timestamp" timestamp with time zone NOT NULL,
    payload_pb bytea NOT NULL,
    exported_at timestamp with time zone,
    tenant_id integer
);


--
-- Name: temporary_settings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.temporary_settings (
    id integer NOT NULL,
    user_id integer NOT NULL,
    contents jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    tenant_id integer
);


--
-- Name: TABLE temporary_settings; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.temporary_settings IS 'Stores per-user temporary settings used in the UI, for example, which modals have been dimissed or what theme is preferred.';


--
-- Name: COLUMN temporary_settings.user_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.temporary_settings.user_id IS 'The ID of the user the settings will be saved for.';


--
-- Name: COLUMN temporary_settings.contents; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.temporary_settings.contents IS 'JSON-encoded temporary settings.';


--
-- Name: temporary_settings_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.temporary_settings_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: temporary_settings_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.temporary_settings_id_seq OWNED BY public.temporary_settings.id;


--
-- Name: tenants; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.tenants (
    id bigint NOT NULL,
    name text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT tenant_name_length CHECK (((char_length(name) <= 32) AND (char_length(name) >= 3))),
    CONSTRAINT tenant_name_valid_chars CHECK ((name ~ '^[a-z](?:[a-z0-9\_-])*[a-z0-9]$'::text))
);


--
-- Name: TABLE tenants; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.tenants IS 'The table that holds all tenants known to the instance. In enterprise instances, this table will only contain the "default" tenant.';


--
-- Name: COLUMN tenants.id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.tenants.id IS 'The ID of the tenant. To keep tenants globally addressable, and be able to move them aronud instances more easily, the ID is NOT a serial and has to be specified explicitly. The creator of the tenant is responsible for choosing a unique ID, if it cares.';


--
-- Name: COLUMN tenants.name; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.tenants.name IS 'The name of the tenant. This may be displayed to the user and must be unique.';


--
-- Name: tracking_changeset_specs_and_changesets; Type: VIEW; Schema: public; Owner: -
--

CREATE VIEW public.tracking_changeset_specs_and_changesets AS
 SELECT changeset_specs.id AS changeset_spec_id,
    COALESCE(changesets.id, (0)::bigint) AS changeset_id,
    changeset_specs.repo_id,
    changeset_specs.batch_spec_id,
    repo.name AS repo_name,
    COALESCE((changesets.metadata ->> 'Title'::text), (changesets.metadata ->> 'title'::text)) AS changeset_name,
    changesets.external_state,
    changesets.publication_state,
    changesets.reconciler_state,
    changesets.computed_state
   FROM ((public.changeset_specs
     LEFT JOIN public.changesets ON (((changesets.repo_id = changeset_specs.repo_id) AND (changesets.external_id = changeset_specs.external_id))))
     JOIN public.repo ON ((changeset_specs.repo_id = repo.id)))
  WHERE ((changeset_specs.external_id IS NOT NULL) AND (repo.deleted_at IS NULL));


--
-- Name: user_credentials; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_credentials (
    id bigint NOT NULL,
    domain text NOT NULL,
    user_id integer NOT NULL,
    external_service_type text NOT NULL,
    external_service_id text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    credential bytea NOT NULL,
    ssh_migration_applied boolean DEFAULT false NOT NULL,
    encryption_key_id text DEFAULT ''::text NOT NULL,
    github_app_id integer,
    tenant_id integer,
    CONSTRAINT check_github_app_id_and_external_service_type_user_credentials CHECK (((github_app_id IS NULL) OR (external_service_type = 'github'::text)))
);


--
-- Name: user_credentials_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.user_credentials_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: user_credentials_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.user_credentials_id_seq OWNED BY public.user_credentials.id;


--
-- Name: user_emails; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_emails (
    user_id integer NOT NULL,
    email public.citext NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    verification_code text,
    verified_at timestamp with time zone,
    last_verification_sent_at timestamp with time zone,
    is_primary boolean DEFAULT false NOT NULL,
    tenant_id integer
);


--
-- Name: user_external_accounts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_external_accounts (
    id integer NOT NULL,
    user_id integer NOT NULL,
    service_type text NOT NULL,
    service_id text NOT NULL,
    account_id text NOT NULL,
    auth_data text,
    account_data text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    client_id text NOT NULL,
    expired_at timestamp with time zone,
    last_valid_at timestamp with time zone,
    encryption_key_id text DEFAULT ''::text NOT NULL,
    tenant_id integer
);


--
-- Name: user_external_accounts_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.user_external_accounts_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: user_external_accounts_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.user_external_accounts_id_seq OWNED BY public.user_external_accounts.id;


--
-- Name: user_onboarding_tour; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_onboarding_tour (
    id integer NOT NULL,
    raw_json text NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_by integer,
    tenant_id integer
);


--
-- Name: user_onboarding_tour_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.user_onboarding_tour_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: user_onboarding_tour_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.user_onboarding_tour_id_seq OWNED BY public.user_onboarding_tour.id;


--
-- Name: user_pending_permissions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_pending_permissions (
    id bigint NOT NULL,
    bind_id text NOT NULL,
    permission text NOT NULL,
    object_type text NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    service_type text NOT NULL,
    service_id text NOT NULL,
    object_ids_ints integer[] DEFAULT '{}'::integer[] NOT NULL,
    tenant_id integer
);


--
-- Name: user_pending_permissions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.user_pending_permissions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: user_pending_permissions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.user_pending_permissions_id_seq OWNED BY public.user_pending_permissions.id;


--
-- Name: user_permissions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_permissions (
    user_id integer NOT NULL,
    permission text NOT NULL,
    object_type text NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    synced_at timestamp with time zone,
    object_ids_ints integer[] DEFAULT '{}'::integer[] NOT NULL,
    migrated boolean DEFAULT true,
    tenant_id integer
);


--
-- Name: user_public_repos; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_public_repos (
    user_id integer NOT NULL,
    repo_uri text NOT NULL,
    repo_id integer NOT NULL,
    tenant_id integer
);


--
-- Name: user_repo_permissions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_repo_permissions (
    id bigint NOT NULL,
    user_id integer,
    repo_id integer NOT NULL,
    user_external_account_id integer,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    source text DEFAULT 'sync'::text NOT NULL,
    tenant_id integer
);


--
-- Name: user_repo_permissions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.user_repo_permissions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: user_repo_permissions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.user_repo_permissions_id_seq OWNED BY public.user_repo_permissions.id;


--
-- Name: user_roles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_roles (
    user_id integer NOT NULL,
    role_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    tenant_id integer
);


--
-- Name: users_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: users_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.users_id_seq OWNED BY public.users.id;


--
-- Name: versions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.versions (
    service text NOT NULL,
    version text NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    first_version text NOT NULL,
    auto_upgrade boolean DEFAULT false NOT NULL
);


--
-- Name: vulnerabilities; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.vulnerabilities (
    id integer NOT NULL,
    source_id text NOT NULL,
    summary text NOT NULL,
    details text NOT NULL,
    cpes text[] NOT NULL,
    cwes text[] NOT NULL,
    aliases text[] NOT NULL,
    related text[] NOT NULL,
    data_source text NOT NULL,
    urls text[] NOT NULL,
    severity text NOT NULL,
    cvss_vector text NOT NULL,
    cvss_score text NOT NULL,
    published_at timestamp with time zone NOT NULL,
    modified_at timestamp with time zone,
    withdrawn_at timestamp with time zone,
    tenant_id integer
);


--
-- Name: vulnerabilities_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.vulnerabilities_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: vulnerabilities_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.vulnerabilities_id_seq OWNED BY public.vulnerabilities.id;


--
-- Name: vulnerability_affected_packages; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.vulnerability_affected_packages (
    id integer NOT NULL,
    vulnerability_id integer NOT NULL,
    package_name text NOT NULL,
    language text NOT NULL,
    namespace text NOT NULL,
    version_constraint text[] NOT NULL,
    fixed boolean NOT NULL,
    fixed_in text,
    tenant_id integer
);


--
-- Name: vulnerability_affected_packages_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.vulnerability_affected_packages_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: vulnerability_affected_packages_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.vulnerability_affected_packages_id_seq OWNED BY public.vulnerability_affected_packages.id;


--
-- Name: vulnerability_affected_symbols; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.vulnerability_affected_symbols (
    id integer NOT NULL,
    vulnerability_affected_package_id integer NOT NULL,
    path text NOT NULL,
    symbols text[] NOT NULL,
    tenant_id integer
);


--
-- Name: vulnerability_affected_symbols_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.vulnerability_affected_symbols_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: vulnerability_affected_symbols_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.vulnerability_affected_symbols_id_seq OWNED BY public.vulnerability_affected_symbols.id;


--
-- Name: vulnerability_matches; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.vulnerability_matches (
    id integer NOT NULL,
    upload_id integer NOT NULL,
    vulnerability_affected_package_id integer NOT NULL,
    tenant_id integer
);


--
-- Name: vulnerability_matches_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.vulnerability_matches_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: vulnerability_matches_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.vulnerability_matches_id_seq OWNED BY public.vulnerability_matches.id;


--
-- Name: webhook_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.webhook_logs (
    id bigint NOT NULL,
    received_at timestamp with time zone DEFAULT now() NOT NULL,
    external_service_id integer,
    status_code integer NOT NULL,
    request bytea NOT NULL,
    response bytea NOT NULL,
    encryption_key_id text NOT NULL,
    webhook_id integer,
    tenant_id integer
);


--
-- Name: webhook_logs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.webhook_logs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: webhook_logs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.webhook_logs_id_seq OWNED BY public.webhook_logs.id;


--
-- Name: webhooks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.webhooks (
    id integer NOT NULL,
    code_host_kind text NOT NULL,
    code_host_urn text NOT NULL,
    secret text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    encryption_key_id text,
    uuid uuid DEFAULT gen_random_uuid() NOT NULL,
    created_by_user_id integer,
    updated_by_user_id integer,
    name text NOT NULL,
    tenant_id integer
);


--
-- Name: TABLE webhooks; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.webhooks IS 'Webhooks registered in Sourcegraph instance.';


--
-- Name: COLUMN webhooks.code_host_kind; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.webhooks.code_host_kind IS 'Kind of an external service for which webhooks are registered.';


--
-- Name: COLUMN webhooks.code_host_urn; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.webhooks.code_host_urn IS 'URN of a code host. This column maps to external_service_id column of repo table.';


--
-- Name: COLUMN webhooks.secret; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.webhooks.secret IS 'Secret used to decrypt webhook payload (if supported by the code host).';


--
-- Name: COLUMN webhooks.created_by_user_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.webhooks.created_by_user_id IS 'ID of a user, who created the webhook. If NULL, then the user does not exist (never existed or was deleted).';


--
-- Name: COLUMN webhooks.updated_by_user_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.webhooks.updated_by_user_id IS 'ID of a user, who updated the webhook. If NULL, then the user does not exist (never existed or was deleted).';


--
-- Name: COLUMN webhooks.name; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.webhooks.name IS 'Descriptive name of a webhook.';


--
-- Name: webhooks_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.webhooks_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: webhooks_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.webhooks_id_seq OWNED BY public.webhooks.id;


--
-- Name: zoekt_repos; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.zoekt_repos (
    repo_id integer NOT NULL,
    branches jsonb DEFAULT '[]'::jsonb NOT NULL,
    index_status text DEFAULT 'not_indexed'::text NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    last_indexed_at timestamp with time zone,
    tenant_id integer
);


--
-- Name: access_requests id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.access_requests ALTER COLUMN id SET DEFAULT nextval('public.access_requests_id_seq'::regclass);


--
-- Name: access_tokens id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.access_tokens ALTER COLUMN id SET DEFAULT nextval('public.access_tokens_id_seq'::regclass);


--
-- Name: assigned_owners id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.assigned_owners ALTER COLUMN id SET DEFAULT nextval('public.assigned_owners_id_seq'::regclass);


--
-- Name: assigned_teams id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.assigned_teams ALTER COLUMN id SET DEFAULT nextval('public.assigned_teams_id_seq'::regclass);


--
-- Name: batch_changes id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_changes ALTER COLUMN id SET DEFAULT nextval('public.batch_changes_id_seq'::regclass);


--
-- Name: batch_changes_site_credentials id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_changes_site_credentials ALTER COLUMN id SET DEFAULT nextval('public.batch_changes_site_credentials_id_seq'::regclass);


--
-- Name: batch_spec_execution_cache_entries id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_spec_execution_cache_entries ALTER COLUMN id SET DEFAULT nextval('public.batch_spec_execution_cache_entries_id_seq'::regclass);


--
-- Name: batch_spec_resolution_jobs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_spec_resolution_jobs ALTER COLUMN id SET DEFAULT nextval('public.batch_spec_resolution_jobs_id_seq'::regclass);


--
-- Name: batch_spec_workspace_execution_jobs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_spec_workspace_execution_jobs ALTER COLUMN id SET DEFAULT nextval('public.batch_spec_workspace_execution_jobs_id_seq'::regclass);


--
-- Name: batch_spec_workspace_files id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_spec_workspace_files ALTER COLUMN id SET DEFAULT nextval('public.batch_spec_workspace_files_id_seq'::regclass);


--
-- Name: batch_spec_workspaces id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_spec_workspaces ALTER COLUMN id SET DEFAULT nextval('public.batch_spec_workspaces_id_seq'::regclass);


--
-- Name: batch_specs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_specs ALTER COLUMN id SET DEFAULT nextval('public.batch_specs_id_seq'::regclass);


--
-- Name: cached_available_indexers id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cached_available_indexers ALTER COLUMN id SET DEFAULT nextval('public.cached_available_indexers_id_seq'::regclass);


--
-- Name: changeset_events id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changeset_events ALTER COLUMN id SET DEFAULT nextval('public.changeset_events_id_seq'::regclass);


--
-- Name: changeset_jobs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changeset_jobs ALTER COLUMN id SET DEFAULT nextval('public.changeset_jobs_id_seq'::regclass);


--
-- Name: changeset_specs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changeset_specs ALTER COLUMN id SET DEFAULT nextval('public.changeset_specs_id_seq'::regclass);


--
-- Name: changesets id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changesets ALTER COLUMN id SET DEFAULT nextval('public.changesets_id_seq'::regclass);


--
-- Name: cm_action_jobs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_action_jobs ALTER COLUMN id SET DEFAULT nextval('public.cm_action_jobs_id_seq'::regclass);


--
-- Name: cm_emails id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_emails ALTER COLUMN id SET DEFAULT nextval('public.cm_emails_id_seq'::regclass);


--
-- Name: cm_monitors id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_monitors ALTER COLUMN id SET DEFAULT nextval('public.cm_monitors_id_seq'::regclass);


--
-- Name: cm_queries id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_queries ALTER COLUMN id SET DEFAULT nextval('public.cm_queries_id_seq'::regclass);


--
-- Name: cm_recipients id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_recipients ALTER COLUMN id SET DEFAULT nextval('public.cm_recipients_id_seq'::regclass);


--
-- Name: cm_slack_webhooks id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_slack_webhooks ALTER COLUMN id SET DEFAULT nextval('public.cm_slack_webhooks_id_seq'::regclass);


--
-- Name: cm_trigger_jobs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_trigger_jobs ALTER COLUMN id SET DEFAULT nextval('public.cm_trigger_jobs_id_seq'::regclass);


--
-- Name: cm_webhooks id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_webhooks ALTER COLUMN id SET DEFAULT nextval('public.cm_webhooks_id_seq'::regclass);


--
-- Name: code_hosts id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.code_hosts ALTER COLUMN id SET DEFAULT nextval('public.code_hosts_id_seq'::regclass);


--
-- Name: codeintel_autoindex_queue id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_autoindex_queue ALTER COLUMN id SET DEFAULT nextval('public.codeintel_autoindex_queue_id_seq'::regclass);


--
-- Name: codeintel_autoindexing_exceptions id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_autoindexing_exceptions ALTER COLUMN id SET DEFAULT nextval('public.codeintel_autoindexing_exceptions_id_seq'::regclass);


--
-- Name: codeintel_initial_path_ranks id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_initial_path_ranks ALTER COLUMN id SET DEFAULT nextval('public.codeintel_initial_path_ranks_id_seq'::regclass);


--
-- Name: codeintel_initial_path_ranks_processed id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_initial_path_ranks_processed ALTER COLUMN id SET DEFAULT nextval('public.codeintel_initial_path_ranks_processed_id_seq'::regclass);


--
-- Name: codeintel_langugage_support_requests id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_langugage_support_requests ALTER COLUMN id SET DEFAULT nextval('public.codeintel_langugage_support_requests_id_seq'::regclass);


--
-- Name: codeintel_path_ranks id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_path_ranks ALTER COLUMN id SET DEFAULT nextval('public.codeintel_path_ranks_id_seq'::regclass);


--
-- Name: codeintel_ranking_definitions id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_ranking_definitions ALTER COLUMN id SET DEFAULT nextval('public.codeintel_ranking_definitions_id_seq'::regclass);


--
-- Name: codeintel_ranking_exports id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_ranking_exports ALTER COLUMN id SET DEFAULT nextval('public.codeintel_ranking_exports_id_seq'::regclass);


--
-- Name: codeintel_ranking_graph_keys id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_ranking_graph_keys ALTER COLUMN id SET DEFAULT nextval('public.codeintel_ranking_graph_keys_id_seq'::regclass);


--
-- Name: codeintel_ranking_path_counts_inputs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_ranking_path_counts_inputs ALTER COLUMN id SET DEFAULT nextval('public.codeintel_ranking_path_counts_inputs_id_seq'::regclass);


--
-- Name: codeintel_ranking_progress id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_ranking_progress ALTER COLUMN id SET DEFAULT nextval('public.codeintel_ranking_progress_id_seq'::regclass);


--
-- Name: codeintel_ranking_references id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_ranking_references ALTER COLUMN id SET DEFAULT nextval('public.codeintel_ranking_references_id_seq'::regclass);


--
-- Name: codeintel_ranking_references_processed id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_ranking_references_processed ALTER COLUMN id SET DEFAULT nextval('public.codeintel_ranking_references_processed_id_seq'::regclass);


--
-- Name: codeowners id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeowners ALTER COLUMN id SET DEFAULT nextval('public.codeowners_id_seq'::regclass);


--
-- Name: codeowners_owners id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeowners_owners ALTER COLUMN id SET DEFAULT nextval('public.codeowners_owners_id_seq'::regclass);


--
-- Name: commit_authors id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.commit_authors ALTER COLUMN id SET DEFAULT nextval('public.commit_authors_id_seq'::regclass);


--
-- Name: configuration_policies_audit_logs sequence; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.configuration_policies_audit_logs ALTER COLUMN sequence SET DEFAULT nextval('public.configuration_policies_audit_logs_seq'::regclass);


--
-- Name: context_detection_embedding_jobs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.context_detection_embedding_jobs ALTER COLUMN id SET DEFAULT nextval('public.context_detection_embedding_jobs_id_seq'::regclass);


--
-- Name: critical_and_site_config id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.critical_and_site_config ALTER COLUMN id SET DEFAULT nextval('public.critical_and_site_config_id_seq'::regclass);


--
-- Name: discussion_comments id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.discussion_comments ALTER COLUMN id SET DEFAULT nextval('public.discussion_comments_id_seq'::regclass);


--
-- Name: discussion_threads id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.discussion_threads ALTER COLUMN id SET DEFAULT nextval('public.discussion_threads_id_seq'::regclass);


--
-- Name: discussion_threads_target_repo id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.discussion_threads_target_repo ALTER COLUMN id SET DEFAULT nextval('public.discussion_threads_target_repo_id_seq'::regclass);


--
-- Name: event_logs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.event_logs ALTER COLUMN id SET DEFAULT nextval('public.event_logs_id_seq'::regclass);


--
-- Name: event_logs_export_allowlist id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.event_logs_export_allowlist ALTER COLUMN id SET DEFAULT nextval('public.event_logs_export_allowlist_id_seq'::regclass);


--
-- Name: event_logs_scrape_state id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.event_logs_scrape_state ALTER COLUMN id SET DEFAULT nextval('public.event_logs_scrape_state_id_seq'::regclass);


--
-- Name: event_logs_scrape_state_own id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.event_logs_scrape_state_own ALTER COLUMN id SET DEFAULT nextval('public.event_logs_scrape_state_own_id_seq'::regclass);


--
-- Name: executor_heartbeats id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.executor_heartbeats ALTER COLUMN id SET DEFAULT nextval('public.executor_heartbeats_id_seq'::regclass);


--
-- Name: executor_job_tokens id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.executor_job_tokens ALTER COLUMN id SET DEFAULT nextval('public.executor_job_tokens_id_seq'::regclass);


--
-- Name: executor_secret_access_logs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.executor_secret_access_logs ALTER COLUMN id SET DEFAULT nextval('public.executor_secret_access_logs_id_seq'::regclass);


--
-- Name: executor_secrets id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.executor_secrets ALTER COLUMN id SET DEFAULT nextval('public.executor_secrets_id_seq'::regclass);


--
-- Name: exhaustive_search_jobs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.exhaustive_search_jobs ALTER COLUMN id SET DEFAULT nextval('public.exhaustive_search_jobs_id_seq'::regclass);


--
-- Name: exhaustive_search_repo_jobs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.exhaustive_search_repo_jobs ALTER COLUMN id SET DEFAULT nextval('public.exhaustive_search_repo_jobs_id_seq'::regclass);


--
-- Name: exhaustive_search_repo_revision_jobs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.exhaustive_search_repo_revision_jobs ALTER COLUMN id SET DEFAULT nextval('public.exhaustive_search_repo_revision_jobs_id_seq'::regclass);


--
-- Name: explicit_permissions_bitbucket_projects_jobs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.explicit_permissions_bitbucket_projects_jobs ALTER COLUMN id SET DEFAULT nextval('public.explicit_permissions_bitbucket_projects_jobs_id_seq'::regclass);


--
-- Name: external_services id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.external_services ALTER COLUMN id SET DEFAULT nextval('public.external_services_id_seq'::regclass);


--
-- Name: github_app_installs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_app_installs ALTER COLUMN id SET DEFAULT nextval('public.github_app_installs_id_seq'::regclass);


--
-- Name: github_apps id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_apps ALTER COLUMN id SET DEFAULT nextval('public.github_apps_id_seq'::regclass);


--
-- Name: gitserver_relocator_jobs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.gitserver_relocator_jobs ALTER COLUMN id SET DEFAULT nextval('public.gitserver_relocator_jobs_id_seq'::regclass);


--
-- Name: insights_query_runner_jobs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.insights_query_runner_jobs ALTER COLUMN id SET DEFAULT nextval('public.insights_query_runner_jobs_id_seq'::regclass);


--
-- Name: insights_query_runner_jobs_dependencies id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.insights_query_runner_jobs_dependencies ALTER COLUMN id SET DEFAULT nextval('public.insights_query_runner_jobs_dependencies_id_seq'::regclass);


--
-- Name: insights_settings_migration_jobs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.insights_settings_migration_jobs ALTER COLUMN id SET DEFAULT nextval('public.insights_settings_migration_jobs_id_seq'::regclass);


--
-- Name: lsif_configuration_policies id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lsif_configuration_policies ALTER COLUMN id SET DEFAULT nextval('public.lsif_configuration_policies_id_seq'::regclass);


--
-- Name: lsif_dependency_indexing_jobs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lsif_dependency_indexing_jobs ALTER COLUMN id SET DEFAULT nextval('public.lsif_dependency_indexing_jobs_id_seq1'::regclass);


--
-- Name: lsif_dependency_repos id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lsif_dependency_repos ALTER COLUMN id SET DEFAULT nextval('public.lsif_dependency_repos_id_seq'::regclass);


--
-- Name: lsif_dependency_syncing_jobs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lsif_dependency_syncing_jobs ALTER COLUMN id SET DEFAULT nextval('public.lsif_dependency_indexing_jobs_id_seq'::regclass);


--
-- Name: lsif_index_configuration id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lsif_index_configuration ALTER COLUMN id SET DEFAULT nextval('public.lsif_index_configuration_id_seq'::regclass);


--
-- Name: lsif_indexes id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lsif_indexes ALTER COLUMN id SET DEFAULT nextval('public.lsif_indexes_id_seq'::regclass);


--
-- Name: lsif_packages id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lsif_packages ALTER COLUMN id SET DEFAULT nextval('public.lsif_packages_id_seq'::regclass);


--
-- Name: lsif_references id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lsif_references ALTER COLUMN id SET DEFAULT nextval('public.lsif_references_id_seq'::regclass);


--
-- Name: lsif_retention_configuration id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lsif_retention_configuration ALTER COLUMN id SET DEFAULT nextval('public.lsif_retention_configuration_id_seq'::regclass);


--
-- Name: lsif_uploads id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lsif_uploads ALTER COLUMN id SET DEFAULT nextval('public.lsif_dumps_id_seq'::regclass);


--
-- Name: lsif_uploads_audit_logs sequence; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lsif_uploads_audit_logs ALTER COLUMN sequence SET DEFAULT nextval('public.lsif_uploads_audit_logs_seq'::regclass);


--
-- Name: lsif_uploads_vulnerability_scan id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lsif_uploads_vulnerability_scan ALTER COLUMN id SET DEFAULT nextval('public.lsif_uploads_vulnerability_scan_id_seq'::regclass);


--
-- Name: namespace_permissions id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.namespace_permissions ALTER COLUMN id SET DEFAULT nextval('public.namespace_permissions_id_seq'::regclass);


--
-- Name: notebooks id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notebooks ALTER COLUMN id SET DEFAULT nextval('public.notebooks_id_seq'::regclass);


--
-- Name: org_invitations id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.org_invitations ALTER COLUMN id SET DEFAULT nextval('public.org_invitations_id_seq'::regclass);


--
-- Name: org_members id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.org_members ALTER COLUMN id SET DEFAULT nextval('public.org_members_id_seq'::regclass);


--
-- Name: orgs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.orgs ALTER COLUMN id SET DEFAULT nextval('public.orgs_id_seq'::regclass);


--
-- Name: out_of_band_migrations id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.out_of_band_migrations ALTER COLUMN id SET DEFAULT nextval('public.out_of_band_migrations_id_seq'::regclass);


--
-- Name: out_of_band_migrations_errors id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.out_of_band_migrations_errors ALTER COLUMN id SET DEFAULT nextval('public.out_of_band_migrations_errors_id_seq'::regclass);


--
-- Name: outbound_webhook_event_types id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.outbound_webhook_event_types ALTER COLUMN id SET DEFAULT nextval('public.outbound_webhook_event_types_id_seq'::regclass);


--
-- Name: outbound_webhook_jobs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.outbound_webhook_jobs ALTER COLUMN id SET DEFAULT nextval('public.outbound_webhook_jobs_id_seq'::regclass);


--
-- Name: outbound_webhook_logs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.outbound_webhook_logs ALTER COLUMN id SET DEFAULT nextval('public.outbound_webhook_logs_id_seq'::regclass);


--
-- Name: outbound_webhooks id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.outbound_webhooks ALTER COLUMN id SET DEFAULT nextval('public.outbound_webhooks_id_seq'::regclass);


--
-- Name: own_aggregate_recent_contribution id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.own_aggregate_recent_contribution ALTER COLUMN id SET DEFAULT nextval('public.own_aggregate_recent_contribution_id_seq'::regclass);


--
-- Name: own_aggregate_recent_view id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.own_aggregate_recent_view ALTER COLUMN id SET DEFAULT nextval('public.own_aggregate_recent_view_id_seq'::regclass);


--
-- Name: own_background_jobs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.own_background_jobs ALTER COLUMN id SET DEFAULT nextval('public.own_background_jobs_id_seq'::regclass);


--
-- Name: own_signal_configurations id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.own_signal_configurations ALTER COLUMN id SET DEFAULT nextval('public.own_signal_configurations_id_seq'::regclass);


--
-- Name: own_signal_recent_contribution id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.own_signal_recent_contribution ALTER COLUMN id SET DEFAULT nextval('public.own_signal_recent_contribution_id_seq'::regclass);


--
-- Name: package_repo_filters id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.package_repo_filters ALTER COLUMN id SET DEFAULT nextval('public.package_repo_filters_id_seq'::regclass);


--
-- Name: package_repo_versions id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.package_repo_versions ALTER COLUMN id SET DEFAULT nextval('public.package_repo_versions_id_seq'::regclass);


--
-- Name: permission_sync_jobs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.permission_sync_jobs ALTER COLUMN id SET DEFAULT nextval('public.permission_sync_jobs_id_seq'::regclass);


--
-- Name: permissions id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.permissions ALTER COLUMN id SET DEFAULT nextval('public.permissions_id_seq'::regclass);


--
-- Name: phabricator_repos id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.phabricator_repos ALTER COLUMN id SET DEFAULT nextval('public.phabricator_repos_id_seq'::regclass);


--
-- Name: prompts id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.prompts ALTER COLUMN id SET DEFAULT nextval('public.prompts_id_seq'::regclass);


--
-- Name: registry_extension_releases id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.registry_extension_releases ALTER COLUMN id SET DEFAULT nextval('public.registry_extension_releases_id_seq'::regclass);


--
-- Name: registry_extensions id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.registry_extensions ALTER COLUMN id SET DEFAULT nextval('public.registry_extensions_id_seq'::regclass);


--
-- Name: repo id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.repo ALTER COLUMN id SET DEFAULT nextval('public.repo_id_seq'::regclass);


--
-- Name: repo_commits_changelists id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.repo_commits_changelists ALTER COLUMN id SET DEFAULT nextval('public.repo_commits_changelists_id_seq'::regclass);


--
-- Name: repo_embedding_jobs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.repo_embedding_jobs ALTER COLUMN id SET DEFAULT nextval('public.repo_embedding_jobs_id_seq'::regclass);


--
-- Name: repo_paths id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.repo_paths ALTER COLUMN id SET DEFAULT nextval('public.repo_paths_id_seq'::regclass);


--
-- Name: roles id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.roles ALTER COLUMN id SET DEFAULT nextval('public.roles_id_seq'::regclass);


--
-- Name: saved_searches id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.saved_searches ALTER COLUMN id SET DEFAULT nextval('public.saved_searches_id_seq'::regclass);


--
-- Name: search_contexts id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.search_contexts ALTER COLUMN id SET DEFAULT nextval('public.search_contexts_id_seq'::regclass);


--
-- Name: security_event_logs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.security_event_logs ALTER COLUMN id SET DEFAULT nextval('public.security_event_logs_id_seq'::regclass);


--
-- Name: settings id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.settings ALTER COLUMN id SET DEFAULT nextval('public.settings_id_seq'::regclass);


--
-- Name: survey_responses id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.survey_responses ALTER COLUMN id SET DEFAULT nextval('public.survey_responses_id_seq'::regclass);


--
-- Name: syntactic_scip_indexing_jobs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.syntactic_scip_indexing_jobs ALTER COLUMN id SET DEFAULT nextval('public.syntactic_scip_indexing_jobs_id_seq'::regclass);


--
-- Name: teams id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.teams ALTER COLUMN id SET DEFAULT nextval('public.teams_id_seq'::regclass);


--
-- Name: temporary_settings id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.temporary_settings ALTER COLUMN id SET DEFAULT nextval('public.temporary_settings_id_seq'::regclass);


--
-- Name: user_credentials id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_credentials ALTER COLUMN id SET DEFAULT nextval('public.user_credentials_id_seq'::regclass);


--
-- Name: user_external_accounts id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_external_accounts ALTER COLUMN id SET DEFAULT nextval('public.user_external_accounts_id_seq'::regclass);


--
-- Name: user_onboarding_tour id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_onboarding_tour ALTER COLUMN id SET DEFAULT nextval('public.user_onboarding_tour_id_seq'::regclass);


--
-- Name: user_pending_permissions id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pending_permissions ALTER COLUMN id SET DEFAULT nextval('public.user_pending_permissions_id_seq'::regclass);


--
-- Name: user_repo_permissions id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_repo_permissions ALTER COLUMN id SET DEFAULT nextval('public.user_repo_permissions_id_seq'::regclass);


--
-- Name: users id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users ALTER COLUMN id SET DEFAULT nextval('public.users_id_seq'::regclass);


--
-- Name: vulnerabilities id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vulnerabilities ALTER COLUMN id SET DEFAULT nextval('public.vulnerabilities_id_seq'::regclass);


--
-- Name: vulnerability_affected_packages id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vulnerability_affected_packages ALTER COLUMN id SET DEFAULT nextval('public.vulnerability_affected_packages_id_seq'::regclass);


--
-- Name: vulnerability_affected_symbols id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vulnerability_affected_symbols ALTER COLUMN id SET DEFAULT nextval('public.vulnerability_affected_symbols_id_seq'::regclass);


--
-- Name: vulnerability_matches id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vulnerability_matches ALTER COLUMN id SET DEFAULT nextval('public.vulnerability_matches_id_seq'::regclass);


--
-- Name: webhook_logs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhook_logs ALTER COLUMN id SET DEFAULT nextval('public.webhook_logs_id_seq'::regclass);


--
-- Name: webhooks id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhooks ALTER COLUMN id SET DEFAULT nextval('public.webhooks_id_seq'::regclass);


--
-- Name: access_requests access_requests_email_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.access_requests
    ADD CONSTRAINT access_requests_email_key UNIQUE (email);


--
-- Name: access_requests access_requests_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.access_requests
    ADD CONSTRAINT access_requests_pkey PRIMARY KEY (id);


--
-- Name: access_tokens access_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.access_tokens
    ADD CONSTRAINT access_tokens_pkey PRIMARY KEY (id);


--
-- Name: access_tokens access_tokens_value_sha256_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.access_tokens
    ADD CONSTRAINT access_tokens_value_sha256_key UNIQUE (value_sha256);


--
-- Name: aggregated_user_statistics aggregated_user_statistics_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.aggregated_user_statistics
    ADD CONSTRAINT aggregated_user_statistics_pkey PRIMARY KEY (user_id);


--
-- Name: assigned_owners assigned_owners_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.assigned_owners
    ADD CONSTRAINT assigned_owners_pkey PRIMARY KEY (id);


--
-- Name: assigned_teams assigned_teams_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.assigned_teams
    ADD CONSTRAINT assigned_teams_pkey PRIMARY KEY (id);


--
-- Name: batch_changes batch_changes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_changes
    ADD CONSTRAINT batch_changes_pkey PRIMARY KEY (id);


--
-- Name: batch_changes_site_credentials batch_changes_site_credentials_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_changes_site_credentials
    ADD CONSTRAINT batch_changes_site_credentials_pkey PRIMARY KEY (id);


--
-- Name: batch_spec_execution_cache_entries batch_spec_execution_cache_entries_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_spec_execution_cache_entries
    ADD CONSTRAINT batch_spec_execution_cache_entries_pkey PRIMARY KEY (id);


--
-- Name: batch_spec_execution_cache_entries batch_spec_execution_cache_entries_user_id_key_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_spec_execution_cache_entries
    ADD CONSTRAINT batch_spec_execution_cache_entries_user_id_key_unique UNIQUE (user_id, key);


--
-- Name: batch_spec_resolution_jobs batch_spec_resolution_jobs_batch_spec_id_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_spec_resolution_jobs
    ADD CONSTRAINT batch_spec_resolution_jobs_batch_spec_id_unique UNIQUE (batch_spec_id);


--
-- Name: batch_spec_resolution_jobs batch_spec_resolution_jobs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_spec_resolution_jobs
    ADD CONSTRAINT batch_spec_resolution_jobs_pkey PRIMARY KEY (id);


--
-- Name: batch_spec_workspace_execution_jobs batch_spec_workspace_execution_jobs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_spec_workspace_execution_jobs
    ADD CONSTRAINT batch_spec_workspace_execution_jobs_pkey PRIMARY KEY (id);


--
-- Name: batch_spec_workspace_execution_last_dequeues batch_spec_workspace_execution_last_dequeues_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_spec_workspace_execution_last_dequeues
    ADD CONSTRAINT batch_spec_workspace_execution_last_dequeues_pkey PRIMARY KEY (user_id);


--
-- Name: batch_spec_workspace_files batch_spec_workspace_files_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_spec_workspace_files
    ADD CONSTRAINT batch_spec_workspace_files_pkey PRIMARY KEY (id);


--
-- Name: batch_spec_workspaces batch_spec_workspaces_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_spec_workspaces
    ADD CONSTRAINT batch_spec_workspaces_pkey PRIMARY KEY (id);


--
-- Name: batch_specs batch_specs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_specs
    ADD CONSTRAINT batch_specs_pkey PRIMARY KEY (id);


--
-- Name: cached_available_indexers cached_available_indexers_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cached_available_indexers
    ADD CONSTRAINT cached_available_indexers_pkey PRIMARY KEY (id);


--
-- Name: changeset_events changeset_events_changeset_id_kind_key_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changeset_events
    ADD CONSTRAINT changeset_events_changeset_id_kind_key_unique UNIQUE (changeset_id, kind, key);


--
-- Name: changeset_events changeset_events_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changeset_events
    ADD CONSTRAINT changeset_events_pkey PRIMARY KEY (id);


--
-- Name: changeset_jobs changeset_jobs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changeset_jobs
    ADD CONSTRAINT changeset_jobs_pkey PRIMARY KEY (id);


--
-- Name: changeset_specs changeset_specs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changeset_specs
    ADD CONSTRAINT changeset_specs_pkey PRIMARY KEY (id);


--
-- Name: changesets changesets_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changesets
    ADD CONSTRAINT changesets_pkey PRIMARY KEY (id);


--
-- Name: changesets changesets_repo_external_id_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changesets
    ADD CONSTRAINT changesets_repo_external_id_unique UNIQUE (repo_id, external_id);


--
-- Name: cm_action_jobs cm_action_jobs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_action_jobs
    ADD CONSTRAINT cm_action_jobs_pkey PRIMARY KEY (id);


--
-- Name: cm_emails cm_emails_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_emails
    ADD CONSTRAINT cm_emails_pkey PRIMARY KEY (id);


--
-- Name: cm_last_searched cm_last_searched_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_last_searched
    ADD CONSTRAINT cm_last_searched_pkey PRIMARY KEY (monitor_id, repo_id);


--
-- Name: cm_monitors cm_monitors_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_monitors
    ADD CONSTRAINT cm_monitors_pkey PRIMARY KEY (id);


--
-- Name: cm_queries cm_queries_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_queries
    ADD CONSTRAINT cm_queries_pkey PRIMARY KEY (id);


--
-- Name: cm_recipients cm_recipients_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_recipients
    ADD CONSTRAINT cm_recipients_pkey PRIMARY KEY (id);


--
-- Name: cm_slack_webhooks cm_slack_webhooks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_slack_webhooks
    ADD CONSTRAINT cm_slack_webhooks_pkey PRIMARY KEY (id);


--
-- Name: cm_trigger_jobs cm_trigger_jobs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_trigger_jobs
    ADD CONSTRAINT cm_trigger_jobs_pkey PRIMARY KEY (id);


--
-- Name: cm_webhooks cm_webhooks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_webhooks
    ADD CONSTRAINT cm_webhooks_pkey PRIMARY KEY (id);


--
-- Name: code_hosts code_hosts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.code_hosts
    ADD CONSTRAINT code_hosts_pkey PRIMARY KEY (id);


--
-- Name: code_hosts code_hosts_url_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.code_hosts
    ADD CONSTRAINT code_hosts_url_key UNIQUE (url);


--
-- Name: codeintel_autoindex_queue codeintel_autoindex_queue_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_autoindex_queue
    ADD CONSTRAINT codeintel_autoindex_queue_pkey PRIMARY KEY (id);


--
-- Name: codeintel_autoindexing_exceptions codeintel_autoindexing_exceptions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_autoindexing_exceptions
    ADD CONSTRAINT codeintel_autoindexing_exceptions_pkey PRIMARY KEY (id);


--
-- Name: codeintel_autoindexing_exceptions codeintel_autoindexing_exceptions_repository_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_autoindexing_exceptions
    ADD CONSTRAINT codeintel_autoindexing_exceptions_repository_id_key UNIQUE (repository_id);


--
-- Name: codeintel_commit_dates codeintel_commit_dates_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_commit_dates
    ADD CONSTRAINT codeintel_commit_dates_pkey PRIMARY KEY (repository_id, commit_bytea);


--
-- Name: codeintel_initial_path_ranks codeintel_initial_path_ranks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_initial_path_ranks
    ADD CONSTRAINT codeintel_initial_path_ranks_pkey PRIMARY KEY (id);


--
-- Name: codeintel_initial_path_ranks_processed codeintel_initial_path_ranks_processed_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_initial_path_ranks_processed
    ADD CONSTRAINT codeintel_initial_path_ranks_processed_pkey PRIMARY KEY (id);


--
-- Name: codeintel_path_ranks codeintel_path_ranks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_path_ranks
    ADD CONSTRAINT codeintel_path_ranks_pkey PRIMARY KEY (id);


--
-- Name: codeintel_ranking_definitions codeintel_ranking_definitions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_ranking_definitions
    ADD CONSTRAINT codeintel_ranking_definitions_pkey PRIMARY KEY (id);


--
-- Name: codeintel_ranking_exports codeintel_ranking_exports_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_ranking_exports
    ADD CONSTRAINT codeintel_ranking_exports_pkey PRIMARY KEY (id);


--
-- Name: codeintel_ranking_graph_keys codeintel_ranking_graph_keys_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_ranking_graph_keys
    ADD CONSTRAINT codeintel_ranking_graph_keys_pkey PRIMARY KEY (id);


--
-- Name: codeintel_ranking_path_counts_inputs codeintel_ranking_path_counts_inputs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_ranking_path_counts_inputs
    ADD CONSTRAINT codeintel_ranking_path_counts_inputs_pkey PRIMARY KEY (id);


--
-- Name: codeintel_ranking_progress codeintel_ranking_progress_graph_key_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_ranking_progress
    ADD CONSTRAINT codeintel_ranking_progress_graph_key_key UNIQUE (graph_key);


--
-- Name: codeintel_ranking_progress codeintel_ranking_progress_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_ranking_progress
    ADD CONSTRAINT codeintel_ranking_progress_pkey PRIMARY KEY (id);


--
-- Name: codeintel_ranking_references codeintel_ranking_references_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_ranking_references
    ADD CONSTRAINT codeintel_ranking_references_pkey PRIMARY KEY (id);


--
-- Name: codeintel_ranking_references_processed codeintel_ranking_references_processed_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_ranking_references_processed
    ADD CONSTRAINT codeintel_ranking_references_processed_pkey PRIMARY KEY (id);


--
-- Name: codeowners_individual_stats codeowners_individual_stats_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeowners_individual_stats
    ADD CONSTRAINT codeowners_individual_stats_pkey PRIMARY KEY (file_path_id, owner_id);


--
-- Name: codeowners_owners codeowners_owners_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeowners_owners
    ADD CONSTRAINT codeowners_owners_pkey PRIMARY KEY (id);


--
-- Name: codeowners codeowners_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeowners
    ADD CONSTRAINT codeowners_pkey PRIMARY KEY (id);


--
-- Name: codeowners codeowners_repo_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeowners
    ADD CONSTRAINT codeowners_repo_id_key UNIQUE (repo_id);


--
-- Name: commit_authors commit_authors_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.commit_authors
    ADD CONSTRAINT commit_authors_pkey PRIMARY KEY (id);


--
-- Name: context_detection_embedding_jobs context_detection_embedding_jobs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.context_detection_embedding_jobs
    ADD CONSTRAINT context_detection_embedding_jobs_pkey PRIMARY KEY (id);


--
-- Name: critical_and_site_config critical_and_site_config_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.critical_and_site_config
    ADD CONSTRAINT critical_and_site_config_pkey PRIMARY KEY (id);


--
-- Name: discussion_comments discussion_comments_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.discussion_comments
    ADD CONSTRAINT discussion_comments_pkey PRIMARY KEY (id);


--
-- Name: discussion_mail_reply_tokens discussion_mail_reply_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.discussion_mail_reply_tokens
    ADD CONSTRAINT discussion_mail_reply_tokens_pkey PRIMARY KEY (token);


--
-- Name: discussion_threads discussion_threads_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.discussion_threads
    ADD CONSTRAINT discussion_threads_pkey PRIMARY KEY (id);


--
-- Name: discussion_threads_target_repo discussion_threads_target_repo_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.discussion_threads_target_repo
    ADD CONSTRAINT discussion_threads_target_repo_pkey PRIMARY KEY (id);


--
-- Name: event_logs_export_allowlist event_logs_export_allowlist_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.event_logs_export_allowlist
    ADD CONSTRAINT event_logs_export_allowlist_pkey PRIMARY KEY (id);


--
-- Name: event_logs event_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.event_logs
    ADD CONSTRAINT event_logs_pkey PRIMARY KEY (id);


--
-- Name: event_logs_scrape_state_own event_logs_scrape_state_own_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.event_logs_scrape_state_own
    ADD CONSTRAINT event_logs_scrape_state_own_pk PRIMARY KEY (id);


--
-- Name: event_logs_scrape_state event_logs_scrape_state_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.event_logs_scrape_state
    ADD CONSTRAINT event_logs_scrape_state_pk PRIMARY KEY (id);


--
-- Name: executor_heartbeats executor_heartbeats_hostname_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.executor_heartbeats
    ADD CONSTRAINT executor_heartbeats_hostname_key UNIQUE (hostname);


--
-- Name: executor_heartbeats executor_heartbeats_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.executor_heartbeats
    ADD CONSTRAINT executor_heartbeats_pkey PRIMARY KEY (id);


--
-- Name: executor_job_tokens executor_job_tokens_job_id_queue_repo_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.executor_job_tokens
    ADD CONSTRAINT executor_job_tokens_job_id_queue_repo_id_key UNIQUE (job_id, queue, repo_id);


--
-- Name: executor_job_tokens executor_job_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.executor_job_tokens
    ADD CONSTRAINT executor_job_tokens_pkey PRIMARY KEY (id);


--
-- Name: executor_job_tokens executor_job_tokens_value_sha256_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.executor_job_tokens
    ADD CONSTRAINT executor_job_tokens_value_sha256_key UNIQUE (value_sha256);


--
-- Name: executor_secret_access_logs executor_secret_access_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.executor_secret_access_logs
    ADD CONSTRAINT executor_secret_access_logs_pkey PRIMARY KEY (id);


--
-- Name: executor_secrets executor_secrets_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.executor_secrets
    ADD CONSTRAINT executor_secrets_pkey PRIMARY KEY (id);


--
-- Name: exhaustive_search_jobs exhaustive_search_jobs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.exhaustive_search_jobs
    ADD CONSTRAINT exhaustive_search_jobs_pkey PRIMARY KEY (id);


--
-- Name: exhaustive_search_repo_jobs exhaustive_search_repo_jobs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.exhaustive_search_repo_jobs
    ADD CONSTRAINT exhaustive_search_repo_jobs_pkey PRIMARY KEY (id);


--
-- Name: exhaustive_search_repo_revision_jobs exhaustive_search_repo_revision_jobs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.exhaustive_search_repo_revision_jobs
    ADD CONSTRAINT exhaustive_search_repo_revision_jobs_pkey PRIMARY KEY (id);


--
-- Name: explicit_permissions_bitbucket_projects_jobs explicit_permissions_bitbucket_projects_jobs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.explicit_permissions_bitbucket_projects_jobs
    ADD CONSTRAINT explicit_permissions_bitbucket_projects_jobs_pkey PRIMARY KEY (id);


--
-- Name: external_service_repos external_service_repos_repo_id_external_service_id_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.external_service_repos
    ADD CONSTRAINT external_service_repos_repo_id_external_service_id_unique UNIQUE (repo_id, external_service_id);


--
-- Name: external_services external_services_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.external_services
    ADD CONSTRAINT external_services_pkey PRIMARY KEY (id);


--
-- Name: feature_flag_overrides feature_flag_overrides_unique_org_flag; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feature_flag_overrides
    ADD CONSTRAINT feature_flag_overrides_unique_org_flag UNIQUE (namespace_org_id, flag_name);


--
-- Name: feature_flag_overrides feature_flag_overrides_unique_user_flag; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feature_flag_overrides
    ADD CONSTRAINT feature_flag_overrides_unique_user_flag UNIQUE (namespace_user_id, flag_name);


--
-- Name: feature_flags feature_flags_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feature_flags
    ADD CONSTRAINT feature_flags_pkey PRIMARY KEY (flag_name);


--
-- Name: github_app_installs github_app_installs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_app_installs
    ADD CONSTRAINT github_app_installs_pkey PRIMARY KEY (id);


--
-- Name: github_apps github_apps_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_apps
    ADD CONSTRAINT github_apps_pkey PRIMARY KEY (id);


--
-- Name: gitserver_relocator_jobs gitserver_relocator_jobs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.gitserver_relocator_jobs
    ADD CONSTRAINT gitserver_relocator_jobs_pkey PRIMARY KEY (id);


--
-- Name: gitserver_repos gitserver_repos_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.gitserver_repos
    ADD CONSTRAINT gitserver_repos_pkey PRIMARY KEY (repo_id);


--
-- Name: gitserver_repos_sync_output gitserver_repos_sync_output_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.gitserver_repos_sync_output
    ADD CONSTRAINT gitserver_repos_sync_output_pkey PRIMARY KEY (repo_id);


--
-- Name: global_state global_state_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.global_state
    ADD CONSTRAINT global_state_pkey PRIMARY KEY (site_id);


--
-- Name: insights_query_runner_jobs_dependencies insights_query_runner_jobs_dependencies_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.insights_query_runner_jobs_dependencies
    ADD CONSTRAINT insights_query_runner_jobs_dependencies_pkey PRIMARY KEY (id);


--
-- Name: insights_query_runner_jobs insights_query_runner_jobs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.insights_query_runner_jobs
    ADD CONSTRAINT insights_query_runner_jobs_pkey PRIMARY KEY (id);


--
-- Name: lsif_configuration_policies lsif_configuration_policies_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lsif_configuration_policies
    ADD CONSTRAINT lsif_configuration_policies_pkey PRIMARY KEY (id);


--
-- Name: lsif_configuration_policies_repository_pattern_lookup lsif_configuration_policies_repository_pattern_lookup_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lsif_configuration_policies_repository_pattern_lookup
    ADD CONSTRAINT lsif_configuration_policies_repository_pattern_lookup_pkey PRIMARY KEY (policy_id, repo_id);


--
-- Name: lsif_dependency_syncing_jobs lsif_dependency_indexing_jobs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lsif_dependency_syncing_jobs
    ADD CONSTRAINT lsif_dependency_indexing_jobs_pkey PRIMARY KEY (id);


--
-- Name: lsif_dependency_indexing_jobs lsif_dependency_indexing_jobs_pkey1; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lsif_dependency_indexing_jobs
    ADD CONSTRAINT lsif_dependency_indexing_jobs_pkey1 PRIMARY KEY (id);


--
-- Name: lsif_dependency_repos lsif_dependency_repos_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lsif_dependency_repos
    ADD CONSTRAINT lsif_dependency_repos_pkey PRIMARY KEY (id);


--
-- Name: lsif_dirty_repositories lsif_dirty_repositories_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lsif_dirty_repositories
    ADD CONSTRAINT lsif_dirty_repositories_pkey PRIMARY KEY (repository_id);


--
-- Name: lsif_index_configuration lsif_index_configuration_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lsif_index_configuration
    ADD CONSTRAINT lsif_index_configuration_pkey PRIMARY KEY (id);


--
-- Name: lsif_index_configuration lsif_index_configuration_repository_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lsif_index_configuration
    ADD CONSTRAINT lsif_index_configuration_repository_id_key UNIQUE (repository_id);


--
-- Name: lsif_indexes lsif_indexes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lsif_indexes
    ADD CONSTRAINT lsif_indexes_pkey PRIMARY KEY (id);


--
-- Name: lsif_last_index_scan lsif_last_index_scan_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lsif_last_index_scan
    ADD CONSTRAINT lsif_last_index_scan_pkey PRIMARY KEY (repository_id);


--
-- Name: lsif_last_retention_scan lsif_last_retention_scan_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lsif_last_retention_scan
    ADD CONSTRAINT lsif_last_retention_scan_pkey PRIMARY KEY (repository_id);


--
-- Name: lsif_packages lsif_packages_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lsif_packages
    ADD CONSTRAINT lsif_packages_pkey PRIMARY KEY (id);


--
-- Name: lsif_references lsif_references_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lsif_references
    ADD CONSTRAINT lsif_references_pkey PRIMARY KEY (id);


--
-- Name: lsif_retention_configuration lsif_retention_configuration_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lsif_retention_configuration
    ADD CONSTRAINT lsif_retention_configuration_pkey PRIMARY KEY (id);


--
-- Name: lsif_retention_configuration lsif_retention_configuration_repository_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lsif_retention_configuration
    ADD CONSTRAINT lsif_retention_configuration_repository_id_key UNIQUE (repository_id);


--
-- Name: lsif_uploads lsif_uploads_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lsif_uploads
    ADD CONSTRAINT lsif_uploads_pkey PRIMARY KEY (id);


--
-- Name: lsif_uploads_reference_counts lsif_uploads_reference_counts_upload_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lsif_uploads_reference_counts
    ADD CONSTRAINT lsif_uploads_reference_counts_upload_id_key UNIQUE (upload_id);


--
-- Name: lsif_uploads_vulnerability_scan lsif_uploads_vulnerability_scan_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lsif_uploads_vulnerability_scan
    ADD CONSTRAINT lsif_uploads_vulnerability_scan_pkey PRIMARY KEY (id);


--
-- Name: names names_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.names
    ADD CONSTRAINT names_pkey PRIMARY KEY (name);


--
-- Name: namespace_permissions namespace_permissions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.namespace_permissions
    ADD CONSTRAINT namespace_permissions_pkey PRIMARY KEY (id);


--
-- Name: notebook_stars notebook_stars_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notebook_stars
    ADD CONSTRAINT notebook_stars_pkey PRIMARY KEY (notebook_id, user_id);


--
-- Name: notebooks notebooks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notebooks
    ADD CONSTRAINT notebooks_pkey PRIMARY KEY (id);


--
-- Name: org_invitations org_invitations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.org_invitations
    ADD CONSTRAINT org_invitations_pkey PRIMARY KEY (id);


--
-- Name: org_members org_members_org_id_user_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.org_members
    ADD CONSTRAINT org_members_org_id_user_id_key UNIQUE (org_id, user_id);


--
-- Name: org_members org_members_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.org_members
    ADD CONSTRAINT org_members_pkey PRIMARY KEY (id);


--
-- Name: org_stats org_stats_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.org_stats
    ADD CONSTRAINT org_stats_pkey PRIMARY KEY (org_id);


--
-- Name: orgs_open_beta_stats orgs_open_beta_stats_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.orgs_open_beta_stats
    ADD CONSTRAINT orgs_open_beta_stats_pkey PRIMARY KEY (id);


--
-- Name: orgs orgs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.orgs
    ADD CONSTRAINT orgs_pkey PRIMARY KEY (id);


--
-- Name: out_of_band_migrations_errors out_of_band_migrations_errors_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.out_of_band_migrations_errors
    ADD CONSTRAINT out_of_band_migrations_errors_pkey PRIMARY KEY (id);


--
-- Name: out_of_band_migrations out_of_band_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.out_of_band_migrations
    ADD CONSTRAINT out_of_band_migrations_pkey PRIMARY KEY (id);


--
-- Name: outbound_webhook_event_types outbound_webhook_event_types_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.outbound_webhook_event_types
    ADD CONSTRAINT outbound_webhook_event_types_pkey PRIMARY KEY (id);


--
-- Name: outbound_webhook_jobs outbound_webhook_jobs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.outbound_webhook_jobs
    ADD CONSTRAINT outbound_webhook_jobs_pkey PRIMARY KEY (id);


--
-- Name: outbound_webhook_logs outbound_webhook_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.outbound_webhook_logs
    ADD CONSTRAINT outbound_webhook_logs_pkey PRIMARY KEY (id);


--
-- Name: outbound_webhooks outbound_webhooks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.outbound_webhooks
    ADD CONSTRAINT outbound_webhooks_pkey PRIMARY KEY (id);


--
-- Name: own_aggregate_recent_contribution own_aggregate_recent_contribution_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.own_aggregate_recent_contribution
    ADD CONSTRAINT own_aggregate_recent_contribution_pkey PRIMARY KEY (id);


--
-- Name: own_aggregate_recent_view own_aggregate_recent_view_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.own_aggregate_recent_view
    ADD CONSTRAINT own_aggregate_recent_view_pkey PRIMARY KEY (id);


--
-- Name: own_background_jobs own_background_jobs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.own_background_jobs
    ADD CONSTRAINT own_background_jobs_pkey PRIMARY KEY (id);


--
-- Name: own_signal_configurations own_signal_configurations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.own_signal_configurations
    ADD CONSTRAINT own_signal_configurations_pkey PRIMARY KEY (id);


--
-- Name: own_signal_recent_contribution own_signal_recent_contribution_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.own_signal_recent_contribution
    ADD CONSTRAINT own_signal_recent_contribution_pkey PRIMARY KEY (id);


--
-- Name: ownership_path_stats ownership_path_stats_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ownership_path_stats
    ADD CONSTRAINT ownership_path_stats_pkey PRIMARY KEY (file_path_id);


--
-- Name: package_repo_filters package_repo_filters_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.package_repo_filters
    ADD CONSTRAINT package_repo_filters_pkey PRIMARY KEY (id);


--
-- Name: package_repo_versions package_repo_versions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.package_repo_versions
    ADD CONSTRAINT package_repo_versions_pkey PRIMARY KEY (id);


--
-- Name: permission_sync_jobs permission_sync_jobs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.permission_sync_jobs
    ADD CONSTRAINT permission_sync_jobs_pkey PRIMARY KEY (id);


--
-- Name: permissions permissions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.permissions
    ADD CONSTRAINT permissions_pkey PRIMARY KEY (id);


--
-- Name: phabricator_repos phabricator_repos_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.phabricator_repos
    ADD CONSTRAINT phabricator_repos_pkey PRIMARY KEY (id);


--
-- Name: phabricator_repos phabricator_repos_repo_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.phabricator_repos
    ADD CONSTRAINT phabricator_repos_repo_name_key UNIQUE (repo_name);


--
-- Name: product_licenses product_licenses_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.product_licenses
    ADD CONSTRAINT product_licenses_pkey PRIMARY KEY (id);


--
-- Name: product_subscriptions product_subscriptions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.product_subscriptions
    ADD CONSTRAINT product_subscriptions_pkey PRIMARY KEY (id);


--
-- Name: prompts prompts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.prompts
    ADD CONSTRAINT prompts_pkey PRIMARY KEY (id);


--
-- Name: redis_key_value redis_key_value_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.redis_key_value
    ADD CONSTRAINT redis_key_value_pkey PRIMARY KEY (namespace, key) INCLUDE (value);


--
-- Name: registry_extension_releases registry_extension_releases_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.registry_extension_releases
    ADD CONSTRAINT registry_extension_releases_pkey PRIMARY KEY (id);


--
-- Name: registry_extensions registry_extensions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.registry_extensions
    ADD CONSTRAINT registry_extensions_pkey PRIMARY KEY (id);


--
-- Name: repo_commits_changelists repo_commits_changelists_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.repo_commits_changelists
    ADD CONSTRAINT repo_commits_changelists_pkey PRIMARY KEY (id);


--
-- Name: repo_embedding_job_stats repo_embedding_job_stats_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.repo_embedding_job_stats
    ADD CONSTRAINT repo_embedding_job_stats_pkey PRIMARY KEY (job_id);


--
-- Name: repo_embedding_jobs repo_embedding_jobs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.repo_embedding_jobs
    ADD CONSTRAINT repo_embedding_jobs_pkey PRIMARY KEY (id);


--
-- Name: repo_kvps repo_kvps_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.repo_kvps
    ADD CONSTRAINT repo_kvps_pkey PRIMARY KEY (repo_id, key) INCLUDE (value);


--
-- Name: repo repo_name_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.repo
    ADD CONSTRAINT repo_name_unique UNIQUE (name) DEFERRABLE;


--
-- Name: repo_paths repo_paths_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.repo_paths
    ADD CONSTRAINT repo_paths_pkey PRIMARY KEY (id);


--
-- Name: repo_pending_permissions repo_pending_permissions_perm_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.repo_pending_permissions
    ADD CONSTRAINT repo_pending_permissions_perm_unique UNIQUE (repo_id, permission);


--
-- Name: repo_permissions repo_permissions_perm_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.repo_permissions
    ADD CONSTRAINT repo_permissions_perm_unique UNIQUE (repo_id, permission);


--
-- Name: repo repo_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.repo
    ADD CONSTRAINT repo_pkey PRIMARY KEY (id);


--
-- Name: role_permissions role_permissions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_permissions
    ADD CONSTRAINT role_permissions_pkey PRIMARY KEY (permission_id, role_id);


--
-- Name: roles roles_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.roles
    ADD CONSTRAINT roles_pkey PRIMARY KEY (id);


--
-- Name: saved_searches saved_searches_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.saved_searches
    ADD CONSTRAINT saved_searches_pkey PRIMARY KEY (id);


--
-- Name: search_context_default search_context_default_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.search_context_default
    ADD CONSTRAINT search_context_default_pkey PRIMARY KEY (user_id);


--
-- Name: search_context_repos search_context_repos_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.search_context_repos
    ADD CONSTRAINT search_context_repos_unique UNIQUE (repo_id, search_context_id, revision);


--
-- Name: search_context_stars search_context_stars_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.search_context_stars
    ADD CONSTRAINT search_context_stars_pkey PRIMARY KEY (search_context_id, user_id);


--
-- Name: search_contexts search_contexts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.search_contexts
    ADD CONSTRAINT search_contexts_pkey PRIMARY KEY (id);


--
-- Name: security_event_logs security_event_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.security_event_logs
    ADD CONSTRAINT security_event_logs_pkey PRIMARY KEY (id);


--
-- Name: settings settings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.settings
    ADD CONSTRAINT settings_pkey PRIMARY KEY (id);


--
-- Name: survey_responses survey_responses_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.survey_responses
    ADD CONSTRAINT survey_responses_pkey PRIMARY KEY (id);


--
-- Name: syntactic_scip_indexing_jobs syntactic_scip_indexing_jobs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.syntactic_scip_indexing_jobs
    ADD CONSTRAINT syntactic_scip_indexing_jobs_pkey PRIMARY KEY (id);


--
-- Name: syntactic_scip_last_index_scan syntactic_scip_last_index_scan_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.syntactic_scip_last_index_scan
    ADD CONSTRAINT syntactic_scip_last_index_scan_pkey PRIMARY KEY (repository_id);


--
-- Name: team_members team_members_team_id_user_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team_members
    ADD CONSTRAINT team_members_team_id_user_id_key PRIMARY KEY (team_id, user_id);


--
-- Name: teams teams_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.teams
    ADD CONSTRAINT teams_pkey PRIMARY KEY (id);


--
-- Name: telemetry_events_export_queue telemetry_events_export_queue_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.telemetry_events_export_queue
    ADD CONSTRAINT telemetry_events_export_queue_pkey PRIMARY KEY (id);


--
-- Name: temporary_settings temporary_settings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.temporary_settings
    ADD CONSTRAINT temporary_settings_pkey PRIMARY KEY (id);


--
-- Name: temporary_settings temporary_settings_user_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.temporary_settings
    ADD CONSTRAINT temporary_settings_user_id_key UNIQUE (user_id);


--
-- Name: tenants tenants_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenants
    ADD CONSTRAINT tenants_name_key UNIQUE (name);


--
-- Name: tenants tenants_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tenants
    ADD CONSTRAINT tenants_pkey PRIMARY KEY (id);


--
-- Name: github_app_installs unique_app_install; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_app_installs
    ADD CONSTRAINT unique_app_install UNIQUE (app_id, installation_id);


--
-- Name: user_credentials user_credentials_domain_user_id_external_service_type_exter_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_credentials
    ADD CONSTRAINT user_credentials_domain_user_id_external_service_type_exter_key UNIQUE (domain, user_id, external_service_type, external_service_id);


--
-- Name: user_credentials user_credentials_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_credentials
    ADD CONSTRAINT user_credentials_pkey PRIMARY KEY (id);


--
-- Name: user_emails user_emails_no_duplicates_per_user; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_emails
    ADD CONSTRAINT user_emails_no_duplicates_per_user UNIQUE (user_id, email);


--
-- Name: user_emails user_emails_unique_verified_email; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_emails
    ADD CONSTRAINT user_emails_unique_verified_email EXCLUDE USING btree (email WITH OPERATOR(public.=)) WHERE ((verified_at IS NOT NULL));


--
-- Name: user_external_accounts user_external_accounts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_external_accounts
    ADD CONSTRAINT user_external_accounts_pkey PRIMARY KEY (id);


--
-- Name: user_onboarding_tour user_onboarding_tour_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_onboarding_tour
    ADD CONSTRAINT user_onboarding_tour_pkey PRIMARY KEY (id);


--
-- Name: user_pending_permissions user_pending_permissions_service_perm_object_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pending_permissions
    ADD CONSTRAINT user_pending_permissions_service_perm_object_unique UNIQUE (service_type, service_id, permission, object_type, bind_id);


--
-- Name: user_permissions user_permissions_perm_object_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_permissions
    ADD CONSTRAINT user_permissions_perm_object_unique UNIQUE (user_id, permission, object_type);


--
-- Name: user_public_repos user_public_repos_user_id_repo_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_public_repos
    ADD CONSTRAINT user_public_repos_user_id_repo_id_key UNIQUE (user_id, repo_id);


--
-- Name: user_repo_permissions user_repo_permissions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_repo_permissions
    ADD CONSTRAINT user_repo_permissions_pkey PRIMARY KEY (id);


--
-- Name: user_roles user_roles_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_roles
    ADD CONSTRAINT user_roles_pkey PRIMARY KEY (user_id, role_id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: versions versions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.versions
    ADD CONSTRAINT versions_pkey PRIMARY KEY (service);


--
-- Name: vulnerabilities vulnerabilities_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vulnerabilities
    ADD CONSTRAINT vulnerabilities_pkey PRIMARY KEY (id);


--
-- Name: vulnerability_affected_packages vulnerability_affected_packages_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vulnerability_affected_packages
    ADD CONSTRAINT vulnerability_affected_packages_pkey PRIMARY KEY (id);


--
-- Name: vulnerability_affected_symbols vulnerability_affected_symbols_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vulnerability_affected_symbols
    ADD CONSTRAINT vulnerability_affected_symbols_pkey PRIMARY KEY (id);


--
-- Name: vulnerability_matches vulnerability_matches_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vulnerability_matches
    ADD CONSTRAINT vulnerability_matches_pkey PRIMARY KEY (id);


--
-- Name: webhook_logs webhook_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhook_logs
    ADD CONSTRAINT webhook_logs_pkey PRIMARY KEY (id);


--
-- Name: webhooks webhooks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhooks
    ADD CONSTRAINT webhooks_pkey PRIMARY KEY (id);


--
-- Name: webhooks webhooks_uuid_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhooks
    ADD CONSTRAINT webhooks_uuid_key UNIQUE (uuid);


--
-- Name: zoekt_repos zoekt_repos_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.zoekt_repos
    ADD CONSTRAINT zoekt_repos_pkey PRIMARY KEY (repo_id);


--
-- Name: access_requests_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX access_requests_created_at ON public.access_requests USING btree (created_at);


--
-- Name: access_requests_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX access_requests_status ON public.access_requests USING btree (status);


--
-- Name: access_tokens_lookup; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX access_tokens_lookup ON public.access_tokens USING hash (value_sha256) WHERE (deleted_at IS NULL);


--
-- Name: access_tokens_lookup_double_hash; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX access_tokens_lookup_double_hash ON public.access_tokens USING hash (public.digest(value_sha256, 'sha256'::text)) WHERE (deleted_at IS NULL);


--
-- Name: app_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX app_id_idx ON public.github_app_installs USING btree (app_id);


--
-- Name: assigned_owners_file_path_owner; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX assigned_owners_file_path_owner ON public.assigned_owners USING btree (file_path_id, owner_user_id);


--
-- Name: assigned_teams_file_path_owner; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX assigned_teams_file_path_owner ON public.assigned_teams USING btree (file_path_id, owner_team_id);


--
-- Name: batch_changes_namespace_org_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX batch_changes_namespace_org_id ON public.batch_changes USING btree (namespace_org_id);


--
-- Name: batch_changes_namespace_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX batch_changes_namespace_user_id ON public.batch_changes USING btree (namespace_user_id);


--
-- Name: batch_changes_site_credentials_credential_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX batch_changes_site_credentials_credential_idx ON public.batch_changes_site_credentials USING btree (((encryption_key_id = ANY (ARRAY[''::text, 'previously-migrated'::text]))));


--
-- Name: batch_changes_site_credentials_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX batch_changes_site_credentials_unique ON public.batch_changes_site_credentials USING btree (external_service_type, external_service_id);


--
-- Name: batch_changes_unique_org_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX batch_changes_unique_org_id ON public.batch_changes USING btree (name, namespace_org_id) WHERE (namespace_org_id IS NOT NULL);


--
-- Name: batch_changes_unique_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX batch_changes_unique_user_id ON public.batch_changes USING btree (name, namespace_user_id) WHERE (namespace_user_id IS NOT NULL);


--
-- Name: batch_spec_resolution_jobs_state; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX batch_spec_resolution_jobs_state ON public.batch_spec_resolution_jobs USING btree (state);


--
-- Name: batch_spec_workspace_execution_jobs_batch_spec_workspace_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX batch_spec_workspace_execution_jobs_batch_spec_workspace_id ON public.batch_spec_workspace_execution_jobs USING btree (batch_spec_workspace_id);


--
-- Name: batch_spec_workspace_execution_jobs_cancel; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX batch_spec_workspace_execution_jobs_cancel ON public.batch_spec_workspace_execution_jobs USING btree (cancel);


--
-- Name: batch_spec_workspace_execution_jobs_last_dequeue; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX batch_spec_workspace_execution_jobs_last_dequeue ON public.batch_spec_workspace_execution_jobs USING btree (user_id, started_at DESC);


--
-- Name: batch_spec_workspace_execution_jobs_state; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX batch_spec_workspace_execution_jobs_state ON public.batch_spec_workspace_execution_jobs USING btree (state);


--
-- Name: batch_spec_workspace_files_batch_spec_id_filename_path; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX batch_spec_workspace_files_batch_spec_id_filename_path ON public.batch_spec_workspace_files USING btree (batch_spec_id, filename, path);


--
-- Name: batch_spec_workspace_files_rand_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX batch_spec_workspace_files_rand_id ON public.batch_spec_workspace_files USING btree (rand_id);


--
-- Name: batch_spec_workspaces_batch_spec_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX batch_spec_workspaces_batch_spec_id ON public.batch_spec_workspaces USING btree (batch_spec_id);


--
-- Name: batch_spec_workspaces_id_batch_spec_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX batch_spec_workspaces_id_batch_spec_id ON public.batch_spec_workspaces USING btree (id, batch_spec_id);


--
-- Name: batch_specs_unique_rand_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX batch_specs_unique_rand_id ON public.batch_specs USING btree (rand_id);


--
-- Name: cached_available_indexers_num_events; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX cached_available_indexers_num_events ON public.cached_available_indexers USING btree (num_events DESC) WHERE ((available_indexers)::text <> '{}'::text);


--
-- Name: cached_available_indexers_repository_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX cached_available_indexers_repository_id ON public.cached_available_indexers USING btree (repository_id);


--
-- Name: changeset_jobs_bulk_group_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX changeset_jobs_bulk_group_idx ON public.changeset_jobs USING btree (bulk_group);


--
-- Name: changeset_jobs_state_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX changeset_jobs_state_idx ON public.changeset_jobs USING btree (state);


--
-- Name: changeset_specs_batch_spec_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX changeset_specs_batch_spec_id ON public.changeset_specs USING btree (batch_spec_id);


--
-- Name: changeset_specs_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX changeset_specs_created_at ON public.changeset_specs USING btree (created_at);


--
-- Name: changeset_specs_external_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX changeset_specs_external_id ON public.changeset_specs USING btree (external_id);


--
-- Name: changeset_specs_head_ref; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX changeset_specs_head_ref ON public.changeset_specs USING btree (head_ref);


--
-- Name: changeset_specs_title; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX changeset_specs_title ON public.changeset_specs USING btree (title);


--
-- Name: changeset_specs_unique_rand_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX changeset_specs_unique_rand_id ON public.changeset_specs USING btree (rand_id);


--
-- Name: changesets_batch_change_ids; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX changesets_batch_change_ids ON public.changesets USING gin (batch_change_ids);


--
-- Name: changesets_bitbucket_cloud_metadata_source_commit_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX changesets_bitbucket_cloud_metadata_source_commit_idx ON public.changesets USING btree (((((metadata -> 'source'::text) -> 'commit'::text) ->> 'hash'::text)));


--
-- Name: changesets_changeset_specs; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX changesets_changeset_specs ON public.changesets USING btree (current_spec_id, previous_spec_id);


--
-- Name: changesets_computed_state; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX changesets_computed_state ON public.changesets USING btree (computed_state);


--
-- Name: changesets_detached_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX changesets_detached_at ON public.changesets USING btree (detached_at);


--
-- Name: changesets_external_state_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX changesets_external_state_idx ON public.changesets USING btree (external_state);


--
-- Name: changesets_external_title_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX changesets_external_title_idx ON public.changesets USING btree (external_title);


--
-- Name: changesets_publication_state_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX changesets_publication_state_idx ON public.changesets USING btree (publication_state);


--
-- Name: changesets_reconciler_state_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX changesets_reconciler_state_idx ON public.changesets USING btree (reconciler_state);


--
-- Name: cm_action_jobs_state_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX cm_action_jobs_state_idx ON public.cm_action_jobs USING btree (state);


--
-- Name: cm_action_jobs_trigger_event; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX cm_action_jobs_trigger_event ON public.cm_action_jobs USING btree (trigger_event);


--
-- Name: cm_slack_webhooks_monitor; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX cm_slack_webhooks_monitor ON public.cm_slack_webhooks USING btree (monitor);


--
-- Name: cm_trigger_jobs_finished_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX cm_trigger_jobs_finished_at ON public.cm_trigger_jobs USING btree (finished_at);


--
-- Name: cm_trigger_jobs_state_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX cm_trigger_jobs_state_idx ON public.cm_trigger_jobs USING btree (state);


--
-- Name: cm_webhooks_monitor; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX cm_webhooks_monitor ON public.cm_webhooks USING btree (monitor);


--
-- Name: codeintel_autoindex_queue_repository_id_commit; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX codeintel_autoindex_queue_repository_id_commit ON public.codeintel_autoindex_queue USING btree (repository_id, rev);


--
-- Name: codeintel_initial_path_ranks_exported_upload_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX codeintel_initial_path_ranks_exported_upload_id ON public.codeintel_initial_path_ranks USING btree (exported_upload_id);


--
-- Name: codeintel_initial_path_ranks_graph_key_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX codeintel_initial_path_ranks_graph_key_id ON public.codeintel_initial_path_ranks USING btree (graph_key, id);


--
-- Name: codeintel_initial_path_ranks_processed_cgraph_key_codeintel_ini; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX codeintel_initial_path_ranks_processed_cgraph_key_codeintel_ini ON public.codeintel_initial_path_ranks_processed USING btree (graph_key, codeintel_initial_path_ranks_id);


--
-- Name: codeintel_initial_path_ranks_processed_codeintel_initial_path_r; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX codeintel_initial_path_ranks_processed_codeintel_initial_path_r ON public.codeintel_initial_path_ranks_processed USING btree (codeintel_initial_path_ranks_id);


--
-- Name: codeintel_langugage_support_requests_user_id_language; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX codeintel_langugage_support_requests_user_id_language ON public.codeintel_langugage_support_requests USING btree (user_id, language_id);


--
-- Name: codeintel_path_ranks_graph_key; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX codeintel_path_ranks_graph_key ON public.codeintel_path_ranks USING btree (graph_key, updated_at NULLS FIRST, id);


--
-- Name: codeintel_path_ranks_graph_key_repository_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX codeintel_path_ranks_graph_key_repository_id ON public.codeintel_path_ranks USING btree (graph_key, repository_id);


--
-- Name: codeintel_path_ranks_repository_id_updated_at_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX codeintel_path_ranks_repository_id_updated_at_id ON public.codeintel_path_ranks USING btree (repository_id, updated_at NULLS FIRST, id);


--
-- Name: codeintel_ranking_definitions_exported_upload_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX codeintel_ranking_definitions_exported_upload_id ON public.codeintel_ranking_definitions USING btree (exported_upload_id);


--
-- Name: codeintel_ranking_definitions_graph_key_symbol_checksum_search; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX codeintel_ranking_definitions_graph_key_symbol_checksum_search ON public.codeintel_ranking_definitions USING btree (graph_key, symbol_checksum, exported_upload_id, document_path);


--
-- Name: codeintel_ranking_exports_graph_key_deleted_at_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX codeintel_ranking_exports_graph_key_deleted_at_id ON public.codeintel_ranking_exports USING btree (graph_key, deleted_at DESC, id);


--
-- Name: codeintel_ranking_exports_graph_key_last_scanned_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX codeintel_ranking_exports_graph_key_last_scanned_at ON public.codeintel_ranking_exports USING btree (graph_key, last_scanned_at NULLS FIRST, id);


--
-- Name: codeintel_ranking_exports_graph_key_upload_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX codeintel_ranking_exports_graph_key_upload_id ON public.codeintel_ranking_exports USING btree (graph_key, upload_id);


--
-- Name: codeintel_ranking_path_counts_inputs_graph_key_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX codeintel_ranking_path_counts_inputs_graph_key_id ON public.codeintel_ranking_path_counts_inputs USING btree (graph_key, id);


--
-- Name: codeintel_ranking_path_counts_inputs_graph_key_unique_definitio; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX codeintel_ranking_path_counts_inputs_graph_key_unique_definitio ON public.codeintel_ranking_path_counts_inputs USING btree (graph_key, definition_id) WHERE (NOT processed);


--
-- Name: codeintel_ranking_references_exported_upload_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX codeintel_ranking_references_exported_upload_id ON public.codeintel_ranking_references USING btree (exported_upload_id);


--
-- Name: codeintel_ranking_references_graph_key_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX codeintel_ranking_references_graph_key_id ON public.codeintel_ranking_references USING btree (graph_key, id);


--
-- Name: codeintel_ranking_references_processed_graph_key_codeintel_rank; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX codeintel_ranking_references_processed_graph_key_codeintel_rank ON public.codeintel_ranking_references_processed USING btree (graph_key, codeintel_ranking_reference_id);


--
-- Name: codeintel_ranking_references_processed_reference_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX codeintel_ranking_references_processed_reference_id ON public.codeintel_ranking_references_processed USING btree (codeintel_ranking_reference_id);


--
-- Name: codeowners_owners_reference; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX codeowners_owners_reference ON public.codeowners_owners USING btree (reference);


--
-- Name: commit_authors_email_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX commit_authors_email_name ON public.commit_authors USING btree (email, name);


--
-- Name: configuration_policies_audit_logs_policy_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX configuration_policies_audit_logs_policy_id ON public.configuration_policies_audit_logs USING btree (policy_id);


--
-- Name: configuration_policies_audit_logs_timestamp; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX configuration_policies_audit_logs_timestamp ON public.configuration_policies_audit_logs USING brin (log_timestamp);


--
-- Name: critical_and_site_config_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX critical_and_site_config_unique ON public.critical_and_site_config USING btree (id, type);


--
-- Name: discussion_comments_author_user_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX discussion_comments_author_user_id_idx ON public.discussion_comments USING btree (author_user_id);


--
-- Name: discussion_comments_reports_array_length_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX discussion_comments_reports_array_length_idx ON public.discussion_comments USING btree (array_length(reports, 1));


--
-- Name: discussion_comments_thread_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX discussion_comments_thread_id_idx ON public.discussion_comments USING btree (thread_id);


--
-- Name: discussion_mail_reply_tokens_user_id_thread_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX discussion_mail_reply_tokens_user_id_thread_id_idx ON public.discussion_mail_reply_tokens USING btree (user_id, thread_id);


--
-- Name: discussion_threads_author_user_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX discussion_threads_author_user_id_idx ON public.discussion_threads USING btree (author_user_id);


--
-- Name: discussion_threads_target_repo_repo_id_path_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX discussion_threads_target_repo_repo_id_path_idx ON public.discussion_threads_target_repo USING btree (repo_id, path);


--
-- Name: event_logs_anonymous_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX event_logs_anonymous_user_id ON public.event_logs USING btree (anonymous_user_id);


--
-- Name: event_logs_export_allowlist_event_name_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX event_logs_export_allowlist_event_name_idx ON public.event_logs_export_allowlist USING btree (event_name);


--
-- Name: event_logs_name_timestamp; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX event_logs_name_timestamp ON public.event_logs USING btree (name, "timestamp" DESC);


--
-- Name: event_logs_source; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX event_logs_source ON public.event_logs USING btree (source);


--
-- Name: event_logs_timestamp; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX event_logs_timestamp ON public.event_logs USING btree ("timestamp");


--
-- Name: event_logs_timestamp_at_utc; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX event_logs_timestamp_at_utc ON public.event_logs USING btree (date(timezone('UTC'::text, "timestamp")));


--
-- Name: event_logs_user_id_name; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX event_logs_user_id_name ON public.event_logs USING btree (user_id, name);


--
-- Name: event_logs_user_id_timestamp; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX event_logs_user_id_timestamp ON public.event_logs USING btree (user_id, "timestamp");


--
-- Name: executor_secrets_unique_key_global; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX executor_secrets_unique_key_global ON public.executor_secrets USING btree (key, scope) WHERE ((namespace_user_id IS NULL) AND (namespace_org_id IS NULL));


--
-- Name: executor_secrets_unique_key_namespace_org; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX executor_secrets_unique_key_namespace_org ON public.executor_secrets USING btree (key, namespace_org_id, scope) WHERE (namespace_org_id IS NOT NULL);


--
-- Name: executor_secrets_unique_key_namespace_user; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX executor_secrets_unique_key_namespace_user ON public.executor_secrets USING btree (key, namespace_user_id, scope) WHERE (namespace_user_id IS NOT NULL);


--
-- Name: exhaustive_search_jobs_state; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX exhaustive_search_jobs_state ON public.exhaustive_search_jobs USING btree (state);


--
-- Name: exhaustive_search_repo_jobs_state; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX exhaustive_search_repo_jobs_state ON public.exhaustive_search_repo_jobs USING btree (state);


--
-- Name: exhaustive_search_repo_revision_jobs_state; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX exhaustive_search_repo_revision_jobs_state ON public.exhaustive_search_repo_revision_jobs USING btree (state);


--
-- Name: explicit_permissions_bitbucket_projects_jobs_project_key_extern; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX explicit_permissions_bitbucket_projects_jobs_project_key_extern ON public.explicit_permissions_bitbucket_projects_jobs USING btree (project_key, external_service_id, state);


--
-- Name: explicit_permissions_bitbucket_projects_jobs_queued_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX explicit_permissions_bitbucket_projects_jobs_queued_at_idx ON public.explicit_permissions_bitbucket_projects_jobs USING btree (queued_at);


--
-- Name: explicit_permissions_bitbucket_projects_jobs_state_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX explicit_permissions_bitbucket_projects_jobs_state_idx ON public.explicit_permissions_bitbucket_projects_jobs USING btree (state);


--
-- Name: external_service_repos_clone_url_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX external_service_repos_clone_url_idx ON public.external_service_repos USING btree (clone_url);


--
-- Name: external_service_repos_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX external_service_repos_idx ON public.external_service_repos USING btree (external_service_id, repo_id);


--
-- Name: external_service_sync_jobs_state_external_service_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX external_service_sync_jobs_state_external_service_id ON public.external_service_sync_jobs USING btree (state, external_service_id) INCLUDE (finished_at);


--
-- Name: external_services_has_webhooks_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX external_services_has_webhooks_idx ON public.external_services USING btree (has_webhooks);


--
-- Name: feature_flag_overrides_org_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX feature_flag_overrides_org_id ON public.feature_flag_overrides USING btree (namespace_org_id) WHERE (namespace_org_id IS NOT NULL);


--
-- Name: feature_flag_overrides_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX feature_flag_overrides_user_id ON public.feature_flag_overrides USING btree (namespace_user_id) WHERE (namespace_user_id IS NOT NULL);


--
-- Name: finished_at_insights_query_runner_jobs_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX finished_at_insights_query_runner_jobs_idx ON public.insights_query_runner_jobs USING btree (finished_at);


--
-- Name: github_app_installs_account_login; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX github_app_installs_account_login ON public.github_app_installs USING btree (account_login);


--
-- Name: github_apps_app_id_slug_base_url_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX github_apps_app_id_slug_base_url_unique ON public.github_apps USING btree (app_id, slug, base_url);


--
-- Name: gitserver_relocator_jobs_state; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX gitserver_relocator_jobs_state ON public.gitserver_relocator_jobs USING btree (state);


--
-- Name: gitserver_repo_size_bytes; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX gitserver_repo_size_bytes ON public.gitserver_repos USING btree (repo_size_bytes);


--
-- Name: gitserver_repos_cloned_status_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX gitserver_repos_cloned_status_idx ON public.gitserver_repos USING btree (repo_id) WHERE (clone_status = 'cloned'::text);


--
-- Name: gitserver_repos_cloning_status_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX gitserver_repos_cloning_status_idx ON public.gitserver_repos USING btree (repo_id) WHERE (clone_status = 'cloning'::text);


--
-- Name: gitserver_repos_last_changed_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX gitserver_repos_last_changed_idx ON public.gitserver_repos USING btree (last_changed, repo_id);


--
-- Name: gitserver_repos_last_error_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX gitserver_repos_last_error_idx ON public.gitserver_repos USING btree (repo_id) WHERE (last_error IS NOT NULL);


--
-- Name: gitserver_repos_not_cloned_status_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX gitserver_repos_not_cloned_status_idx ON public.gitserver_repos USING btree (repo_id) WHERE (clone_status = 'not_cloned'::text);


--
-- Name: gitserver_repos_not_explicitly_cloned_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX gitserver_repos_not_explicitly_cloned_idx ON public.gitserver_repos USING btree (repo_id) WHERE (clone_status <> 'cloned'::text);


--
-- Name: gitserver_repos_shard_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX gitserver_repos_shard_id ON public.gitserver_repos USING btree (shard_id, repo_id);


--
-- Name: gitserver_repos_statistics_shard_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX gitserver_repos_statistics_shard_id ON public.gitserver_repos_statistics USING btree (shard_id);


--
-- Name: idx_repo_topics; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_repo_topics ON public.repo USING gin (topics);


--
-- Name: insights_query_runner_jobs_cost_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX insights_query_runner_jobs_cost_idx ON public.insights_query_runner_jobs USING btree (cost);


--
-- Name: insights_query_runner_jobs_dependencies_job_id_fk_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX insights_query_runner_jobs_dependencies_job_id_fk_idx ON public.insights_query_runner_jobs_dependencies USING btree (job_id);


--
-- Name: insights_query_runner_jobs_priority_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX insights_query_runner_jobs_priority_idx ON public.insights_query_runner_jobs USING btree (priority);


--
-- Name: insights_query_runner_jobs_processable_priority_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX insights_query_runner_jobs_processable_priority_id ON public.insights_query_runner_jobs USING btree (priority, id) WHERE ((state = 'queued'::text) OR (state = 'errored'::text));


--
-- Name: insights_query_runner_jobs_series_id_state; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX insights_query_runner_jobs_series_id_state ON public.insights_query_runner_jobs USING btree (series_id, state);


--
-- Name: insights_query_runner_jobs_state_btree; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX insights_query_runner_jobs_state_btree ON public.insights_query_runner_jobs USING btree (state);


--
-- Name: installation_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX installation_id_idx ON public.github_app_installs USING btree (installation_id);


--
-- Name: kind_cloud_default; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX kind_cloud_default ON public.external_services USING btree (kind, cloud_default) WHERE ((cloud_default = true) AND (deleted_at IS NULL));


--
-- Name: lsif_configuration_policies_repository_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX lsif_configuration_policies_repository_id ON public.lsif_configuration_policies USING btree (repository_id);


--
-- Name: lsif_dependency_indexing_jobs_state; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX lsif_dependency_indexing_jobs_state ON public.lsif_dependency_indexing_jobs USING btree (state);


--
-- Name: lsif_dependency_indexing_jobs_upload_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX lsif_dependency_indexing_jobs_upload_id ON public.lsif_dependency_syncing_jobs USING btree (upload_id);


--
-- Name: lsif_dependency_repos_blocked; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX lsif_dependency_repos_blocked ON public.lsif_dependency_repos USING btree (blocked);


--
-- Name: lsif_dependency_repos_last_checked_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX lsif_dependency_repos_last_checked_at ON public.lsif_dependency_repos USING btree (last_checked_at NULLS FIRST);


--
-- Name: lsif_dependency_repos_name_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX lsif_dependency_repos_name_gin ON public.lsif_dependency_repos USING gin (name public.gin_trgm_ops);


--
-- Name: lsif_dependency_repos_name_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX lsif_dependency_repos_name_id ON public.lsif_dependency_repos USING btree (name, id);


--
-- Name: lsif_dependency_repos_scheme_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX lsif_dependency_repos_scheme_id ON public.lsif_dependency_repos USING btree (scheme, id);


--
-- Name: lsif_dependency_repos_unique_scheme_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX lsif_dependency_repos_unique_scheme_name ON public.lsif_dependency_repos USING btree (scheme, name);


--
-- Name: lsif_dependency_syncing_jobs_state; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX lsif_dependency_syncing_jobs_state ON public.lsif_dependency_syncing_jobs USING btree (state);


--
-- Name: lsif_indexes_commit_last_checked_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX lsif_indexes_commit_last_checked_at ON public.lsif_indexes USING btree (commit_last_checked_at) WHERE (state <> 'deleted'::text);


--
-- Name: lsif_indexes_dequeue_order_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX lsif_indexes_dequeue_order_idx ON public.lsif_indexes USING btree (((enqueuer_user_id > 0)) DESC, queued_at DESC, id) WHERE ((state = 'queued'::text) OR (state = 'errored'::text));


--
-- Name: lsif_indexes_queued_at_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX lsif_indexes_queued_at_id ON public.lsif_indexes USING btree (queued_at DESC, id);


--
-- Name: lsif_indexes_repository_id_commit; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX lsif_indexes_repository_id_commit ON public.lsif_indexes USING btree (repository_id, commit);


--
-- Name: lsif_indexes_repository_id_indexer; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX lsif_indexes_repository_id_indexer ON public.lsif_indexes USING btree (repository_id, indexer);


--
-- Name: lsif_indexes_state; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX lsif_indexes_state ON public.lsif_indexes USING btree (state);


--
-- Name: lsif_nearest_uploads_links_repository_id_ancestor_commit_bytea; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX lsif_nearest_uploads_links_repository_id_ancestor_commit_bytea ON public.lsif_nearest_uploads_links USING btree (repository_id, ancestor_commit_bytea);


--
-- Name: lsif_nearest_uploads_links_repository_id_commit_bytea; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX lsif_nearest_uploads_links_repository_id_commit_bytea ON public.lsif_nearest_uploads_links USING btree (repository_id, commit_bytea);


--
-- Name: lsif_nearest_uploads_repository_id_commit_bytea; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX lsif_nearest_uploads_repository_id_commit_bytea ON public.lsif_nearest_uploads USING btree (repository_id, commit_bytea);


--
-- Name: lsif_nearest_uploads_uploads; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX lsif_nearest_uploads_uploads ON public.lsif_nearest_uploads USING gin (uploads);


--
-- Name: lsif_packages_dump_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX lsif_packages_dump_id ON public.lsif_packages USING btree (dump_id);


--
-- Name: lsif_packages_scheme_name_version_dump_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX lsif_packages_scheme_name_version_dump_id ON public.lsif_packages USING btree (scheme, name, version, dump_id);


--
-- Name: lsif_references_dump_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX lsif_references_dump_id ON public.lsif_references USING btree (dump_id);


--
-- Name: lsif_references_scheme_name_version_dump_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX lsif_references_scheme_name_version_dump_id ON public.lsif_references USING btree (scheme, name, version, dump_id);


--
-- Name: lsif_uploads_associated_index_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX lsif_uploads_associated_index_id ON public.lsif_uploads USING btree (associated_index_id);


--
-- Name: lsif_uploads_audit_logs_timestamp; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX lsif_uploads_audit_logs_timestamp ON public.lsif_uploads_audit_logs USING brin (log_timestamp);


--
-- Name: lsif_uploads_audit_logs_upload_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX lsif_uploads_audit_logs_upload_id ON public.lsif_uploads_audit_logs USING btree (upload_id);


--
-- Name: lsif_uploads_commit_last_checked_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX lsif_uploads_commit_last_checked_at ON public.lsif_uploads USING btree (commit_last_checked_at) WHERE (state <> 'deleted'::text);


--
-- Name: lsif_uploads_committed_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX lsif_uploads_committed_at ON public.lsif_uploads USING btree (committed_at) WHERE (state = 'completed'::text);


--
-- Name: lsif_uploads_last_reconcile_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX lsif_uploads_last_reconcile_at ON public.lsif_uploads USING btree (last_reconcile_at, id) WHERE (state = 'completed'::text);


--
-- Name: lsif_uploads_repository_id_commit; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX lsif_uploads_repository_id_commit ON public.lsif_uploads USING btree (repository_id, commit);


--
-- Name: lsif_uploads_repository_id_commit_root_indexer; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX lsif_uploads_repository_id_commit_root_indexer ON public.lsif_uploads USING btree (repository_id, commit, root, indexer) WHERE (state = 'completed'::text);


--
-- Name: lsif_uploads_repository_id_indexer; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX lsif_uploads_repository_id_indexer ON public.lsif_uploads USING btree (repository_id, indexer);


--
-- Name: lsif_uploads_state; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX lsif_uploads_state ON public.lsif_uploads USING btree (state);


--
-- Name: lsif_uploads_uploaded_at_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX lsif_uploads_uploaded_at_id ON public.lsif_uploads USING btree (uploaded_at DESC, id) WHERE (state <> 'deleted'::text);


--
-- Name: lsif_uploads_visible_at_tip_is_default_branch; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX lsif_uploads_visible_at_tip_is_default_branch ON public.lsif_uploads_visible_at_tip USING btree (upload_id) WHERE is_default_branch;


--
-- Name: lsif_uploads_visible_at_tip_repository_id_upload_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX lsif_uploads_visible_at_tip_repository_id_upload_id ON public.lsif_uploads_visible_at_tip USING btree (repository_id, upload_id);


--
-- Name: lsif_uploads_vulnerability_scan_upload_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX lsif_uploads_vulnerability_scan_upload_id ON public.lsif_uploads_vulnerability_scan USING btree (upload_id);


--
-- Name: notebook_stars_user_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX notebook_stars_user_id_idx ON public.notebook_stars USING btree (user_id);


--
-- Name: notebooks_blocks_tsvector_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX notebooks_blocks_tsvector_idx ON public.notebooks USING gin (blocks_tsvector);


--
-- Name: notebooks_namespace_org_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX notebooks_namespace_org_id_idx ON public.notebooks USING btree (namespace_org_id);


--
-- Name: notebooks_namespace_user_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX notebooks_namespace_user_id_idx ON public.notebooks USING btree (namespace_user_id);


--
-- Name: notebooks_title_trgm_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX notebooks_title_trgm_idx ON public.notebooks USING gin (title public.gin_trgm_ops);


--
-- Name: org_invitations_org_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX org_invitations_org_id ON public.org_invitations USING btree (org_id) WHERE (deleted_at IS NULL);


--
-- Name: org_invitations_recipient_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX org_invitations_recipient_user_id ON public.org_invitations USING btree (recipient_user_id) WHERE (deleted_at IS NULL);


--
-- Name: orgs_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX orgs_name ON public.orgs USING btree (name) WHERE (deleted_at IS NULL);


--
-- Name: outbound_webhook_event_types_event_type_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX outbound_webhook_event_types_event_type_idx ON public.outbound_webhook_event_types USING btree (event_type, scope);


--
-- Name: outbound_webhook_jobs_state_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX outbound_webhook_jobs_state_idx ON public.outbound_webhook_jobs USING btree (state);


--
-- Name: outbound_webhook_logs_outbound_webhook_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX outbound_webhook_logs_outbound_webhook_id_idx ON public.outbound_webhook_logs USING btree (outbound_webhook_id);


--
-- Name: outbound_webhook_payload_process_after_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX outbound_webhook_payload_process_after_idx ON public.outbound_webhook_jobs USING btree (process_after);


--
-- Name: outbound_webhooks_logs_status_code_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX outbound_webhooks_logs_status_code_idx ON public.outbound_webhook_logs USING btree (status_code);


--
-- Name: own_aggregate_recent_contribution_file_author; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX own_aggregate_recent_contribution_file_author ON public.own_aggregate_recent_contribution USING btree (changed_file_path_id, commit_author_id);


--
-- Name: own_aggregate_recent_view_viewer; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX own_aggregate_recent_view_viewer ON public.own_aggregate_recent_view USING btree (viewed_file_path_id, viewer_id);


--
-- Name: own_background_jobs_repo_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX own_background_jobs_repo_id_idx ON public.own_background_jobs USING btree (repo_id);


--
-- Name: own_background_jobs_state_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX own_background_jobs_state_idx ON public.own_background_jobs USING btree (state);


--
-- Name: own_signal_configurations_name_uidx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX own_signal_configurations_name_uidx ON public.own_signal_configurations USING btree (name);


--
-- Name: package_repo_filters_unique_matcher_per_scheme; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX package_repo_filters_unique_matcher_per_scheme ON public.package_repo_filters USING btree (scheme, matcher);


--
-- Name: package_repo_versions_blocked; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX package_repo_versions_blocked ON public.package_repo_versions USING btree (blocked);


--
-- Name: package_repo_versions_last_checked_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX package_repo_versions_last_checked_at ON public.package_repo_versions USING btree (last_checked_at NULLS FIRST);


--
-- Name: package_repo_versions_unique_version_per_package; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX package_repo_versions_unique_version_per_package ON public.package_repo_versions USING btree (package_id, version);


--
-- Name: permission_sync_jobs_process_after; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX permission_sync_jobs_process_after ON public.permission_sync_jobs USING btree (process_after);


--
-- Name: permission_sync_jobs_repository_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX permission_sync_jobs_repository_id ON public.permission_sync_jobs USING btree (repository_id);


--
-- Name: permission_sync_jobs_state; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX permission_sync_jobs_state ON public.permission_sync_jobs USING btree (state);


--
-- Name: permission_sync_jobs_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX permission_sync_jobs_unique ON public.permission_sync_jobs USING btree (priority, user_id, repository_id, cancel, process_after) WHERE (state = 'queued'::text);


--
-- Name: permission_sync_jobs_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX permission_sync_jobs_user_id ON public.permission_sync_jobs USING btree (user_id);


--
-- Name: permissions_unique_namespace_action; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX permissions_unique_namespace_action ON public.permissions USING btree (namespace, action);


--
-- Name: process_after_insights_query_runner_jobs_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX process_after_insights_query_runner_jobs_idx ON public.insights_query_runner_jobs USING btree (process_after);


--
-- Name: product_licenses_license_check_token_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX product_licenses_license_check_token_idx ON public.product_licenses USING btree (license_check_token);


--
-- Name: prompts_name_is_unique_in_owner_org; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX prompts_name_is_unique_in_owner_org ON public.prompts USING btree (owner_org_id, name) WHERE (owner_org_id IS NOT NULL);


--
-- Name: prompts_name_is_unique_in_owner_user; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX prompts_name_is_unique_in_owner_user ON public.prompts USING btree (owner_user_id, name) WHERE (owner_user_id IS NOT NULL);


--
-- Name: registry_extension_releases_registry_extension_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX registry_extension_releases_registry_extension_id ON public.registry_extension_releases USING btree (registry_extension_id, release_tag, created_at DESC) WHERE (deleted_at IS NULL);


--
-- Name: registry_extension_releases_registry_extension_id_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX registry_extension_releases_registry_extension_id_created_at ON public.registry_extension_releases USING btree (registry_extension_id, created_at) WHERE (deleted_at IS NULL);


--
-- Name: registry_extension_releases_version; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX registry_extension_releases_version ON public.registry_extension_releases USING btree (registry_extension_id, release_version) WHERE (release_version IS NOT NULL);


--
-- Name: registry_extensions_publisher_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX registry_extensions_publisher_name ON public.registry_extensions USING btree (COALESCE(publisher_user_id, 0), COALESCE(publisher_org_id, 0), name) WHERE (deleted_at IS NULL);


--
-- Name: registry_extensions_uuid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX registry_extensions_uuid ON public.registry_extensions USING btree (uuid);


--
-- Name: repo_archived; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX repo_archived ON public.repo USING btree (archived);


--
-- Name: repo_blocked_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX repo_blocked_idx ON public.repo USING btree (((blocked IS NOT NULL)));


--
-- Name: repo_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX repo_created_at ON public.repo USING btree (created_at);


--
-- Name: repo_description_trgm_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX repo_description_trgm_idx ON public.repo USING gin (lower(description) public.gin_trgm_ops);


--
-- Name: repo_dotcom_indexable_repos_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX repo_dotcom_indexable_repos_idx ON public.repo USING btree (stars DESC NULLS LAST) INCLUDE (id, name) WHERE ((deleted_at IS NULL) AND (blocked IS NULL) AND (((stars >= 5) AND (NOT COALESCE(fork, false)) AND (NOT archived)) OR (lower((name)::text) ~ '^(src\.fedoraproject\.org|maven|npm|jdk)'::text)));


--
-- Name: repo_embedding_jobs_repo; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX repo_embedding_jobs_repo ON public.repo_embedding_jobs USING btree (repo_id, revision);


--
-- Name: repo_external_unique_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX repo_external_unique_idx ON public.repo USING btree (external_service_type, external_service_id, external_id);


--
-- Name: repo_fork; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX repo_fork ON public.repo USING btree (fork);


--
-- Name: repo_hashed_name_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX repo_hashed_name_idx ON public.repo USING btree (sha256((lower((name)::text))::bytea)) WHERE (deleted_at IS NULL);


--
-- Name: repo_id_perforce_changelist_id_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX repo_id_perforce_changelist_id_unique ON public.repo_commits_changelists USING btree (repo_id, perforce_changelist_id);


--
-- Name: repo_is_not_blocked_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX repo_is_not_blocked_idx ON public.repo USING btree (((blocked IS NULL)));


--
-- Name: repo_kvps_trgm_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX repo_kvps_trgm_idx ON public.repo_kvps USING gin (key public.gin_trgm_ops, value public.gin_trgm_ops);


--
-- Name: repo_metadata_gin_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX repo_metadata_gin_idx ON public.repo USING gin (metadata);


--
-- Name: repo_name_case_sensitive_trgm_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX repo_name_case_sensitive_trgm_idx ON public.repo USING gin (((name)::text) public.gin_trgm_ops);


--
-- Name: repo_name_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX repo_name_idx ON public.repo USING btree (lower((name)::text) COLLATE "C");


--
-- Name: repo_name_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX repo_name_trgm ON public.repo USING gin (lower((name)::text) public.gin_trgm_ops);


--
-- Name: repo_non_deleted_id_name_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX repo_non_deleted_id_name_idx ON public.repo USING btree (id, name) WHERE (deleted_at IS NULL);


--
-- Name: repo_paths_index_absolute_path; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX repo_paths_index_absolute_path ON public.repo_paths USING btree (repo_id, absolute_path);


--
-- Name: repo_permissions_unrestricted_true_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX repo_permissions_unrestricted_true_idx ON public.repo_permissions USING btree (unrestricted) WHERE unrestricted;


--
-- Name: repo_private; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX repo_private ON public.repo USING btree (private);


--
-- Name: repo_stars_desc_id_desc_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX repo_stars_desc_id_desc_idx ON public.repo USING btree (stars DESC NULLS LAST, id DESC) WHERE ((deleted_at IS NULL) AND (blocked IS NULL));


--
-- Name: repo_stars_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX repo_stars_idx ON public.repo USING btree (stars DESC NULLS LAST);


--
-- Name: repo_uri_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX repo_uri_idx ON public.repo USING btree (uri);


--
-- Name: search_contexts_name_namespace_org_id_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX search_contexts_name_namespace_org_id_unique ON public.search_contexts USING btree (name, namespace_org_id) WHERE (namespace_org_id IS NOT NULL);


--
-- Name: search_contexts_name_namespace_user_id_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX search_contexts_name_namespace_user_id_unique ON public.search_contexts USING btree (name, namespace_user_id) WHERE (namespace_user_id IS NOT NULL);


--
-- Name: search_contexts_name_without_namespace_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX search_contexts_name_without_namespace_unique ON public.search_contexts USING btree (name) WHERE ((namespace_user_id IS NULL) AND (namespace_org_id IS NULL));


--
-- Name: search_contexts_query_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX search_contexts_query_idx ON public.search_contexts USING btree (query);


--
-- Name: security_event_logs_timestamp; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX security_event_logs_timestamp ON public.security_event_logs USING btree ("timestamp");


--
-- Name: settings_global_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX settings_global_id ON public.settings USING btree (id DESC) WHERE ((user_id IS NULL) AND (org_id IS NULL));


--
-- Name: settings_org_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX settings_org_id_idx ON public.settings USING btree (org_id);


--
-- Name: settings_user_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX settings_user_id_idx ON public.settings USING btree (user_id);


--
-- Name: sub_repo_permissions_repo_id_user_id_version_uindex; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX sub_repo_permissions_repo_id_user_id_version_uindex ON public.sub_repo_permissions USING btree (repo_id, user_id, version);


--
-- Name: sub_repo_perms_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX sub_repo_perms_user_id ON public.sub_repo_permissions USING btree (user_id);


--
-- Name: syntactic_scip_indexing_jobs_dequeue_order_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX syntactic_scip_indexing_jobs_dequeue_order_idx ON public.syntactic_scip_indexing_jobs USING btree (((enqueuer_user_id > 0)) DESC, queued_at DESC, id) WHERE ((state = 'queued'::text) OR (state = 'errored'::text));


--
-- Name: syntactic_scip_indexing_jobs_queued_at_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX syntactic_scip_indexing_jobs_queued_at_id ON public.syntactic_scip_indexing_jobs USING btree (queued_at DESC, id);


--
-- Name: syntactic_scip_indexing_jobs_repository_id_commit; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX syntactic_scip_indexing_jobs_repository_id_commit ON public.syntactic_scip_indexing_jobs USING btree (repository_id, commit);


--
-- Name: syntactic_scip_indexing_jobs_state; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX syntactic_scip_indexing_jobs_state ON public.syntactic_scip_indexing_jobs USING btree (state);


--
-- Name: teams_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX teams_name ON public.teams USING btree (name);


--
-- Name: unique_resource_permission; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX unique_resource_permission ON public.namespace_permissions USING btree (namespace, resource_id, user_id);


--
-- Name: unique_role_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX unique_role_name ON public.roles USING btree (name);


--
-- Name: user_credentials_credential_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_credentials_credential_idx ON public.user_credentials USING btree (((encryption_key_id = ANY (ARRAY[''::text, 'previously-migrated'::text]))));


--
-- Name: user_emails_user_id_is_primary_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX user_emails_user_id_is_primary_idx ON public.user_emails USING btree (user_id, is_primary) WHERE (is_primary = true);


--
-- Name: user_external_accounts_account; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX user_external_accounts_account ON public.user_external_accounts USING btree (service_type, service_id, client_id, account_id) WHERE (deleted_at IS NULL);


--
-- Name: user_external_accounts_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_external_accounts_user_id ON public.user_external_accounts USING btree (user_id) WHERE (deleted_at IS NULL);


--
-- Name: user_external_accounts_user_id_scim_service_type; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX user_external_accounts_user_id_scim_service_type ON public.user_external_accounts USING btree (user_id, service_type) WHERE ((service_type = 'scim'::text) AND (deleted_at IS NULL));


--
-- Name: user_repo_permissions_perms_unique_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX user_repo_permissions_perms_unique_idx ON public.user_repo_permissions USING btree (user_id, user_external_account_id, repo_id);


--
-- Name: user_repo_permissions_repo_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_repo_permissions_repo_id_idx ON public.user_repo_permissions USING btree (repo_id);


--
-- Name: user_repo_permissions_source_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_repo_permissions_source_idx ON public.user_repo_permissions USING btree (source);


--
-- Name: user_repo_permissions_updated_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_repo_permissions_updated_at_idx ON public.user_repo_permissions USING btree (updated_at);


--
-- Name: user_repo_permissions_user_external_account_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_repo_permissions_user_external_account_id_idx ON public.user_repo_permissions USING btree (user_external_account_id);


--
-- Name: users_billing_customer_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX users_billing_customer_id ON public.users USING btree (billing_customer_id) WHERE (deleted_at IS NULL);


--
-- Name: users_created_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX users_created_at_idx ON public.users USING btree (created_at);


--
-- Name: users_username; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX users_username ON public.users USING btree (username) WHERE (deleted_at IS NULL);


--
-- Name: vulnerabilities_source_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX vulnerabilities_source_id ON public.vulnerabilities USING btree (source_id);


--
-- Name: vulnerability_affected_packages_vulnerability_id_package_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX vulnerability_affected_packages_vulnerability_id_package_name ON public.vulnerability_affected_packages USING btree (vulnerability_id, package_name);


--
-- Name: vulnerability_affected_symbols_vulnerability_affected_package_i; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX vulnerability_affected_symbols_vulnerability_affected_package_i ON public.vulnerability_affected_symbols USING btree (vulnerability_affected_package_id, path);


--
-- Name: vulnerability_matches_upload_id_vulnerability_affected_package_; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX vulnerability_matches_upload_id_vulnerability_affected_package_ ON public.vulnerability_matches USING btree (upload_id, vulnerability_affected_package_id);


--
-- Name: vulnerability_matches_vulnerability_affected_package_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX vulnerability_matches_vulnerability_affected_package_id ON public.vulnerability_matches USING btree (vulnerability_affected_package_id);


--
-- Name: webhook_logs_external_service_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX webhook_logs_external_service_id_idx ON public.webhook_logs USING btree (external_service_id);


--
-- Name: webhook_logs_received_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX webhook_logs_received_at_idx ON public.webhook_logs USING btree (received_at);


--
-- Name: webhook_logs_status_code_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX webhook_logs_status_code_idx ON public.webhook_logs USING btree (status_code);


--
-- Name: zoekt_repos_index_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX zoekt_repos_index_status ON public.zoekt_repos USING btree (index_status);


--
-- Name: batch_spec_workspace_execution_jobs batch_spec_workspace_execution_last_dequeues_insert; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER batch_spec_workspace_execution_last_dequeues_insert AFTER INSERT ON public.batch_spec_workspace_execution_jobs REFERENCING NEW TABLE AS newtab FOR EACH STATEMENT EXECUTE FUNCTION public.batch_spec_workspace_execution_last_dequeues_upsert();


--
-- Name: batch_spec_workspace_execution_jobs batch_spec_workspace_execution_last_dequeues_update; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER batch_spec_workspace_execution_last_dequeues_update AFTER UPDATE ON public.batch_spec_workspace_execution_jobs REFERENCING NEW TABLE AS newtab FOR EACH STATEMENT EXECUTE FUNCTION public.batch_spec_workspace_execution_last_dequeues_upsert();


--
-- Name: changesets changesets_update_computed_state; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER changesets_update_computed_state BEFORE INSERT OR UPDATE ON public.changesets FOR EACH ROW EXECUTE FUNCTION public.changesets_computed_state_ensure();


--
-- Name: codeintel_path_ranks insert_codeintel_path_ranks_statistics; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER insert_codeintel_path_ranks_statistics BEFORE INSERT ON public.codeintel_path_ranks FOR EACH ROW EXECUTE FUNCTION public.update_codeintel_path_ranks_statistics_columns();


--
-- Name: repo trig_create_zoekt_repo_on_repo_insert; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trig_create_zoekt_repo_on_repo_insert AFTER INSERT ON public.repo FOR EACH ROW EXECUTE FUNCTION public.func_insert_zoekt_repo();


--
-- Name: batch_changes trig_delete_batch_change_reference_on_changesets; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trig_delete_batch_change_reference_on_changesets AFTER DELETE ON public.batch_changes FOR EACH ROW EXECUTE FUNCTION public.delete_batch_change_reference_on_changesets();


--
-- Name: repo trig_delete_repo_ref_on_external_service_repos; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trig_delete_repo_ref_on_external_service_repos AFTER UPDATE OF deleted_at ON public.repo FOR EACH ROW EXECUTE FUNCTION public.delete_repo_ref_on_external_service_repos();


--
-- Name: user_external_accounts trig_delete_user_repo_permissions_on_external_account_soft_dele; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trig_delete_user_repo_permissions_on_external_account_soft_dele AFTER UPDATE OF deleted_at ON public.user_external_accounts FOR EACH ROW WHEN (((new.deleted_at IS NOT NULL) AND (old.deleted_at IS NULL))) EXECUTE FUNCTION public.delete_user_repo_permissions_on_external_account_soft_delete();


--
-- Name: repo trig_delete_user_repo_permissions_on_repo_soft_delete; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trig_delete_user_repo_permissions_on_repo_soft_delete AFTER UPDATE OF deleted_at ON public.repo FOR EACH ROW WHEN (((new.deleted_at IS NOT NULL) AND (old.deleted_at IS NULL))) EXECUTE FUNCTION public.delete_user_repo_permissions_on_repo_soft_delete();


--
-- Name: users trig_delete_user_repo_permissions_on_user_soft_delete; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trig_delete_user_repo_permissions_on_user_soft_delete AFTER UPDATE OF deleted_at ON public.users FOR EACH ROW WHEN (((new.deleted_at IS NOT NULL) AND (old.deleted_at IS NULL))) EXECUTE FUNCTION public.delete_user_repo_permissions_on_user_soft_delete();


--
-- Name: users trig_invalidate_session_on_password_change; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trig_invalidate_session_on_password_change BEFORE UPDATE OF passwd ON public.users FOR EACH ROW EXECUTE FUNCTION public.invalidate_session_for_userid_on_password_change();


--
-- Name: gitserver_repos trig_recalc_gitserver_repos_statistics_on_delete; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trig_recalc_gitserver_repos_statistics_on_delete AFTER DELETE ON public.gitserver_repos REFERENCING OLD TABLE AS oldtab FOR EACH STATEMENT EXECUTE FUNCTION public.recalc_gitserver_repos_statistics_on_delete();


--
-- Name: gitserver_repos trig_recalc_gitserver_repos_statistics_on_insert; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trig_recalc_gitserver_repos_statistics_on_insert AFTER INSERT ON public.gitserver_repos REFERENCING NEW TABLE AS newtab FOR EACH STATEMENT EXECUTE FUNCTION public.recalc_gitserver_repos_statistics_on_insert();


--
-- Name: gitserver_repos trig_recalc_gitserver_repos_statistics_on_update; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trig_recalc_gitserver_repos_statistics_on_update AFTER UPDATE ON public.gitserver_repos REFERENCING OLD TABLE AS oldtab NEW TABLE AS newtab FOR EACH STATEMENT EXECUTE FUNCTION public.recalc_gitserver_repos_statistics_on_update();


--
-- Name: repo trig_recalc_repo_statistics_on_repo_delete; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trig_recalc_repo_statistics_on_repo_delete AFTER DELETE ON public.repo REFERENCING OLD TABLE AS oldtab FOR EACH STATEMENT EXECUTE FUNCTION public.recalc_repo_statistics_on_repo_delete();


--
-- Name: repo trig_recalc_repo_statistics_on_repo_insert; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trig_recalc_repo_statistics_on_repo_insert AFTER INSERT ON public.repo REFERENCING NEW TABLE AS newtab FOR EACH STATEMENT EXECUTE FUNCTION public.recalc_repo_statistics_on_repo_insert();


--
-- Name: repo trig_recalc_repo_statistics_on_repo_update; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trig_recalc_repo_statistics_on_repo_update AFTER UPDATE ON public.repo REFERENCING OLD TABLE AS oldtab NEW TABLE AS newtab FOR EACH STATEMENT EXECUTE FUNCTION public.recalc_repo_statistics_on_repo_update();


--
-- Name: lsif_configuration_policies trigger_configuration_policies_delete; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_configuration_policies_delete AFTER DELETE ON public.lsif_configuration_policies REFERENCING OLD TABLE AS old FOR EACH STATEMENT EXECUTE FUNCTION public.func_configuration_policies_delete();


--
-- Name: lsif_configuration_policies trigger_configuration_policies_insert; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_configuration_policies_insert AFTER INSERT ON public.lsif_configuration_policies FOR EACH ROW EXECUTE FUNCTION public.func_configuration_policies_insert();


--
-- Name: lsif_configuration_policies trigger_configuration_policies_update; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_configuration_policies_update BEFORE UPDATE OF name, pattern, retention_enabled, retention_duration_hours, type, retain_intermediate_commits ON public.lsif_configuration_policies FOR EACH ROW EXECUTE FUNCTION public.func_configuration_policies_update();


--
-- Name: repo trigger_gitserver_repo_insert; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_gitserver_repo_insert AFTER INSERT ON public.repo FOR EACH ROW EXECUTE FUNCTION public.func_insert_gitserver_repo();


--
-- Name: lsif_uploads trigger_lsif_uploads_delete; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_lsif_uploads_delete AFTER DELETE ON public.lsif_uploads REFERENCING OLD TABLE AS old FOR EACH STATEMENT EXECUTE FUNCTION public.func_lsif_uploads_delete();


--
-- Name: lsif_uploads trigger_lsif_uploads_insert; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_lsif_uploads_insert AFTER INSERT ON public.lsif_uploads FOR EACH ROW EXECUTE FUNCTION public.func_lsif_uploads_insert();


--
-- Name: lsif_uploads trigger_lsif_uploads_update; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_lsif_uploads_update BEFORE UPDATE OF state, num_resets, num_failures, worker_hostname, expired, committed_at ON public.lsif_uploads FOR EACH ROW EXECUTE FUNCTION public.func_lsif_uploads_update();


--
-- Name: package_repo_filters trigger_package_repo_filters_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_package_repo_filters_updated_at BEFORE UPDATE ON public.package_repo_filters FOR EACH ROW WHEN ((old.* IS DISTINCT FROM new.*)) EXECUTE FUNCTION public.func_package_repo_filters_updated_at();


--
-- Name: codeintel_path_ranks update_codeintel_path_ranks_statistics; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_codeintel_path_ranks_statistics BEFORE UPDATE ON public.codeintel_path_ranks FOR EACH ROW WHEN ((new.* IS DISTINCT FROM old.*)) EXECUTE FUNCTION public.update_codeintel_path_ranks_statistics_columns();


--
-- Name: codeintel_path_ranks update_codeintel_path_ranks_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_codeintel_path_ranks_updated_at BEFORE UPDATE ON public.codeintel_path_ranks FOR EACH ROW WHEN ((new.* IS DISTINCT FROM old.*)) EXECUTE FUNCTION public.update_codeintel_path_ranks_updated_at_column();


--
-- Name: own_signal_recent_contribution update_own_aggregate_recent_contribution; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_own_aggregate_recent_contribution AFTER INSERT ON public.own_signal_recent_contribution FOR EACH ROW EXECUTE FUNCTION public.update_own_aggregate_recent_contribution();


--
-- Name: versions versions_insert; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER versions_insert BEFORE INSERT ON public.versions FOR EACH ROW EXECUTE FUNCTION public.versions_insert_row_trigger();


--
-- Name: access_requests access_requests_decision_by_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.access_requests
    ADD CONSTRAINT access_requests_decision_by_user_id_fkey FOREIGN KEY (decision_by_user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: access_requests access_requests_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.access_requests
    ADD CONSTRAINT access_requests_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: access_tokens access_tokens_creator_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.access_tokens
    ADD CONSTRAINT access_tokens_creator_user_id_fkey FOREIGN KEY (creator_user_id) REFERENCES public.users(id);


--
-- Name: access_tokens access_tokens_subject_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.access_tokens
    ADD CONSTRAINT access_tokens_subject_user_id_fkey FOREIGN KEY (subject_user_id) REFERENCES public.users(id);


--
-- Name: access_tokens access_tokens_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.access_tokens
    ADD CONSTRAINT access_tokens_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: aggregated_user_statistics aggregated_user_statistics_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.aggregated_user_statistics
    ADD CONSTRAINT aggregated_user_statistics_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: aggregated_user_statistics aggregated_user_statistics_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.aggregated_user_statistics
    ADD CONSTRAINT aggregated_user_statistics_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: assigned_owners assigned_owners_file_path_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.assigned_owners
    ADD CONSTRAINT assigned_owners_file_path_id_fkey FOREIGN KEY (file_path_id) REFERENCES public.repo_paths(id);


--
-- Name: assigned_owners assigned_owners_owner_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.assigned_owners
    ADD CONSTRAINT assigned_owners_owner_user_id_fkey FOREIGN KEY (owner_user_id) REFERENCES public.users(id) ON DELETE CASCADE DEFERRABLE;


--
-- Name: assigned_owners assigned_owners_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.assigned_owners
    ADD CONSTRAINT assigned_owners_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: assigned_owners assigned_owners_who_assigned_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.assigned_owners
    ADD CONSTRAINT assigned_owners_who_assigned_user_id_fkey FOREIGN KEY (who_assigned_user_id) REFERENCES public.users(id) ON DELETE SET NULL DEFERRABLE;


--
-- Name: assigned_teams assigned_teams_file_path_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.assigned_teams
    ADD CONSTRAINT assigned_teams_file_path_id_fkey FOREIGN KEY (file_path_id) REFERENCES public.repo_paths(id);


--
-- Name: assigned_teams assigned_teams_owner_team_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.assigned_teams
    ADD CONSTRAINT assigned_teams_owner_team_id_fkey FOREIGN KEY (owner_team_id) REFERENCES public.teams(id) ON DELETE CASCADE DEFERRABLE;


--
-- Name: assigned_teams assigned_teams_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.assigned_teams
    ADD CONSTRAINT assigned_teams_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: assigned_teams assigned_teams_who_assigned_team_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.assigned_teams
    ADD CONSTRAINT assigned_teams_who_assigned_team_id_fkey FOREIGN KEY (who_assigned_team_id) REFERENCES public.users(id) ON DELETE SET NULL DEFERRABLE;


--
-- Name: batch_changes batch_changes_batch_spec_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_changes
    ADD CONSTRAINT batch_changes_batch_spec_id_fkey FOREIGN KEY (batch_spec_id) REFERENCES public.batch_specs(id) DEFERRABLE;


--
-- Name: batch_changes batch_changes_initial_applier_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_changes
    ADD CONSTRAINT batch_changes_initial_applier_id_fkey FOREIGN KEY (creator_id) REFERENCES public.users(id) ON DELETE SET NULL DEFERRABLE;


--
-- Name: batch_changes batch_changes_last_applier_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_changes
    ADD CONSTRAINT batch_changes_last_applier_id_fkey FOREIGN KEY (last_applier_id) REFERENCES public.users(id) ON DELETE SET NULL DEFERRABLE;


--
-- Name: batch_changes batch_changes_namespace_org_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_changes
    ADD CONSTRAINT batch_changes_namespace_org_id_fkey FOREIGN KEY (namespace_org_id) REFERENCES public.orgs(id) ON DELETE CASCADE DEFERRABLE;


--
-- Name: batch_changes batch_changes_namespace_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_changes
    ADD CONSTRAINT batch_changes_namespace_user_id_fkey FOREIGN KEY (namespace_user_id) REFERENCES public.users(id) ON DELETE CASCADE DEFERRABLE;


--
-- Name: batch_changes_site_credentials batch_changes_site_credentials_github_app_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_changes_site_credentials
    ADD CONSTRAINT batch_changes_site_credentials_github_app_id_fkey FOREIGN KEY (github_app_id) REFERENCES public.github_apps(id) ON DELETE CASCADE;


--
-- Name: batch_changes_site_credentials batch_changes_site_credentials_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_changes_site_credentials
    ADD CONSTRAINT batch_changes_site_credentials_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: batch_changes batch_changes_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_changes
    ADD CONSTRAINT batch_changes_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: batch_spec_execution_cache_entries batch_spec_execution_cache_entries_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_spec_execution_cache_entries
    ADD CONSTRAINT batch_spec_execution_cache_entries_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: batch_spec_execution_cache_entries batch_spec_execution_cache_entries_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_spec_execution_cache_entries
    ADD CONSTRAINT batch_spec_execution_cache_entries_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE DEFERRABLE;


--
-- Name: batch_spec_resolution_jobs batch_spec_resolution_jobs_batch_spec_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_spec_resolution_jobs
    ADD CONSTRAINT batch_spec_resolution_jobs_batch_spec_id_fkey FOREIGN KEY (batch_spec_id) REFERENCES public.batch_specs(id) ON DELETE CASCADE DEFERRABLE;


--
-- Name: batch_spec_resolution_jobs batch_spec_resolution_jobs_initiator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_spec_resolution_jobs
    ADD CONSTRAINT batch_spec_resolution_jobs_initiator_id_fkey FOREIGN KEY (initiator_id) REFERENCES public.users(id) ON UPDATE CASCADE ON DELETE CASCADE DEFERRABLE;


--
-- Name: batch_spec_resolution_jobs batch_spec_resolution_jobs_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_spec_resolution_jobs
    ADD CONSTRAINT batch_spec_resolution_jobs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: batch_spec_workspace_execution_jobs batch_spec_workspace_execution_job_batch_spec_workspace_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_spec_workspace_execution_jobs
    ADD CONSTRAINT batch_spec_workspace_execution_job_batch_spec_workspace_id_fkey FOREIGN KEY (batch_spec_workspace_id) REFERENCES public.batch_spec_workspaces(id) ON DELETE CASCADE DEFERRABLE;


--
-- Name: batch_spec_workspace_execution_jobs batch_spec_workspace_execution_jobs_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_spec_workspace_execution_jobs
    ADD CONSTRAINT batch_spec_workspace_execution_jobs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: batch_spec_workspace_execution_last_dequeues batch_spec_workspace_execution_last_dequeues_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_spec_workspace_execution_last_dequeues
    ADD CONSTRAINT batch_spec_workspace_execution_last_dequeues_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: batch_spec_workspace_execution_last_dequeues batch_spec_workspace_execution_last_dequeues_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_spec_workspace_execution_last_dequeues
    ADD CONSTRAINT batch_spec_workspace_execution_last_dequeues_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON UPDATE CASCADE ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;


--
-- Name: batch_spec_workspace_files batch_spec_workspace_files_batch_spec_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_spec_workspace_files
    ADD CONSTRAINT batch_spec_workspace_files_batch_spec_id_fkey FOREIGN KEY (batch_spec_id) REFERENCES public.batch_specs(id) ON DELETE CASCADE;


--
-- Name: batch_spec_workspace_files batch_spec_workspace_files_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_spec_workspace_files
    ADD CONSTRAINT batch_spec_workspace_files_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: batch_spec_workspaces batch_spec_workspaces_batch_spec_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_spec_workspaces
    ADD CONSTRAINT batch_spec_workspaces_batch_spec_id_fkey FOREIGN KEY (batch_spec_id) REFERENCES public.batch_specs(id) ON DELETE CASCADE DEFERRABLE;


--
-- Name: batch_spec_workspaces batch_spec_workspaces_repo_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_spec_workspaces
    ADD CONSTRAINT batch_spec_workspaces_repo_id_fkey FOREIGN KEY (repo_id) REFERENCES public.repo(id) DEFERRABLE;


--
-- Name: batch_spec_workspaces batch_spec_workspaces_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_spec_workspaces
    ADD CONSTRAINT batch_spec_workspaces_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: batch_specs batch_specs_batch_change_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_specs
    ADD CONSTRAINT batch_specs_batch_change_id_fkey FOREIGN KEY (batch_change_id) REFERENCES public.batch_changes(id) ON DELETE SET NULL DEFERRABLE;


--
-- Name: batch_specs batch_specs_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_specs
    ADD CONSTRAINT batch_specs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: batch_specs batch_specs_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.batch_specs
    ADD CONSTRAINT batch_specs_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE SET NULL DEFERRABLE;


--
-- Name: cached_available_indexers cached_available_indexers_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cached_available_indexers
    ADD CONSTRAINT cached_available_indexers_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: changeset_events changeset_events_changeset_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changeset_events
    ADD CONSTRAINT changeset_events_changeset_id_fkey FOREIGN KEY (changeset_id) REFERENCES public.changesets(id) ON DELETE CASCADE DEFERRABLE;


--
-- Name: changeset_events changeset_events_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changeset_events
    ADD CONSTRAINT changeset_events_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: changeset_jobs changeset_jobs_batch_change_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changeset_jobs
    ADD CONSTRAINT changeset_jobs_batch_change_id_fkey FOREIGN KEY (batch_change_id) REFERENCES public.batch_changes(id) ON DELETE CASCADE DEFERRABLE;


--
-- Name: changeset_jobs changeset_jobs_changeset_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changeset_jobs
    ADD CONSTRAINT changeset_jobs_changeset_id_fkey FOREIGN KEY (changeset_id) REFERENCES public.changesets(id) ON DELETE CASCADE DEFERRABLE;


--
-- Name: changeset_jobs changeset_jobs_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changeset_jobs
    ADD CONSTRAINT changeset_jobs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: changeset_jobs changeset_jobs_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changeset_jobs
    ADD CONSTRAINT changeset_jobs_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE DEFERRABLE;


--
-- Name: changeset_specs changeset_specs_batch_spec_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changeset_specs
    ADD CONSTRAINT changeset_specs_batch_spec_id_fkey FOREIGN KEY (batch_spec_id) REFERENCES public.batch_specs(id) ON DELETE CASCADE DEFERRABLE;


--
-- Name: changeset_specs changeset_specs_repo_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changeset_specs
    ADD CONSTRAINT changeset_specs_repo_id_fkey FOREIGN KEY (repo_id) REFERENCES public.repo(id) DEFERRABLE;


--
-- Name: changeset_specs changeset_specs_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changeset_specs
    ADD CONSTRAINT changeset_specs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: changeset_specs changeset_specs_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changeset_specs
    ADD CONSTRAINT changeset_specs_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE SET NULL DEFERRABLE;


--
-- Name: changesets changesets_changeset_spec_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changesets
    ADD CONSTRAINT changesets_changeset_spec_id_fkey FOREIGN KEY (current_spec_id) REFERENCES public.changeset_specs(id) DEFERRABLE;


--
-- Name: changesets changesets_owned_by_batch_spec_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changesets
    ADD CONSTRAINT changesets_owned_by_batch_spec_id_fkey FOREIGN KEY (owned_by_batch_change_id) REFERENCES public.batch_changes(id) ON DELETE SET NULL DEFERRABLE;


--
-- Name: changesets changesets_previous_spec_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changesets
    ADD CONSTRAINT changesets_previous_spec_id_fkey FOREIGN KEY (previous_spec_id) REFERENCES public.changeset_specs(id) DEFERRABLE;


--
-- Name: changesets changesets_repo_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changesets
    ADD CONSTRAINT changesets_repo_id_fkey FOREIGN KEY (repo_id) REFERENCES public.repo(id) ON DELETE CASCADE DEFERRABLE;


--
-- Name: changesets changesets_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changesets
    ADD CONSTRAINT changesets_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: cm_action_jobs cm_action_jobs_email_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_action_jobs
    ADD CONSTRAINT cm_action_jobs_email_fk FOREIGN KEY (email) REFERENCES public.cm_emails(id) ON DELETE CASCADE;


--
-- Name: cm_action_jobs cm_action_jobs_slack_webhook_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_action_jobs
    ADD CONSTRAINT cm_action_jobs_slack_webhook_fkey FOREIGN KEY (slack_webhook) REFERENCES public.cm_slack_webhooks(id) ON DELETE CASCADE;


--
-- Name: cm_action_jobs cm_action_jobs_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_action_jobs
    ADD CONSTRAINT cm_action_jobs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: cm_action_jobs cm_action_jobs_trigger_event_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_action_jobs
    ADD CONSTRAINT cm_action_jobs_trigger_event_fk FOREIGN KEY (trigger_event) REFERENCES public.cm_trigger_jobs(id) ON DELETE CASCADE;


--
-- Name: cm_action_jobs cm_action_jobs_webhook_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_action_jobs
    ADD CONSTRAINT cm_action_jobs_webhook_fkey FOREIGN KEY (webhook) REFERENCES public.cm_webhooks(id) ON DELETE CASCADE;


--
-- Name: cm_emails cm_emails_changed_by_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_emails
    ADD CONSTRAINT cm_emails_changed_by_fk FOREIGN KEY (changed_by) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: cm_emails cm_emails_created_by_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_emails
    ADD CONSTRAINT cm_emails_created_by_fk FOREIGN KEY (created_by) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: cm_emails cm_emails_monitor; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_emails
    ADD CONSTRAINT cm_emails_monitor FOREIGN KEY (monitor) REFERENCES public.cm_monitors(id) ON DELETE CASCADE;


--
-- Name: cm_emails cm_emails_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_emails
    ADD CONSTRAINT cm_emails_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: cm_last_searched cm_last_searched_monitor_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_last_searched
    ADD CONSTRAINT cm_last_searched_monitor_id_fkey FOREIGN KEY (monitor_id) REFERENCES public.cm_monitors(id) ON DELETE CASCADE;


--
-- Name: cm_last_searched cm_last_searched_repo_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_last_searched
    ADD CONSTRAINT cm_last_searched_repo_id_fkey FOREIGN KEY (repo_id) REFERENCES public.repo(id) ON DELETE CASCADE;


--
-- Name: cm_last_searched cm_last_searched_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_last_searched
    ADD CONSTRAINT cm_last_searched_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: cm_monitors cm_monitors_changed_by_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_monitors
    ADD CONSTRAINT cm_monitors_changed_by_fk FOREIGN KEY (changed_by) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: cm_monitors cm_monitors_created_by_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_monitors
    ADD CONSTRAINT cm_monitors_created_by_fk FOREIGN KEY (created_by) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: cm_monitors cm_monitors_org_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_monitors
    ADD CONSTRAINT cm_monitors_org_id_fk FOREIGN KEY (namespace_org_id) REFERENCES public.orgs(id) ON DELETE CASCADE;


--
-- Name: cm_monitors cm_monitors_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_monitors
    ADD CONSTRAINT cm_monitors_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: cm_monitors cm_monitors_user_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_monitors
    ADD CONSTRAINT cm_monitors_user_id_fk FOREIGN KEY (namespace_user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: cm_queries cm_queries_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_queries
    ADD CONSTRAINT cm_queries_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: cm_recipients cm_recipients_emails; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_recipients
    ADD CONSTRAINT cm_recipients_emails FOREIGN KEY (email) REFERENCES public.cm_emails(id) ON DELETE CASCADE;


--
-- Name: cm_recipients cm_recipients_org_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_recipients
    ADD CONSTRAINT cm_recipients_org_id_fk FOREIGN KEY (namespace_org_id) REFERENCES public.orgs(id) ON DELETE CASCADE;


--
-- Name: cm_recipients cm_recipients_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_recipients
    ADD CONSTRAINT cm_recipients_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: cm_recipients cm_recipients_user_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_recipients
    ADD CONSTRAINT cm_recipients_user_id_fk FOREIGN KEY (namespace_user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: cm_slack_webhooks cm_slack_webhooks_changed_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_slack_webhooks
    ADD CONSTRAINT cm_slack_webhooks_changed_by_fkey FOREIGN KEY (changed_by) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: cm_slack_webhooks cm_slack_webhooks_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_slack_webhooks
    ADD CONSTRAINT cm_slack_webhooks_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: cm_slack_webhooks cm_slack_webhooks_monitor_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_slack_webhooks
    ADD CONSTRAINT cm_slack_webhooks_monitor_fkey FOREIGN KEY (monitor) REFERENCES public.cm_monitors(id) ON DELETE CASCADE;


--
-- Name: cm_slack_webhooks cm_slack_webhooks_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_slack_webhooks
    ADD CONSTRAINT cm_slack_webhooks_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: cm_trigger_jobs cm_trigger_jobs_query_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_trigger_jobs
    ADD CONSTRAINT cm_trigger_jobs_query_fk FOREIGN KEY (query) REFERENCES public.cm_queries(id) ON DELETE CASCADE;


--
-- Name: cm_trigger_jobs cm_trigger_jobs_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_trigger_jobs
    ADD CONSTRAINT cm_trigger_jobs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: cm_queries cm_triggers_changed_by_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_queries
    ADD CONSTRAINT cm_triggers_changed_by_fk FOREIGN KEY (changed_by) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: cm_queries cm_triggers_created_by_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_queries
    ADD CONSTRAINT cm_triggers_created_by_fk FOREIGN KEY (created_by) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: cm_queries cm_triggers_monitor; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_queries
    ADD CONSTRAINT cm_triggers_monitor FOREIGN KEY (monitor) REFERENCES public.cm_monitors(id) ON DELETE CASCADE;


--
-- Name: cm_webhooks cm_webhooks_changed_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_webhooks
    ADD CONSTRAINT cm_webhooks_changed_by_fkey FOREIGN KEY (changed_by) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: cm_webhooks cm_webhooks_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_webhooks
    ADD CONSTRAINT cm_webhooks_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: cm_webhooks cm_webhooks_monitor_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_webhooks
    ADD CONSTRAINT cm_webhooks_monitor_fkey FOREIGN KEY (monitor) REFERENCES public.cm_monitors(id) ON DELETE CASCADE;


--
-- Name: cm_webhooks cm_webhooks_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cm_webhooks
    ADD CONSTRAINT cm_webhooks_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: code_hosts code_hosts_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.code_hosts
    ADD CONSTRAINT code_hosts_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: codeintel_autoindex_queue codeintel_autoindex_queue_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_autoindex_queue
    ADD CONSTRAINT codeintel_autoindex_queue_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: codeintel_autoindexing_exceptions codeintel_autoindexing_exceptions_repository_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_autoindexing_exceptions
    ADD CONSTRAINT codeintel_autoindexing_exceptions_repository_id_fkey FOREIGN KEY (repository_id) REFERENCES public.repo(id) ON DELETE CASCADE;


--
-- Name: codeintel_autoindexing_exceptions codeintel_autoindexing_exceptions_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_autoindexing_exceptions
    ADD CONSTRAINT codeintel_autoindexing_exceptions_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: codeintel_commit_dates codeintel_commit_dates_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_commit_dates
    ADD CONSTRAINT codeintel_commit_dates_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: codeintel_inference_scripts codeintel_inference_scripts_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_inference_scripts
    ADD CONSTRAINT codeintel_inference_scripts_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: codeintel_initial_path_ranks codeintel_initial_path_ranks_exported_upload_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_initial_path_ranks
    ADD CONSTRAINT codeintel_initial_path_ranks_exported_upload_id_fkey FOREIGN KEY (exported_upload_id) REFERENCES public.codeintel_ranking_exports(id) ON DELETE CASCADE;


--
-- Name: codeintel_initial_path_ranks_processed codeintel_initial_path_ranks_processed_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_initial_path_ranks_processed
    ADD CONSTRAINT codeintel_initial_path_ranks_processed_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: codeintel_initial_path_ranks codeintel_initial_path_ranks_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_initial_path_ranks
    ADD CONSTRAINT codeintel_initial_path_ranks_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: codeintel_langugage_support_requests codeintel_langugage_support_requests_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_langugage_support_requests
    ADD CONSTRAINT codeintel_langugage_support_requests_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: codeintel_path_ranks codeintel_path_ranks_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_path_ranks
    ADD CONSTRAINT codeintel_path_ranks_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: codeintel_ranking_definitions codeintel_ranking_definitions_exported_upload_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_ranking_definitions
    ADD CONSTRAINT codeintel_ranking_definitions_exported_upload_id_fkey FOREIGN KEY (exported_upload_id) REFERENCES public.codeintel_ranking_exports(id) ON DELETE CASCADE;


--
-- Name: codeintel_ranking_definitions codeintel_ranking_definitions_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_ranking_definitions
    ADD CONSTRAINT codeintel_ranking_definitions_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: codeintel_ranking_exports codeintel_ranking_exports_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_ranking_exports
    ADD CONSTRAINT codeintel_ranking_exports_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: codeintel_ranking_exports codeintel_ranking_exports_upload_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_ranking_exports
    ADD CONSTRAINT codeintel_ranking_exports_upload_id_fkey FOREIGN KEY (upload_id) REFERENCES public.lsif_uploads(id) ON DELETE SET NULL;


--
-- Name: codeintel_ranking_graph_keys codeintel_ranking_graph_keys_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_ranking_graph_keys
    ADD CONSTRAINT codeintel_ranking_graph_keys_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: codeintel_ranking_path_counts_inputs codeintel_ranking_path_counts_inputs_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_ranking_path_counts_inputs
    ADD CONSTRAINT codeintel_ranking_path_counts_inputs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: codeintel_ranking_progress codeintel_ranking_progress_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_ranking_progress
    ADD CONSTRAINT codeintel_ranking_progress_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: codeintel_ranking_references codeintel_ranking_references_exported_upload_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_ranking_references
    ADD CONSTRAINT codeintel_ranking_references_exported_upload_id_fkey FOREIGN KEY (exported_upload_id) REFERENCES public.codeintel_ranking_exports(id) ON DELETE CASCADE;


--
-- Name: codeintel_ranking_references_processed codeintel_ranking_references_processed_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_ranking_references_processed
    ADD CONSTRAINT codeintel_ranking_references_processed_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: codeintel_ranking_references codeintel_ranking_references_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_ranking_references
    ADD CONSTRAINT codeintel_ranking_references_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: codeowners_individual_stats codeowners_individual_stats_file_path_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeowners_individual_stats
    ADD CONSTRAINT codeowners_individual_stats_file_path_id_fkey FOREIGN KEY (file_path_id) REFERENCES public.repo_paths(id);


--
-- Name: codeowners_individual_stats codeowners_individual_stats_owner_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeowners_individual_stats
    ADD CONSTRAINT codeowners_individual_stats_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES public.codeowners_owners(id);


--
-- Name: codeowners_individual_stats codeowners_individual_stats_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeowners_individual_stats
    ADD CONSTRAINT codeowners_individual_stats_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: codeowners_owners codeowners_owners_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeowners_owners
    ADD CONSTRAINT codeowners_owners_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: codeowners codeowners_repo_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeowners
    ADD CONSTRAINT codeowners_repo_id_fkey FOREIGN KEY (repo_id) REFERENCES public.repo(id) ON DELETE CASCADE;


--
-- Name: codeowners codeowners_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeowners
    ADD CONSTRAINT codeowners_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: commit_authors commit_authors_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.commit_authors
    ADD CONSTRAINT commit_authors_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: configuration_policies_audit_logs configuration_policies_audit_logs_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.configuration_policies_audit_logs
    ADD CONSTRAINT configuration_policies_audit_logs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: context_detection_embedding_jobs context_detection_embedding_jobs_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.context_detection_embedding_jobs
    ADD CONSTRAINT context_detection_embedding_jobs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: discussion_comments discussion_comments_author_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.discussion_comments
    ADD CONSTRAINT discussion_comments_author_user_id_fkey FOREIGN KEY (author_user_id) REFERENCES public.users(id) ON DELETE RESTRICT;


--
-- Name: discussion_comments discussion_comments_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.discussion_comments
    ADD CONSTRAINT discussion_comments_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: discussion_comments discussion_comments_thread_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.discussion_comments
    ADD CONSTRAINT discussion_comments_thread_id_fkey FOREIGN KEY (thread_id) REFERENCES public.discussion_threads(id) ON DELETE CASCADE;


--
-- Name: discussion_mail_reply_tokens discussion_mail_reply_tokens_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.discussion_mail_reply_tokens
    ADD CONSTRAINT discussion_mail_reply_tokens_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: discussion_mail_reply_tokens discussion_mail_reply_tokens_thread_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.discussion_mail_reply_tokens
    ADD CONSTRAINT discussion_mail_reply_tokens_thread_id_fkey FOREIGN KEY (thread_id) REFERENCES public.discussion_threads(id) ON DELETE CASCADE;


--
-- Name: discussion_mail_reply_tokens discussion_mail_reply_tokens_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.discussion_mail_reply_tokens
    ADD CONSTRAINT discussion_mail_reply_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE RESTRICT;


--
-- Name: discussion_threads discussion_threads_author_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.discussion_threads
    ADD CONSTRAINT discussion_threads_author_user_id_fkey FOREIGN KEY (author_user_id) REFERENCES public.users(id) ON DELETE RESTRICT;


--
-- Name: discussion_threads discussion_threads_target_repo_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.discussion_threads
    ADD CONSTRAINT discussion_threads_target_repo_id_fk FOREIGN KEY (target_repo_id) REFERENCES public.discussion_threads_target_repo(id) ON DELETE CASCADE;


--
-- Name: discussion_threads_target_repo discussion_threads_target_repo_repo_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.discussion_threads_target_repo
    ADD CONSTRAINT discussion_threads_target_repo_repo_id_fkey FOREIGN KEY (repo_id) REFERENCES public.repo(id) ON DELETE CASCADE;


--
-- Name: discussion_threads_target_repo discussion_threads_target_repo_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.discussion_threads_target_repo
    ADD CONSTRAINT discussion_threads_target_repo_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: discussion_threads_target_repo discussion_threads_target_repo_thread_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.discussion_threads_target_repo
    ADD CONSTRAINT discussion_threads_target_repo_thread_id_fkey FOREIGN KEY (thread_id) REFERENCES public.discussion_threads(id) ON DELETE CASCADE;


--
-- Name: discussion_threads discussion_threads_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.discussion_threads
    ADD CONSTRAINT discussion_threads_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: event_logs_export_allowlist event_logs_export_allowlist_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.event_logs_export_allowlist
    ADD CONSTRAINT event_logs_export_allowlist_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: event_logs_scrape_state_own event_logs_scrape_state_own_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.event_logs_scrape_state_own
    ADD CONSTRAINT event_logs_scrape_state_own_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: event_logs_scrape_state event_logs_scrape_state_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.event_logs_scrape_state
    ADD CONSTRAINT event_logs_scrape_state_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: event_logs event_logs_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.event_logs
    ADD CONSTRAINT event_logs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: executor_heartbeats executor_heartbeats_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.executor_heartbeats
    ADD CONSTRAINT executor_heartbeats_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: executor_job_tokens executor_job_tokens_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.executor_job_tokens
    ADD CONSTRAINT executor_job_tokens_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: executor_secret_access_logs executor_secret_access_logs_executor_secret_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.executor_secret_access_logs
    ADD CONSTRAINT executor_secret_access_logs_executor_secret_id_fkey FOREIGN KEY (executor_secret_id) REFERENCES public.executor_secrets(id) ON DELETE CASCADE;


--
-- Name: executor_secret_access_logs executor_secret_access_logs_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.executor_secret_access_logs
    ADD CONSTRAINT executor_secret_access_logs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: executor_secret_access_logs executor_secret_access_logs_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.executor_secret_access_logs
    ADD CONSTRAINT executor_secret_access_logs_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: executor_secrets executor_secrets_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.executor_secrets
    ADD CONSTRAINT executor_secrets_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: executor_secrets executor_secrets_namespace_org_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.executor_secrets
    ADD CONSTRAINT executor_secrets_namespace_org_id_fkey FOREIGN KEY (namespace_org_id) REFERENCES public.orgs(id) ON DELETE CASCADE;


--
-- Name: executor_secrets executor_secrets_namespace_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.executor_secrets
    ADD CONSTRAINT executor_secrets_namespace_user_id_fkey FOREIGN KEY (namespace_user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: executor_secrets executor_secrets_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.executor_secrets
    ADD CONSTRAINT executor_secrets_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: exhaustive_search_jobs exhaustive_search_jobs_initiator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.exhaustive_search_jobs
    ADD CONSTRAINT exhaustive_search_jobs_initiator_id_fkey FOREIGN KEY (initiator_id) REFERENCES public.users(id) ON UPDATE CASCADE ON DELETE CASCADE DEFERRABLE;


--
-- Name: exhaustive_search_jobs exhaustive_search_jobs_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.exhaustive_search_jobs
    ADD CONSTRAINT exhaustive_search_jobs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: exhaustive_search_repo_jobs exhaustive_search_repo_jobs_repo_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.exhaustive_search_repo_jobs
    ADD CONSTRAINT exhaustive_search_repo_jobs_repo_id_fkey FOREIGN KEY (repo_id) REFERENCES public.repo(id) ON DELETE CASCADE;


--
-- Name: exhaustive_search_repo_jobs exhaustive_search_repo_jobs_search_job_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.exhaustive_search_repo_jobs
    ADD CONSTRAINT exhaustive_search_repo_jobs_search_job_id_fkey FOREIGN KEY (search_job_id) REFERENCES public.exhaustive_search_jobs(id) ON DELETE CASCADE;


--
-- Name: exhaustive_search_repo_jobs exhaustive_search_repo_jobs_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.exhaustive_search_repo_jobs
    ADD CONSTRAINT exhaustive_search_repo_jobs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: exhaustive_search_repo_revision_jobs exhaustive_search_repo_revision_jobs_search_repo_job_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.exhaustive_search_repo_revision_jobs
    ADD CONSTRAINT exhaustive_search_repo_revision_jobs_search_repo_job_id_fkey FOREIGN KEY (search_repo_job_id) REFERENCES public.exhaustive_search_repo_jobs(id) ON DELETE CASCADE;


--
-- Name: exhaustive_search_repo_revision_jobs exhaustive_search_repo_revision_jobs_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.exhaustive_search_repo_revision_jobs
    ADD CONSTRAINT exhaustive_search_repo_revision_jobs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: explicit_permissions_bitbucket_projects_jobs explicit_permissions_bitbucket_projects_jobs_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.explicit_permissions_bitbucket_projects_jobs
    ADD CONSTRAINT explicit_permissions_bitbucket_projects_jobs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: external_service_repos external_service_repos_external_service_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.external_service_repos
    ADD CONSTRAINT external_service_repos_external_service_id_fkey FOREIGN KEY (external_service_id) REFERENCES public.external_services(id) ON DELETE CASCADE DEFERRABLE;


--
-- Name: external_service_repos external_service_repos_repo_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.external_service_repos
    ADD CONSTRAINT external_service_repos_repo_id_fkey FOREIGN KEY (repo_id) REFERENCES public.repo(id) ON DELETE CASCADE DEFERRABLE;


--
-- Name: external_service_repos external_service_repos_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.external_service_repos
    ADD CONSTRAINT external_service_repos_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: external_service_sync_jobs external_service_sync_jobs_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.external_service_sync_jobs
    ADD CONSTRAINT external_service_sync_jobs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: external_services external_services_code_host_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.external_services
    ADD CONSTRAINT external_services_code_host_id_fkey FOREIGN KEY (code_host_id) REFERENCES public.code_hosts(id) ON UPDATE CASCADE ON DELETE SET NULL DEFERRABLE INITIALLY DEFERRED;


--
-- Name: external_services external_services_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.external_services
    ADD CONSTRAINT external_services_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.users(id) ON DELETE SET NULL DEFERRABLE;


--
-- Name: external_service_sync_jobs external_services_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.external_service_sync_jobs
    ADD CONSTRAINT external_services_id_fk FOREIGN KEY (external_service_id) REFERENCES public.external_services(id) ON DELETE CASCADE;


--
-- Name: external_services external_services_last_updater_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.external_services
    ADD CONSTRAINT external_services_last_updater_id_fkey FOREIGN KEY (last_updater_id) REFERENCES public.users(id) ON DELETE SET NULL DEFERRABLE;


--
-- Name: external_services external_services_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.external_services
    ADD CONSTRAINT external_services_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: feature_flag_overrides feature_flag_overrides_namespace_org_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feature_flag_overrides
    ADD CONSTRAINT feature_flag_overrides_namespace_org_id_fkey FOREIGN KEY (namespace_org_id) REFERENCES public.orgs(id) ON DELETE CASCADE;


--
-- Name: feature_flag_overrides feature_flag_overrides_namespace_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feature_flag_overrides
    ADD CONSTRAINT feature_flag_overrides_namespace_user_id_fkey FOREIGN KEY (namespace_user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: feature_flag_overrides feature_flag_overrides_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feature_flag_overrides
    ADD CONSTRAINT feature_flag_overrides_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: feature_flags feature_flags_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feature_flags
    ADD CONSTRAINT feature_flags_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: codeintel_initial_path_ranks_processed fk_codeintel_initial_path_ranks; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_initial_path_ranks_processed
    ADD CONSTRAINT fk_codeintel_initial_path_ranks FOREIGN KEY (codeintel_initial_path_ranks_id) REFERENCES public.codeintel_initial_path_ranks(id) ON DELETE CASCADE;


--
-- Name: codeintel_ranking_references_processed fk_codeintel_ranking_reference; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.codeintel_ranking_references_processed
    ADD CONSTRAINT fk_codeintel_ranking_reference FOREIGN KEY (codeintel_ranking_reference_id) REFERENCES public.codeintel_ranking_references(id) ON DELETE CASCADE;


--
-- Name: vulnerability_matches fk_upload; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vulnerability_matches
    ADD CONSTRAINT fk_upload FOREIGN KEY (upload_id) REFERENCES public.lsif_uploads(id) ON DELETE CASCADE;


--
-- Name: lsif_uploads_vulnerability_scan fk_upload_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lsif_uploads_vulnerability_scan
    ADD CONSTRAINT fk_upload_id FOREIGN KEY (upload_id) REFERENCES public.lsif_uploads(id) ON DELETE CASCADE;


--
-- Name: vulnerability_affected_packages fk_vulnerabilities; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vulnerability_affected_packages
    ADD CONSTRAINT fk_vulnerabilities FOREIGN KEY (vulnerability_id) REFERENCES public.vulnerabilities(id) ON DELETE CASCADE;


--
-- Name: vulnerability_affected_symbols fk_vulnerability_affected_packages; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vulnerability_affected_symbols
    ADD CONSTRAINT fk_vulnerability_affected_packages FOREIGN KEY (vulnerability_affected_package_id) REFERENCES public.vulnerability_affected_packages(id) ON DELETE CASCADE;


--
-- Name: vulnerability_matches fk_vulnerability_affected_packages; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vulnerability_matches
    ADD CONSTRAINT fk_vulnerability_affected_packages FOREIGN KEY (vulnerability_affected_package_id) REFERENCES public.vulnerability_affected_packages(id) ON DELETE CASCADE;


--
-- Name: github_app_installs github_app_installs_app_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_app_installs
    ADD CONSTRAINT github_app_installs_app_id_fkey FOREIGN KEY (app_id) REFERENCES public.github_apps(id) ON DELETE CASCADE;


--
-- Name: github_app_installs github_app_installs_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_app_installs
    ADD CONSTRAINT github_app_installs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: github_apps github_apps_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_apps
    ADD CONSTRAINT github_apps_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: github_apps github_apps_webhook_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_apps
    ADD CONSTRAINT github_apps_webhook_id_fkey FOREIGN KEY (webhook_id) REFERENCES public.webhooks(id) ON DELETE SET NULL;


--
-- Name: gitserver_relocator_jobs gitserver_relocator_jobs_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.gitserver_relocator_jobs
    ADD CONSTRAINT gitserver_relocator_jobs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: gitserver_repos gitserver_repos_repo_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.gitserver_repos
    ADD CONSTRAINT gitserver_repos_repo_id_fkey FOREIGN KEY (repo_id) REFERENCES public.repo(id) ON DELETE CASCADE;


--
-- Name: gitserver_repos_statistics gitserver_repos_statistics_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.gitserver_repos_statistics
    ADD CONSTRAINT gitserver_repos_statistics_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: gitserver_repos_sync_output gitserver_repos_sync_output_repo_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.gitserver_repos_sync_output
    ADD CONSTRAINT gitserver_repos_sync_output_repo_id_fkey FOREIGN KEY (repo_id) REFERENCES public.repo(id) ON DELETE CASCADE;


--
-- Name: gitserver_repos_sync_output gitserver_repos_sync_output_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.gitserver_repos_sync_output
    ADD CONSTRAINT gitserver_repos_sync_output_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: gitserver_repos gitserver_repos_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.gitserver_repos
    ADD CONSTRAINT gitserver_repos_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: global_state global_state_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.global_state
    ADD CONSTRAINT global_state_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: insights_query_runner_jobs_dependencies insights_query_runner_jobs_dependencies_fk_job_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.insights_query_runner_jobs_dependencies
    ADD CONSTRAINT insights_query_runner_jobs_dependencies_fk_job_id FOREIGN KEY (job_id) REFERENCES public.insights_query_runner_jobs(id) ON DELETE CASCADE;


--
-- Name: insights_query_runner_jobs_dependencies insights_query_runner_jobs_dependencies_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.insights_query_runner_jobs_dependencies
    ADD CONSTRAINT insights_query_runner_jobs_dependencies_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: insights_query_runner_jobs insights_query_runner_jobs_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.insights_query_runner_jobs
    ADD CONSTRAINT insights_query_runner_jobs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: insights_settings_migration_jobs insights_settings_migration_jobs_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.insights_settings_migration_jobs
    ADD CONSTRAINT insights_settings_migration_jobs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: lsif_dependency_syncing_jobs lsif_dependency_indexing_jobs_upload_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lsif_dependency_syncing_jobs
    ADD CONSTRAINT lsif_dependency_indexing_jobs_upload_id_fkey FOREIGN KEY (upload_id) REFERENCES public.lsif_uploads(id) ON DELETE CASCADE;


--
-- Name: lsif_dependency_indexing_jobs lsif_dependency_indexing_jobs_upload_id_fkey1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lsif_dependency_indexing_jobs
    ADD CONSTRAINT lsif_dependency_indexing_jobs_upload_id_fkey1 FOREIGN KEY (upload_id) REFERENCES public.lsif_uploads(id) ON DELETE CASCADE;


--
-- Name: lsif_index_configuration lsif_index_configuration_repository_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lsif_index_configuration
    ADD CONSTRAINT lsif_index_configuration_repository_id_fkey FOREIGN KEY (repository_id) REFERENCES public.repo(id) ON DELETE CASCADE;


--
-- Name: lsif_packages lsif_packages_dump_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lsif_packages
    ADD CONSTRAINT lsif_packages_dump_id_fkey FOREIGN KEY (dump_id) REFERENCES public.lsif_uploads(id) ON DELETE CASCADE;


--
-- Name: lsif_references lsif_references_dump_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lsif_references
    ADD CONSTRAINT lsif_references_dump_id_fkey FOREIGN KEY (dump_id) REFERENCES public.lsif_uploads(id) ON DELETE CASCADE;


--
-- Name: lsif_retention_configuration lsif_retention_configuration_repository_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lsif_retention_configuration
    ADD CONSTRAINT lsif_retention_configuration_repository_id_fkey FOREIGN KEY (repository_id) REFERENCES public.repo(id) ON DELETE CASCADE;


--
-- Name: lsif_uploads_reference_counts lsif_uploads_reference_counts_upload_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lsif_uploads_reference_counts
    ADD CONSTRAINT lsif_uploads_reference_counts_upload_id_fk FOREIGN KEY (upload_id) REFERENCES public.lsif_uploads(id) ON DELETE CASCADE;


--
-- Name: names names_org_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.names
    ADD CONSTRAINT names_org_id_fkey FOREIGN KEY (org_id) REFERENCES public.orgs(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: names names_team_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.names
    ADD CONSTRAINT names_team_id_fkey FOREIGN KEY (team_id) REFERENCES public.teams(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: names names_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.names
    ADD CONSTRAINT names_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: names names_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.names
    ADD CONSTRAINT names_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: namespace_permissions namespace_permissions_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.namespace_permissions
    ADD CONSTRAINT namespace_permissions_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: namespace_permissions namespace_permissions_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.namespace_permissions
    ADD CONSTRAINT namespace_permissions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE DEFERRABLE;


--
-- Name: notebook_stars notebook_stars_notebook_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notebook_stars
    ADD CONSTRAINT notebook_stars_notebook_id_fkey FOREIGN KEY (notebook_id) REFERENCES public.notebooks(id) ON DELETE CASCADE DEFERRABLE;


--
-- Name: notebook_stars notebook_stars_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notebook_stars
    ADD CONSTRAINT notebook_stars_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: notebook_stars notebook_stars_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notebook_stars
    ADD CONSTRAINT notebook_stars_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE DEFERRABLE;


--
-- Name: notebooks notebooks_creator_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notebooks
    ADD CONSTRAINT notebooks_creator_user_id_fkey FOREIGN KEY (creator_user_id) REFERENCES public.users(id) ON DELETE SET NULL DEFERRABLE;


--
-- Name: notebooks notebooks_namespace_org_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notebooks
    ADD CONSTRAINT notebooks_namespace_org_id_fkey FOREIGN KEY (namespace_org_id) REFERENCES public.orgs(id) ON DELETE SET NULL DEFERRABLE;


--
-- Name: notebooks notebooks_namespace_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notebooks
    ADD CONSTRAINT notebooks_namespace_user_id_fkey FOREIGN KEY (namespace_user_id) REFERENCES public.users(id) ON DELETE SET NULL DEFERRABLE;


--
-- Name: notebooks notebooks_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notebooks
    ADD CONSTRAINT notebooks_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: notebooks notebooks_updater_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notebooks
    ADD CONSTRAINT notebooks_updater_user_id_fkey FOREIGN KEY (updater_user_id) REFERENCES public.users(id) ON DELETE SET NULL DEFERRABLE;


--
-- Name: org_invitations org_invitations_org_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.org_invitations
    ADD CONSTRAINT org_invitations_org_id_fkey FOREIGN KEY (org_id) REFERENCES public.orgs(id);


--
-- Name: org_invitations org_invitations_recipient_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.org_invitations
    ADD CONSTRAINT org_invitations_recipient_user_id_fkey FOREIGN KEY (recipient_user_id) REFERENCES public.users(id);


--
-- Name: org_invitations org_invitations_sender_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.org_invitations
    ADD CONSTRAINT org_invitations_sender_user_id_fkey FOREIGN KEY (sender_user_id) REFERENCES public.users(id);


--
-- Name: org_invitations org_invitations_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.org_invitations
    ADD CONSTRAINT org_invitations_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: org_members org_members_references_orgs; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.org_members
    ADD CONSTRAINT org_members_references_orgs FOREIGN KEY (org_id) REFERENCES public.orgs(id) ON DELETE RESTRICT;


--
-- Name: org_members org_members_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.org_members
    ADD CONSTRAINT org_members_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: org_members org_members_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.org_members
    ADD CONSTRAINT org_members_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE RESTRICT;


--
-- Name: org_stats org_stats_org_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.org_stats
    ADD CONSTRAINT org_stats_org_id_fkey FOREIGN KEY (org_id) REFERENCES public.orgs(id) ON DELETE CASCADE DEFERRABLE;


--
-- Name: org_stats org_stats_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.org_stats
    ADD CONSTRAINT org_stats_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: orgs_open_beta_stats orgs_open_beta_stats_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.orgs_open_beta_stats
    ADD CONSTRAINT orgs_open_beta_stats_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: orgs orgs_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.orgs
    ADD CONSTRAINT orgs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: out_of_band_migrations_errors out_of_band_migrations_errors_migration_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.out_of_band_migrations_errors
    ADD CONSTRAINT out_of_band_migrations_errors_migration_id_fkey FOREIGN KEY (migration_id) REFERENCES public.out_of_band_migrations(id) ON DELETE CASCADE;


--
-- Name: out_of_band_migrations_errors out_of_band_migrations_errors_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.out_of_band_migrations_errors
    ADD CONSTRAINT out_of_band_migrations_errors_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: out_of_band_migrations out_of_band_migrations_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.out_of_band_migrations
    ADD CONSTRAINT out_of_band_migrations_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: outbound_webhook_event_types outbound_webhook_event_types_outbound_webhook_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.outbound_webhook_event_types
    ADD CONSTRAINT outbound_webhook_event_types_outbound_webhook_id_fkey FOREIGN KEY (outbound_webhook_id) REFERENCES public.outbound_webhooks(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: outbound_webhook_event_types outbound_webhook_event_types_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.outbound_webhook_event_types
    ADD CONSTRAINT outbound_webhook_event_types_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: outbound_webhook_jobs outbound_webhook_jobs_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.outbound_webhook_jobs
    ADD CONSTRAINT outbound_webhook_jobs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: outbound_webhook_logs outbound_webhook_logs_job_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.outbound_webhook_logs
    ADD CONSTRAINT outbound_webhook_logs_job_id_fkey FOREIGN KEY (job_id) REFERENCES public.outbound_webhook_jobs(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: outbound_webhook_logs outbound_webhook_logs_outbound_webhook_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.outbound_webhook_logs
    ADD CONSTRAINT outbound_webhook_logs_outbound_webhook_id_fkey FOREIGN KEY (outbound_webhook_id) REFERENCES public.outbound_webhooks(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: outbound_webhook_logs outbound_webhook_logs_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.outbound_webhook_logs
    ADD CONSTRAINT outbound_webhook_logs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: outbound_webhooks outbound_webhooks_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.outbound_webhooks
    ADD CONSTRAINT outbound_webhooks_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: outbound_webhooks outbound_webhooks_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.outbound_webhooks
    ADD CONSTRAINT outbound_webhooks_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: outbound_webhooks outbound_webhooks_updated_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.outbound_webhooks
    ADD CONSTRAINT outbound_webhooks_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: own_aggregate_recent_contribution own_aggregate_recent_contribution_changed_file_path_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.own_aggregate_recent_contribution
    ADD CONSTRAINT own_aggregate_recent_contribution_changed_file_path_id_fkey FOREIGN KEY (changed_file_path_id) REFERENCES public.repo_paths(id);


--
-- Name: own_aggregate_recent_contribution own_aggregate_recent_contribution_commit_author_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.own_aggregate_recent_contribution
    ADD CONSTRAINT own_aggregate_recent_contribution_commit_author_id_fkey FOREIGN KEY (commit_author_id) REFERENCES public.commit_authors(id);


--
-- Name: own_aggregate_recent_contribution own_aggregate_recent_contribution_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.own_aggregate_recent_contribution
    ADD CONSTRAINT own_aggregate_recent_contribution_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: own_aggregate_recent_view own_aggregate_recent_view_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.own_aggregate_recent_view
    ADD CONSTRAINT own_aggregate_recent_view_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: own_aggregate_recent_view own_aggregate_recent_view_viewed_file_path_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.own_aggregate_recent_view
    ADD CONSTRAINT own_aggregate_recent_view_viewed_file_path_id_fkey FOREIGN KEY (viewed_file_path_id) REFERENCES public.repo_paths(id);


--
-- Name: own_aggregate_recent_view own_aggregate_recent_view_viewer_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.own_aggregate_recent_view
    ADD CONSTRAINT own_aggregate_recent_view_viewer_id_fkey FOREIGN KEY (viewer_id) REFERENCES public.users(id) ON DELETE CASCADE DEFERRABLE;


--
-- Name: own_background_jobs own_background_jobs_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.own_background_jobs
    ADD CONSTRAINT own_background_jobs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: own_signal_configurations own_signal_configurations_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.own_signal_configurations
    ADD CONSTRAINT own_signal_configurations_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: own_signal_recent_contribution own_signal_recent_contribution_changed_file_path_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.own_signal_recent_contribution
    ADD CONSTRAINT own_signal_recent_contribution_changed_file_path_id_fkey FOREIGN KEY (changed_file_path_id) REFERENCES public.repo_paths(id);


--
-- Name: own_signal_recent_contribution own_signal_recent_contribution_commit_author_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.own_signal_recent_contribution
    ADD CONSTRAINT own_signal_recent_contribution_commit_author_id_fkey FOREIGN KEY (commit_author_id) REFERENCES public.commit_authors(id);


--
-- Name: own_signal_recent_contribution own_signal_recent_contribution_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.own_signal_recent_contribution
    ADD CONSTRAINT own_signal_recent_contribution_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: ownership_path_stats ownership_path_stats_file_path_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ownership_path_stats
    ADD CONSTRAINT ownership_path_stats_file_path_id_fkey FOREIGN KEY (file_path_id) REFERENCES public.repo_paths(id);


--
-- Name: ownership_path_stats ownership_path_stats_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ownership_path_stats
    ADD CONSTRAINT ownership_path_stats_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: package_repo_versions package_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.package_repo_versions
    ADD CONSTRAINT package_id_fk FOREIGN KEY (package_id) REFERENCES public.lsif_dependency_repos(id) ON DELETE CASCADE;


--
-- Name: package_repo_filters package_repo_filters_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.package_repo_filters
    ADD CONSTRAINT package_repo_filters_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: package_repo_versions package_repo_versions_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.package_repo_versions
    ADD CONSTRAINT package_repo_versions_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: permission_sync_jobs permission_sync_jobs_repository_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.permission_sync_jobs
    ADD CONSTRAINT permission_sync_jobs_repository_id_fkey FOREIGN KEY (repository_id) REFERENCES public.repo(id) ON DELETE CASCADE;


--
-- Name: permission_sync_jobs permission_sync_jobs_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.permission_sync_jobs
    ADD CONSTRAINT permission_sync_jobs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: permission_sync_jobs permission_sync_jobs_triggered_by_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.permission_sync_jobs
    ADD CONSTRAINT permission_sync_jobs_triggered_by_user_id_fkey FOREIGN KEY (triggered_by_user_id) REFERENCES public.users(id) ON DELETE SET NULL DEFERRABLE;


--
-- Name: permission_sync_jobs permission_sync_jobs_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.permission_sync_jobs
    ADD CONSTRAINT permission_sync_jobs_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: permissions permissions_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.permissions
    ADD CONSTRAINT permissions_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: phabricator_repos phabricator_repos_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.phabricator_repos
    ADD CONSTRAINT phabricator_repos_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: product_licenses product_licenses_product_subscription_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.product_licenses
    ADD CONSTRAINT product_licenses_product_subscription_id_fkey FOREIGN KEY (product_subscription_id) REFERENCES public.product_subscriptions(id);


--
-- Name: product_licenses product_licenses_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.product_licenses
    ADD CONSTRAINT product_licenses_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: product_subscriptions product_subscriptions_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.product_subscriptions
    ADD CONSTRAINT product_subscriptions_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: product_subscriptions product_subscriptions_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.product_subscriptions
    ADD CONSTRAINT product_subscriptions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: prompts prompts_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.prompts
    ADD CONSTRAINT prompts_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: prompts prompts_owner_org_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.prompts
    ADD CONSTRAINT prompts_owner_org_id_fkey FOREIGN KEY (owner_org_id) REFERENCES public.orgs(id) ON DELETE CASCADE;


--
-- Name: prompts prompts_owner_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.prompts
    ADD CONSTRAINT prompts_owner_user_id_fkey FOREIGN KEY (owner_user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: prompts prompts_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.prompts
    ADD CONSTRAINT prompts_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: prompts prompts_updated_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.prompts
    ADD CONSTRAINT prompts_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: query_runner_state query_runner_state_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.query_runner_state
    ADD CONSTRAINT query_runner_state_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: redis_key_value redis_key_value_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.redis_key_value
    ADD CONSTRAINT redis_key_value_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: registry_extension_releases registry_extension_releases_creator_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.registry_extension_releases
    ADD CONSTRAINT registry_extension_releases_creator_user_id_fkey FOREIGN KEY (creator_user_id) REFERENCES public.users(id);


--
-- Name: registry_extension_releases registry_extension_releases_registry_extension_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.registry_extension_releases
    ADD CONSTRAINT registry_extension_releases_registry_extension_id_fkey FOREIGN KEY (registry_extension_id) REFERENCES public.registry_extensions(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: registry_extension_releases registry_extension_releases_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.registry_extension_releases
    ADD CONSTRAINT registry_extension_releases_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: registry_extensions registry_extensions_publisher_org_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.registry_extensions
    ADD CONSTRAINT registry_extensions_publisher_org_id_fkey FOREIGN KEY (publisher_org_id) REFERENCES public.orgs(id);


--
-- Name: registry_extensions registry_extensions_publisher_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.registry_extensions
    ADD CONSTRAINT registry_extensions_publisher_user_id_fkey FOREIGN KEY (publisher_user_id) REFERENCES public.users(id);


--
-- Name: registry_extensions registry_extensions_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.registry_extensions
    ADD CONSTRAINT registry_extensions_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: repo_commits_changelists repo_commits_changelists_repo_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.repo_commits_changelists
    ADD CONSTRAINT repo_commits_changelists_repo_id_fkey FOREIGN KEY (repo_id) REFERENCES public.repo(id) ON DELETE CASCADE DEFERRABLE;


--
-- Name: repo_commits_changelists repo_commits_changelists_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.repo_commits_changelists
    ADD CONSTRAINT repo_commits_changelists_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: repo_embedding_job_stats repo_embedding_job_stats_job_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.repo_embedding_job_stats
    ADD CONSTRAINT repo_embedding_job_stats_job_id_fkey FOREIGN KEY (job_id) REFERENCES public.repo_embedding_jobs(id) ON DELETE CASCADE DEFERRABLE;


--
-- Name: repo_embedding_job_stats repo_embedding_job_stats_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.repo_embedding_job_stats
    ADD CONSTRAINT repo_embedding_job_stats_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: repo_embedding_jobs repo_embedding_jobs_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.repo_embedding_jobs
    ADD CONSTRAINT repo_embedding_jobs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: repo_kvps repo_kvps_repo_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.repo_kvps
    ADD CONSTRAINT repo_kvps_repo_id_fkey FOREIGN KEY (repo_id) REFERENCES public.repo(id) ON DELETE CASCADE;


--
-- Name: repo_kvps repo_kvps_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.repo_kvps
    ADD CONSTRAINT repo_kvps_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: repo_paths repo_paths_parent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.repo_paths
    ADD CONSTRAINT repo_paths_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES public.repo_paths(id);


--
-- Name: repo_paths repo_paths_repo_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.repo_paths
    ADD CONSTRAINT repo_paths_repo_id_fkey FOREIGN KEY (repo_id) REFERENCES public.repo(id) ON DELETE CASCADE DEFERRABLE;


--
-- Name: repo_paths repo_paths_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.repo_paths
    ADD CONSTRAINT repo_paths_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: repo_pending_permissions repo_pending_permissions_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.repo_pending_permissions
    ADD CONSTRAINT repo_pending_permissions_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: repo_permissions repo_permissions_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.repo_permissions
    ADD CONSTRAINT repo_permissions_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: repo_statistics repo_statistics_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.repo_statistics
    ADD CONSTRAINT repo_statistics_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: repo repo_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.repo
    ADD CONSTRAINT repo_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: role_permissions role_permissions_permission_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_permissions
    ADD CONSTRAINT role_permissions_permission_id_fkey FOREIGN KEY (permission_id) REFERENCES public.permissions(id) ON DELETE CASCADE DEFERRABLE;


--
-- Name: role_permissions role_permissions_role_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_permissions
    ADD CONSTRAINT role_permissions_role_id_fkey FOREIGN KEY (role_id) REFERENCES public.roles(id) ON DELETE CASCADE DEFERRABLE;


--
-- Name: role_permissions role_permissions_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_permissions
    ADD CONSTRAINT role_permissions_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: roles roles_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.roles
    ADD CONSTRAINT roles_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: saved_searches saved_searches_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.saved_searches
    ADD CONSTRAINT saved_searches_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: saved_searches saved_searches_org_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.saved_searches
    ADD CONSTRAINT saved_searches_org_id_fkey FOREIGN KEY (org_id) REFERENCES public.orgs(id);


--
-- Name: saved_searches saved_searches_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.saved_searches
    ADD CONSTRAINT saved_searches_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: saved_searches saved_searches_updated_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.saved_searches
    ADD CONSTRAINT saved_searches_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: saved_searches saved_searches_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.saved_searches
    ADD CONSTRAINT saved_searches_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: search_context_default search_context_default_search_context_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.search_context_default
    ADD CONSTRAINT search_context_default_search_context_id_fkey FOREIGN KEY (search_context_id) REFERENCES public.search_contexts(id) ON DELETE CASCADE DEFERRABLE;


--
-- Name: search_context_default search_context_default_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.search_context_default
    ADD CONSTRAINT search_context_default_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: search_context_default search_context_default_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.search_context_default
    ADD CONSTRAINT search_context_default_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE DEFERRABLE;


--
-- Name: search_context_repos search_context_repos_repo_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.search_context_repos
    ADD CONSTRAINT search_context_repos_repo_id_fk FOREIGN KEY (repo_id) REFERENCES public.repo(id) ON DELETE CASCADE;


--
-- Name: search_context_repos search_context_repos_search_context_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.search_context_repos
    ADD CONSTRAINT search_context_repos_search_context_id_fk FOREIGN KEY (search_context_id) REFERENCES public.search_contexts(id) ON DELETE CASCADE;


--
-- Name: search_context_repos search_context_repos_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.search_context_repos
    ADD CONSTRAINT search_context_repos_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: search_context_stars search_context_stars_search_context_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.search_context_stars
    ADD CONSTRAINT search_context_stars_search_context_id_fkey FOREIGN KEY (search_context_id) REFERENCES public.search_contexts(id) ON DELETE CASCADE DEFERRABLE;


--
-- Name: search_context_stars search_context_stars_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.search_context_stars
    ADD CONSTRAINT search_context_stars_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: search_context_stars search_context_stars_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.search_context_stars
    ADD CONSTRAINT search_context_stars_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE DEFERRABLE;


--
-- Name: search_contexts search_contexts_namespace_org_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.search_contexts
    ADD CONSTRAINT search_contexts_namespace_org_id_fk FOREIGN KEY (namespace_org_id) REFERENCES public.orgs(id) ON DELETE CASCADE;


--
-- Name: search_contexts search_contexts_namespace_user_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.search_contexts
    ADD CONSTRAINT search_contexts_namespace_user_id_fk FOREIGN KEY (namespace_user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: search_contexts search_contexts_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.search_contexts
    ADD CONSTRAINT search_contexts_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: security_event_logs security_event_logs_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.security_event_logs
    ADD CONSTRAINT security_event_logs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: settings settings_author_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.settings
    ADD CONSTRAINT settings_author_user_id_fkey FOREIGN KEY (author_user_id) REFERENCES public.users(id) ON DELETE RESTRICT;


--
-- Name: settings settings_references_orgs; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.settings
    ADD CONSTRAINT settings_references_orgs FOREIGN KEY (org_id) REFERENCES public.orgs(id) ON DELETE RESTRICT;


--
-- Name: settings settings_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.settings
    ADD CONSTRAINT settings_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: settings settings_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.settings
    ADD CONSTRAINT settings_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE RESTRICT;


--
-- Name: sub_repo_permissions sub_repo_permissions_repo_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sub_repo_permissions
    ADD CONSTRAINT sub_repo_permissions_repo_id_fk FOREIGN KEY (repo_id) REFERENCES public.repo(id) ON DELETE CASCADE;


--
-- Name: sub_repo_permissions sub_repo_permissions_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sub_repo_permissions
    ADD CONSTRAINT sub_repo_permissions_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: sub_repo_permissions sub_repo_permissions_users_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sub_repo_permissions
    ADD CONSTRAINT sub_repo_permissions_users_id_fk FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: survey_responses survey_responses_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.survey_responses
    ADD CONSTRAINT survey_responses_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: survey_responses survey_responses_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.survey_responses
    ADD CONSTRAINT survey_responses_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: syntactic_scip_indexing_jobs syntactic_scip_indexing_jobs_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.syntactic_scip_indexing_jobs
    ADD CONSTRAINT syntactic_scip_indexing_jobs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: syntactic_scip_last_index_scan syntactic_scip_last_index_scan_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.syntactic_scip_last_index_scan
    ADD CONSTRAINT syntactic_scip_last_index_scan_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: team_members team_members_team_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team_members
    ADD CONSTRAINT team_members_team_id_fkey FOREIGN KEY (team_id) REFERENCES public.teams(id) ON DELETE CASCADE;


--
-- Name: team_members team_members_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team_members
    ADD CONSTRAINT team_members_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: team_members team_members_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team_members
    ADD CONSTRAINT team_members_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: teams teams_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.teams
    ADD CONSTRAINT teams_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: teams teams_parent_team_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.teams
    ADD CONSTRAINT teams_parent_team_id_fkey FOREIGN KEY (parent_team_id) REFERENCES public.teams(id) ON DELETE CASCADE;


--
-- Name: teams teams_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.teams
    ADD CONSTRAINT teams_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: telemetry_events_export_queue telemetry_events_export_queue_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.telemetry_events_export_queue
    ADD CONSTRAINT telemetry_events_export_queue_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: temporary_settings temporary_settings_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.temporary_settings
    ADD CONSTRAINT temporary_settings_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: temporary_settings temporary_settings_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.temporary_settings
    ADD CONSTRAINT temporary_settings_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: user_credentials user_credentials_github_app_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_credentials
    ADD CONSTRAINT user_credentials_github_app_id_fkey FOREIGN KEY (github_app_id) REFERENCES public.github_apps(id) ON DELETE CASCADE;


--
-- Name: user_credentials user_credentials_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_credentials
    ADD CONSTRAINT user_credentials_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: user_credentials user_credentials_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_credentials
    ADD CONSTRAINT user_credentials_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE DEFERRABLE;


--
-- Name: user_emails user_emails_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_emails
    ADD CONSTRAINT user_emails_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: user_emails user_emails_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_emails
    ADD CONSTRAINT user_emails_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: user_external_accounts user_external_accounts_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_external_accounts
    ADD CONSTRAINT user_external_accounts_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: user_external_accounts user_external_accounts_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_external_accounts
    ADD CONSTRAINT user_external_accounts_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: user_onboarding_tour user_onboarding_tour_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_onboarding_tour
    ADD CONSTRAINT user_onboarding_tour_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: user_onboarding_tour user_onboarding_tour_users_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_onboarding_tour
    ADD CONSTRAINT user_onboarding_tour_users_fk FOREIGN KEY (updated_by) REFERENCES public.users(id);


--
-- Name: user_pending_permissions user_pending_permissions_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_pending_permissions
    ADD CONSTRAINT user_pending_permissions_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: user_permissions user_permissions_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_permissions
    ADD CONSTRAINT user_permissions_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: user_public_repos user_public_repos_repo_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_public_repos
    ADD CONSTRAINT user_public_repos_repo_id_fkey FOREIGN KEY (repo_id) REFERENCES public.repo(id) ON DELETE CASCADE;


--
-- Name: user_public_repos user_public_repos_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_public_repos
    ADD CONSTRAINT user_public_repos_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: user_public_repos user_public_repos_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_public_repos
    ADD CONSTRAINT user_public_repos_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: user_repo_permissions user_repo_permissions_repo_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_repo_permissions
    ADD CONSTRAINT user_repo_permissions_repo_id_fkey FOREIGN KEY (repo_id) REFERENCES public.repo(id) ON DELETE CASCADE;


--
-- Name: user_repo_permissions user_repo_permissions_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_repo_permissions
    ADD CONSTRAINT user_repo_permissions_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: user_repo_permissions user_repo_permissions_user_external_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_repo_permissions
    ADD CONSTRAINT user_repo_permissions_user_external_account_id_fkey FOREIGN KEY (user_external_account_id) REFERENCES public.user_external_accounts(id) ON DELETE CASCADE;


--
-- Name: user_repo_permissions user_repo_permissions_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_repo_permissions
    ADD CONSTRAINT user_repo_permissions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: user_roles user_roles_role_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_roles
    ADD CONSTRAINT user_roles_role_id_fkey FOREIGN KEY (role_id) REFERENCES public.roles(id) ON DELETE CASCADE DEFERRABLE;


--
-- Name: user_roles user_roles_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_roles
    ADD CONSTRAINT user_roles_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: user_roles user_roles_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_roles
    ADD CONSTRAINT user_roles_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE DEFERRABLE;


--
-- Name: users users_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: vulnerabilities vulnerabilities_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vulnerabilities
    ADD CONSTRAINT vulnerabilities_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: vulnerability_affected_packages vulnerability_affected_packages_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vulnerability_affected_packages
    ADD CONSTRAINT vulnerability_affected_packages_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: vulnerability_affected_symbols vulnerability_affected_symbols_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vulnerability_affected_symbols
    ADD CONSTRAINT vulnerability_affected_symbols_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: vulnerability_matches vulnerability_matches_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.vulnerability_matches
    ADD CONSTRAINT vulnerability_matches_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: webhook_logs webhook_logs_external_service_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhook_logs
    ADD CONSTRAINT webhook_logs_external_service_id_fkey FOREIGN KEY (external_service_id) REFERENCES public.external_services(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: webhook_logs webhook_logs_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhook_logs
    ADD CONSTRAINT webhook_logs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: webhook_logs webhook_logs_webhook_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhook_logs
    ADD CONSTRAINT webhook_logs_webhook_id_fkey FOREIGN KEY (webhook_id) REFERENCES public.webhooks(id) ON DELETE CASCADE;


--
-- Name: webhooks webhooks_created_by_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhooks
    ADD CONSTRAINT webhooks_created_by_user_id_fkey FOREIGN KEY (created_by_user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: webhooks webhooks_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhooks
    ADD CONSTRAINT webhooks_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: webhooks webhooks_updated_by_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhooks
    ADD CONSTRAINT webhooks_updated_by_user_id_fkey FOREIGN KEY (updated_by_user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: zoekt_repos zoekt_repos_repo_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.zoekt_repos
    ADD CONSTRAINT zoekt_repos_repo_id_fkey FOREIGN KEY (repo_id) REFERENCES public.repo(id) ON DELETE CASCADE;


--
-- Name: zoekt_repos zoekt_repos_tenant_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.zoekt_repos
    ADD CONSTRAINT zoekt_repos_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

