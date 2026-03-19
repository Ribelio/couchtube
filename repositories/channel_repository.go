package repo

import (
	"database/sql"
	"fmt"

	dbmodels "github.com/ozencb/couchtube/models/db"
)

type ChannelRepository interface {
	FetchAllChannels() ([]dbmodels.Channel, error)
	FetchAllChannelsWithVideos() ([]dbmodels.Channel, []dbmodels.Video, error)
	InsertChannel(tx *sql.Tx, channelName string) (int, error)
	RenameChannel(channelID int, name string) error
	DeleteChannel(channelID int) error
	DeleteAllChannels(tx *sql.Tx) error
	ReorderChannels(channelIDs []int) error
}

type channelRepository struct {
	db *sql.DB
}

func NewChannelRepository(db *sql.DB) ChannelRepository {
	return &channelRepository{db: db}
}

func (r *channelRepository) BeginTx() (*sql.Tx, error) {
	return r.db.Begin()
}

func (r *channelRepository) FetchAllChannels() ([]dbmodels.Channel, error) {
	query := `
    SELECT id, name, position
    FROM channels
    WHERE EXISTS (
        SELECT 1 FROM channel_videos
        WHERE channel_videos.channel_id = channels.id
    )
    ORDER BY position;`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []dbmodels.Channel
	for rows.Next() {
		var channel dbmodels.Channel
		if err := rows.Scan(&channel.ID, &channel.Name, &channel.Position); err != nil {
			return nil, err
		}
		channels = append(channels, channel)
	}

	return channels, rows.Err()
}

func (r *channelRepository) FetchAllChannelsWithVideos() ([]dbmodels.Channel, []dbmodels.Video, error) {
	channelRows, err := r.db.Query(`SELECT id, name, position FROM channels ORDER BY position`)
	if err != nil {
		return nil, nil, err
	}
	defer channelRows.Close()

	var channels []dbmodels.Channel
	for channelRows.Next() {
		var channel dbmodels.Channel
		if err := channelRows.Scan(&channel.ID, &channel.Name, &channel.Position); err != nil {
			return nil, nil, err
		}
		channels = append(channels, channel)
	}
	if err := channelRows.Err(); err != nil {
		return nil, nil, err
	}

	videoRows, err := r.db.Query(`
		SELECT cv.channel_id, cv.video_id, cv.section_start, cv.section_end, cv.position
		FROM channel_videos cv
		ORDER BY cv.channel_id, cv.position
	`)
	if err != nil {
		return nil, nil, err
	}
	defer videoRows.Close()

	var videos []dbmodels.Video
	for videoRows.Next() {
		var video dbmodels.Video
		if err := videoRows.Scan(&video.ChannelID, &video.ID, &video.SectionStart, &video.SectionEnd, &video.Position); err != nil {
			return nil, nil, err
		}
		videos = append(videos, video)
	}
	if err := videoRows.Err(); err != nil {
		return nil, nil, err
	}

	return channels, videos, nil
}

func (r *channelRepository) InsertChannel(tx *sql.Tx, channelName string) (int, error) {
	exec := r.db.Exec
	if tx != nil {
		exec = tx.Exec
	}

	result, err := exec("INSERT INTO channels (name, position) VALUES (?, (SELECT COALESCE(MAX(position), -1) + 1 FROM channels)) RETURNING id", channelName)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	return int(id), err
}

func (r *channelRepository) RenameChannel(channelID int, name string) error {
	result, err := r.db.Exec(`UPDATE channels SET name = ? WHERE id = ?`, name, channelID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("no channel found with id %d", channelID)
	}
	return nil
}

func (r *channelRepository) DeleteChannel(channelID int) error {
	result, err := r.db.Exec(`DELETE FROM channels WHERE id = ?`, channelID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("no channel found with id %d", channelID)
	}
	return nil
}

func (r *channelRepository) DeleteAllChannels(tx *sql.Tx) error {
	exec := r.db.Exec
	if tx != nil {
		exec = tx.Exec
	}

	_, err := exec("DELETE FROM channels")
	return err
}

func (r *channelRepository) ReorderChannels(channelIDs []int) error {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM channels`).Scan(&count)
	if err != nil {
		return err
	}
	if count != len(channelIDs) {
		return fmt.Errorf("channel count mismatch: expected %d, got %d", count, len(channelIDs))
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	stmt, err := tx.Prepare(`UPDATE channels SET position = ? WHERE id = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i, channelID := range channelIDs {
		result, err := stmt.Exec(i, channelID)
		if err != nil {
			return err
		}
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			return fmt.Errorf("channel %d not found", channelID)
		}
	}

	return tx.Commit()
}
