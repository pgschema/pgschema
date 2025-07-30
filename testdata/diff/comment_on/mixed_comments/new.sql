CREATE TABLE posts (
    id INTEGER PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    content TEXT,
    author_id INTEGER NOT NULL,
    published_at TIMESTAMP
);
COMMENT ON TABLE posts IS 'Blog posts and articles';
COMMENT ON COLUMN posts.id IS 'Unique post identifier';
COMMENT ON COLUMN posts.title IS 'Post title, max 200 characters';
COMMENT ON COLUMN posts.content IS 'Post body in markdown format';
COMMENT ON COLUMN posts.author_id IS 'Foreign key to users table';
COMMENT ON COLUMN posts.published_at IS 'Publication timestamp, NULL for drafts';

CREATE INDEX idx_posts_author ON posts (author_id);
COMMENT ON INDEX idx_posts_author IS 'Index for finding posts by author';

CREATE INDEX idx_posts_published ON posts (published_at);
COMMENT ON INDEX idx_posts_published IS 'Partial index for published posts only';

CREATE TABLE categories (
    id INTEGER PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    parent_id INTEGER REFERENCES categories(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
COMMENT ON TABLE categories IS 'Hierarchical category system for posts';
COMMENT ON COLUMN categories.id IS 'Category unique identifier';
COMMENT ON COLUMN categories.name IS 'Category display name';
COMMENT ON COLUMN categories.description IS 'Optional category description';
COMMENT ON COLUMN categories.parent_id IS 'Parent category for hierarchical structure';
COMMENT ON COLUMN categories.created_at IS 'Category creation timestamp';

CREATE INDEX idx_categories_parent ON categories (parent_id);
COMMENT ON INDEX idx_categories_parent IS 'Index for hierarchical category queries';