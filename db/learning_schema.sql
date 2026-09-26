CREATE TABLE IF NOT EXISTS learning_articles (
    id SERIAL PRIMARY KEY,
    slug VARCHAR(120) NOT NULL UNIQUE,
    title VARCHAR(180) NOT NULL,
    summary VARCHAR(300) NOT NULL,
    category VARCHAR(40) NOT NULL DEFAULT 'Guide',
    body TEXT NOT NULL,
    published_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS learning_comments (
    id SERIAL PRIMARY KEY,
    article_id INT NOT NULL REFERENCES learning_articles(id) ON DELETE CASCADE,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    body VARCHAR(1200) NOT NULL,
    minute_bucket BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, minute_bucket)
);
CREATE INDEX IF NOT EXISTS idx_learning_comments_article ON learning_comments(article_id, created_at, id);

INSERT INTO learning_articles (slug, title, summary, category, body)
VALUES (
    'start-an-agentic-workflow',
    'Start an agentic workflow you can trust',
    'A practical checklist for giving an AI coding assistant a clear task, useful context, and a way to verify its work.',
    'Practical guide',
    E'1. Choose one small outcome. Describe what should work when the task is complete.\n\n2. Give the agent context: the relevant files, project rules, constraints, and what you have already tried. Keep credentials out of prompts and commits.\n\n3. Define a check before it edits anything. For code, this can be a failing test, a build command, or a manual flow to verify.\n\n4. Ask it to inspect the existing implementation, make a focused change, and explain its choices. Review the diff yourself before merging.\n\n5. Run the checks, test the real user flow, and record what passed or remains uncertain. For changes to accounts or production data, review permissions carefully.\n\nStart with a task small enough to review in one sitting. As you gain confidence, give the agent more responsibility while keeping the verification step.'
) ON CONFLICT (slug) DO NOTHING;
