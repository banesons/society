-- Migration: 20261029000000_create_user_profiles_schema.sql
-- Description: Canonical global user profiles table for authenticated user identity independent of society tenancy

CREATE TABLE IF NOT EXISTS user_profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    first_name TEXT NOT NULL DEFAULT '',
    last_name TEXT NOT NULL DEFAULT '',
    email TEXT,
    profile_photo_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS user_profiles_user_id_idx ON user_profiles(user_id);

-- Backfill: ensure every existing user has a canonical profile record
INSERT INTO user_profiles (user_id, first_name, last_name, email, created_at, updated_at)
SELECT u.id, '', '', NULL, u.created_at, u.updated_at
FROM users u
ON CONFLICT (user_id) DO NOTHING;

-- Populate canonical name and email for existing users from active society_members if available
UPDATE user_profiles up
SET first_name = sm.first_name,
    last_name = COALESCE(sm.last_name, ''),
    email = sm.email,
    updated_at = NOW()
FROM (
    SELECT DISTINCT ON (user_id) user_id, first_name, last_name, email
    FROM society_members
    WHERE user_id IS NOT NULL AND status = 'ACTIVE'
    ORDER BY user_id, updated_at DESC
) sm
WHERE up.user_id = sm.user_id AND up.first_name = '';
