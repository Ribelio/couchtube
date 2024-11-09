package services

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/ozencb/couchtube/db"
	dbmodels "github.com/ozencb/couchtube/models/db"
	jsonmodels "github.com/ozencb/couchtube/models/json"
	repo "github.com/ozencb/couchtube/repositories"
)

type MediaService struct {
	TxManager   repo.TxManager
	ChannelRepo repo.ChannelRepository
	VideoRepo   repo.VideoRepository
}

func NewMediaService(txManager repo.TxManager, channelRepo repo.ChannelRepository, videoRepo repo.VideoRepository) *MediaService {
	return &MediaService{
		TxManager:   txManager,
		ChannelRepo: channelRepo,
		VideoRepo:   videoRepo,
	}
}

func (s *MediaService) FetchAllChannels() ([]dbmodels.Channel, error) {
	channels, err := s.ChannelRepo.FetchAllChannels()

	if err != nil {
		return nil, err
	}

	return channels, nil
}

func (s *MediaService) GetCurrentVideoByChannelId(channelId int) (*dbmodels.Video, error) {
	videos, err := s.VideoRepo.GetVideosByChannelID(channelId)
	if err != nil {
		return nil, err
	}

	if len(videos) == 0 {
		return nil, nil
	}

	totalLength := int64(0)
	for _, video := range videos {
		totalLength += int64(video.SectionEnd - video.SectionStart)
	}

	currentPoint := time.Now().UTC().Unix() % totalLength
	videoIndex := -1

	for i := range videos {
		video := &videos[i]
		sectionLength := int64(video.SectionEnd - video.SectionStart)
		if currentPoint < sectionLength {
			videoIndex = i
			video.SectionStart += int(currentPoint) // Adjust start to match the current second
			break
		}
		currentPoint -= sectionLength
	}

	if videoIndex == -1 {
		return &videos[0], nil
	}

	return &videos[videoIndex], nil
}

func (s *MediaService) FetchNextVideo(channelId int, videoId int) *dbmodels.Video {
	video, err := s.VideoRepo.FetchNextVideo(channelId, videoId)
	if err != nil {
		return nil
	}

	return video
}

func (s *MediaService) InvalidateVideo(videoId int) error {
	return db.WithTransaction(s.TxManager.GetDB(), func(tx *sql.Tx) error {
		return s.VideoRepo.DeleteVideo(tx, videoId)
	})
}

func (s *MediaService) SubmitList(list jsonmodels.SubmitListRequestJson) (bool, error) {
	videoListUrl := list.VideoListUrl

	if videoListUrl == "" {
		return false, nil
	}

	// Fetch the video list from the provided URL
	response, err := http.Get(videoListUrl)
	if err != nil {
		return false, err
	}
	defer response.Body.Close()

	var videoList jsonmodels.ChannelsJson
	if err := json.NewDecoder(response.Body).Decode(&videoList); err != nil {
		return false, err
	}

	if len(videoList.Channels) == 0 {
		return false, nil
	}

	err = db.WithTransaction(s.TxManager.GetDB(), func(tx *sql.Tx) error {
		if err := s.ChannelRepo.DeleteAllChannels(tx); err != nil {
			return err
		}
		if err := s.VideoRepo.DeleteAllVideos(tx); err != nil {
			return err
		}

		for _, channel := range videoList.Channels {
			channelID, err := s.ChannelRepo.InsertChannel(tx, channel.Name)
			if err != nil {
				return err
			}
			for _, video := range channel.Videos {
				if err := s.VideoRepo.SaveVideo(tx, channelID, video.Id, video.SectionStart, video.SectionEnd); err != nil {
					return err
				}
			}
		}
		return nil // No error, commit
	})

	if err != nil {
		return false, err
	}

	return true, nil
}
