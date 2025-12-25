package local_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/azizndao/glib/storage"
	"github.com/azizndao/glib/storage/local"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestStorage(t *testing.T) (*local.LocalStorage, string) {
	tmpDir := t.TempDir()

	store, err := local.New(local.Options{
		Root:       tmpDir,
		BaseURL:    "http://localhost:8080/storage",
		URLSecret:  "test-secret-key-for-signing",
		CreateRoot: true,
	})
	require.NoError(t, err)

	return store, tmpDir
}

func TestNew(t *testing.T) {
	t.Run("creates storage with valid options", func(t *testing.T) {
		tmpDir := t.TempDir()

		store, err := local.New(local.Options{
			Root:       tmpDir,
			CreateRoot: true,
		})

		assert.NoError(t, err)
		assert.NotNil(t, store)
	})

	t.Run("returns error for empty root", func(t *testing.T) {
		_, err := local.New(local.Options{})
		assert.Error(t, err)
	})

	t.Run("creates root directory if CreateRoot is true", func(t *testing.T) {
		tmpDir := filepath.Join(t.TempDir(), "newdir")

		_, err := local.New(local.Options{
			Root:       tmpDir,
			CreateRoot: true,
		})

		assert.NoError(t, err)
		assert.DirExists(t, tmpDir)
	})
}

func TestPut(t *testing.T) {
	store, _ := setupTestStorage(t)
	ctx := context.Background()

	t.Run("stores file successfully", func(t *testing.T) {
		content := []byte("test content")
		err := store.Put(ctx, "test.txt", bytes.NewReader(content))

		assert.NoError(t, err)

		// Verify file exists
		exists, err := store.Exists(ctx, "test.txt")
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("creates nested directories", func(t *testing.T) {
		content := []byte("nested content")
		err := store.Put(ctx, "dir1/dir2/file.txt", bytes.NewReader(content))

		assert.NoError(t, err)

		exists, err := store.Exists(ctx, "dir1/dir2/file.txt")
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("overwrites existing file", func(t *testing.T) {
		err := store.Put(ctx, "overwrite.txt", bytes.NewReader([]byte("old")))
		require.NoError(t, err)

		err = store.Put(ctx, "overwrite.txt", bytes.NewReader([]byte("new")))
		assert.NoError(t, err)

		data, err := store.Get(ctx, "overwrite.txt")
		assert.NoError(t, err)
		assert.Equal(t, []byte("new"), data)
	})
}

func TestGet(t *testing.T) {
	store, _ := setupTestStorage(t)
	ctx := context.Background()

	t.Run("retrieves file content", func(t *testing.T) {
		content := []byte("test content")
		err := store.Put(ctx, "test.txt", bytes.NewReader(content))
		require.NoError(t, err)

		data, err := store.Get(ctx, "test.txt")
		assert.NoError(t, err)
		assert.Equal(t, content, data)
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		_, err := store.Get(ctx, "nonexistent.txt")
		assert.ErrorIs(t, err, storage.ErrFileNotFound)
	})
}

func TestGetStream(t *testing.T) {
	store, _ := setupTestStorage(t)
	ctx := context.Background()

	content := []byte("stream content")
	err := store.Put(ctx, "stream.txt", bytes.NewReader(content))
	require.NoError(t, err)

	stream, err := store.GetStream(ctx, "stream.txt")
	require.NoError(t, err)
	defer stream.Close()

	data, err := io.ReadAll(stream)
	assert.NoError(t, err)
	assert.Equal(t, content, data)
}

func TestExists(t *testing.T) {
	store, _ := setupTestStorage(t)
	ctx := context.Background()

	err := store.Put(ctx, "exists.txt", bytes.NewReader([]byte("content")))
	require.NoError(t, err)

	exists, err := store.Exists(ctx, "exists.txt")
	assert.NoError(t, err)
	assert.True(t, exists)

	exists, err = store.Exists(ctx, "notexists.txt")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestMissing(t *testing.T) {
	store, _ := setupTestStorage(t)
	ctx := context.Background()

	missing, err := store.Missing(ctx, "notexists.txt")
	assert.NoError(t, err)
	assert.True(t, missing)

	err = store.Put(ctx, "exists.txt", bytes.NewReader([]byte("content")))
	require.NoError(t, err)

	missing, err = store.Missing(ctx, "exists.txt")
	assert.NoError(t, err)
	assert.False(t, missing)
}

func TestSize(t *testing.T) {
	store, _ := setupTestStorage(t)
	ctx := context.Background()

	content := []byte("test content")
	err := store.Put(ctx, "sized.txt", bytes.NewReader(content))
	require.NoError(t, err)

	size, err := store.Size(ctx, "sized.txt")
	assert.NoError(t, err)
	assert.Equal(t, int64(len(content)), size)
}

func TestLastModified(t *testing.T) {
	store, _ := setupTestStorage(t)
	ctx := context.Background()

	before := time.Now()
	err := store.Put(ctx, "timed.txt", bytes.NewReader([]byte("content")))
	require.NoError(t, err)
	after := time.Now()

	modTime, err := store.LastModified(ctx, "timed.txt")
	assert.NoError(t, err)
	assert.True(t, modTime.After(before) || modTime.Equal(before))
	assert.True(t, modTime.Before(after) || modTime.Equal(after))
}

func TestDelete(t *testing.T) {
	store, _ := setupTestStorage(t)
	ctx := context.Background()

	err := store.Put(ctx, "delete.txt", bytes.NewReader([]byte("content")))
	require.NoError(t, err)

	err = store.Delete(ctx, "delete.txt")
	assert.NoError(t, err)

	exists, err := store.Exists(ctx, "delete.txt")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestDeleteDirectory(t *testing.T) {
	store, _ := setupTestStorage(t)
	ctx := context.Background()

	// Create multiple files in directory
	err := store.Put(ctx, "deldir/file1.txt", bytes.NewReader([]byte("1")))
	require.NoError(t, err)
	err = store.Put(ctx, "deldir/file2.txt", bytes.NewReader([]byte("2")))
	require.NoError(t, err)
	err = store.Put(ctx, "deldir/sub/file3.txt", bytes.NewReader([]byte("3")))
	require.NoError(t, err)

	err = store.DeleteDirectory(ctx, "deldir")
	assert.NoError(t, err)

	// Verify all files are gone
	exists, err := store.Exists(ctx, "deldir/file1.txt")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestCopy(t *testing.T) {
	store, _ := setupTestStorage(t)
	ctx := context.Background()

	content := []byte("copy content")
	err := store.Put(ctx, "source.txt", bytes.NewReader(content))
	require.NoError(t, err)

	err = store.Copy(ctx, "source.txt", "dest.txt")
	assert.NoError(t, err)

	// Verify both files exist
	sourceData, err := store.Get(ctx, "source.txt")
	assert.NoError(t, err)
	assert.Equal(t, content, sourceData)

	destData, err := store.Get(ctx, "dest.txt")
	assert.NoError(t, err)
	assert.Equal(t, content, destData)
}

func TestMove(t *testing.T) {
	store, _ := setupTestStorage(t)
	ctx := context.Background()

	content := []byte("move content")
	err := store.Put(ctx, "source.txt", bytes.NewReader(content))
	require.NoError(t, err)

	err = store.Move(ctx, "source.txt", "dest.txt")
	assert.NoError(t, err)

	// Source should not exist
	exists, err := store.Exists(ctx, "source.txt")
	assert.NoError(t, err)
	assert.False(t, exists)

	// Destination should exist with same content
	destData, err := store.Get(ctx, "dest.txt")
	assert.NoError(t, err)
	assert.Equal(t, content, destData)
}

func TestFiles(t *testing.T) {
	store, _ := setupTestStorage(t)
	ctx := context.Background()

	// Create test structure
	err := store.Put(ctx, "root.txt", bytes.NewReader([]byte("1")))
	require.NoError(t, err)
	err = store.Put(ctx, "dir/file1.txt", bytes.NewReader([]byte("2")))
	require.NoError(t, err)
	err = store.Put(ctx, "dir/file2.txt", bytes.NewReader([]byte("3")))
	require.NoError(t, err)
	err = store.Put(ctx, "dir/sub/file3.txt", bytes.NewReader([]byte("4")))
	require.NoError(t, err)

	files, err := store.Files(ctx, "dir")
	assert.NoError(t, err)
	assert.Len(t, files, 2)
	assert.Contains(t, files, "dir/file1.txt")
	assert.Contains(t, files, "dir/file2.txt")
}

func TestAllFiles(t *testing.T) {
	store, _ := setupTestStorage(t)
	ctx := context.Background()

	// Create test structure
	err := store.Put(ctx, "dir/file1.txt", bytes.NewReader([]byte("1")))
	require.NoError(t, err)
	err = store.Put(ctx, "dir/sub/file2.txt", bytes.NewReader([]byte("2")))
	require.NoError(t, err)
	err = store.Put(ctx, "dir/sub/deep/file3.txt", bytes.NewReader([]byte("3")))
	require.NoError(t, err)

	files, err := store.AllFiles(ctx, "dir")
	assert.NoError(t, err)
	assert.Len(t, files, 3)
}

func TestDirectories(t *testing.T) {
	store, _ := setupTestStorage(t)
	ctx := context.Background()

	// Create test structure
	err := store.Put(ctx, "root/sub1/file.txt", bytes.NewReader([]byte("1")))
	require.NoError(t, err)
	err = store.Put(ctx, "root/sub2/file.txt", bytes.NewReader([]byte("2")))
	require.NoError(t, err)
	err = store.Put(ctx, "root/sub2/deep/file.txt", bytes.NewReader([]byte("3")))
	require.NoError(t, err)

	dirs, err := store.Directories(ctx, "root")
	assert.NoError(t, err)
	assert.Len(t, dirs, 2)
	assert.Contains(t, dirs, "root/sub1")
	assert.Contains(t, dirs, "root/sub2")
}

func TestMakeDirectory(t *testing.T) {
	store, tmpDir := setupTestStorage(t)
	ctx := context.Background()

	err := store.MakeDirectory(ctx, "newdir/subdir")
	assert.NoError(t, err)

	// Verify directory exists
	dirPath := filepath.Join(tmpDir, "newdir/subdir")
	info, err := os.Stat(dirPath)
	assert.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestURL(t *testing.T) {
	store, _ := setupTestStorage(t)
	ctx := context.Background()

	url, err := store.URL(ctx, "test/file.txt")
	assert.NoError(t, err)
	assert.Equal(t, "http://localhost:8080/storage/test/file.txt", url)
}

func TestTemporaryURL(t *testing.T) {
	store, _ := setupTestStorage(t)
	ctx := context.Background()

	expiration := time.Now().Add(1 * time.Hour)
	url, err := store.TemporaryURL(ctx, "test/file.txt", expiration)

	assert.NoError(t, err)
	assert.Contains(t, url, "expires=")
	assert.Contains(t, url, "signature=")
	assert.Contains(t, url, "http://localhost:8080/storage/test/file.txt")
}

func TestVisibility(t *testing.T) {
	store, _ := setupTestStorage(t)
	ctx := context.Background()

	err := store.Put(ctx, "visible.txt", bytes.NewReader([]byte("content")))
	require.NoError(t, err)

	t.Run("set and get public visibility", func(t *testing.T) {
		err := store.SetVisibility(ctx, "visible.txt", storage.VisibilityPublic)
		assert.NoError(t, err)

		visibility, err := store.GetVisibility(ctx, "visible.txt")
		assert.NoError(t, err)
		assert.Equal(t, storage.VisibilityPublic, visibility)
	})

	t.Run("set and get private visibility", func(t *testing.T) {
		err := store.SetVisibility(ctx, "visible.txt", storage.VisibilityPrivate)
		assert.NoError(t, err)

		visibility, err := store.GetVisibility(ctx, "visible.txt")
		assert.NoError(t, err)
		assert.Equal(t, storage.VisibilityPrivate, visibility)
	})
}

func TestPutFile(t *testing.T) {
	store, _ := setupTestStorage(t)
	ctx := context.Background()

	file := storage.NewFile("test.txt", 12, bytes.NewReader([]byte("file content")))
	file.WithVisibility(storage.VisibilityPublic)

	err := store.PutFile(ctx, "uploads/test.txt", file)
	assert.NoError(t, err)

	// Verify file exists and has correct visibility
	exists, err := store.Exists(ctx, "uploads/test.txt")
	assert.NoError(t, err)
	assert.True(t, exists)

	visibility, err := store.GetVisibility(ctx, "uploads/test.txt")
	assert.NoError(t, err)
	assert.Equal(t, storage.VisibilityPublic, visibility)
}
