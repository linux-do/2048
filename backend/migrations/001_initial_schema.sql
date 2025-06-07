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
    game_mode VARCHAR(20) NOT NULL DEFAULT 'classic',
    disabled_cell JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for games table
CREATE INDEX IF NOT EXISTS idx_games_user_id ON games(user_id);
CREATE INDEX IF NOT EXISTS idx_games_score ON games(score DESC);
CREATE INDEX IF NOT EXISTS idx_games_created_at ON games(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_games_active ON games(user_id, game_over, victory) WHERE game_over = FALSE AND victory = FALSE;
CREATE INDEX IF NOT EXISTS idx_games_mode ON games(game_mode);

-- Create composite index for leaderboard queries (separated by game mode)
CREATE INDEX IF NOT EXISTS idx_games_leaderboard_daily ON games(game_mode, created_at, score DESC) WHERE (game_over = TRUE OR victory = TRUE);
CREATE INDEX IF NOT EXISTS idx_games_leaderboard_weekly ON games(game_mode, created_at, score DESC) WHERE (game_over = TRUE OR victory = TRUE);
CREATE INDEX IF NOT EXISTS idx_games_leaderboard_all ON games(game_mode, score DESC) WHERE (game_over = TRUE OR victory = TRUE);

-- Create leaderboard cache tables for better performance
CREATE TABLE IF NOT EXISTS leaderboard_daily (
    user_id VARCHAR(255) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_name VARCHAR(255) NOT NULL,
    user_avatar VARCHAR(500),
    score INTEGER NOT NULL,
    rank INTEGER NOT NULL,
    game_id UUID NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    game_mode VARCHAR(20) NOT NULL DEFAULT 'classic',
    date DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (date, user_id, game_mode)
);

CREATE TABLE IF NOT EXISTS leaderboard_weekly (
    user_id VARCHAR(255) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_name VARCHAR(255) NOT NULL,
    user_avatar VARCHAR(500),
    score INTEGER NOT NULL,
    rank INTEGER NOT NULL,
    game_id UUID NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    game_mode VARCHAR(20) NOT NULL DEFAULT 'classic',
    week_start DATE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (week_start, user_id, game_mode)
);

CREATE TABLE IF NOT EXISTS leaderboard_monthly (
    user_id VARCHAR(255) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_name VARCHAR(255) NOT NULL,
    user_avatar VARCHAR(500),
    score INTEGER NOT NULL,
    rank INTEGER NOT NULL,
    game_id UUID NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    game_mode VARCHAR(20) NOT NULL DEFAULT 'classic',
    month_start DATE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (month_start, user_id, game_mode)
);

-- Create indexes for leaderboard cache tables
CREATE INDEX IF NOT EXISTS idx_leaderboard_daily_score ON leaderboard_daily(date, game_mode, score DESC);
CREATE INDEX IF NOT EXISTS idx_leaderboard_weekly_score ON leaderboard_weekly(week_start, game_mode, score DESC);
CREATE INDEX IF NOT EXISTS idx_leaderboard_monthly_score ON leaderboard_monthly(month_start, game_mode, score DESC);

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

-- Create function to refresh leaderboard cache (updated for game modes)
CREATE OR REPLACE FUNCTION refresh_daily_leaderboard(mode VARCHAR(20) DEFAULT 'classic')
RETURNS VOID AS $$
BEGIN
    -- Clear today's leaderboard for the specified mode
    DELETE FROM leaderboard_daily WHERE date = CURRENT_DATE AND game_mode = mode;

    -- Insert today's top scores for the specified mode
    INSERT INTO leaderboard_daily (user_id, user_name, user_avatar, score, rank, game_id, game_mode, date)
    SELECT
        g.user_id,
        u.name,
        u.avatar,
        g.score,
        ROW_NUMBER() OVER (ORDER BY g.score DESC),
        g.id,
        g.game_mode,
        CURRENT_DATE
    FROM games g
    JOIN users u ON g.user_id = u.id
    WHERE (g.game_over = TRUE OR g.victory = TRUE)
        AND g.game_mode = mode
        AND g.created_at >= CURRENT_DATE
        AND g.created_at < CURRENT_DATE + INTERVAL '1 day'
    ORDER BY g.score DESC
    LIMIT 100;
END;
$$ LANGUAGE plpgsql;

-- Create function to refresh weekly leaderboard (updated for game modes)
CREATE OR REPLACE FUNCTION refresh_weekly_leaderboard(mode VARCHAR(20) DEFAULT 'classic')
RETURNS VOID AS $$
DECLARE
    week_start DATE := DATE_TRUNC('week', CURRENT_DATE)::DATE;
BEGIN
    -- Clear this week's leaderboard for the specified mode
    DELETE FROM leaderboard_weekly WHERE week_start = week_start AND game_mode = mode;

    -- Insert this week's top scores for the specified mode
    INSERT INTO leaderboard_weekly (user_id, user_name, user_avatar, score, rank, game_id, game_mode, week_start)
    SELECT
        g.user_id,
        u.name,
        u.avatar,
        g.score,
        ROW_NUMBER() OVER (ORDER BY g.score DESC),
        g.id,
        g.game_mode,
        week_start
    FROM games g
    JOIN users u ON g.user_id = u.id
    WHERE (g.game_over = TRUE OR g.victory = TRUE)
        AND g.game_mode = mode
        AND g.created_at >= week_start
        AND g.created_at < week_start + INTERVAL '1 week'
    ORDER BY g.score DESC
    LIMIT 100;
END;
$$ LANGUAGE plpgsql;

-- Create function to refresh monthly leaderboard (updated for game modes)
CREATE OR REPLACE FUNCTION refresh_monthly_leaderboard(mode VARCHAR(20) DEFAULT 'classic')
RETURNS VOID AS $$
DECLARE
    month_start DATE := DATE_TRUNC('month', CURRENT_DATE)::DATE;
BEGIN
    -- Clear this month's leaderboard for the specified mode
    DELETE FROM leaderboard_monthly WHERE month_start = month_start AND game_mode = mode;

    -- Insert this month's top scores for the specified mode
    INSERT INTO leaderboard_monthly (user_id, user_name, user_avatar, score, rank, game_id, game_mode, month_start)
    SELECT
        g.user_id,
        u.name,
        u.avatar,
        g.score,
        ROW_NUMBER() OVER (ORDER BY g.score DESC),
        g.id,
        g.game_mode,
        month_start
    FROM games g
    JOIN users u ON g.user_id = u.id
    WHERE (g.game_over = TRUE OR g.victory = TRUE)
        AND g.game_mode = mode
        AND g.created_at >= month_start
        AND g.created_at < month_start + INTERVAL '1 month'
    ORDER BY g.score DESC
    LIMIT 100;
END;
$$ LANGUAGE plpgsql;
