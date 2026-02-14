-- Remove branding image fields from grid_settings
ALTER TABLE grid_settings DROP COLUMN branding_image_url;
ALTER TABLE grid_settings DROP COLUMN branding_image_alt;
