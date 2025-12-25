package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/azizndao/glib/cache"
	"github.com/azizndao/glib/cache/memory"
)

// User represents a user in the system
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// UserRepository simulates a database
type UserRepository struct {
	users map[int]User
}

func NewUserRepository() *UserRepository {
	return &UserRepository{
		users: map[int]User{
			1: {ID: 1, Name: "Alice", Email: "alice@example.com"},
			2: {ID: 2, Name: "Bob", Email: "bob@example.com"},
			3: {ID: 3, Name: "Charlie", Email: "charlie@example.com"},
		},
	}
}

func (r *UserRepository) Find(id int) (*User, error) {
	// Simulate slow database query
	time.Sleep(100 * time.Millisecond)

	if user, exists := r.users[id]; exists {
		return &user, nil
	}

	return nil, fmt.Errorf("user not found")
}

// UserService handles user-related operations with caching
type UserService struct {
	repo  *UserRepository
	cache cache.Store
}

func NewUserService(repo *UserRepository, cacheManager *cache.Manager) *UserService {
	store, _ := cacheManager.Default()
	return &UserService{
		repo:  repo,
		cache: store,
	}
}

// GetUser retrieves a user with caching
func (s *UserService) GetUser(ctx context.Context, id int) (*User, error) {
	cacheKey := fmt.Sprintf("user:%d", id)

	var user User
	err := s.cache.Remember(ctx, cacheKey, &user, 5*time.Minute, func() (any, error) {
		log.Printf("Cache miss - fetching user %d from database", id)
		return s.repo.Find(id)
	})

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// InvalidateUser removes user from cache
func (s *UserService) InvalidateUser(ctx context.Context, id int) error {
	cacheKey := fmt.Sprintf("user:%d", id)
	return s.cache.Forget(ctx, cacheKey)
}

// UserHandler handles HTTP requests for users
type UserHandler struct {
	service *UserService
	cache   *cache.Manager
}

func NewUserHandler(service *UserService, cacheManager *cache.Manager) *UserHandler {
	return &UserHandler{
		service: service,
		cache:   cacheManager,
	}
}

// GetUser handles GET /users/:id
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	// Parse user ID from query
	userID := 1 // Simplified for example
	if id := r.URL.Query().Get("id"); id != "" {
		fmt.Sscanf(id, "%d", &userID)
	}

	// Get user
	user, err := h.service.GetUser(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Return JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// InvalidateCache handles POST /users/:id/invalidate
func (h *UserHandler) InvalidateCache(w http.ResponseWriter, r *http.Request) {
	userID := 1
	if id := r.URL.Query().Get("id"); id != "" {
		fmt.Sscanf(id, "%d", &userID)
	}

	err := h.service.InvalidateUser(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "cache invalidated"})
}

// GetStats handles GET /cache/stats
func (h *UserHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	store, _ := h.cache.Default()

	// Type assert to get memory-specific methods
	if memStore, ok := store.(*memory.Memory); ok {
		stats := memStore.GetStats()
		size := memStore.Size()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"hits":      stats.Hits,
			"misses":    stats.Misses,
			"writes":    stats.Writes,
			"deletes":   stats.Deletes,
			"hit_ratio": stats.HitRatio(),
			"size":      size,
		})
	} else {
		http.Error(w, "statistics not available", http.StatusNotImplemented)
	}
}

func main() {
	// Initialize dependencies
	repo := NewUserRepository()

	// Setup cache
	cacheManager := cache.NewManager()
	cacheManager.RegisterDriver("memory", func() cache.Store {
		return memory.New()
	})

	// Setup service and handler
	service := NewUserService(repo, cacheManager)
	handler := NewUserHandler(service, cacheManager)

	// Setup routes
	http.HandleFunc("/users", handler.GetUser)
	http.HandleFunc("/users/invalidate", handler.InvalidateCache)
	http.HandleFunc("/cache/stats", handler.GetStats)

	// Start server
	log.Println("Starting server on :8080")
	log.Println("Try:")
	log.Println("  curl http://localhost:8080/users?id=1")
	log.Println("  curl http://localhost:8080/cache/stats")
	log.Println("  curl -X POST http://localhost:8080/users/invalidate?id=1")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
