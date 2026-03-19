package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

func columnExists(db *sql.DB, table, column string) bool {
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return false
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull int
		var dfltValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			return false
		}
		if name == column {
			return true
		}
	}
	return false
}

func tableExists(db *sql.DB, table string) bool {
	var name string
	err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
	return err == nil
}

func migrateSchema(db *sql.DB) error {
	if !tableExists(db, "channel_videos") {
		return nil
	}
	if columnExists(db, "channel_videos", "section_start") {
		return nil
	}
	if !columnExists(db, "videos", "section_start") {
		return nil
	}

	log.Println("Migrating schema: moving section times from videos to channel_videos...")

	if tableExists(db, "channel_videos_new") {
		db.Exec("DROP TABLE channel_videos_new")
	}
	if tableExists(db, "videos_new") && !tableExists(db, "videos") {
		db.Exec("ALTER TABLE videos_new RENAME TO videos")
		log.Println("Schema migration recovered from partial state.")
		return nil
	}
	if tableExists(db, "videos_new") && tableExists(db, "videos") {
		db.Exec("DROP TABLE videos_new")
	}

	_, err := db.Exec(`CREATE TABLE channel_videos_new (
		"channel_id" INTEGER NOT NULL,
		"video_id" TEXT NOT NULL,
		"section_start" INTEGER NOT NULL,
		"section_end" INTEGER NOT NULL,
		"position" INTEGER NOT NULL DEFAULT 0,
		FOREIGN KEY(channel_id) REFERENCES channels(id) ON DELETE CASCADE,
		FOREIGN KEY(video_id) REFERENCES videos(id),
		UNIQUE(channel_id, video_id),
		CHECK (section_end > section_start)
	)`)
	if err != nil {
		return fmt.Errorf("create channel_videos_new: %w", err)
	}

	_, err = db.Exec(`INSERT INTO channel_videos_new (channel_id, video_id, section_start, section_end, position)
		SELECT cv.channel_id, cv.video_id, v.section_start, v.section_end,
			(SELECT COUNT(*) FROM channel_videos cv2
			 WHERE cv2.channel_id = cv.channel_id AND cv2.rowid < cv.rowid) as pos
		FROM channel_videos cv
		JOIN videos v ON cv.video_id = v.id`)
	if err != nil {
		db.Exec("DROP TABLE channel_videos_new")
		return fmt.Errorf("copy channel_videos data: %w", err)
	}

	if _, err = db.Exec("DROP TABLE channel_videos"); err != nil {
		return fmt.Errorf("drop old channel_videos: %w", err)
	}
	if _, err = db.Exec("ALTER TABLE channel_videos_new RENAME TO channel_videos"); err != nil {
		return fmt.Errorf("rename channel_videos_new: %w", err)
	}

	if _, err = db.Exec(`CREATE TABLE videos_new ("id" TEXT NOT NULL PRIMARY KEY)`); err != nil {
		return fmt.Errorf("create videos_new: %w", err)
	}
	if _, err = db.Exec(`INSERT INTO videos_new (id) SELECT id FROM videos`); err != nil {
		db.Exec("DROP TABLE videos_new")
		return fmt.Errorf("copy videos data: %w", err)
	}
	if _, err = db.Exec("DROP TABLE videos"); err != nil {
		return fmt.Errorf("drop old videos: %w", err)
	}
	if _, err = db.Exec("ALTER TABLE videos_new RENAME TO videos"); err != nil {
		return fmt.Errorf("rename videos_new: %w", err)
	}

	db.Exec("CREATE INDEX IF NOT EXISTS idx_videos_channel_id ON channel_videos(channel_id, video_id)")
	log.Println("Schema migration completed successfully.")
	return nil
}

func createTables(db *sql.DB) error {
	createVideosTableQuery := `CREATE TABLE IF NOT EXISTS videos (
		"id" TEXT NOT NULL PRIMARY KEY
	);`
	createChannelsTableQuery := `CREATE TABLE IF NOT EXISTS channels (
		"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		"name" TEXT,
		UNIQUE(name)
	);`
	createChannelVideosTableQuery := `CREATE TABLE IF NOT EXISTS channel_videos (
		"channel_id" INTEGER NOT NULL,
		"video_id" TEXT NOT NULL,
		"section_start" INTEGER NOT NULL,
		"section_end" INTEGER NOT NULL,
		"position" INTEGER NOT NULL DEFAULT 0,
		FOREIGN KEY(channel_id) REFERENCES channels(id) ON DELETE CASCADE,
		FOREIGN KEY(video_id) REFERENCES videos(id),
		UNIQUE(channel_id, video_id),
		CHECK (section_end > section_start)
	);`
	createIndexesQuery := `CREATE INDEX IF NOT EXISTS idx_videos_channel_id ON channel_videos(channel_id, video_id);`

	_, err := db.Exec(createChannelsTableQuery + createVideosTableQuery + createChannelVideosTableQuery + createIndexesQuery)
	if err != nil {
		log.Fatal(err)
		return err
	}

	log.Println("Database initialized and tables created successfully.")
	return nil
}


func InitDatabase(db *sql.DB) {
	if err := migrateSchema(db); err != nil {
		log.Fatal("Failed to migrate schema:", err)
	}
	if err := createTables(db); err != nil {
		log.Fatal("Failed to create tables:", err)
	}
}
