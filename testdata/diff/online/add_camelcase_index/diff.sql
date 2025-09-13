CREATE INDEX IF NOT EXISTS idx_invite_assignedTo ON invite ("assignedTo");

CREATE INDEX IF NOT EXISTS idx_invite_created_invited ON invite ("createdAt", "invitedBy");