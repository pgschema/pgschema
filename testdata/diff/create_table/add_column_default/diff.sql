ALTER TABLE events ADD COLUMN status text DEFAULT 'active' NOT NULL;
ALTER TABLE events ADD COLUMN priority integer DEFAULT 0;
ALTER TABLE events ADD COLUMN score numeric DEFAULT 0.0;
ALTER TABLE events ADD COLUMN is_active boolean DEFAULT true;
ALTER TABLE events ADD COLUMN created_at timestamp DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE events ADD COLUMN updated_at timestamp DEFAULT now();
ALTER TABLE events ADD COLUMN config jsonb DEFAULT '{}';
ALTER TABLE events ADD COLUMN tags text[] DEFAULT '{}';
