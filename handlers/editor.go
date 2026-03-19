package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	jsonmodels "github.com/ozencb/couchtube/models/json"
	"github.com/ozencb/couchtube/services"
)

type Editor struct {
	Service *services.EditorService
}

func NewEditorHandler(service *services.EditorService) *Editor {
	return &Editor{Service: service}
}

func ServeEditor() http.Handler {
	fs := http.FileServer(http.Dir("./static/editor"))
	return http.StripPrefix("/editor/", fs)
}

func (h *Editor) HandleChannels(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		channels, err := h.Service.FetchAllChannels()
		if err != nil {
			http.Error(w, "Failed to load channels", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"channels": channels})

	case http.MethodPost:
		var body struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}
		channel, err := h.Service.CreateChannel(body.Name)
		if err != nil {
			http.Error(w, "Failed to create channel", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{"channel": channel})

	case http.MethodPut:
		channelIDStr := r.URL.Query().Get("channel-id")
		if channelIDStr == "" {
			http.Error(w, "channel-id is required", http.StatusBadRequest)
			return
		}
		channelID, err := strconv.Atoi(channelIDStr)
		if err != nil {
			http.Error(w, "invalid channel-id", http.StatusBadRequest)
			return
		}
		var body struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}
		channel, err := h.Service.RenameChannel(channelID, body.Name)
		if err != nil {
			http.Error(w, "Failed to rename channel", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"channel": channel})

	case http.MethodDelete:
		channelIDStr := r.URL.Query().Get("channel-id")
		if channelIDStr == "" {
			http.Error(w, "channel-id is required", http.StatusBadRequest)
			return
		}
		channelID, err := strconv.Atoi(channelIDStr)
		if err != nil {
			http.Error(w, "invalid channel-id", http.StatusBadRequest)
			return
		}
		if err := h.Service.DeleteChannel(channelID); err != nil {
			http.Error(w, "Failed to delete channel", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Editor) HandleVideos(w http.ResponseWriter, r *http.Request) {
	channelIDStr := r.URL.Query().Get("channel-id")
	if channelIDStr == "" {
		http.Error(w, "channel-id is required", http.StatusBadRequest)
		return
	}
	channelID, err := strconv.Atoi(channelIDStr)
	if err != nil {
		http.Error(w, "invalid channel-id", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPost:
		var body struct {
			VideoID      string `json:"videoId"`
			SectionStart int    `json:"sectionStart"`
			SectionEnd   int    `json:"sectionEnd"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.VideoID == "" || body.SectionEnd < body.SectionStart+1 {
			http.Error(w, "videoId required, sectionEnd must be at least 1 second after sectionStart", http.StatusBadRequest)
			return
		}
		video, err := h.Service.AddVideo(channelID, body.VideoID, body.SectionStart, body.SectionEnd)
		if err != nil {
			http.Error(w, "Failed to add video", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{"video": video})

	case http.MethodPut:
		videoID := r.URL.Query().Get("video-id")
		if videoID == "" {
			http.Error(w, "video-id is required", http.StatusBadRequest)
			return
		}
		var body struct {
			SectionStart int `json:"sectionStart"`
			SectionEnd   int `json:"sectionEnd"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		if body.SectionEnd < body.SectionStart+1 {
			http.Error(w, "sectionEnd must be at least 1 second after sectionStart", http.StatusBadRequest)
			return
		}
		video, err := h.Service.UpdateVideo(channelID, videoID, body.SectionStart, body.SectionEnd)
		if err != nil {
			http.Error(w, "Failed to update video", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"video": video})

	case http.MethodDelete:
		videoID := r.URL.Query().Get("video-id")
		if videoID == "" {
			http.Error(w, "video-id is required", http.StatusBadRequest)
			return
		}
		if err := h.Service.RemoveVideo(channelID, videoID); err != nil {
			http.Error(w, "Failed to remove video", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Editor) ReorderVideos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	channelIDStr := r.URL.Query().Get("channel-id")
	if channelIDStr == "" {
		http.Error(w, "channel-id is required", http.StatusBadRequest)
		return
	}
	channelID, err := strconv.Atoi(channelIDStr)
	if err != nil {
		http.Error(w, "invalid channel-id", http.StatusBadRequest)
		return
	}

	var body struct {
		VideoIDs []string `json:"videoIds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.VideoIDs) == 0 {
		http.Error(w, "videoIds is required", http.StatusBadRequest)
		return
	}

	if err := h.Service.ReorderVideos(channelID, body.VideoIDs); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

func (h *Editor) ExportJSON(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	data, err := h.Service.ExportJSON()
	if err != nil {
		http.Error(w, "Failed to export", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="couchtube-export.json"`)
	json.NewEncoder(w).Encode(data)
}

func (h *Editor) ImportJSON(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var data jsonmodels.ChannelsJson
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if len(data.Channels) == 0 {
		http.Error(w, "channels is required", http.StatusBadRequest)
		return
	}

	if err := h.Service.ImportJSON(data); err != nil {
		http.Error(w, "Failed to import", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

func (h *Editor) LoadDefaults(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := h.Service.LoadDefaults(); err != nil {
		http.Error(w, "Failed to load defaults", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

func (h *Editor) ImportPlaylist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct{ URL string `json:"url"` }
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.URL == "" {
		http.Error(w, "url is required", http.StatusBadRequest)
		return
	}
	videoIDs, err := h.Service.ImportPlaylist(body.URL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	type entry struct{ ID string `json:"id"` }
	videos := make([]entry, len(videoIDs))
	for i, id := range videoIDs {
		videos[i] = entry{ID: id}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"videos": videos})
}
