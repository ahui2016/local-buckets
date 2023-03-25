package stmt

const CreateTables = `
CREATE TABLE IF NOT EXISTS bucket
(
	id             TEXT      PRIMARY KEY COLLATE NOCASE,
	title          TEXT      NOT NULL COLLATE NOCASE UNIQUE,
	subtitle       TEXT      NOT NULL,
	capacity       INTEGER   NOT NULL,
	encrypted      BOOLEAN   NOT NULL
);

CREATE TABLE IF NOT EXISTS file
(
	id         INTEGER   PRIMARY KEY,
	checksum   TEXT      NOT NULL COLLATE NOCASE UNIQUE,
	bucketid   TEXT      REFERENCES bucket(id) ON UPDATE CASCADE,
	name       TEXT      NOT NULL COLLATE NOCASE UNIQUE,
	notes      TEXT      NOT NULL,
	keywords   TEXT      NOT NULL,
	size       INTEGER   NOT NULL,
	type       TEXT      NOT NULL,
	like       INTEGER   NOT NULL,
	ctime      TEXT      NOT NULL,
	utime      TEXT      NOT NULL,
	checked    TEXT      NOT NULL,
	damaged    BOOLEAN   NOT NULL,
	deleted    BOOLEAN   NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_file_bucketid ON file(bucketid);
CREATE INDEX IF NOT EXISTS idx_file_notes ON file(notes);
CREATE INDEX IF NOT EXISTS idx_file_keywords ON file(keywords);
CREATE INDEX IF NOT EXISTS idx_file_ctime ON file(ctime);
CREATE INDEX IF NOT EXISTS idx_file_utime ON file(utime);
CREATE INDEX IF NOT EXISTS idx_file_checked ON file(checked);
`

const InsertBucket = `INSERT INTO bucket (
	id, title, subtitle, capacity, encrypted
) VALUES (?, ?, ?, ?, ?);`

const GetAllBuckets = `SELECT * FROM bucket;`

const GetBucket = `SELECT * FROM bucket WHERE id=?;`

const CountFilesInBucket = `SELECT count(*) FROM file WHERE bucketid=?;`

const InsertFile = `INSERT INTO file (
	checksum, bucketid, name,  notes, keywords, size,
	type,     like,     ctime, utime, checked,  damaged, deleted
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`

const UpdateFileContent = `UPDATE file
	SET checksum=?, size=?, utime=?, damaged=FALSE WHERE id=?;`

const UpdateFileInfo = `UPDATE file SET name=?, notes=?,
	keywords=?, type=?, like=?, ctime=?, utime=? WHERE id=?;`

const MoveFileToBucket = `UPDATE file SET bucketid=? WHERE id=?;`

const GetFileByID = `SELECT * FROM file WHERE id=?;`

const GetFileByName = `SELECT * FROM file WHERE name=?;`

const GetFileByChecksum = `SELECT * FROM file WHERE checksum=?;`

const GetRecentFiles = `SELECT * FROM file ORDER BY utime DESC LIMIT ?;`
