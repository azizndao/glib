package orm

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Test model structs
type TestUser struct {
	Model
	Name   string
	Email  string
	Age    int
	Active bool
}

type TestPost struct {
	Model
	Title     string
	Content   string
	Published bool
	UserID    uuid.UUID
}

func TestModel_IsNew(t *testing.T) {
	tests := []struct {
		name     string
		model    Model
		expected bool
	}{
		{
			name:     "new model with nil UUID",
			model:    Model{ID: uuid.Nil},
			expected: true,
		},
		{
			name:     "existing model with non-nil UUID",
			model:    Model{ID: uuid.New()},
			expected: false,
		},
		{
			name:     "existing model with specific UUID",
			model:    Model{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.model.IsNew(); got != tt.expected {
				t.Errorf("IsNew() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestModel_IsDeleted(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		model    Model
		expected bool
	}{
		{
			name:     "not deleted - nil DeletedAt",
			model:    Model{DeletedAt: gorm.DeletedAt{}},
			expected: false,
		},
		{
			name: "deleted - has DeletedAt",
			model: Model{
				DeletedAt: gorm.DeletedAt{
					Time:  now,
					Valid: true,
				},
			},
			expected: true,
		},
		{
			name: "not deleted - DeletedAt invalid",
			model: Model{
				DeletedAt: gorm.DeletedAt{
					Time:  now,
					Valid: false,
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.model.IsDeleted(); got != tt.expected {
				t.Errorf("IsDeleted() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestModel_ID(t *testing.T) {
	id := uuid.New()
	model := Model{ID: id}

	if model.ID != id {
		t.Errorf("Expected ID %v, got %v", id, model.ID)
	}
}

func TestModel_Timestamps(t *testing.T) {
	now := time.Now()
	model := &Model{
		CreatedAt: now,
		UpdatedAt: now.Add(1 * time.Hour),
	}

	if !model.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt = %v, want %v", model.CreatedAt, now)
	}

	expected := now.Add(1 * time.Hour)
	if !model.UpdatedAt.Equal(expected) {
		t.Errorf("UpdatedAt = %v, want %v", model.UpdatedAt, expected)
	}
}

func TestModel_DeletedAt(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		model    Model
		expected gorm.DeletedAt
	}{
		{
			name:     "not deleted",
			model:    Model{DeletedAt: gorm.DeletedAt{Valid: false}},
			expected: gorm.DeletedAt{Valid: false},
		},
		{
			name: "deleted",
			model: Model{
				DeletedAt: gorm.DeletedAt{
					Time:  now,
					Valid: true,
				},
			},
			expected: gorm.DeletedAt{Time: now, Valid: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.model.DeletedAt
			if got.Valid != tt.expected.Valid {
				t.Errorf("DeletedAt.Valid = %v, want %v", got.Valid, tt.expected.Valid)
			}
			if tt.expected.Valid && !got.Time.Equal(tt.expected.Time) {
				t.Errorf("DeletedAt.Time = %v, want %v", got.Time, tt.expected.Time)
			}
		})
	}
}

func TestModel_UpdatedAtModification(t *testing.T) {
	model := &Model{
		UpdatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	oldUpdatedAt := model.UpdatedAt

	// Manually update UpdatedAt to simulate what GORM would do
	model.UpdatedAt = time.Now()

	if !model.UpdatedAt.After(oldUpdatedAt) {
		t.Errorf("UpdatedAt modification failed. Old: %v, New: %v", oldUpdatedAt, model.UpdatedAt)
	}

	// Verify it's approximately now
	if time.Since(model.UpdatedAt) > time.Second {
		t.Errorf("UpdatedAt set to %v, expected time close to now", model.UpdatedAt)
	}
}

func TestModel_EmbeddedInStruct(t *testing.T) {
	// Test that Model can be embedded in other structs
	id := uuid.New()
	user := TestUser{
		Model: Model{ID: id},
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   30,
	}

	if user.ID != id {
		t.Errorf("Embedded Model ID = %v, want %v", user.ID, id)
	}

	// ID is set, so not new
	if user.IsNew() {
		t.Error("User with ID should not be new")
	}

	// Test direct field access
	newID := uuid.New()
	user.ID = newID
	if user.ID != newID {
		t.Errorf("Setting embedded Model ID failed, ID = %v, want %v", user.ID, newID)
	}

	if user.ID != newID {
		t.Errorf("ID after setting = %v, want %v", user.ID, newID)
	}
}

func TestModel_SoftDelete(t *testing.T) {
	model := &Model{}

	// Initially not deleted
	if model.IsDeleted() {
		t.Error("New model should not be deleted")
	}

	deletedAt := model.DeletedAt
	if deletedAt.Valid {
		t.Errorf("New model DeletedAt.Valid = true, want false")
	}

	// Simulate soft delete
	now := time.Now()
	model.DeletedAt = gorm.DeletedAt{
		Time:  now,
		Valid: true,
	}

	if !model.IsDeleted() {
		t.Error("Model with DeletedAt should be deleted")
	}

	deletedAt = model.DeletedAt
	if !deletedAt.Valid {
		t.Error("DeletedAt.Valid should be true after soft delete")
	}
	if !deletedAt.Time.Equal(now) {
		t.Errorf("DeletedAt.Time = %v, want %v", deletedAt.Time, now)
	}
}

func TestTimestamps_EmbeddedStruct(t *testing.T) {
	type CustomModel struct {
		ID uuid.UUID
		Timestamps
		Name string
	}

	now := time.Now()
	model := CustomModel{
		ID: uuid.New(),
		Timestamps: Timestamps{
			CreatedAt: now,
			UpdatedAt: now,
		},
		Name: "Test",
	}

	if !model.CreatedAt.Equal(now) {
		t.Errorf("Timestamps.CreatedAt = %v, want %v", model.CreatedAt, now)
	}

	if !model.UpdatedAt.Equal(now) {
		t.Errorf("Timestamps.UpdatedAt = %v, want %v", model.UpdatedAt, now)
	}
}

func TestSoftDeletes_EmbeddedStruct(t *testing.T) {
	type CustomModel struct {
		ID uuid.UUID
		SoftDeletes
		Name string
	}

	model := CustomModel{
		ID:   uuid.New(),
		Name: "Test",
	}

	// Initially not deleted
	if model.IsDeleted() {
		t.Error("New model should not be deleted")
	}

	// Simulate soft delete
	now := time.Now()
	model.DeletedAt = gorm.DeletedAt{
		Time:  now,
		Valid: true,
	}

	if !model.IsDeleted() {
		t.Error("Model with DeletedAt should be deleted")
	}
}

func TestModel_BeforeCreateHook(t *testing.T) {
	model := &Model{}

	// ID should be Nil initially
	if model.ID != uuid.Nil {
		t.Errorf("New model ID should be Nil, got %v", model.ID)
	}

	// Simulate BeforeCreate hook
	err := model.BeforeCreate(nil)
	if err != nil {
		t.Errorf("BeforeCreate returned error: %v", err)
	}

	// ID should now be set
	if model.ID == uuid.Nil {
		t.Error("BeforeCreate should have set ID")
	}
}

func TestModel_BeforeCreateHook_PresetID(t *testing.T) {
	presetID := uuid.New()
	model := &Model{ID: presetID}

	// Simulate BeforeCreate hook
	err := model.BeforeCreate(nil)
	if err != nil {
		t.Errorf("BeforeCreate returned error: %v", err)
	}

	// ID should remain the same
	if model.ID != presetID {
		t.Errorf("BeforeCreate should not override preset ID, got %v, want %v", model.ID, presetID)
	}
}
