package orm

import (
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Test model with active flag
type TestArticle struct {
	Model
	Title     string
	Content   string
	Active    bool
	Published bool
	UserID    uuid.UUID
}

func TestScopes_Single(t *testing.T) {
	db := setupTestDB(t)
	db.AutoMigrate(&TestArticle{})

	user1ID := uuid.New()
	user2ID := uuid.New()

	// Seed articles
	articles := []TestArticle{
		{Title: "Article 1", Active: true, Published: true, UserID: user1ID},
		{Title: "Article 2", Active: false, Published: true, UserID: user1ID},
		{Title: "Article 3", Active: true, Published: false, UserID: user2ID},
		{Title: "Article 4", Active: false, Published: false, UserID: user2ID},
	}

	for _, article := range articles {
		if err := db.Create(&article).Error; err != nil {
			t.Fatalf("Failed to seed article: %v", err)
		}
	}

	// Test applying a single scope
	var results []TestArticle
	err := WhereColumn("active", true)(db).Find(&results).Error

	if err != nil {
		t.Fatalf("Scopes query failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 active articles, got %d", len(results))
	}
}

func TestScopes_Multiple(t *testing.T) {
	db := setupTestDB(t)
	db.AutoMigrate(&TestArticle{})

	user1ID := uuid.New()
	user2ID := uuid.New()

	// Seed articles
	articles := []TestArticle{
		{Title: "Article 1", Active: true, Published: true, UserID: user1ID},
		{Title: "Article 2", Active: false, Published: true, UserID: user1ID},
		{Title: "Article 3", Active: true, Published: false, UserID: user2ID},
		{Title: "Article 4", Active: false, Published: false, UserID: user2ID},
	}

	for _, article := range articles {
		if err := db.Create(&article).Error; err != nil {
			t.Fatalf("Failed to seed article: %v", err)
		}
	}

	// Test applying multiple scopes
	var results []TestArticle
	scopedDB := WhereColumn("active", true)(db)
	scopedDB = WhereColumn("published", true)(scopedDB)
	err := scopedDB.Find(&results).Error

	if err != nil {
		t.Fatalf("Multiple scopes query failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 active and published article, got %d", len(results))
	}
}

func TestScope_WhereColumn(t *testing.T) {
	db := setupTestDB(t)
	db.AutoMigrate(&TestArticle{})

	// Seed articles
	db.Create(&TestArticle{Title: "Active Article", Active: true})
	db.Create(&TestArticle{Title: "Inactive Article", Active: false})

	// Test that scope function works
	var results []TestArticle
	err := WhereColumn("active", true)(db).Find(&results).Error

	if err != nil {
		t.Fatalf("WhereColumn scope query failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 active article, got %d", len(results))
	}

	if !results[0].Active {
		t.Error("Expected article to be active")
	}
}

func TestScope_WhereNotColumn(t *testing.T) {
	db := setupTestDB(t)
	db.AutoMigrate(&TestArticle{})

	// Seed articles
	db.Create(&TestArticle{Title: "Active Article", Active: true})
	db.Create(&TestArticle{Title: "Inactive Article", Active: false})

	var results []TestArticle
	err := WhereNotColumn("active", true)(db).Find(&results).Error

	if err != nil {
		t.Fatalf("WhereNotColumn scope query failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 inactive article, got %d", len(results))
	}

	if results[0].Active {
		t.Error("Expected article to be inactive")
	}
}

func TestScope_Published(t *testing.T) {
	db := setupTestDB(t)
	db.AutoMigrate(&TestArticle{})

	// Seed articles
	db.Create(&TestArticle{Title: "Published Article", Published: true})
	db.Create(&TestArticle{Title: "Draft Article", Published: false})

	var results []TestArticle
	err := WhereColumn("published", true)(db).Find(&results).Error

	if err != nil {
		t.Fatalf("Published scope query failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 published article, got %d", len(results))
	}

	if !results[0].Published {
		t.Error("Expected article to be published")
	}
}

func TestScope_Draft(t *testing.T) {
	db := setupTestDB(t)
	db.AutoMigrate(&TestArticle{})

	// Seed articles
	db.Create(&TestArticle{Title: "Published Article", Published: true})
	db.Create(&TestArticle{Title: "Draft Article", Published: false})

	var results []TestArticle
	err := WhereColumn("published", false)(db).Find(&results).Error

	if err != nil {
		t.Fatalf("Draft scope query failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 draft article, got %d", len(results))
	}

	if results[0].Published {
		t.Error("Expected article to be draft (not published)")
	}
}

func TestScope_OrderByCreatedAt(t *testing.T) {
	db := setupTestDB(t)
	db.AutoMigrate(&TestArticle{})

	// Seed articles
	db.Create(&TestArticle{Title: "First"})
	db.Create(&TestArticle{Title: "Second"})
	db.Create(&TestArticle{Title: "Third"})

	t.Run("ascending", func(t *testing.T) {
		var results []TestArticle
		err := OrderByCreatedAt("ASC")(db).Find(&results).Error

		if err != nil {
			t.Fatalf("OrderByCreatedAt ASC failed: %v", err)
		}

		if len(results) != 3 {
			t.Errorf("Expected 3 articles, got %d", len(results))
		}

		// Check order (oldest first)
		for i := 0; i < len(results)-1; i++ {
			if results[i].CreatedAt.After(results[i+1].CreatedAt) {
				t.Error("Articles not ordered by created_at ASC")
			}
		}
	})

	t.Run("descending", func(t *testing.T) {
		var results []TestArticle
		err := OrderByCreatedAt("DESC")(db).Find(&results).Error

		if err != nil {
			t.Fatalf("OrderByCreatedAt DESC failed: %v", err)
		}

		// Check order (newest first)
		for i := 0; i < len(results)-1; i++ {
			if results[i].CreatedAt.Before(results[i+1].CreatedAt) {
				t.Error("Articles not ordered by created_at DESC")
			}
		}
	})
}

func TestScope_OrderByUpdatedAt(t *testing.T) {
	db := setupTestDB(t)
	db.AutoMigrate(&TestArticle{})

	// Seed articles
	db.Create(&TestArticle{Title: "First"})
	db.Create(&TestArticle{Title: "Second"})

	var results []TestArticle
	err := OrderByUpdatedAt("DESC")(db).Find(&results).Error

	if err != nil {
		t.Fatalf("OrderByUpdatedAt failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 articles, got %d", len(results))
	}

	// Verify order exists
	for i := 0; i < len(results)-1; i++ {
		if results[i].UpdatedAt.Before(results[i+1].UpdatedAt) {
			t.Error("Articles not ordered by updated_at DESC")
		}
	}
}

func TestScope_OrderByColumn(t *testing.T) {
	db := setupTestDB(t)
	db.AutoMigrate(&TestArticle{})

	// Seed articles
	db.Create(&TestArticle{Title: "Zebra"})
	db.Create(&TestArticle{Title: "Apple"})
	db.Create(&TestArticle{Title: "Mango"})

	var results []TestArticle
	err := OrderByColumn("title", "ASC")(db).Find(&results).Error

	if err != nil {
		t.Fatalf("OrderByColumn failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 articles, got %d", len(results))
	}

	// Check alphabetical order
	if results[0].Title != "Apple" || results[1].Title != "Mango" || results[2].Title != "Zebra" {
		t.Errorf("Articles not ordered by title ASC: got %s, %s, %s", results[0].Title, results[1].Title, results[2].Title)
	}
}

func TestScope_PaginateScope(t *testing.T) {
	db := setupTestDB(t)
	db.AutoMigrate(&TestArticle{})

	// Seed 10 articles
	for i := 1; i <= 10; i++ {
		db.Create(&TestArticle{Title: "Article"})
	}

	t.Run("first page", func(t *testing.T) {
		var results []TestArticle
		err := db.Model(&TestArticle{}).Scopes(PaginateScope(1, 3)).Order("created_at ASC").Find(&results).Error

		if err != nil {
			t.Fatalf("PaginateScope page 1 failed: %v", err)
		}

		if len(results) != 3 {
			t.Errorf("Expected 3 articles on page 1, got %d", len(results))
		}
	})

	t.Run("second page", func(t *testing.T) {
		var results []TestArticle
		err := db.Model(&TestArticle{}).Scopes(PaginateScope(2, 3)).Order("created_at ASC").Find(&results).Error

		if err != nil {
			t.Fatalf("PaginateScope page 2 failed: %v", err)
		}

		if len(results) != 3 {
			t.Errorf("Expected 3 articles on page 2, got %d", len(results))
		}
	})
}

func TestScope_Search(t *testing.T) {
	db := setupTestDB(t)
	db.AutoMigrate(&TestArticle{})

	// Seed articles
	db.Create(&TestArticle{Title: "Go Programming", Content: "Learn Go language"})
	db.Create(&TestArticle{Title: "Python Guide", Content: "Python tutorial"})
	db.Create(&TestArticle{Title: "JavaScript Basics", Content: "JS fundamentals with Go mention"})

	t.Run("search with results", func(t *testing.T) {
		var results []TestArticle
		err := Search([]string{"title", "content"}, "Go")(db).Find(&results).Error

		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 articles matching 'Go', got %d", len(results))
		}
	})

	t.Run("search with no results", func(t *testing.T) {
		var results []TestArticle
		err := Search([]string{"title", "content"}, "Ruby")(db).Find(&results).Error

		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if len(results) != 0 {
			t.Errorf("Expected 0 articles matching 'Ruby', got %d", len(results))
		}
	})

	t.Run("search with empty term", func(t *testing.T) {
		var results []TestArticle
		err := Search([]string{"title", "content"}, "")(db).Find(&results).Error

		if err != nil {
			t.Fatalf("Search with empty term failed: %v", err)
		}

		// Should return all articles
		if len(results) != 3 {
			t.Errorf("Expected 3 articles with empty search, got %d", len(results))
		}
	})
}

func TestScope_BelongsTo(t *testing.T) {
	db := setupTestDB(t)
	db.AutoMigrate(&TestArticle{})

	userID1 := uuid.New()
	userID2 := uuid.New()

	// Seed articles for different users
	db.Create(&TestArticle{Title: "User 1 Article 1", UserID: userID1})
	db.Create(&TestArticle{Title: "User 1 Article 2", UserID: userID1})
	db.Create(&TestArticle{Title: "User 2 Article 1", UserID: userID2})

	var results []TestArticle
	err := BelongsTo("user_id", userID1)(db).Find(&results).Error

	if err != nil {
		t.Fatalf("BelongsTo scope failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 articles for user 1, got %d", len(results))
	}

	for _, article := range results {
		if article.UserID != userID1 {
			t.Errorf("Expected article to belong to user %v, got user %v", userID1, article.UserID)
		}
	}
}

func TestScope_CustomScope(t *testing.T) {
	db := setupTestDB(t)
	db.AutoMigrate(&TestArticle{})

	// Seed articles
	db.Create(&TestArticle{Title: "Short", Content: "A"})
	db.Create(&TestArticle{Title: "Medium Title", Content: "B"})
	db.Create(&TestArticle{Title: "Very Long Title Here", Content: "C"})

	// Create a custom scope
	longTitles := func(db *gorm.DB) *gorm.DB {
		return db.Where("LENGTH(title) > ?", 10)
	}

	var results []TestArticle
	err := longTitles(db).Find(&results).Error

	if err != nil {
		t.Fatalf("Custom scope failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 articles with long titles, got %d", len(results))
	}
}

func TestScope_CombinedScopes(t *testing.T) {
	db := setupTestDB(t)
	db.AutoMigrate(&TestArticle{})

	userID1 := uuid.New()
	userID2 := uuid.New()

	// Seed complex data
	db.Create(&TestArticle{Title: "Active Published Article", Active: true, Published: true, UserID: userID1})
	db.Create(&TestArticle{Title: "Active Draft Article", Active: true, Published: false, UserID: userID1})
	db.Create(&TestArticle{Title: "Inactive Published Article", Active: false, Published: true, UserID: userID2})
	db.Create(&TestArticle{Title: "Inactive Draft Article", Active: false, Published: false, UserID: userID2})

	// Test combining multiple scopes
	var results []TestArticle
	scopedDB := WhereColumn("active", true)(db)
	scopedDB = WhereColumn("published", true)(scopedDB)
	scopedDB = BelongsTo("user_id", userID1)(scopedDB)
	err := scopedDB.Find(&results).Error

	if err != nil {
		t.Fatalf("Combined scopes query failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 active, published article for user 1, got %d", len(results))
	}

	if results[0].Title != "Active Published Article" {
		t.Errorf("Got wrong article: %s", results[0].Title)
	}
}

func TestScope_ScopeWithMethods(t *testing.T) {
	db := setupTestDB(t)
	db.AutoMigrate(&TestArticle{})

	// Seed articles
	for i := 1; i <= 5; i++ {
		db.Create(&TestArticle{
			Title:     "Article",
			Active:    i%2 == 0, // Even IDs are active
			Published: true,
			UserID:    uuid.New(),
		})
	}

	// Test combining scopes with GORM methods using Scopes method
	var results []TestArticle
	err := db.Model(&TestArticle{}).
		Scopes(WhereColumn("active", true), WhereColumn("published", true)).
		Order("created_at DESC").
		Limit(1).
		Find(&results).Error

	if err != nil {
		t.Fatalf("Scope with methods failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	// Should be the most recent active and published article
	if !results[0].Active || !results[0].Published {
		t.Error("Expected article to be active and published")
	}
}
