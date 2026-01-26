-- Add branding image fields to grid_settings
ALTER TABLE grid_settings ADD COLUMN branding_image_url TEXT;
ALTER TABLE grid_settings ADD COLUMN branding_image_alt TEXT;
