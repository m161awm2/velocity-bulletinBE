-- Preserve notice contents as ordinary posts.
UPDATE posts SET category = 'GENERAL' WHERE category = 'NOTICE';
ALTER TABLE posts DROP CONSTRAINT posts_category_check;
ALTER TABLE posts ADD CONSTRAINT posts_category_check CHECK (category IN ('GENERAL', 'QUESTION'));
DROP TABLE likes;
ALTER TABLE posts DROP COLUMN view_count, DROP COLUMN like_count;
