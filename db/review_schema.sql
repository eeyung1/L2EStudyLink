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
CREATE TABLE IF NOT EXISTS reviews (
 id SERIAL PRIMARY KEY,
 booking_id INT NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
 reviewer_id INT NOT NULL REFERENCES users(id),
 reviewee_id INT NOT NULL REFERENCES users(id),
 rating INT NOT NULL CHECK (rating BETWEEN 1 AND 5),
 comment TEXT NOT NULL,
 tags TEXT[],
 would_book_again BOOLEAN,
 created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
 UNIQUE(booking_id, reviewer_id)
);
CREATE INDEX IF NOT EXISTS idx_reviews_reviewee ON reviews(reviewee_id);
