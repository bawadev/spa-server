package server

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/andybalholm/brotli"
	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/bawadev/spa-server/internal/config"
	"github.com/bawadev/spa-server/internal/middleware"
)

// Compression type
type Compression int

const (
	None Compression = iota
	Gzip
	Brotli
)

// ResponseItem represents a cached file response
type ResponseItem struct {
	Name        string
	Path        string
	ModTime     time.Time
	Content     []byte
	ContentType string
}

// Server handles HTTP requests for static files
type Server struct {
	config     *config.Config
	httpServer *http.Server
	cache      *lru.TwoQueueCache[string, any]
	startTime  time.Time
}

// New creates a new Server instance
func New(cfg *config.Config) *Server {
	var cache *lru.TwoQueueCache[string, any]

	if cfg.CacheEnabled && cfg.CacheSize > 0 {
		var err error
		cache, err = lru.New2Q[string, any](cfg.CacheSize)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to create cache: %v\n", err)
		}
	}

	return &Server{
		config:    cfg,
		cache:     cache,
		startTime: time.Now(),
	}
}

// Listen starts the HTTP server
func (s *Server) Listen() error {
	mux := http.NewServeMux()

	// Health endpoint
	if s.config.HealthEndpoint {
		mux.HandleFunc("/healthz", s.healthHandler)
		mux.HandleFunc("/health", s.healthHandler)
		mux.HandleFunc("/ready", s.healthHandler)
	}

	// Main handler for all other routes
	mux.HandleFunc("/", s.fileHandler)

	// Build middleware chain
	var handler http.Handler = mux

	if s.config.SecurityHeaders {
		handler = middleware.SecurityHeaders(handler)
	}

	if s.config.CORS {
		handler = middleware.CORS(handler, s.config.CORSOrigin)
	}

	if s.config.Logger {
		handler = middleware.Logger(handler, s.config.LogPretty)
	}

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.config.Address, s.config.Port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	fmt.Printf("spa-server listening on http://%s\n", s.httpServer.Addr)
	fmt.Printf("  Directory: %s\n", s.config.Directory)
	fmt.Printf("  SPA Mode: %v\n", s.config.SpaMode)
	if s.config.Gzip || s.config.Brotli {
		fmt.Printf("  Compression: gzip=%v, brotli=%v\n", s.config.Gzip, s.config.Brotli)
	}
	if s.config.HealthEndpoint {
		fmt.Printf("  Health: /healthz\n")
	}

	err := s.httpServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

// Shutdown gracefully stops the server
func (s *Server) Shutdown() error {
	if s.httpServer == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return s.httpServer.Shutdown(ctx)
}

// healthHandler returns server health status
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	uptime := time.Since(s.startTime).Round(time.Second)
	fmt.Fprintf(w, `{"status":"ok","uptime":"%s"}`, uptime)
}

// fileHandler serves static files with SPA support
func (s *Server) fileHandler(w http.ResponseWriter, r *http.Request) {
	// Validate and get file path
	requestedPath, valid := s.getFilePath(r.URL.Path)
	if !valid {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	// Get response item (from cache or disk)
	responseItem, errorCode := s.getOrCreateResponseItem(requestedPath, None, nil)
	if errorCode != 0 {
		http.Error(w, http.StatusText(errorCode), errorCode)
		return
	}

	// Handle range requests or skip compression for certain files
	if r.Header.Get("Range") != "" || s.shouldSkipCompression(requestedPath) {
		if responseItem.ContentType != "" {
			w.Header().Set("Content-Type", responseItem.ContentType)
		}
		http.ServeContent(w, r, responseItem.Name, responseItem.ModTime, bytes.NewReader(responseItem.Content))
		return
	}

	// Set cache control headers
	s.setCacheHeaders(w, r.URL.Path, responseItem.Name)

	// Handle compression
	s.serveWithCompression(w, r, responseItem)
}

// getFilePath validates and returns the absolute file path
func (s *Server) getFilePath(urlPath string) (string, bool) {
	requestedPath := path.Join(s.config.Directory, urlPath)

	// Resolve symlinks if file exists
	if _, err := os.Stat(requestedPath); err == nil {
		resolved, err := filepath.EvalSymlinks(requestedPath)
		if err == nil {
			requestedPath = resolved
		}
	}

	// Security check: ensure path is within directory
	if !strings.HasPrefix(requestedPath, s.config.Directory) {
		return "", false
	}

	return requestedPath, true
}

// getOrCreateResponseItem retrieves a file from cache or loads it from disk
func (s *Server) getOrCreateResponseItem(requestedPath string, compression Compression, actualContentType *string) (*ResponseItem, int) {
	fallbackPath := path.Join(s.config.Directory, s.config.FallbackFile)

	// Add compression extension
	switch compression {
	case Gzip:
		requestedPath += ".gz"
	case Brotli:
		requestedPath += ".br"
	}

	// Check cache
	if s.cache != nil {
		if cached, ok := s.cache.Get(requestedPath); ok {
			if item, ok := cached.(ResponseItem); ok {
				return &item, 0
			}
			// It's a redirect path
			if redirectPath, ok := cached.(string); ok {
				return s.getOrCreateResponseItem(redirectPath, compression, actualContentType)
			}
		}
	}

	// Open file
	dirPath, fileName := filepath.Split(requestedPath)
	dir := http.Dir(dirPath)

	file, err := dir.Open(fileName)
	if err != nil {
		// SPA mode: serve fallback file
		if s.config.SpaMode && compression == None && requestedPath != fallbackPath {
			if s.cache != nil {
				s.cache.Add(requestedPath, fallbackPath)
			}
			return s.getOrCreateResponseItem(fallbackPath, compression, actualContentType)
		}
		return nil, http.StatusNotFound
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		if s.config.SpaMode && compression == None && requestedPath != fallbackPath {
			if s.cache != nil {
				s.cache.Add(requestedPath, fallbackPath)
			}
			return s.getOrCreateResponseItem(fallbackPath, compression, actualContentType)
		}
		return nil, http.StatusNotFound
	}

	// Handle directories
	if stat.IsDir() {
		if requestedPath != fallbackPath {
			if s.config.SpaMode && compression == None {
				if s.cache != nil {
					s.cache.Add(requestedPath, fallbackPath)
				}
				return s.getOrCreateResponseItem(fallbackPath, compression, actualContentType)
			}
			// Try index.html in directory
			indexPath := path.Join(requestedPath, "index.html")
			if s.cache != nil {
				s.cache.Add(requestedPath, indexPath)
			}
			return s.getOrCreateResponseItem(indexPath, compression, actualContentType)
		}
		return nil, http.StatusNotFound
	}

	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, http.StatusInternalServerError
	}

	// Determine content type
	var contentType string
	if compression == None {
		contentType = mime.TypeByExtension(filepath.Ext(stat.Name()))
	} else if actualContentType != nil {
		contentType = *actualContentType
	}

	item := ResponseItem{
		Path:        requestedPath,
		Name:        stat.Name(),
		ModTime:     stat.ModTime(),
		Content:     content,
		ContentType: contentType,
	}

	// Cache the item
	if s.cache != nil {
		s.cache.Add(requestedPath, item)
	}

	return &item, 0
}

// setCacheHeaders sets appropriate cache control headers
func (s *Server) setCacheHeaders(w http.ResponseWriter, urlPath string, fileName string) {
	// No cache for HTML files or specified paths
	if slices.Contains(s.config.NoCachePaths, urlPath) || path.Ext(fileName) == ".html" {
		w.Header().Set("Cache-Control", "no-store")
	} else if s.config.CacheMaxAge >= 0 {
		w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", s.config.CacheMaxAge))
	}
}

// serveWithCompression serves content with appropriate compression
func (s *Server) serveWithCompression(w http.ResponseWriter, r *http.Request, item *ResponseItem) {
	acceptEncoding := r.Header.Get("Accept-Encoding")
	overThreshold := int64(len(item.Content)) > s.config.Threshold

	// Try Brotli first (better compression)
	if s.config.Brotli && overThreshold && strings.Contains(acceptEncoding, "br") {
		if compressed, _ := s.getOrCreateResponseItem(item.Path, Brotli, &item.ContentType); compressed != nil {
			w.Header().Set("Content-Encoding", "br")
			item = compressed
		}
	} else if s.config.Gzip && overThreshold && strings.Contains(acceptEncoding, "gzip") {
		if compressed, _ := s.getOrCreateResponseItem(item.Path, Gzip, &item.ContentType); compressed != nil {
			w.Header().Set("Content-Encoding", "gzip")
			item = compressed
		}
	}

	if item.ContentType != "" {
		w.Header().Set("Content-Type", item.ContentType)
	}

	http.ServeContent(w, r, item.Name, item.ModTime, bytes.NewReader(item.Content))
}

// shouldSkipCompression checks if a file should skip compression
func (s *Server) shouldSkipCompression(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	for _, skip := range s.config.NoCompress {
		if strings.ToLower(skip) == ext {
			return true
		}
	}
	return false
}

// CompressFiles pre-compresses static files
func (s *Server) CompressFiles() {
	if !s.config.Gzip && !s.config.Brotli {
		return
	}

	err := filepath.Walk(s.config.Directory, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		ext := path.Ext(filePath)
		// Skip already compressed files
		if ext == ".br" || ext == ".gz" {
			return nil
		}

		// Skip files in no-compress list
		if s.shouldSkipCompression(filePath) {
			return nil
		}

		// Only compress files over threshold
		if info.Size() <= s.config.Threshold {
			return nil
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to read %s: %v\n", filePath, err)
			return nil
		}

		if s.config.Gzip {
			s.compressGzip(filePath, data)
		}

		if s.config.Brotli {
			s.compressBrotli(filePath, data)
		}

		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: error during compression: %v\n", err)
	}
}

func (s *Server) compressGzip(filePath string, data []byte) {
	outPath := filePath + ".gz"
	file, err := os.Create(outPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to create %s: %v\n", outPath, err)
		return
	}
	defer file.Close()

	writer := gzip.NewWriter(file)
	if _, err := writer.Write(data); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to write %s: %v\n", outPath, err)
	}
	writer.Close()
}

func (s *Server) compressBrotli(filePath string, data []byte) {
	outPath := filePath + ".br"
	file, err := os.Create(outPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to create %s: %v\n", outPath, err)
		return
	}
	defer file.Close()

	writer := brotli.NewWriter(file)
	if _, err := writer.Write(data); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to write %s: %v\n", outPath, err)
	}
	writer.Close()
}
