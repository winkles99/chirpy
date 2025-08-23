-- up.sql
CREATE TABLE chirps (
    id UUID PRIMARY KEY,
    body TEXT NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

-- down.sql
DROP TABLE chirps;