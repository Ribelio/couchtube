package repo

import (
	"database/sql"
	"fmt"

	dbmodels "github.com/ozencb/couchtube/models/db"
)

type VideoRepository interface {
	GetVideosByChannelID(channelID int) ([]dbmodels.Video, error)
	FetchNextVideo(videoID string, channelID int) (*dbmodels.Video, error)
	SaveVideo(tx *sql.Tx, channelID int, videoUrl string, sectionStart int, sectionEnd int) error
	AddVideoToChannel(channelID int, videoID string, sectionStart int, sectionEnd int) (*dbmodels.Video, error)
	UpdateVideoInChannel(channelID int, videoID string, sectionStart int, sectionEnd int) (*dbmodels.Video, error)
	RemoveVideoFromChannel(channelID int, videoID string) error
	ReorderVideos(channelID int, videoIDs []string) error
	DeleteVideo(tx *sql.Tx, videoID string) error
	DeleteAllChannelVideos(tx *sql.Tx) error
	DeleteAllVideos(tx *sql.Tx) error
}

type videoRepository struct {
	db *sql.DB
}

func NewVideoRepository(db *sql.DB) VideoRepository {
	return &videoRepository{db: db}
}

func (r *videoRepository) GetVideosByChannelID(channelID int) ([]dbmodels.Video, error) {
	rows, err := r.db.Query(`
        SELECT cv.video_id, cv.section_start, cv.section_end, cv.position
        FROM channel_videos cv
        WHERE cv.channel_id = ?
        ORDER BY cv.position
    `, channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var videos []dbmodels.Video
	for rows.Next() {
		var video dbmodels.Video
		if err := rows.Scan(&video.ID, &video.SectionStart, &video.SectionEnd, &video.Position); err != nil {
			return nil, err
		}
		videos = append(videos, video)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return videos, nil
}

func (r *videoRepository) FetchNextVideo(videoID string, channelID int) (*dbmodels.Video, error) {
	var currentPosition int
	err := r.db.QueryRow(`SELECT position FROM channel_videos WHERE channel_id = ? AND video_id = ?`,
		channelID, videoID).Scan(&currentPosition)
	if err != nil {
		currentPosition = -1
	}
	row := r.db.QueryRow(`
		SELECT cv.video_id, cv.section_start, cv.section_end, cv.position
		FROM channel_videos cv
		WHERE cv.channel_id = ? AND cv.position > ?
		ORDER BY cv.position ASC
		LIMIT 1
	`, channelID, currentPosition)
	var video dbmodels.Video
	err = row.Scan(&video.ID, &video.SectionStart, &video.SectionEnd, &video.Position)
	if err == sql.ErrNoRows {
		row = r.db.QueryRow(`
			SELECT cv.video_id, cv.section_start, cv.section_end, cv.position
			FROM channel_videos cv
			WHERE cv.channel_id = ?
			ORDER BY cv.position ASC
			LIMIT 1
		`, channelID)
		err = row.Scan(&video.ID, &video.SectionStart, &video.SectionEnd, &video.Position)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	return &video, nil
}

func (r *videoRepository) SaveVideo(tx *sql.Tx, channelID int, videoId string, sectionStart int, sectionEnd int) error {
	dbExec := r.db.Exec
	if tx != nil {
		dbExec = tx.Exec
	}
	_, err := dbExec(`INSERT OR IGNORE INTO videos (id) VALUES (?)`, videoId)
	if err != nil {
		return err
	}
	_, err = dbExec(`
        INSERT OR IGNORE INTO channel_videos (channel_id, video_id, section_start, section_end, position)
        VALUES (?, ?, ?, ?, (SELECT COALESCE(MAX(position), -1) + 1 FROM channel_videos WHERE channel_id = ?))
    `, channelID, videoId, sectionStart, sectionEnd, channelID)
	return err
}

func (r *videoRepository) AddVideoToChannel(channelID int, videoID string, sectionStart int, sectionEnd int) (*dbmodels.Video, error) {
	_, err := r.db.Exec(`INSERT OR IGNORE INTO videos (id) VALUES (?)`, videoID)
	if err != nil {
		return nil, err
	}
	var position int
	err = r.db.QueryRow(`
		INSERT INTO channel_videos (channel_id, video_id, section_start, section_end, position)
		VALUES (?, ?, ?, ?, (SELECT COALESCE(MAX(position), -1) + 1 FROM channel_videos WHERE channel_id = ?))
		RETURNING position
	`, channelID, videoID, sectionStart, sectionEnd, channelID).Scan(&position)
	if err != nil {
		return nil, err
	}
	return &dbmodels.Video{
		ID: videoID, ChannelID: channelID,
		SectionStart: sectionStart, SectionEnd: sectionEnd, Position: position,
	}, nil
}

func (r *videoRepository) UpdateVideoInChannel(channelID int, videoID string, sectionStart int, sectionEnd int) (*dbmodels.Video, error) {
	var position int
	err := r.db.QueryRow(`
		UPDATE channel_videos SET section_start = ?, section_end = ?
		WHERE channel_id = ? AND video_id = ?
		RETURNING position
	`, sectionStart, sectionEnd, channelID, videoID).Scan(&position)
	if err != nil {
		return nil, err
	}
	return &dbmodels.Video{
		ID: videoID, ChannelID: channelID,
		SectionStart: sectionStart, SectionEnd: sectionEnd, Position: position,
	}, nil
}

func (r *videoRepository) RemoveVideoFromChannel(channelID int, videoID string) error {
	result, err := r.db.Exec(`DELETE FROM channel_videos WHERE channel_id = ? AND video_id = ?`, channelID, videoID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("no video %s found in channel %d", videoID, channelID)
	}
	return nil
}

func (r *videoRepository) ReorderVideos(channelID int, videoIDs []string) error {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM channel_videos WHERE channel_id = ?`, channelID).Scan(&count)
	if err != nil {
		return err
	}
	if count != len(videoIDs) {
		return fmt.Errorf("video count mismatch: expected %d, got %d", count, len(videoIDs))
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

	stmt, err := tx.Prepare(`UPDATE channel_videos SET position = ? WHERE channel_id = ? AND video_id = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i, videoID := range videoIDs {
		result, err := stmt.Exec(i, channelID, videoID)
		if err != nil {
			return err
		}
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			return fmt.Errorf("video %s not found in channel %d", videoID, channelID)
		}
	}

	return tx.Commit()
}

func (r *videoRepository) DeleteVideo(tx *sql.Tx, videoID string) error {
	dbExec := r.db.Exec
	if tx != nil {
		dbExec = tx.Exec
	}
	result, err := dbExec(`DELETE FROM channel_videos WHERE video_id = ?`, videoID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("no video found with id %s", videoID)
	}
	return nil
}

func (r *videoRepository) DeleteAllChannelVideos(tx *sql.Tx) error {
	exec := r.db.Exec
	if tx != nil {
		exec = tx.Exec
	}
	_, err := exec("DELETE FROM channel_videos")
	return err
}

func (r *videoRepository) DeleteAllVideos(tx *sql.Tx) error {
	exec := r.db.Exec
	if tx != nil {
		exec = tx.Exec
	}

	_, err := exec("DELETE FROM videos")
	return err
}
