-- Insert default timetable blocks for all users who don't have any
INSERT INTO timetable (user_id, day_of_week, start_time, end_time, activity, goal)
SELECT 
    u.id,
    d.day,
    d.start_time,
    d.end_time,
    'Study Time',
    'Focus on learning'
FROM users u
CROSS JOIN (
    VALUES 
        (0, '09:00', '11:00'),
        (1, '09:00', '11:00'),
        (2, '09:00', '11:00'),
        (3, '09:00', '11:00'),
        (4, '09:00', '11:00'),
        (5, '10:00', '12:00')
) AS d(day, start_time, end_time)
WHERE NOT EXISTS (
    SELECT 1 FROM timetable t WHERE t.user_id = u.id
);
