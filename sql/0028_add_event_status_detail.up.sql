-- Add status_detail column to sports_events table to track ESPN status description (e.g., "Halftime", "End of 1st Quarter")
ALTER TABLE sports_events ADD COLUMN status_detail VARCHAR(100);
