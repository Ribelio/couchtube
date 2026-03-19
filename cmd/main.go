package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/ozencb/couchtube/config"
	"github.com/ozencb/couchtube/db"
	"github.com/ozencb/couchtube/handlers"

	"github.com/ozencb/couchtube/middleware"
	repo "github.com/ozencb/couchtube/repositories"
	"github.com/ozencb/couchtube/services"
)

type Route struct {
	Path       string
	Handler    http.HandlerFunc
	EditorOnly bool
}

func registerRoutes(mux *http.ServeMux, routes []Route, editorGuard func(http.HandlerFunc) http.HandlerFunc) {
	for _, route := range routes {
		handler := route.Handler

		if route.EditorOnly {
			handler = editorGuard(handler)
		}

		mux.Handle(route.Path, handler)
	}
}

func main() {
	// Initialize the database
	dbInstance, err := db.GetDbConnection()
	if err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	defer db.CloseConnector()

	db.InitDatabase(dbInstance)

	// Initialize Repositories
	txManager := repo.NewTxManager(dbInstance)
	channelRepo := repo.NewChannelRepository(dbInstance)
	videoRepo := repo.NewVideoRepository(dbInstance)

	// Initialize Services
	mediaService := services.NewMediaService(txManager, channelRepo, videoRepo)
	editorService := services.NewEditorService(txManager, channelRepo, videoRepo)

	// Initialize Handlers with services
	mediaHandler := handlers.NewMediaHandler(mediaService)
	editorHandler := handlers.NewEditorHandler(editorService)

	var editorGuard func(http.HandlerFunc) http.HandlerFunc
	if config.GetEditorMode() {
		editorGuard = middleware.EditorGuardEnabled
	} else {
		editorGuard = middleware.EditorGuard
	}

	staticFs := http.FileServer(http.Dir("./static"))
	rootHandler := staticFs.ServeHTTP
	if !config.GetEditorMode() {
		rootHandler = func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/editor") {
				http.NotFound(w, r)
				return
			}
			staticFs.ServeHTTP(w, r)
		}
	}

	routes := []Route{
		{Path: "/", Handler: rootHandler},
		{Path: "/api/channels", Handler: mediaHandler.FetchAllChannels},
		{Path: "/api/current-video", Handler: mediaHandler.GetCurrentVideo},
		{Path: "/api/submit-list", Handler: mediaHandler.SubmitList},
		{Path: "/api/invalidate-video", Handler: mediaHandler.InvalidateVideo},
		{Path: "/api/config", Handler: handlers.GetConfigs},
		{Path: "/api/load-defaults", Handler: editorHandler.LoadDefaults},
		{Path: "/api/editor/channels", Handler: editorHandler.HandleChannels, EditorOnly: true},
		{Path: "/api/editor/channels/videos", Handler: editorHandler.HandleVideos, EditorOnly: true},
		{Path: "/api/editor/channels/videos/reorder", Handler: editorHandler.ReorderVideos, EditorOnly: true},
		{Path: "/api/editor/export", Handler: editorHandler.ExportJSON, EditorOnly: true},
		{Path: "/api/editor/import", Handler: editorHandler.ImportJSON, EditorOnly: true},
		{Path: "/api/editor/import-playlist", Handler: editorHandler.ImportPlaylist, EditorOnly: true},
	}
	registerRoutes(http.DefaultServeMux, routes, editorGuard)

	if config.GetEditorMode() {
		editorFs := handlers.ServeEditor()
		http.DefaultServeMux.Handle("/editor/", editorFs)
	}

	port := config.GetPort()

	log.Println("Server starting on port", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
