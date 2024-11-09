package repo

import (
	"database/sql"
	"fmt"

	dbmodels "github.com/ozencb/couchtube/models/db"
)

type VideoRepository interface {
	GetVideosByChannelID(channelID int) ([]dbmodels.Video, error)
	FetchNextVideo(videoID int, channelID int) (*dbmodels.Video, error)
	SaveVideo(tx *sql.Tx, channelID int, videoUrl string, sectionStart int, sectionEnd int) error
	DeleteVideo(tx *sql.Tx, videoID int) error
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
        SELECT id, section_start, section_end
        FROM videos
		JOIN channel_videos ON videos.id = channel_videos.video_id
		WHERE channel_videos.channel_id = ?
    `, channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var videos []dbmodels.Video
	for rows.Next() {
		var video dbmodels.Video
		if err := rows.Scan(&video.ID, &video.SectionStart, &video.SectionEnd); err != nil {
			return nil, err
		}
		videos = append(videos, video)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return videos, nil
}

func (r *videoRepository) FetchNextVideo(videoID int, channelID int) (*dbmodels.Video, error) {
	row := r.db.QueryRow(`
		SELECT id, section_start, section_end
		FROM videos
		JOIN channel_videos ON videos.id = channel_videos.video_id
		WHERE channel_videos.channel_id = ? AND videos.id > ?
		ORDER BY id ASC
		LIMIT 1
	`, channelID, videoID)

	var video dbmodels.Video
	err := row.Scan(&video.ID, &video.SectionStart, &video.SectionEnd)
	if err == sql.ErrNoRows {
		// If no next video is found, get the first video instead
		row = r.db.QueryRow(`
			SELECT id, section_start, section_end
			FROM videos
			JOIN channel_videos ON videos.id = channel_videos.video_id
			WHERE channels.id = ?
			ORDER BY id ASC
			LIMIT 1
		`, channelID)

		err = row.Scan(&video.ID, &video.SectionStart, &video.SectionEnd)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	return &video, nil
}

func (r *videoRepository) SaveVideo(tx *sql.Tx, channelID int, videoId string, sectionStart int, sectionEnd int) error {
	exec := r.db.Exec
	if tx != nil {
		exec = tx.Exec
	}

	_, err := exec(`
        INSERT OR IGNORE INTO videos (id, section_start, section_end)
        VALUES (?, ?, ?)
    `, videoId, sectionStart, sectionEnd)
	if err != nil {
		return err
	}

	_, err = exec(`
        INSERT OR IGNORE INTO channel_videos (channel_id, video_id)
        VALUES (?, ?)
    `, channelID, videoId)

	return err
}

func (r *videoRepository) DeleteVideo(tx *sql.Tx, videoID int) error {
	exec := r.db.Exec
	if tx != nil {
		exec = tx.Exec
	}

	result, err := exec(`
        DELETE FROM videos
        WHERE id = ?
    `, videoID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("no video found with id %d", videoID)
	}

	return nil
}

func (r *videoRepository) DeleteAllVideos(tx *sql.Tx) error {
	exec := r.db.Exec
	if tx != nil {
		exec = tx.Exec
	}

	_, err := exec("DELETE FROM videos")
	return err
}
