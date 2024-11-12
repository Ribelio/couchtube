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
		"id" TEXT NOT NULL PRIMARY KEY,		
		"section_start" INTEGER NOT NULL,
		"section_end" INTEGER NOT NULL,
		CHECK (section_end > section_start)
	);`
	createChannelsTableQuery := `CREATE TABLE IF NOT EXISTS channels (
		"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		"name" TEXT,
		UNIQUE(name)
	);`
	createChannelVideosTableQuery := `CREATE TABLE IF NOT EXISTS channel_videos (
		"channel_id" INTEGER NOT NULL,
		"video_id" TEXT NOT NULL,
		FOREIGN KEY(channel_id) REFERENCES channels(id) ON DELETE CASCADE,
		FOREIGN KEY(video_id) REFERENCES videos(id) ON DELETE CASCADE,
		UNIQUE(channel_id, video_id)
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

func populateDatabase(db *sql.DB) error {
	// Parse the JSON file and insert data into the database, if channels are not defined.
	channels, err := helpers.LoadJSONFromFile[jsonmodels.ChannelsJson]("/videos.json")
	if err != nil {
		log.Fatal(err)
		return err
	}

	// Check if any channels already exist to avoid re-population.
	var exists int
	err = db.QueryRow(`SELECT EXISTS(SELECT 1 FROM channels LIMIT 1);`).Scan(&exists)
	if err != nil {
		log.Fatal(err)
		return err
	}

	if exists == 1 {
		log.Println("Data already exists in the database. Skipping population.")
		return nil
	}

	return WithTransaction(db, func(tx *sql.Tx) error {
		insertChannelQuery := `INSERT OR IGNORE INTO channels (name) VALUES (?)`
		insertVideoQuery := `INSERT OR IGNORE INTO videos (id, section_start, section_end) VALUES (?, ?, ?)`
		insertChannelVideoQuery := `INSERT OR IGNORE INTO channel_videos (channel_id, video_id) VALUES (?, ?)`

		for _, channel := range channels.Channels {
			if len(channel.Videos) == 0 {
				log.Printf("Channel %s has no videos. Skipping.\n", channel.Name)
				continue
			}

			channelID, err := insertOrGetChannelID(tx, channel.Name, insertChannelQuery)
			if err != nil {
				return err
			}

			for _, video := range channel.Videos {
				videoID, err := insertOrGetVideoID(tx, video, insertVideoQuery)
				if err != nil {
					return err
				}

				// Insert channel-video relationship
				_, err = tx.Exec(insertChannelVideoQuery, channelID, videoID)
				if err != nil {
					return err
				}
			}
		}

		log.Println("Data inserted successfully.")
		return nil
	})
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

func insertOrGetVideoID(tx *sql.Tx, video jsonmodels.VideoJson, query string) (string, error) {
	result, err := tx.Exec(query, video.Id, video.SectionStart, video.SectionEnd)
	if err != nil {
		return "", err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return "", err
	}

	if rowsAffected > 0 {
		return video.Id, nil
	}

	var existingID string
	getIDQuery := `SELECT id FROM videos WHERE id = ?`
	err = tx.QueryRow(getIDQuery, video.Id).Scan(&existingID)
	if err != nil {
		return "", err
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
