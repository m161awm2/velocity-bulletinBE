CREATE TABLE users (
    id uuid PRIMARY KEY,
    email varchar(320) NOT NULL UNIQUE,
    display_name varchar(40) NOT NULL,
    password_hash text NOT NULL,
    role varchar(16) NOT NULL DEFAULT 'USER' CHECK (role IN ('USER', 'ADMIN')),
    active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE INDEX idx_users_deleted_at ON users (deleted_at);

CREATE TABLE posts (
    id uuid PRIMARY KEY,
    author_id uuid NOT NULL REFERENCES users(id),
    title varchar(200) NOT NULL,
    body text NOT NULL,
    category varchar(16) NOT NULL CHECK (category IN ('GENERAL', 'QUESTION', 'NOTICE')),
    view_count bigint NOT NULL DEFAULT 0 CHECK (view_count >= 0),
    like_count bigint NOT NULL DEFAULT 0 CHECK (like_count >= 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE INDEX idx_posts_author_id ON posts (author_id);
CREATE INDEX idx_posts_category ON posts (category);
CREATE INDEX idx_posts_created_at ON posts (created_at DESC);
CREATE INDEX idx_posts_deleted_at ON posts (deleted_at);

CREATE TABLE post_images (
    id uuid PRIMARY KEY,
    post_id uuid NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    object_key varchar(512) NOT NULL,
    url varchar(1024) NOT NULL,
    position integer NOT NULL,
    UNIQUE (post_id, position)
);
CREATE INDEX idx_post_images_post_id ON post_images (post_id);

CREATE TABLE comments (
    id uuid PRIMARY KEY,
    post_id uuid NOT NULL REFERENCES posts(id),
    author_id uuid NOT NULL REFERENCES users(id),
    body text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE INDEX idx_comments_post_id ON comments (post_id);
CREATE INDEX idx_comments_author_id ON comments (author_id);
CREATE INDEX idx_comments_deleted_at ON comments (deleted_at);

CREATE TABLE likes (
    user_id uuid NOT NULL REFERENCES users(id),
    post_id uuid NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, post_id)
);
CREATE INDEX idx_likes_post_id ON likes (post_id);
