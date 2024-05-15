--
-- Schema for very simple serial log.
--
CREATE TABLE IF NOT EXISTS log (
	ts INTEGER,
	device TEXT,
	msg TEXT
);