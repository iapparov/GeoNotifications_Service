CREATE TABLE IF NOT EXISTS location_check_history (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid,
    latitude DOUBLE PRECISION NOT NULL,
    longitude DOUBLE PRECISION NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    in_dangerous_area BOOLEAN DEFAULT FALSE
);