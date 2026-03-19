package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/ozencb/couchtube/config"
)

func GetConfigs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"editorMode": config.GetEditorMode()})
}
