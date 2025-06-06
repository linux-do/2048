-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(255) PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    avatar VARCHAR(500),
    provider VARCHAR(50) NOT NULL,
    provider_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(provider, provider_id)
);

-- Create index on email for faster lookups
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_provider ON users(provider, provider_id);

-- Create games table
CREATE TABLE IF NOT EXISTS games (
    id UUID PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    board JSONB NOT NULL,
    score INTEGER NOT NULL DEFAULT 0,
    game_over BOOLEAN NOT NULL DEFAULT FALSE,
    victory BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for games table
CREATE INDEX IF NOT EXISTS idx_games_user_id ON games(user_id);
CREATE INDEX IF NOT EXISTS idx_games_score ON games(score DESC);
CREATE INDEX IF NOT EXISTS idx_games_created_at ON games(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_games_active ON games(user_id, game_over, victory) WHERE game_over = FALSE AND victory = FALSE;

-- Create composite index for leaderboard queries
CREATE INDEX IF NOT EXISTS idx_games_leaderboard_daily ON games(created_at, score DESC) WHERE (game_over = TRUE OR victory = TRUE);
CREATE INDEX IF NOT EXISTS idx_games_leaderboard_weekly ON games(created_at, score DESC) WHERE (game_over = TRUE OR victory = TRUE);
CREATE INDEX IF NOT EXISTS idx_games_leaderboard_all ON games(score DESC) WHERE (game_over = TRUE OR victory = TRUE);

-- Create leaderboard cache tables for better performance
CREATE TABLE IF NOT EXISTS leaderboard_daily (
    user_id VARCHAR(255) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_name VARCHAR(255) NOT NULL,
    user_avatar VARCHAR(500),
    score INTEGER NOT NULL,
    rank INTEGER NOT NULL,
    game_id UUID NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    date DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    PRIMARY KEY (date, user_id)
);

CREATE TABLE IF NOT EXISTS leaderboard_weekly (
    user_id VARCHAR(255) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_name VARCHAR(255) NOT NULL,
    user_avatar VARCHAR(500),
    score INTEGER NOT NULL,
    rank INTEGER NOT NULL,
    game_id UUID NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    week_start DATE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    PRIMARY KEY (week_start, user_id)
);

CREATE TABLE IF NOT EXISTS leaderboard_monthly (
    user_id VARCHAR(255) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_name VARCHAR(255) NOT NULL,
    user_avatar VARCHAR(500),
    score INTEGER NOT NULL,
    rank INTEGER NOT NULL,
    game_id UUID NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    month_start DATE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    PRIMARY KEY (month_start, user_id)
);

-- Create indexes for leaderboard cache tables
CREATE INDEX IF NOT EXISTS idx_leaderboard_daily_score ON leaderboard_daily(date, score DESC);
CREATE INDEX IF NOT EXISTS idx_leaderboard_weekly_score ON leaderboard_weekly(week_start, score DESC);
CREATE INDEX IF NOT EXISTS idx_leaderboard_monthly_score ON leaderboard_monthly(month_start, score DESC);

-- Create function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers to automatically update updated_at
CREATE TRIGGER update_users_updated_at 
    BEFORE UPDATE ON users 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_games_updated_at 
    BEFORE UPDATE ON games 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Create function to refresh leaderboard cache
CREATE OR REPLACE FUNCTION refresh_daily_leaderboard()
RETURNS VOID AS $$
BEGIN
    -- Clear today's leaderboard
    DELETE FROM leaderboard_daily WHERE date = CURRENT_DATE;
    
    -- Insert today's top scores
    INSERT INTO leaderboard_daily (user_id, user_name, user_avatar, score, rank, game_id, date)
    SELECT 
        g.user_id,
        u.name,
        u.avatar,
        g.score,
        ROW_NUMBER() OVER (ORDER BY g.score DESC),
        g.id,
        CURRENT_DATE
    FROM games g
    JOIN users u ON g.user_id = u.id
    WHERE (g.game_over = TRUE OR g.victory = TRUE)
        AND g.created_at >= CURRENT_DATE
        AND g.created_at < CURRENT_DATE + INTERVAL '1 day'
    ORDER BY g.score DESC
    LIMIT 100;
END;
$$ LANGUAGE plpgsql;

-- Create function to refresh weekly leaderboard
CREATE OR REPLACE FUNCTION refresh_weekly_leaderboard()
RETURNS VOID AS $$
DECLARE
    week_start DATE := DATE_TRUNC('week', CURRENT_DATE)::DATE;
BEGIN
    -- Clear this week's leaderboard
    DELETE FROM leaderboard_weekly WHERE week_start = week_start;
    
    -- Insert this week's top scores
    INSERT INTO leaderboard_weekly (user_id, user_name, user_avatar, score, rank, game_id, week_start)
    SELECT 
        g.user_id,
        u.name,
        u.avatar,
        g.score,
        ROW_NUMBER() OVER (ORDER BY g.score DESC),
        g.id,
        week_start
    FROM games g
    JOIN users u ON g.user_id = u.id
    WHERE (g.game_over = TRUE OR g.victory = TRUE)
        AND g.created_at >= week_start
        AND g.created_at < week_start + INTERVAL '1 week'
    ORDER BY g.score DESC
    LIMIT 100;
END;
$$ LANGUAGE plpgsql;

-- Create function to refresh monthly leaderboard
CREATE OR REPLACE FUNCTION refresh_monthly_leaderboard()
RETURNS VOID AS $$
DECLARE
    month_start DATE := DATE_TRUNC('month', CURRENT_DATE)::DATE;
BEGIN
    -- Clear this month's leaderboard
    DELETE FROM leaderboard_monthly WHERE month_start = month_start;
    
    -- Insert this month's top scores
    INSERT INTO leaderboard_monthly (user_id, user_name, user_avatar, score, rank, game_id, month_start)
    SELECT 
        g.user_id,
        u.name,
        u.avatar,
        g.score,
        ROW_NUMBER() OVER (ORDER BY g.score DESC),
        g.id,
        month_start
    FROM games g
    JOIN users u ON g.user_id = u.id
    WHERE (g.game_over = TRUE OR g.victory = TRUE)
        AND g.created_at >= month_start
        AND g.created_at < month_start + INTERVAL '1 month'
    ORDER BY g.score DESC
    LIMIT 100;
END;
$$ LANGUAGE plpgsql;
