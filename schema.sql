-- L2EStudyLink Database Schema

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    discord_username VARCHAR(50),
    bio TEXT,
    meeting_preferences TEXT[],
    rating DECIMAL(3,2) DEFAULT 0.00,
    total_reviews INT DEFAULT 0,
    total_sessions INT DEFAULT 0,
    no_show_count INT DEFAULT 0,
    is_suspended BOOLEAN DEFAULT FALSE,
    is_admin BOOLEAN DEFAULT FALSE,
    marketing_opt_in_at TIMESTAMPTZ,
    public_handle VARCHAR(30),
    portfolio_public BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Skills table
CREATE TABLE IF NOT EXISTS skills (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    skill_name VARCHAR(50) NOT NULL,
    proficiency VARCHAR(20) CHECK (proficiency IN ('beginner', 'intermediate', 'advanced')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, skill_name)
);

-- Availability table
CREATE TABLE IF NOT EXISTS availability (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    day_of_week INT CHECK (day_of_week >= 0 AND day_of_week <= 6),
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, day_of_week, start_time)
);

-- Bookings table
CREATE TABLE IF NOT EXISTS bookings (
    id SERIAL PRIMARY KEY,
    tutor_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    student_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_date DATE NOT NULL,
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    topic VARCHAR(200) NOT NULL,
    meeting_type VARCHAR(20) CHECK (meeting_type IN ('in_person', 'online')),
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'confirmed', 'completed', 'cancelled', 'no_show')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT no_self_booking CHECK (tutor_id != student_id)
);

-- Reviews table
CREATE TABLE IF NOT EXISTS login_attempts (
    subject_key TEXT PRIMARY KEY,
    attempts INT NOT NULL DEFAULT 0,
    window_started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    locked_until TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_login_attempts_window ON login_attempts(window_started_at);

-- Reviews table
CREATE TABLE IF NOT EXISTS reviews (
    id SERIAL PRIMARY KEY,
    booking_id INT NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    reviewer_id INT NOT NULL REFERENCES users(id),
    reviewee_id INT NOT NULL REFERENCES users(id),
    rating INT NOT NULL CHECK (rating >= 1 AND rating <= 5),
    comment TEXT NOT NULL,
    tags TEXT[],
    would_book_again BOOLEAN,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(booking_id, reviewer_id)
);

-- Password reset tokens
CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    used BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_password_reset_token ON password_reset_tokens(token);

-- Timetable table (personal weekly study blocks)
CREATE TABLE IF NOT EXISTS timetable (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    day_of_week INT CHECK (day_of_week >= 0 AND day_of_week <= 6),
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    activity VARCHAR(100) NOT NULL,
    goal VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Activity logs (reflections on timetable blocks)
CREATE TABLE IF NOT EXISTS activity_logs (
    id SERIAL PRIMARY KEY,
    timetable_id INT REFERENCES timetable(id) ON DELETE SET NULL,
    activity_name VARCHAR(255) NOT NULL DEFAULT '',
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    log_date DATE NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    challenges TEXT NOT NULL DEFAULT '',
    learnings TEXT NOT NULL DEFAULT '',
    audio_url TEXT,
    completed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(timetable_id, user_id, log_date)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_skills_user ON skills(user_id);
CREATE INDEX IF NOT EXISTS idx_skills_name ON skills(skill_name);
CREATE INDEX IF NOT EXISTS idx_availability_user ON availability(user_id);
CREATE INDEX IF NOT EXISTS idx_bookings_tutor ON bookings(tutor_id);
CREATE INDEX IF NOT EXISTS idx_bookings_student ON bookings(student_id);
CREATE INDEX IF NOT EXISTS idx_bookings_date ON bookings(session_date);
CREATE INDEX IF NOT EXISTS idx_reviews_reviewee ON reviews(reviewee_id);
CREATE INDEX IF NOT EXISTS idx_timetable_user ON timetable(user_id);
CREATE INDEX IF NOT EXISTS idx_activity_logs_timetable ON activity_logs(timetable_id);
CREATE INDEX IF NOT EXISTS idx_activity_logs_user ON activity_logs(user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_public_handle ON users (lower(public_handle));
-- Project collaboration
CREATE TABLE IF NOT EXISTS collaboration_preferences (
    user_id INT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    open_to_invites BOOLEAN NOT NULL DEFAULT FALSE
);
CREATE TABLE IF NOT EXISTS projects (
    id SERIAL PRIMARY KEY,
    owner_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(120) NOT NULL,
    description TEXT NOT NULL,
    roles_needed VARCHAR(240) NOT NULL,
    time_commitment VARCHAR(120) NOT NULL,
    repo_url TEXT NOT NULL DEFAULT '',
    status VARCHAR(10) NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_projects_owner ON projects(owner_id);
CREATE TABLE IF NOT EXISTS project_requests (
    id SERIAL PRIMARY KEY,
    project_id INT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind VARCHAR(12) NOT NULL CHECK (kind IN ('application', 'invitation')),
    note VARCHAR(500) NOT NULL DEFAULT '',
    status VARCHAR(10) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'declined')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_project_requests_user ON project_requests(user_id);
CREATE TABLE IF NOT EXISTS project_members (
    project_id INT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    joined_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY(project_id, user_id)
);

-- Accountability buddy pairing. The expression index prevents two users from
-- opening duplicate requests in opposite directions.
CREATE TABLE IF NOT EXISTS buddy_connections (
    id SERIAL PRIMARY KEY,
    requester_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    recipient_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(10) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'active', 'declined')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (requester_id <> recipient_id)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_buddy_connections_pair
    ON buddy_connections (LEAST(requester_id, recipient_id), GREATEST(requester_id, recipient_id));
CREATE INDEX IF NOT EXISTS idx_buddy_connections_requester ON buddy_connections(requester_id, status);
CREATE INDEX IF NOT EXISTS idx_buddy_connections_recipient ON buddy_connections(recipient_id, status);

-- Public Learning Hub. Existing Postgres databases create these on startup.
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
CREATE INDEX IF NOT EXISTS idx_learning_comments_article ON learning_comments(article_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_learning_comments_parent ON learning_comments(parent_id);
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

-- In-app notifications. Event inserts share the source transaction.
CREATE TABLE IF NOT EXISTS notifications (
    id BIGSERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_key VARCHAR(160) NOT NULL,
    title VARCHAR(120) NOT NULL,
    message TEXT NOT NULL DEFAULT '',
    link_url VARCHAR(255) NOT NULL DEFAULT '/dashboard',
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id,event_key),
    CHECK (link_url LIKE '/%' AND link_url NOT LIKE '//%')
);
CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications(user_id,id DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_unread ON notifications(user_id,is_read);
ALTER TABLE notifications ENABLE ROW LEVEL SECURITY;
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='anon') THEN
        REVOKE ALL ON notifications FROM anon;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='authenticated') THEN
        REVOKE ALL ON notifications FROM authenticated;
    END IF;
END $$;
CREATE OR REPLACE FUNCTION l2e_notify_event() RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    IF TG_TABLE_NAME='bookings' THEN
        IF TG_OP='UPDATE' THEN
            IF NEW.status IS NOT DISTINCT FROM OLD.status THEN RETURN NEW; END IF;
        END IF;
        INSERT INTO notifications(user_id,event_key,title,message,link_url)
        SELECT id,'booking:'||NEW.id||':'||COALESCE(NEW.status,'pending'),
            CASE WHEN TG_OP='INSERT' THEN 'Study booking requested' ELSE 'Study booking updated' END,
            CASE WHEN TG_OP='INSERT' THEN 'A study booking request was created.'
                 ELSE 'Booking status: '||COALESCE(NEW.status,'pending')||'.' END,'/my-bookings'
        FROM users WHERE id IN (NEW.tutor_id,NEW.student_id) AND COALESCE(is_suspended,FALSE)=FALSE
        ON CONFLICT(user_id,event_key) DO NOTHING;
    ELSIF TG_TABLE_NAME='buddy_connections' THEN
        IF TG_OP='INSERT' THEN
            IF NEW.status='pending' THEN
                INSERT INTO notifications(user_id,event_key,title,message,link_url)
                SELECT id,'buddy:'||NEW.id||':pending','New buddy request',
                    'A fellow would like to be your accountability buddy.','/buddies'
                FROM users WHERE id=NEW.recipient_id AND COALESCE(is_suspended,FALSE)=FALSE
                ON CONFLICT(user_id,event_key) DO NOTHING;
            END IF;
        ELSIF NEW.status IS DISTINCT FROM OLD.status AND NEW.status IN ('active','declined') THEN
            INSERT INTO notifications(user_id,event_key,title,message,link_url)
            SELECT id,'buddy:'||NEW.id||':'||NEW.status,'Buddy request updated',
                CASE WHEN NEW.status='active' THEN 'Your buddy connection is now active.'
                     ELSE 'Your buddy request was declined.' END,'/buddies'
            FROM users WHERE (id=NEW.requester_id OR (NEW.status='active' AND id=NEW.recipient_id))
                AND COALESCE(is_suspended,FALSE)=FALSE
            ON CONFLICT(user_id,event_key) DO NOTHING;
        END IF;
    ELSIF TG_TABLE_NAME='activity_logs' THEN
        IF NEW.completed IS NOT TRUE THEN RETURN NEW; END IF;
        IF TG_OP='UPDATE' THEN
            IF OLD.completed IS TRUE THEN RETURN NEW; END IF;
        END IF;
        INSERT INTO notifications(user_id,event_key,title,message,link_url)
        SELECT id,'reflection:'||NEW.id||':saved','Reflection saved',
            'Your learning reflection is ready in your history.','/reflections'
        FROM users WHERE id=NEW.user_id AND COALESCE(is_suspended,FALSE)=FALSE
        ON CONFLICT(user_id,event_key) DO NOTHING;
        INSERT INTO notifications(user_id,event_key,title,message,link_url)
        SELECT u.id,'reflection:'||NEW.id||':buddy','Your buddy reflected',
            'An accountability buddy logged a learning reflection.','/buddies'
        FROM buddy_connections bc JOIN users u ON u.id=
            CASE WHEN bc.requester_id=NEW.user_id THEN bc.recipient_id ELSE bc.requester_id END
        WHERE bc.status='active' AND (bc.requester_id=NEW.user_id OR bc.recipient_id=NEW.user_id)
            AND COALESCE(u.is_suspended,FALSE)=FALSE
        ON CONFLICT(user_id,event_key) DO NOTHING;
    END IF;
    RETURN NEW;
END $$;
DO $$
DECLARE source_table TEXT;
BEGIN
    FOREACH source_table IN ARRAY ARRAY['bookings','buddy_connections','activity_logs'] LOOP
        IF to_regclass(source_table) IS NOT NULL THEN
            EXECUTE format('DROP TRIGGER IF EXISTS l2e_notification_event ON %I',source_table);
            EXECUTE format('CREATE TRIGGER l2e_notification_event AFTER INSERT OR UPDATE ON %I FOR EACH ROW EXECUTE FUNCTION l2e_notify_event()',source_table);
        END IF;
    END LOOP;
END $$;
