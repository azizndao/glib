package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/azizndao/glib/storage"
	"github.com/azizndao/glib/storage/local"
	"github.com/azizndao/glib/storage/s3"
)

var manager *storage.Manager

func main() {
	// Initialize storage manager
	manager = storage.NewManager()

	// Get configuration from environment
	storageRoot := getEnv("STORAGE_ROOT", "./storage")
	baseURL := getEnv("BASE_URL", "http://localhost:8080/files")
	urlSecret := getEnv("URL_SECRET", "change-me-in-production")

	// Register local disk (always available)
	manager.RegisterDisk("local", func() storage.Storage {
		disk, err := local.New(local.Options{
			Root:       storageRoot,
			BaseURL:    baseURL,
			URLSecret:  urlSecret,
			CreateRoot: true,
		})
		if err != nil {
			log.Fatalf("Failed to create local disk: %v", err)
		}
		log.Printf("✓ Local disk registered (root: %s)", storageRoot)
		return disk
	})

	// Register S3 disk if configured
	if s3Endpoint := os.Getenv("S3_ENDPOINT"); s3Endpoint != "" {
		manager.RegisterDisk("s3", func() storage.Storage {
			disk, err := s3.New(s3.Options{
				Endpoint:  s3Endpoint,
				AccessKey: os.Getenv("S3_ACCESS_KEY"),
				SecretKey: os.Getenv("S3_SECRET_KEY"),
				Bucket:    getEnv("S3_BUCKET", "glib-storage"),
				Region:    getEnv("S3_REGION", "us-east-1"),
				UseSSL:    getEnvBool("S3_USE_SSL", true),
				Prefix:    os.Getenv("S3_PREFIX"),
			})
			if err != nil {
				log.Fatalf("Failed to create S3 disk: %v", err)
			}
			log.Printf("✓ S3 disk registered (endpoint: %s, bucket: %s)", s3Endpoint, getEnv("S3_BUCKET", "glib-storage"))
			return disk
		})
	} else {
		log.Println("ℹ S3 disk not configured (set S3_ENDPOINT to enable)")
	}

	// Set default disk
	manager.SetDefaultDisk("local")

	// Setup HTTP routes
	mux := http.NewServeMux()

	// File operations (ordered from most specific to least specific)
	mux.HandleFunc("POST /upload", corsMiddleware(uploadHandler))
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /files", corsMiddleware(listFilesHandler))

	// API endpoints for operations
	mux.HandleFunc("POST /api/copy", corsMiddleware(copyHandler))
	mux.HandleFunc("POST /api/move", corsMiddleware(moveHandler))
	mux.HandleFunc("GET /api/temp-url", corsMiddleware(tempURLHandler))
	mux.HandleFunc("POST /api/visibility", corsMiddleware(visibilityHandler))
	mux.HandleFunc("GET /api/info", corsMiddleware(infoHandler))
	mux.HandleFunc("DELETE /api/delete", corsMiddleware(deleteHandler))

	// File download (must be last - catches all GET /files/*)
	mux.HandleFunc("GET /files/{path...}", corsMiddleware(downloadHandler))

	// Documentation
	mux.HandleFunc("GET /", docsHandler)

	port := getEnv("PORT", "8080")
	log.Printf("🚀 Storage API server starting on port %s", port)
	log.Printf("📚 API documentation: http://localhost:%s/", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

// uploadHandler handles file uploads via multipart form
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (32MB max)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		jsonError(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get file from form
	formFile, header, err := r.FormFile("file")
	if err != nil {
		jsonError(w, "No file provided", http.StatusBadRequest)
		return
	}
	defer formFile.Close()

	// Get optional parameters
	diskName := r.FormValue("disk")
	if diskName == "" {
		diskName = "local"
	}

	path := r.FormValue("path")
	if path == "" {
		path = "uploads/" + header.Filename
	}

	visibility := r.FormValue("visibility")
	if visibility == "" {
		visibility = string(storage.VisibilityPublic)
	}

	// Get disk
	disk, err := manager.Disk(diskName)
	if err != nil {
		jsonError(w, fmt.Sprintf("Invalid disk: %v", err), http.StatusBadRequest)
		return
	}

	// Create file from multipart
	file, err := storage.NewFileFromMultipart(header)
	if err != nil {
		jsonError(w, fmt.Sprintf("Invalid file: %v", err), http.StatusBadRequest)
		return
	}

	// Upload file
	ctx := r.Context()
	if err := disk.PutFile(ctx, path, file); err != nil {
		jsonError(w, fmt.Sprintf("Upload failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Set visibility
	if err := disk.SetVisibility(ctx, path, storage.Visibility(visibility)); err != nil {
		log.Printf("Warning: Failed to set visibility: %v", err)
	}

	// Get file info
	url, _ := disk.URL(ctx, path)
	size, _ := disk.Size(ctx, path)
	modified, _ := disk.LastModified(ctx, path)

	jsonResponse(w, map[string]interface{}{
		"success":  true,
		"path":     path,
		"url":      url,
		"size":     size,
		"disk":     diskName,
		"modified": modified,
	}, http.StatusCreated)
}

// downloadHandler streams files to the client
func downloadHandler(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("path")
	diskName := r.URL.Query().Get("disk")
	if diskName == "" {
		diskName = "local"
	}

	disk, err := manager.Disk(diskName)
	if err != nil {
		jsonError(w, "Invalid disk", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Check if file exists
	exists, err := disk.Exists(ctx, path)
	if err != nil || !exists {
		jsonError(w, "File not found", http.StatusNotFound)
		return
	}

	// Get file stream
	reader, err := disk.GetStream(ctx, path)
	if err != nil {
		jsonError(w, "Failed to read file", http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	// Set headers
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filepath.Base(path)))

	// Stream file
	if _, err := io.Copy(w, reader); err != nil {
		log.Printf("Error streaming file: %v", err)
	}
}

// listFilesHandler lists all files in a directory
func listFilesHandler(w http.ResponseWriter, r *http.Request) {
	diskName := r.URL.Query().Get("disk")
	if diskName == "" {
		diskName = "local"
	}

	dir := r.URL.Query().Get("dir")
	recursive := r.URL.Query().Get("recursive") == "true"

	disk, err := manager.Disk(diskName)
	if err != nil {
		jsonError(w, "Invalid disk", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	var files []string

	if recursive {
		files, err = disk.AllFiles(ctx, dir)
	} else {
		files, err = disk.Files(ctx, dir)
	}

	if err != nil {
		jsonError(w, fmt.Sprintf("Failed to list files: %v", err), http.StatusInternalServerError)
		return
	}

	// Get file info for each file
	fileInfos := make([]map[string]interface{}, 0, len(files))
	for _, path := range files {
		info := make(map[string]interface{})
		info["path"] = path
		info["url"], _ = disk.URL(ctx, path)
		info["size"], _ = disk.Size(ctx, path)
		info["modified"], _ = disk.LastModified(ctx, path)
		info["visibility"], _ = disk.GetVisibility(ctx, path)
		fileInfos = append(fileInfos, info)
	}

	jsonResponse(w, map[string]interface{}{
		"success": true,
		"disk":    diskName,
		"dir":     dir,
		"count":   len(files),
		"files":   fileInfos,
	}, http.StatusOK)
}

// deleteHandler deletes a file or directory
func deleteHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		jsonError(w, "Path parameter required", http.StatusBadRequest)
		return
	}

	diskName := r.URL.Query().Get("disk")
	if diskName == "" {
		diskName = "local"
	}

	disk, err := manager.Disk(diskName)
	if err != nil {
		jsonError(w, "Invalid disk", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Check if it's a directory
	isDir := r.URL.Query().Get("directory") == "true"

	if isDir {
		err = disk.DeleteDirectory(ctx, path)
	} else {
		err = disk.Delete(ctx, path)
	}

	if err != nil {
		jsonError(w, fmt.Sprintf("Delete failed: %v", err), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]interface{}{
		"success": true,
		"path":    path,
		"message": "Deleted successfully",
	}, http.StatusOK)
}

// copyHandler copies a file
func copyHandler(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Source      string `json:"source"`
		Destination string `json:"destination"`
		Disk        string `json:"disk"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if body.Source == "" {
		jsonError(w, "Source path required", http.StatusBadRequest)
		return
	}

	if body.Destination == "" {
		jsonError(w, "Destination path required", http.StatusBadRequest)
		return
	}

	diskName := body.Disk
	if diskName == "" {
		diskName = "local"
	}

	disk, err := manager.Disk(diskName)
	if err != nil {
		jsonError(w, "Invalid disk", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	if err := disk.Copy(ctx, body.Source, body.Destination); err != nil {
		jsonError(w, fmt.Sprintf("Copy failed: %v", err), http.StatusInternalServerError)
		return
	}

	url, _ := disk.URL(ctx, body.Destination)

	jsonResponse(w, map[string]interface{}{
		"success":     true,
		"source":      body.Source,
		"destination": body.Destination,
		"url":         url,
	}, http.StatusOK)
}

// moveHandler moves a file
func moveHandler(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Source      string `json:"source"`
		Destination string `json:"destination"`
		Disk        string `json:"disk"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if body.Source == "" {
		jsonError(w, "Source path required", http.StatusBadRequest)
		return
	}

	if body.Destination == "" {
		jsonError(w, "Destination path required", http.StatusBadRequest)
		return
	}

	diskName := body.Disk
	if diskName == "" {
		diskName = "local"
	}

	disk, err := manager.Disk(diskName)
	if err != nil {
		jsonError(w, "Invalid disk", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	if err := disk.Move(ctx, body.Source, body.Destination); err != nil {
		jsonError(w, fmt.Sprintf("Move failed: %v", err), http.StatusInternalServerError)
		return
	}

	url, _ := disk.URL(ctx, body.Destination)

	jsonResponse(w, map[string]interface{}{
		"success":     true,
		"source":      body.Source,
		"destination": body.Destination,
		"url":         url,
	}, http.StatusOK)
}

// tempURLHandler generates a temporary signed URL
func tempURLHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		jsonError(w, "Path parameter required", http.StatusBadRequest)
		return
	}

	diskName := r.URL.Query().Get("disk")
	if diskName == "" {
		diskName = "local"
	}

	expiresIn := r.URL.Query().Get("expires")
	if expiresIn == "" {
		expiresIn = "3600" // 1 hour default
	}

	seconds, err := strconv.ParseInt(expiresIn, 10, 64)
	if err != nil {
		jsonError(w, "Invalid expires parameter", http.StatusBadRequest)
		return
	}

	disk, err := manager.Disk(diskName)
	if err != nil {
		jsonError(w, "Invalid disk", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	expiration := time.Now().Add(time.Duration(seconds) * time.Second)

	url, err := disk.TemporaryURL(ctx, path, expiration)
	if err != nil {
		jsonError(w, fmt.Sprintf("Failed to generate URL: %v", err), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]interface{}{
		"success":    true,
		"path":       path,
		"url":        url,
		"expires_at": expiration,
		"expires_in": seconds,
	}, http.StatusOK)
}

// visibilityHandler sets file visibility
func visibilityHandler(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path       string `json:"path"`
		Visibility string `json:"visibility"`
		Disk       string `json:"disk"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if body.Path == "" {
		jsonError(w, "Path required", http.StatusBadRequest)
		return
	}

	if body.Visibility != string(storage.VisibilityPublic) && body.Visibility != string(storage.VisibilityPrivate) {
		jsonError(w, "Invalid visibility (must be 'public' or 'private')", http.StatusBadRequest)
		return
	}

	diskName := body.Disk
	if diskName == "" {
		diskName = "local"
	}

	disk, err := manager.Disk(diskName)
	if err != nil {
		jsonError(w, "Invalid disk", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	if err := disk.SetVisibility(ctx, body.Path, storage.Visibility(body.Visibility)); err != nil {
		jsonError(w, fmt.Sprintf("Failed to set visibility: %v", err), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]interface{}{
		"success":    true,
		"path":       body.Path,
		"visibility": body.Visibility,
	}, http.StatusOK)
}

// infoHandler returns file metadata
func infoHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		jsonError(w, "Path parameter required", http.StatusBadRequest)
		return
	}

	diskName := r.URL.Query().Get("disk")
	if diskName == "" {
		diskName = "local"
	}

	disk, err := manager.Disk(diskName)
	if err != nil {
		jsonError(w, "Invalid disk", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	exists, _ := disk.Exists(ctx, path)
	if !exists {
		jsonError(w, "File not found", http.StatusNotFound)
		return
	}

	size, _ := disk.Size(ctx, path)
	modified, _ := disk.LastModified(ctx, path)
	visibility, _ := disk.GetVisibility(ctx, path)
	url, _ := disk.URL(ctx, path)

	jsonResponse(w, map[string]interface{}{
		"success":    true,
		"path":       path,
		"disk":       diskName,
		"exists":     exists,
		"size":       size,
		"modified":   modified,
		"visibility": visibility,
		"url":        url,
	}, http.StatusOK)
}

// healthHandler returns server health status
func healthHandler(w http.ResponseWriter, r *http.Request) {
	disks := make([]string, 0)
	if _, err := manager.Disk("local"); err == nil {
		disks = append(disks, "local")
	}
	if _, err := manager.Disk("s3"); err == nil {
		disks = append(disks, "s3")
	}

	jsonResponse(w, map[string]interface{}{
		"status": "healthy",
		"disks":  disks,
		"time":   time.Now(),
	}, http.StatusOK)
}

// docsHandler returns API documentation
func docsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head>
    <title>Glib Storage API</title>
    <style>
        body { font-family: -apple-system, system-ui, sans-serif; max-width: 900px; margin: 40px auto; padding: 0 20px; line-height: 1.6; }
        h1 { color: #333; }
        h2 { color: #666; margin-top: 30px; }
        code { background: #f4f4f4; padding: 2px 6px; border-radius: 3px; }
        pre { background: #f4f4f4; padding: 15px; border-radius: 5px; overflow-x: auto; }
        .endpoint { margin: 20px 0; padding: 15px; border-left: 3px solid #4CAF50; background: #f9f9f9; }
        .method { display: inline-block; padding: 3px 8px; border-radius: 3px; font-weight: bold; color: white; }
        .post { background: #4CAF50; }
        .get { background: #2196F3; }
        .delete { background: #f44336; }
    </style>
</head>
<body>
    <h1>🗄️ Glib Storage API</h1>
    <p>A Laravel-inspired file storage API for Go</p>

    <h2>📌 Endpoints</h2>

    <div class="endpoint">
        <span class="method post">POST</span> <code>/upload</code>
        <p>Upload a file via multipart form</p>
        <pre>curl -F "file=@test.txt" -F "path=uploads/test.txt" -F "disk=local" http://localhost:8080/upload</pre>
    </div>

    <div class="endpoint">
        <span class="method get">GET</span> <code>/files/{path}</code>
        <p>Download a file</p>
        <pre>curl http://localhost:8080/files/uploads/test.txt</pre>
    </div>

    <div class="endpoint">
        <span class="method get">GET</span> <code>/files</code>
        <p>List files in a directory</p>
        <pre>curl "http://localhost:8080/files?disk=local&dir=uploads&recursive=true"</pre>
    </div>

    <div class="endpoint">
        <span class="method delete">DELETE</span> <code>/api/delete</code>
        <p>Delete a file</p>
        <pre>curl -X DELETE "http://localhost:8080/api/delete?path=uploads/test.txt&disk=local"</pre>
    </div>

    <div class="endpoint">
        <span class="method post">POST</span> <code>/api/copy</code>
        <p>Copy a file</p>
        <pre>curl -X POST -H "Content-Type: application/json" -d '{"source":"uploads/test.txt","destination":"backup/test.txt"}' http://localhost:8080/api/copy</pre>
    </div>

    <div class="endpoint">
        <span class="method post">POST</span> <code>/api/move</code>
        <p>Move a file</p>
        <pre>curl -X POST -H "Content-Type: application/json" -d '{"source":"uploads/test.txt","destination":"archive/test.txt"}' http://localhost:8080/api/move</pre>
    </div>

    <div class="endpoint">
        <span class="method get">GET</span> <code>/api/temp-url</code>
        <p>Generate a temporary signed URL</p>
        <pre>curl "http://localhost:8080/api/temp-url?path=uploads/test.txt&expires=3600&disk=local"</pre>
    </div>

    <div class="endpoint">
        <span class="method post">POST</span> <code>/api/visibility</code>
        <p>Set file visibility</p>
        <pre>curl -X POST -H "Content-Type: application/json" -d '{"path":"uploads/test.txt","visibility":"private"}' http://localhost:8080/api/visibility</pre>
    </div>

    <div class="endpoint">
        <span class="method get">GET</span> <code>/api/info</code>
        <p>Get file metadata</p>
        <pre>curl "http://localhost:8080/api/info?path=uploads/test.txt&disk=local"</pre>
    </div>

    <div class="endpoint">
        <span class="method get">GET</span> <code>/health</code>
        <p>Check server health</p>
        <pre>curl http://localhost:8080/health</pre>
    </div>

    <h2>🔧 Configuration</h2>
    <p>Configure via environment variables:</p>
    <pre>
# Local Storage
STORAGE_ROOT=./storage
BASE_URL=http://localhost:8080/files
URL_SECRET=your-secret-key

# S3 Storage (optional)
S3_ENDPOINT=s3.amazonaws.com
S3_ACCESS_KEY=your-access-key
S3_SECRET_KEY=your-secret-key
S3_BUCKET=my-bucket
S3_REGION=us-east-1
S3_USE_SSL=true
S3_PREFIX=uploads/

# Server
PORT=8080
    </pre>

    <h2>📚 Documentation</h2>
    <p>See <a href="https://github.com/azizndao/glib/tree/main/storage">storage module README</a> for full documentation</p>
</body>
</html>`)
}

// Helper functions

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func jsonResponse(w http.ResponseWriter, data any, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, message string, status int) {
	jsonResponse(w, map[string]any{
		"success": false,
		"error":   message,
	}, status)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return strings.ToLower(value) == "true" || value == "1"
	}
	return defaultValue
}
