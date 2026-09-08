-- Schema rollback only: deleted likes/counts and former notice categories cannot be recovered.
ALTER TABLE posts DROP CONSTRAINT posts_category_check;
ALTER TABLE posts ADD CONSTRAINT posts_category_check CHECK (category IN ('GENERAL', 'QUESTION', 'NOTICE'));
ALTER TABLE posts ADD COLUMN view_count bigint NOT NULL DEFAULT 0 CHECK (view_count >= 0),
    ADD COLUMN like_count bigint NOT NULL DEFAULT 0 CHECK (like_count >= 0);
CREATE TABLE likes (
    user_id uuid NOT NULL REFERENCES users(id),
    post_id uuid NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, post_id)
);
CREATE INDEX idx_likes_post_id ON likes (post_id);
