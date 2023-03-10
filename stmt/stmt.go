package stmt

const CreateTables = `
CREATE TABLE IF NOT EXISTS bucket
(
	id             TEXT      PRIMARY KEY COLLATE NOCASE,
	title          TEXT      NOT NULL,
	subtitle       TEXT      NOT NULL,
	capacity       INTEGER   NOT NULL,
	max_filesize   INTEGER   NOT NULL,
	encrypted      BOOLEAN   NOT NULL
);
`

const InsertBucket = `INSERT INTO bucket (
	id, title, subtitle, capacity, max_filesize, encrypted
) VALUES (?, ?, ?, ?, ?, ?);`

const GetAllBuckets = `SELECT * FROM bucket;`
