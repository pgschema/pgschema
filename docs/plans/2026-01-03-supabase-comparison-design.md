# Design: Supabase Declarative Schema vs pgschema Comparison Article

## Overview

A comparison article targeting developers already using Supabase who are curious about alternatives. The article takes a neutral comparison approach while naturally highlighting where pgschema offers advantages.

## Target Audience

Developers already using Supabase's declarative schema workflow, curious about alternatives.

## Tone

Neutral and factual. Acknowledge Supabase is a great platform while showing pgschema's broader PostgreSQL support. Let the comparison speak for itself.

## Article Structure

### 1. Introduction

- Hook: Reference users migrating from Supabase to pgschema (existing draft content)
- Set up the comparison: both tools offer declarative schema workflows

### 2. Workflow Comparison: Adding a Column and Rolling Back

Show the same task side-by-side to illustrate workflow differences.

**Supabase workflow:**
1. Edit `supabase/schemas/users.sql` to add the column
2. Run `supabase db diff -f add_phone_column` to generate migration
3. Run `supabase migration up` to apply locally
4. Rollback: `supabase db reset --version <previous_timestamp>`

**pgschema workflow:**
1. Edit `schema.sql` to add the column
2. Run `pgschema plan --from <db> --to schema.sql` to see the plan
3. Run `pgschema apply` to execute
4. Rollback: Edit `schema.sql` to remove the column, run plan + apply again

**Key difference:** Supabase rollback uses migration versioning (go back to a timestamp). pgschema rollback is "change the desired state and apply" - no migration history to manage.

### 3. Where pgschema Goes Further

Bullet list comparing capabilities.

**Supabase/migra documented limitations:**
- Row-level security (RLS) policy modifications
- Partitioned tables
- Comments on objects
- Column-specific privileges
- Schema privileges
- View ownership and security properties

**pgschema supports all of the above, plus:**
- Triggers (with WHEN conditions, constraint triggers, REFERENCING clauses)
- Procedures
- Custom aggregates
- CREATE INDEX CONCURRENTLY (online, non-blocking)
- ADD CONSTRAINT ... NOT VALID (online constraint addition)
- Works with any PostgreSQL database (not Supabase-specific)

### 4. Conclusion

Brief wrap-up (2-3 sentences):
- Supabase declarative schemas work well for simpler use cases within the Supabase ecosystem
- pgschema is the choice when you need comprehensive PostgreSQL support or want to use any PostgreSQL database
- No hard sell - let readers decide based on their needs

## Key Differentiators to Emphasize

1. **Comprehensive object support** - pgschema handles RLS policies, triggers, procedures, etc. that Supabase's migra can't track
2. **No Supabase lock-in** - works with any PostgreSQL database
3. **Online migration features** - CREATE INDEX CONCURRENTLY, NOT VALID constraints for zero-downtime changes

## Technical Level

Practical with examples - show actual SQL/commands side-by-side demonstrating the differences.

## Output Location

`docs/blog/supabase-declarative-schema-vs-pgschema.mdx`
