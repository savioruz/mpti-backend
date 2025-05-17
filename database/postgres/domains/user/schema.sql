CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    password VARCHAR(255) DEFAULT NULL,
    level CHAR(1) NOT NULL DEFAULT '1',
    google_id VARCHAR(255) UNIQUE,
    full_name VARCHAR(255) DEFAULT NULL,
    profile_image TEXT DEFAULT NULL,
    is_verified BOOLEAN DEFAULT FALSE,
    last_login TIMESTAMP DEFAULT NULL,
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now(),
    deleted_at TIMESTAMP DEFAULT NULL
);

CREATE TABLE IF NOT EXISTS email_verifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMP DEFAULT now() + INTERVAL '1 hours',
    created_at TIMESTAMP DEFAULT now()
);

CREATE TABLE IF NOT EXISTS password_resets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMP DEFAULT now() + INTERVAL '1 hours',
    created_at TIMESTAMP DEFAULT now()
);
