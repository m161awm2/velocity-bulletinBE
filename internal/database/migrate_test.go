package database

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/m161awm2/velocity-bulletinBE/internal/config"
)

func TestMigratePostgres(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	db, err := Open(context.Background(), config.Config{DatabaseURL: url, DBMaxOpenConns: 2, DBMaxIdleConns: 2})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	for _, version := range []int{0, 1, 2} {
		t.Run(fmt.Sprintf("version_%d", version), func(t *testing.T) {
			tx := db.Begin()
			if tx.Error != nil {
				t.Fatal(tx.Error)
			}
			defer tx.Rollback()
			schema := fmt.Sprintf("migration_test_%d", time.Now().UnixNano())
			for _, query := range []string{"CREATE SCHEMA " + schema, "SET LOCAL search_path TO " + schema} {
				if err := tx.Exec(query).Error; err != nil {
					t.Fatal(err)
				}
			}
			if version > 0 {
				if err := tx.Exec(legacySchema).Error; err != nil {
					t.Fatal(err)
				}
				if err := tx.Exec("CREATE TABLE schema_migrations (version bigint PRIMARY KEY, dirty boolean NOT NULL)").Error; err != nil {
					t.Fatal(err)
				}
				if err := tx.Exec("INSERT INTO schema_migrations VALUES (?, false)", version).Error; err != nil {
					t.Fatal(err)
				}
				if err := tx.Exec("INSERT INTO users (id,email,display_name,password_hash) VALUES ('00000000-0000-0000-0000-000000000001','test@example.com','test','hash')").Error; err != nil {
					t.Fatal(err)
				}
				if err := tx.Exec("INSERT INTO posts (id,author_id,title,body,category,deleted_at) VALUES ('00000000-0000-0000-0000-000000000002','00000000-0000-0000-0000-000000000001','test','body','NOTICE',now())").Error; err != nil {
					t.Fatal(err)
				}
				if version == 2 {
					if err := tx.Exec("UPDATE posts SET category='GENERAL'; ALTER TABLE posts DROP CONSTRAINT posts_category_check; ALTER TABLE posts ADD CONSTRAINT posts_category_check CHECK (category IN ('GENERAL','QUESTION')); DROP TABLE likes; ALTER TABLE posts DROP COLUMN view_count, DROP COLUMN like_count").Error; err != nil {
						t.Fatal(err)
					}
				}
				if err := tx.Exec("UPDATE schema_migrations SET dirty=true").Error; err != nil {
					t.Fatal(err)
				}
				if err := Migrate(tx); err == nil {
					t.Fatal("dirty migration must be rejected")
				}
				if err := tx.Exec("UPDATE schema_migrations SET dirty=false").Error; err != nil {
					t.Fatal(err)
				}
			}
			for i := 0; i < 2; i++ {
				if err := Migrate(tx); err != nil {
					t.Fatal(err)
				}
			}
			for _, table := range []string{"schema_migrations", "likes"} {
				if tx.Migrator().HasTable(table) {
					t.Fatalf("obsolete table %s remains", table)
				}
			}
			for _, column := range []string{"view_count", "like_count"} {
				if tx.Migrator().HasColumn("posts", column) {
					t.Fatalf("obsolete column %s remains", column)
				}
			}
			if version > 0 {
				var category string
				if err := tx.Raw("SELECT category FROM posts").Scan(&category).Error; err != nil {
					t.Fatal(err)
				}
				if category != "GENERAL" {
					t.Fatalf("category=%s", category)
				}
			}
			// Verify database enforcement, not only GORM's model metadata.
			checks := []string{
				"INSERT INTO users (id,email,display_name,password_hash,role) VALUES (gen_random_uuid(),'bad@example.com','bad','hash','INVALID')",
				"INSERT INTO comments (id,post_id,author_id,body) VALUES (gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),'orphan')",
			}
			for i, query := range checks {
				savepoint := fmt.Sprintf("check_%d", i)
				tx.SavePoint(savepoint)
				if err := tx.Exec(query).Error; err == nil {
					t.Fatal("invalid data was accepted")
				}
				tx.RollbackTo(savepoint)
			}
		})
	}
}

const legacySchema = `CREATE TABLE users (
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
CREATE INDEX idx_likes_post_id ON likes (post_id);`
