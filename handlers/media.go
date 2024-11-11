package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	dbmodels "github.com/ozencb/couchtube/models/db"
	jsonmodels "github.com/ozencb/couchtube/models/json"
	"github.com/ozencb/couchtube/services"
)

type Media struct {
	Service *services.MediaService
}

func NewMediaHandler(service *services.MediaService) *Media {
	return &Media{Service: service}
}

func (h *Media) FetchAllChannels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	channels, err := h.Service.FetchAllChannels()
	if err != nil {
		http.Error(w, "Failed to load channels", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"channels": channels})
}

func (h *Media) GetCurrentVideo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	channelID := r.URL.Query().Get("channel-id")
	if channelID == "" {
		http.Error(w, "channel-id is required", http.StatusBadRequest)
		return
	}
	channelIDInt, err := strconv.Atoi(channelID)
	if err != nil {
		http.Error(w, "Invalid channel-id", http.StatusBadRequest)
		return
	}

	videoID := r.URL.Query().Get("video-id")

	var video *dbmodels.Video
	// if videoId is provided, call FetchNextVideo
	if videoID != "" {
		videoIDInt, err := strconv.Atoi(videoID)
		if err != nil {
			http.Error(w, "Invalid video-id", http.StatusBadRequest)
			return
		}
		video = h.Service.FetchNextVideo(channelIDInt, videoIDInt)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"video": video})
		return
	} else {
		// if videoId is not provided, call GetCurrentVideoByChannelId
		video, err = h.Service.GetCurrentVideoByChannelId(channelIDInt)
		if err != nil {
			println(err.Error())
			http.Error(w, "Failed to load video", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"video": video})
}

func (h *Media) InvalidateVideo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	videoID := r.URL.Query().Get("video-id")
	if videoID == "" {
		http.Error(w, "video-id is required", http.StatusBadRequest)
		return
	}

	err := h.Service.InvalidateVideo(videoID)
	if err != nil {
		http.Error(w, "Failed to invalidate video", http.StatusInternalServerError)
		return
	}

	log.Default().Println("Video invalidated: ", videoID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})

}

func (h *Media) SubmitList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var list jsonmodels.SubmitListRequestJson
	err := json.NewDecoder(r.Body).Decode(&list)
	if err != nil {
		http.Error(w, "Failed to parse list", http.StatusBadRequest)
		return
	}

	if list.VideoListUrl == "" {
		http.Error(w, "videoListUrl is required", http.StatusBadRequest)
		return
	}

	success, err := h.Service.SubmitList(list)
	if err != nil {
		http.Error(w, "Failed to submit list", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"success": success})
}
