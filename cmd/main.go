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
	Path        string
	Handler     http.HandlerFunc
	EditorRead  bool
	EditorWrite bool
}

func registerRoutes(mux *http.ServeMux, routes []Route, readGuard, writeGuard func(http.HandlerFunc) http.HandlerFunc) {
	for _, route := range routes {
		handler := route.Handler

		if route.EditorRead && route.EditorWrite {
			readHandler := readGuard(handler)
			writeHandler := writeGuard(handler)
			handler = func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					readHandler(w, r)
				} else {
					writeHandler(w, r)
				}
			}
		} else if route.EditorRead {
			handler = readGuard(handler)
		} else if route.EditorWrite {
			handler = writeGuard(handler)
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

	// Auto-populate defaults when not in full editor mode
	if config.GetEditorMode() != "full" {
		channels, err := mediaService.FetchAllChannels()
		if err == nil && len(channels) == 0 {
			if err := editorService.LoadDefaults(); err != nil {
				log.Printf("Warning: failed to load default channels: %v", err)
			}
		}
	}

	// Initialize Handlers with services
	mediaHandler := handlers.NewMediaHandler(mediaService)
	editorHandler := handlers.NewEditorHandler(editorService)

	var readGuard, writeGuard func(http.HandlerFunc) http.HandlerFunc
	switch config.GetEditorMode() {
	case "full":
		readGuard = middleware.EditorGuardEnabled
		writeGuard = middleware.EditorGuardEnabled
	case "readonly":
		readGuard = middleware.EditorGuardEnabled
		writeGuard = middleware.EditorGuardReadOnly
	default:
		readGuard = middleware.EditorGuardOff
		writeGuard = middleware.EditorGuardOff
	}

	noCacheStatic := func(next http.Handler) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			next.ServeHTTP(w, r)
		}
	}

	staticFs := http.FileServer(http.Dir("./static"))
	rootHandler := noCacheStatic(staticFs)
	if config.GetEditorMode() == "off" {
		rootHandler = func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/editor") {
				http.NotFound(w, r)
				return
			}
			noCacheStatic(staticFs)(w, r)
		}
	}

	routes := []Route{
		{Path: "/", Handler: rootHandler},
		{Path: "/api/channels", Handler: mediaHandler.FetchAllChannels},
		{Path: "/api/current-video", Handler: mediaHandler.GetCurrentVideo},
		{Path: "/api/current-videos", Handler: mediaHandler.FetchAllCurrentVideos},
		{Path: "/api/submit-list", Handler: mediaHandler.SubmitList},
		{Path: "/api/invalidate-video", Handler: mediaHandler.InvalidateVideo},
		{Path: "/api/config", Handler: handlers.GetConfigs},
		{Path: "/api/load-defaults", Handler: editorHandler.LoadDefaults, EditorWrite: true},
		{Path: "/api/editor/channels", Handler: editorHandler.HandleChannels, EditorRead: true, EditorWrite: true},
		{Path: "/api/editor/channels/reorder", Handler: editorHandler.ReorderChannels, EditorWrite: true},
		{Path: "/api/editor/channels/videos", Handler: editorHandler.HandleVideos, EditorWrite: true},
		{Path: "/api/editor/channels/videos/reorder", Handler: editorHandler.ReorderVideos, EditorWrite: true},
		{Path: "/api/editor/export", Handler: editorHandler.ExportJSON, EditorRead: true},
		{Path: "/api/editor/import", Handler: editorHandler.ImportJSON, EditorWrite: true},
		{Path: "/api/editor/import-playlist", Handler: editorHandler.ImportPlaylist, EditorRead: true},
	}
	registerRoutes(http.DefaultServeMux, routes, readGuard, writeGuard)

	if config.GetEditorMode() != "off" {
		editorFs := handlers.ServeEditor()
		http.DefaultServeMux.Handle("/editor/", editorFs)
	}

	port := config.GetPort()

	log.Println("Server starting on port", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
