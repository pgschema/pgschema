-- Template table for common timestamp fields
CREATE TABLE public._template_timestamps (
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    CHECK (created_at <= updated_at)
);

CREATE INDEX idx_template_created_at ON public._template_timestamps (created_at);

COMMENT ON TABLE public._template_timestamps IS 'Template for timestamp fields';
COMMENT ON COLUMN public._template_timestamps.created_at IS 'Record creation time';