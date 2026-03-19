package handlers

import (
	"net/http"
	"strconv"

	"github.com/ozencb/couchtube/config"
)

func GetConfigs(w http.ResponseWriter, r *http.Request) {
	editorEnabled := config.GetEditorMode()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"editorMode": ` + strconv.FormatBool(editorEnabled) + `}`))
}
