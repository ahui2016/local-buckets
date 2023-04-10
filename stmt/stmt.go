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
	bucket_name TEXT      REFERENCES bucket(name) ON UPDATE CASCADE,
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

CREATE INDEX IF NOT EXISTS idx_file_bucket_name ON file(bucket_name);
CREATE INDEX IF NOT EXISTS idx_file_notes       ON file(notes);
CREATE INDEX IF NOT EXISTS idx_file_keywords    ON file(keywords);
CREATE INDEX IF NOT EXISTS idx_file_ctime       ON file(ctime);
CREATE INDEX IF NOT EXISTS idx_file_utime       ON file(utime);
CREATE INDEX IF NOT EXISTS idx_file_checked     ON file(checked);
`

const InsertBucket = `INSERT INTO bucket (
	name, title, subtitle, encrypted
) VALUES (?, ?, ?, ?);`

const InsertBucketWithID = `INSERT INTO bucket (
	id, name, title, subtitle, encrypted
) VALUES (?, ?, ?, ?, ?);`

const DeleteBucket = `DELETE FROM bucket WHERE id=?;`
const UpdateBucketName = `UPDATE bucket SET name=? WHERE id=?;`
const UpdateBucketTitle = `UPDATE bucket SET title=?, subtitle=? WHERE id=?;`

const GetAllBuckets = `SELECT * FROM bucket;`
const GetPublicBuckets = `SELECT * FROM bucket WHERE encrypted=FALSE;`
const GetPrivateBuckets = `SELECT * FROM bucket WHERE encrypted=TRUE;`
const GetBucket = `SELECT * FROM bucket WHERE id=?;`
const GetBucketByName = `SELECT * FROM bucket WHERE name=?;`
const CountFilesInBucket = `SELECT count(*) FROM file WHERE bucket_name=?;`

const InsertFile = `INSERT INTO file (
	checksum, bucket_name, name,  notes,   keywords, size,   type,
	like,     ctime,       utime, checked, damaged,  deleted
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`

const InsertFileWithID = `INSERT INTO file (
	id,   checksum, bucket_name, name,  notes,   keywords, size,
	type, like,     ctime,       utime, checked, damaged,  deleted
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`

const UpdateFileContent = `UPDATE file
	SET checksum=?, size=?, utime=?, damaged=FALSE WHERE id=?;`

const MoveFileToBucket = `UPDATE file SET bucket_name=? WHERE id=?;`

const UpdateChecksumAndBucket = `UPDATE file
	SET checksum=?, bucket_name=? WHERE id=?;`

const UpdateFileInfo = `UPDATE file SET name=?, notes=?,
	keywords=?, type=?, like=?, ctime=?, utime=? WHERE id=?;`

const UpdateBackupFileInfo = `UPDATE file SET
	checksum=?, bucket_name, name=?,  notes=?, keywords=?, size=?,
	type=?,     like=?,      ctime=?, utime=?, deleted=? WHERE id=?;`

const GetFileByID = `SELECT * FROM file WHERE id=?;`
const GetFileByName = `SELECT * FROM file WHERE name=?;`
const GetFileByChecksum = `SELECT * FROM file WHERE checksum=?;`
const GetAllFiles = `SELECT * FROM file;`
const DeleteFile = `DELETE FROM file WHERE id=?;`

const GetFilePlus = `SELECT file.id, file.checksum, file.bucket_name,
	file.name,    file.notes,   file.keywords, file.size,
	file.type,    file.like,    file.ctime,    file.utime,
	file.checked, file.damaged, file.deleted,  bucket.encrypted
FROM file
	INNER JOIN bucket ON file.bucket_name = bucket.name
	WHERE file.id=?;`

const GetFilePlusByName = `SELECT file.id, file.checksum, file.bucket_name,
	file.name,    file.notes,   file.keywords, file.size,
	file.type,    file.like,    file.ctime,    file.utime,
	file.checked, file.damaged, file.deleted,  bucket.encrypted
FROM file
	INNER JOIN bucket ON file.bucket_name = bucket.name
	WHERE file.name=?;`

const GetAllRecentFiles = `SELECT file.id, file.checksum, file.bucket_name,
	file.name,    file.notes,   file.keywords, file.size,
	file.type,    file.like,    file.ctime,    file.utime,
	file.checked, file.damaged, file.deleted,  bucket.encrypted
FROM file
	INNER JOIN bucket ON file.bucket_name = bucket.name
	ORDER BY utime DESC LIMIT ?;`

const GetAllRecentPics = `SELECT file.id, file.checksum, file.bucket_name,
	file.name,    file.notes,   file.keywords, file.size,
	file.type,    file.like,    file.ctime,    file.utime,
	file.checked, file.damaged, file.deleted,  bucket.encrypted
FROM file
	INNER JOIN bucket ON file.bucket_name = bucket.name
	WHERE file.type LIKE "image/%"
	ORDER BY utime DESC LIMIT ?;`

const GetPublicRecentFiles = `SELECT file.id, file.checksum, file.bucket_name,
	file.name,    file.notes,   file.keywords, file.size,
	file.type,    file.like,    file.ctime,    file.utime,
	file.checked, file.damaged, file.deleted,  bucket.encrypted
FROM file
	INNER JOIN bucket ON file.bucket_name = bucket.name
	WHERE bucket.encrypted=FALSE
	ORDER BY file.utime DESC LIMIT ?;`

const GetPublicRecentPics = `SELECT file.id, file.checksum, file.bucket_name,
	file.name,    file.notes,   file.keywords, file.size,
	file.type,    file.like,    file.ctime,    file.utime,
	file.checked, file.damaged, file.deleted,  bucket.encrypted
FROM file
	INNER JOIN bucket ON file.bucket_name = bucket.name
	WHERE bucket.encrypted=FALSE AND file.type LIKE "image/%"
	ORDER BY file.utime DESC LIMIT ?;`

const CountAllFiles = `SELECT count(*) FROM file;`
const CountFilesNeedCheck = `SELECT count(*) FROM file WHERE checked<?;`
const CountDamagedFiles = `SELECT count(*) FROM file WHERE damaged=TRUE;`
const TotalSize = `SELECT COALESCE(sum(size),0) as totalsize FROM file;`
