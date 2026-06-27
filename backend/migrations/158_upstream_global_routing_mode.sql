INSERT INTO settings (key, value, updated_at)
VALUES ('upstream.routing_mode', 'balanced', NOW())
ON CONFLICT (key) DO NOTHING;
