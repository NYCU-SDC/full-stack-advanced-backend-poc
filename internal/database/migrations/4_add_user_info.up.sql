ALTER TABLE users
ADD COLUMN avatar_url VARCHAR(512);
ALTER TABLE users
ADD COLUMN about_me TEXT;
ALTER TABLE users
ADD COLUMN username VARCHAR(255) UNIQUE NOT NULL default 'user_' || gen_random_uuid();