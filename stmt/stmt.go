package stmt

const CreateTables = `
CREATE TABLE IF NOT EXISTS bucket
(
	id           INTEGER   PRIMARY KEY,
	name         TEXT      NOT NULL COLLATE NOCASE UNIQUE,
	title        TEXT      NOT NULL COLLATE NOCASE UNIQUE,
	subtitle     TEXT      NOT NULL,
	encrypted    BOOLEAN   NOT NULL
);

CREATE TABLE IF NOT EXISTS file
(
	id          INTEGER   PRIMARY KEY,
	checksum    TEXT      NOT NULL COLLATE NOCASE UNIQUE,
	bucketid    INTEGER   REFERENCES bucket(id) ON UPDATE CASCADE,
	bucket_name TEXT      REFERENCES bucket(id) ON UPDATE CASCADE,
	name        TEXT      NOT NULL COLLATE NOCASE UNIQUE,
	notes       TEXT      NOT NULL,
	keywords    TEXT      NOT NULL,
	size        INTEGER   NOT NULL,
	type        TEXT      NOT NULL,
	like        INTEGER   NOT NULL,
	ctime       TEXT      NOT NULL,
	utime       TEXT      NOT NULL,
	checked     TEXT      NOT NULL,
	damaged     BOOLEAN   NOT NULL,
	deleted     BOOLEAN   NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_file_bucketid    ON file(bucketid);
CREATE INDEX IF NOT EXISTS idx_file_bucket_name ON file(bucket_name);
CREATE INDEX IF NOT EXISTS idx_file_notes       ON file(notes);
CREATE INDEX IF NOT EXISTS idx_file_keywords    ON file(keywords);
CREATE INDEX IF NOT EXISTS idx_file_ctime       ON file(ctime);
CREATE INDEX IF NOT EXISTS idx_file_utime       ON file(utime);
CREATE INDEX IF NOT EXISTS idx_file_checked     ON file(checked);
`

const InsertBucket = `INSERT INTO bucket (
	id, name, title, subtitle, encrypted
) VALUES (?, ?, ?, ?, ?);`

const DeleteBucket = `DELETE FROM bucket WHERE id=?;`
const UpdateBucketName = `UPDATE bucket SET name=? WHERE id=?;`
const UpdateBucketTitle = `UPDATE bucket SET title=?, subtitle=? WHERE id=?;`

const GetAllBuckets = `SELECT * FROM bucket;`
const GetBucket = `SELECT * FROM bucket WHERE id=?;`
const GetBucketByName = `SELECT * FROM bucket WHERE name=?;`
const CountFilesInBucket = `SELECT count(*) FROM file WHERE bucketid=?;`

const InsertFile = `INSERT INTO file (
	checksum, bucketid, bucket_name, name,  notes,   keywords, size,
	type,     like,     ctime,       utime, checked, damaged,  deleted
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`

const UpdateFileContent = `UPDATE file
	SET checksum=?, size=?, utime=?, damaged=FALSE WHERE id=?;`

const UpdateFileInfo = `UPDATE file SET name=?, notes=?,
	keywords=?, type=?, like=?, ctime=?, utime=? WHERE id=?;`

const UpdateBackupFileInfo = `UPDATE file SET
	checksum=?, bucketid=?, bucket_name, name=?,  notes=?, keywords=?,
	size=?,     type=?,     like=?,      ctime=?, utime=?, deleted=?
WHERE id=?;`

const MoveFileToBucket = `UPDATE file SET bucketid=?, bucket_name=? WHERE id=?;`

const GetFileByID = `SELECT * FROM file WHERE id=?;`
const GetFileByName = `SELECT * FROM file WHERE name=?;`
const GetFileByChecksum = `SELECT * FROM file WHERE checksum=?;`
const GetRecentFiles = `SELECT * FROM file ORDER BY utime DESC LIMIT ?;`
const GetAllFiles = `SELECT * FROM file;`
const DeleteFile = `DELETE FROM file WHERE id=?;`

const CountAllFiles = `SELECT count(*) FROM file;`
const CountFilesNeedCheck = `SELECT count(*) FROM file WHERE checked<?;`
const CountDamagedFiles = `SELECT count(*) FROM file WHERE damaged=TRUE;`
const TotalSize = `SELECT COALESCE(sum(size),0) as totalsize FROM file;`
