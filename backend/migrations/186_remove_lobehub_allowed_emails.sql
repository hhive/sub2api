-- The LobeHub menu is no longer restricted by email; remove the obsolete setting.
DELETE FROM settings
WHERE key = 'lobehub_allowed_emails';
