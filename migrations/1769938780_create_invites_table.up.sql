-- Create invites table for user invitation system
CREATE TABLE IF NOT EXISTS invites (
    id TEXT PRIMARY KEY,
    family_id TEXT NOT NULL,
    created_by TEXT NOT NULL,
    email TEXT NOT NULL,
    role TEXT NOT NULL CHECK(role IN ('admin', 'member', 'child')),
    token TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending', 'accepted', 'expired', 'revoked')),
    expires_at DATETIME NOT NULL,
    accepted_at DATETIME,
    accepted_by TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (family_id) REFERENCES families(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (accepted_by) REFERENCES users(id) ON DELETE SET NULL
);

-- Create indexes for faster lookups
CREATE INDEX IF NOT EXISTS idx_invites_token ON invites(token);
CREATE INDEX IF NOT EXISTS idx_invites_family_id ON invites(family_id);
CREATE INDEX IF NOT EXISTS idx_invites_email ON invites(email);
CREATE INDEX IF NOT EXISTS idx_invites_status ON invites(status);
CREATE INDEX IF NOT EXISTS idx_invites_expires_at ON invites(expires_at);

-- Create trigger to update updated_at timestamp
CREATE TRIGGER IF NOT EXISTS update_invites_updated_at
AFTER UPDATE ON invites
FOR EACH ROW
BEGIN
    UPDATE invites SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
