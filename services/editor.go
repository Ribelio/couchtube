package services

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/ozencb/couchtube/config"
	"github.com/ozencb/couchtube/db"
	"github.com/ozencb/couchtube/helpers"
	dbmodels "github.com/ozencb/couchtube/models/db"
	jsonmodels "github.com/ozencb/couchtube/models/json"
	repo "github.com/ozencb/couchtube/repositories"
)

var playlistIDRegex = regexp.MustCompile(`[?&]list=([A-Za-z0-9_-]+)`)

type EditorService struct {
	TxManager   repo.TxManager
	ChannelRepo repo.ChannelRepository
	VideoRepo   repo.VideoRepository
}

func NewEditorService(txManager repo.TxManager, channelRepo repo.ChannelRepository, videoRepo repo.VideoRepository) *EditorService {
	return &EditorService{
		TxManager:   txManager,
		ChannelRepo: channelRepo,
		VideoRepo:   videoRepo,
	}
}

type EditorVideo struct {
	VideoID      string `json:"videoId"`
	SectionStart int    `json:"sectionStart"`
	SectionEnd   int    `json:"sectionEnd"`
	Position     int    `json:"position"`
}

type EditorChannel struct {
	ID     int           `json:"id"`
	Name   string        `json:"name"`
	Videos []EditorVideo `json:"videos"`
}

func (s *EditorService) FetchAllChannels() ([]EditorChannel, error) {
	channels, videos, err := s.ChannelRepo.FetchAllChannelsWithVideos()
	if err != nil {
		return nil, err
	}

	channelMap := make(map[int]*EditorChannel, len(channels))
	result := make([]EditorChannel, 0, len(channels))
	for _, ch := range channels {
		ec := EditorChannel{ID: ch.ID, Name: ch.Name, Videos: []EditorVideo{}}
		result = append(result, ec)
		channelMap[ch.ID] = &result[len(result)-1]
	}

	for _, v := range videos {
		if ec, ok := channelMap[v.ChannelID]; ok {
			ec.Videos = append(ec.Videos, EditorVideo{
				VideoID:      v.ID,
				SectionStart: v.SectionStart,
				SectionEnd:   v.SectionEnd,
				Position:     v.Position,
			})
		}
	}

	return result, nil
}

func (s *EditorService) CreateChannel(name string) (*dbmodels.Channel, error) {
	id, err := s.ChannelRepo.InsertChannel(nil, name)
	if err != nil {
		return nil, err
	}
	return &dbmodels.Channel{ID: id, Name: name}, nil
}

func (s *EditorService) RenameChannel(channelID int, name string) (*dbmodels.Channel, error) {
	if err := s.ChannelRepo.RenameChannel(channelID, name); err != nil {
		return nil, err
	}
	return &dbmodels.Channel{ID: channelID, Name: name}, nil
}

func (s *EditorService) DeleteChannel(channelID int) error {
	return s.ChannelRepo.DeleteChannel(channelID)
}

func (s *EditorService) AddVideo(channelID int, videoID string, sectionStart int, sectionEnd int) (*dbmodels.Video, error) {
	return s.VideoRepo.AddVideoToChannel(channelID, videoID, sectionStart, sectionEnd)
}

func (s *EditorService) UpdateVideo(channelID int, videoID string, sectionStart int, sectionEnd int) (*dbmodels.Video, error) {
	return s.VideoRepo.UpdateVideoInChannel(channelID, videoID, sectionStart, sectionEnd)
}

func (s *EditorService) RemoveVideo(channelID int, videoID string) error {
	return s.VideoRepo.RemoveVideoFromChannel(channelID, videoID)
}

func (s *EditorService) ReorderVideos(channelID int, videoIDs []string) error {
	return s.VideoRepo.ReorderVideos(channelID, videoIDs)
}

func (s *EditorService) ExportJSON() (*jsonmodels.ChannelsJson, error) {
	channels, videos, err := s.ChannelRepo.FetchAllChannelsWithVideos()
	if err != nil {
		return nil, err
	}

	videosByChannel := make(map[int][]dbmodels.Video)
	for _, v := range videos {
		videosByChannel[v.ChannelID] = append(videosByChannel[v.ChannelID], v)
	}

	result := jsonmodels.ChannelsJson{Channels: make([]jsonmodels.ChannelJson, 0, len(channels))}
	for _, ch := range channels {
		chJson := jsonmodels.ChannelJson{Name: ch.Name, Videos: []jsonmodels.VideoJson{}}
		for _, v := range videosByChannel[ch.ID] {
			chJson.Videos = append(chJson.Videos, jsonmodels.VideoJson{
				Id:           v.ID,
				SectionStart: v.SectionStart,
				SectionEnd:   v.SectionEnd,
			})
		}
		result.Channels = append(result.Channels, chJson)
	}

	return &result, nil
}

func (s *EditorService) ImportJSON(data jsonmodels.ChannelsJson) error {
	return db.WithTransaction(s.TxManager.GetDB(), func(tx *sql.Tx) error {
		if err := s.VideoRepo.DeleteAllChannelVideos(tx); err != nil {
			return err
		}
		if err := s.ChannelRepo.DeleteAllChannels(tx); err != nil {
			return err
		}
		if err := s.VideoRepo.DeleteAllVideos(tx); err != nil {
			return err
		}

		for _, channel := range data.Channels {
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
		return nil
	})
}

func (s *EditorService) LoadDefaults() error {
	jsonFilePath := config.GetDefaultChannelsPath()
	data, err := helpers.LoadJSONFromFile[jsonmodels.ChannelsJson](jsonFilePath)
	if err != nil {
		return err
	}
	return s.ImportJSON(data)
}

func (s *EditorService) ImportPlaylist(url string) ([]string, error) {
	match := playlistIDRegex.FindStringSubmatch(url)
	if len(match) < 2 {
		return nil, fmt.Errorf("invalid playlist URL")
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://www.youtube.com/playlist?list=" + match[1])
	if err != nil {
		return nil, fmt.Errorf("failed to fetch playlist: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read playlist page: %w", err)
	}
	videoIDRegex := regexp.MustCompile(`"videoId":"([a-zA-Z0-9_-]{11})"`)
	matches := videoIDRegex.FindAllStringSubmatch(string(body), -1)
	seen := make(map[string]bool)
	var videoIDs []string
	for _, m := range matches {
		id := m[1]
		if !seen[id] {
			seen[id] = true
			videoIDs = append(videoIDs, id)
		}
	}
	return videoIDs, nil
}
