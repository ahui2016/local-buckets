package stmt

const CreateTables = `
CREATE TABLE IF NOT EXISTS bucket
(
	id             TEXT      PRIMARY KEY COLLATE NOCASE,
	title          TEXT      NOT NULL COLLATE NOCASE UNIQUE,
	subtitle       TEXT      NOT NULL,
	capacity       INTEGER   NOT NULL,
	max_filesize   INTEGER   NOT NULL,
	encrypted      BOOLEAN   NOT NULL
);

CREATE TABLE IF NOT EXISTS file
(
	id         INTEGER   PRIMARY KEY,
	adler32    TEXT      NOT NULL,
	sha256     TEXT      NOT NULL,
	bucketid   TEXT      REFERENCES bucket(id) ON UPDATE CASCADE,
	name       TEXT      NOT NULL,
	notes      TEXT      NOT NULL,
	keywords   TEXT      NOT NULL,
	size       INTEGER   NOT NULL,
	type       TEXT      NOT NULL,
	like       INTEGER   NOT NULL,
	ctime      TEXT      NOT NULL,
	utime      TEXT      NOT NULL,
	checked    TEXT      NOT NULL,
	damaged    BOOLEAN   NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_file_bucketid ON file(bucketid);
CREATE INDEX IF NOT EXISTS idx_file_name ON file(name);
CREATE INDEX IF NOT EXISTS idx_file_notes ON file(notes);
CREATE INDEX IF NOT EXISTS idx_file_keywords ON file(keywords);
CREATE INDEX IF NOT EXISTS idx_file_ctime ON file(ctime);
CREATE INDEX IF NOT EXISTS idx_file_utime ON file(utime);
CREATE INDEX IF NOT EXISTS idx_file_checked ON file(checked);
`

const InsertBucket = `INSERT INTO bucket (
	id, title, subtitle, capacity, max_filesize, encrypted
) VALUES (?, ?, ?, ?, ?, ?);`

const GetAllBuckets = `SELECT * FROM bucket;`

const InsertFile = `INSERT INTO file (
	adler32, sha256, bucketid, name,  notes,   keywords, size,
	type,    like,   ctime,    utime, checked, damaged
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`
