-- Add clock column to sports_events table to track game clock display
ALTER TABLE sports_events ADD COLUMN clock VARCHAR(20);
