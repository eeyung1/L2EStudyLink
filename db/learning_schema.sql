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
    parent_id INT REFERENCES learning_comments(id) ON DELETE CASCADE,
    body VARCHAR(1200) NOT NULL,
    minute_bucket BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, minute_bucket)
);
ALTER TABLE learning_comments ADD COLUMN IF NOT EXISTS parent_id INT REFERENCES learning_comments(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_learning_comments_article ON learning_comments(article_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_learning_comments_parent ON learning_comments(parent_id);

-- The Go server is the only writer. Keep these tables out of Supabase's
-- client Data API; public reading goes through the deliberately scoped API.
ALTER TABLE learning_articles ENABLE ROW LEVEL SECURITY;
ALTER TABLE learning_comments ENABLE ROW LEVEL SECURITY;
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='anon') THEN
        EXECUTE 'REVOKE ALL ON learning_articles, learning_comments FROM anon';
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='authenticated') THEN
        EXECUTE 'REVOKE ALL ON learning_articles, learning_comments FROM authenticated';
    END IF;
END $$;

INSERT INTO learning_articles (slug, title, summary, category, body)
VALUES (
    'start-an-agentic-workflow',
    'Start an agentic workflow you can trust',
    'A practical checklist for giving an AI coding assistant a clear task, useful context, and a way to verify its work.',
    'Practical guide',
    E'1. Choose one small outcome. Describe what should work when the task is complete.\n\n2. Give the agent context: the relevant files, project rules, constraints, and what you have already tried. Keep credentials out of prompts and commits.\n\n3. Define a check before it edits anything. For code, this can be a failing test, a build command, or a manual flow to verify.\n\n4. Ask it to inspect the existing implementation, make a focused change, and explain its choices. Review the diff yourself before merging.\n\n5. Run the checks, test the real user flow, and record what passed or remains uncertain. For changes to accounts or production data, review permissions carefully.\n\nStart with a task small enough to review in one sitting. As you gain confidence, give the agent more responsibility while keeping the verification step.'
) ON CONFLICT (slug) DO NOTHING;

INSERT INTO learning_articles (slug, title, summary, category, body)
VALUES (
    'practice-agentic-workflows-without-a-paid-plan',
    'Practice an agentic workflow without a paid AI plan',
    'You can learn the workflow before paying for a model. Start with a tiny task, a clear check, and tools you already have.',
    'Getting started',
    E'An agentic workflow is a way to organize work: define an outcome, gather context, make a change, check it, and decide what to do next. An AI assistant can help with those steps, but you can practise the cycle without subscribing to a model.\n\n1. Pick a task you can finish in a sitting. For example, improve an error message or add a small test to a project you already understand.\n\n2. Write the task as if you were briefing a teammate: what the user sees now, what they should see, which files matter, and how you will know it works. That brief is a useful prompt later.\n\n3. Use your editor, terminal, and tests to do one small iteration. Keep a short log of what you tried and what the checks reported.\n\n4. Review your own diff before moving on. Ask: Did I solve the stated problem? Did I change unrelated behavior? What would a fellow need to reproduce the result?\n\nWhen you are ready to try an AI tool, compare available free or local options at that time; availability and limits change. Never paste passwords, API keys, or private user data into a tool you have not evaluated. Keep the same review and verification steps whether a model is involved or not.'
) ON CONFLICT (slug) DO NOTHING;

INSERT INTO learning_articles (slug, title, summary, category, body)
VALUES (
    'turn-weekly-goals-into-evidence',
    'Turn weekly goals into evidence of progress',
    'A simple goal, a protected time block, and an honest reflection can help you see what is improving and what needs a different approach.',
    'Learning practice',
    E'Big goals become easier to act on when you can point to one next step. Instead of writing “learn backend development,” choose something you can show after a week: “build a route that validates an input and returns a useful error.”\n\nPut a realistic study block on your timetable. Name the task you will do in that block and leave room for interruptions. A timetable is a commitment to try; it is not proof that the work happened.\n\nAfter the block, write a short reflection: What did I attempt? What worked? What blocked me? What will I try next? Save the concrete details, including the commands, examples, or questions that helped.\n\nAt the end of the week, read your reflections. Look for a repeated obstacle or a small win. Choose your next goal from that evidence. If you missed a block, record the reason and adjust the plan instead of treating the calendar as a scorecard.\n\nYour private reflections remain yours. Share a lesson publicly only when you choose to write it as a separate comment or guide.'
) ON CONFLICT (slug) DO NOTHING;
