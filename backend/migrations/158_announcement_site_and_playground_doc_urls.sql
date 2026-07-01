-- Add site scoping to announcements so Sub2API, image playground, and video playground
-- only display announcements intended for that surface.
ALTER TABLE announcements
  ADD COLUMN IF NOT EXISTS site VARCHAR(20) NOT NULL DEFAULT 'main';

DO $$
BEGIN
  ALTER TABLE announcements
    ADD CONSTRAINT announcements_site_check CHECK (site IN ('main', 'image', 'video'));
EXCEPTION
  WHEN duplicate_object THEN NULL;
END $$;

CREATE INDEX IF NOT EXISTS idx_announcements_site ON announcements(site);
