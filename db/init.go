package db

import (
	"database/sql"
	"log"

	"github.com/ozencb/couchtube/helpers"
	jsonmodels "github.com/ozencb/couchtube/models/json"
	_ "modernc.org/sqlite"
)

func createTables(db *sql.DB) error {
	createVideosTableQuery := `CREATE TABLE IF NOT EXISTS videos (
		"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,		
		"channel_id" INTEGER NOT NULL,
		"url" TEXT NOT NULL,
		"section_start" INTEGER NOT NULL,
		"section_end" INTEGER NOT NULL,
		UNIQUE(url, channel_id),
		FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE,
		CHECK (section_end > section_start)
	);`
	createChannelsTableQuery := `CREATE TABLE IF NOT EXISTS channels (
		"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		"name" TEXT,
		UNIQUE(name)
	);`
	createIndexesQuery := `CREATE INDEX IF NOT EXISTS idx_videos_channel_id ON videos(channel_id);`

	_, err := db.Exec(createChannelsTableQuery + createVideosTableQuery + createIndexesQuery)
	if err != nil {
		log.Fatal(err)
		return err
	}

	log.Println("Database initialized and tables created successfully.")
	return nil
}

func populateDatabase(db *sql.DB) error {
	// parse the json file and insert the data into the database
	// ignore if there are channels already defined

	channels, err := helpers.LoadJSONFromFile[jsonmodels.ChannelsJson]("/default-channels.json")
	if err != nil {
		log.Fatal(err)
		return err
	}

	// check if anything already exists in channels
	var exists int
	err = db.QueryRow(`SELECT EXISTS(SELECT 1 FROM channels LIMIT 1);`).Scan(&exists)
	if err != nil {
		log.Fatal(err)
		return err
	}

	if exists == 1 {
		log.Println("Data already exists in the database. Skipping db population")
		return nil
	}

	insertChannelQuery := `INSERT OR IGNORE INTO channels (name) VALUES (?)`
	insertVideoQuery := `INSERT OR IGNORE INTO videos (channel_id, url, section_start, section_end) VALUES (?, ?, ?, ?)`

	tx, err := db.Begin()
	if err != nil {
		log.Fatal("Failed to start database transaction:", err)
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
			log.Println("Database transaction rolled back due to error:", err)
		}
	}()

	for _, channel := range channels.Channels {
		if len(channel.Videos) == 0 {
			log.Printf("Channel %s has no videos. Skipping.\n", channel.Name)
			continue
		}

		channelID, err := insertOrGetChannelID(tx, channel.Name, insertChannelQuery)
		if err != nil {
			log.Fatal(err)
			return err
		}

		for _, video := range channel.Videos {
			_, err = tx.Exec(insertVideoQuery, channelID, video.Url, video.SectionStart, video.SectionEnd)
			if err != nil {
				log.Fatal(err)
				return err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		log.Fatal("Failed to commit database transaction:", err)
		return err
	}

	log.Println("Data inserted successfully.")
	return nil
}

func insertOrGetChannelID(tx *sql.Tx, name, query string) (int64, error) {
	result, err := tx.Exec(query, name)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	if rowsAffected > 0 {
		return result.LastInsertId()
	}

	var existingID int64
	getIDQuery := `SELECT id FROM channels WHERE name = ?`
	err = tx.QueryRow(getIDQuery, name).Scan(&existingID)
	if err != nil {
		return 0, err
	}

	return existingID, nil
}

func InitDatabase(db *sql.DB) {
	if err := createTables(db); err != nil {
		log.Fatal("Failed to create tables:", err)
	}
	if err := populateDatabase(db); err != nil {
		log.Println("Database already populated or error occurred:", err)
	}
}
