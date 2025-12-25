// Package s3 implements an S3-compatible storage driver using MinIO SDK.
// Supports AWS S3, MinIO, DigitalOcean Spaces, Backblaze B2, and other S3-compatible services.
package s3

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/azizndao/glib/storage"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Options configures the S3 storage driver.
type Options struct {
	// Endpoint is the S3 endpoint (e.g., "s3.amazonaws.com", "minio.example.com:9000")
	// For AWS S3, use region-specific endpoints like "s3.us-east-1.amazonaws.com"
	Endpoint string

	// AccessKey is the access key ID
	AccessKey string

	// SecretKey is the secret access key
	SecretKey string

	// Bucket is the S3 bucket name (required)
	Bucket string

	// Region is the bucket region (optional, used for AWS S3)
	Region string

	// UseSSL enables HTTPS (default: true for AWS, false for local MinIO)
	UseSSL bool

	// BaseURL is the base URL for accessing files (optional, uses S3 URL if empty)
	BaseURL string

	// DefaultVisibility sets the default visibility for new files
	DefaultVisibility storage.Visibility

	// Prefix is prepended to all paths (optional)
	Prefix string
}

// S3Storage implements S3-compatible storage using MinIO SDK.
type S3Storage struct {
	client            *minio.Client
	bucket            string
	region            string
	baseURL           string
	defaultVisibility storage.Visibility
	prefix            string
}

// New creates a new S3 storage driver.
func New(opts Options) (*S3Storage, error) {
	if opts.Endpoint == "" {
		return nil, fmt.Errorf("storage/s3: endpoint is required")
	}

	if opts.Bucket == "" {
		return nil, fmt.Errorf("storage/s3: bucket is required")
	}

	if opts.DefaultVisibility == "" {
		opts.DefaultVisibility = storage.VisibilityPrivate
	}

	// Create MinIO client
	client, err := minio.New(opts.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(opts.AccessKey, opts.SecretKey, ""),
		Secure: opts.UseSSL,
		Region: opts.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("storage/s3: failed to create client: %w", err)
	}

	return &S3Storage{
		client:            client,
		bucket:            opts.Bucket,
		region:            opts.Region,
		baseURL:           strings.TrimSuffix(opts.BaseURL, "/"),
		defaultVisibility: opts.DefaultVisibility,
		prefix:            strings.Trim(opts.Prefix, "/"),
	}, nil
}

// Put stores a file.
func (s *S3Storage) Put(ctx context.Context, path string, contents io.Reader) error {
	key := s.fullPath(path)

	// Use PutObject with unknown size (-1)
	_, err := s.client.PutObject(ctx, s.bucket, key, contents, -1, minio.PutObjectOptions{
		ContentType: storage.DetectMimeType(path),
	})

	if err != nil {
		return fmt.Errorf("storage/s3: failed to put object: %w", err)
	}

	return nil
}

// PutFile stores a file with metadata.
func (s *S3Storage) PutFile(ctx context.Context, path string, file *storage.File) error {
	key := s.fullPath(path)
	visibility := file.Visibility
	if visibility == "" {
		visibility = s.defaultVisibility
	}

	opts := minio.PutObjectOptions{
		ContentType:  file.MimeType,
		UserMetadata: file.Metadata,
	}

	// Set ACL based on visibility
	if visibility == storage.VisibilityPublic {
		opts.UserMetadata["x-amz-acl"] = "public-read"
	}

	_, err := s.client.PutObject(ctx, s.bucket, key, file.Reader, file.Size, opts)
	if err != nil {
		return fmt.Errorf("storage/s3: failed to put file: %w", err)
	}

	return nil
}

// Get retrieves file contents.
func (s *S3Storage) Get(ctx context.Context, path string) ([]byte, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, s.fullPath(path), minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("storage/s3: failed to get object: %w", err)
	}
	defer obj.Close()

	data, err := io.ReadAll(obj)
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return nil, storage.ErrFileNotFound
		}
		return nil, fmt.Errorf("storage/s3: failed to read object: %w", err)
	}

	return data, nil
}

// GetStream retrieves file as a stream.
func (s *S3Storage) GetStream(ctx context.Context, path string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, s.fullPath(path), minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("storage/s3: failed to get stream: %w", err)
	}

	// Check if object exists by reading stat
	if _, err := obj.Stat(); err != nil {
		obj.Close()
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return nil, storage.ErrFileNotFound
		}
		return nil, fmt.Errorf("storage/s3: failed to stat object: %w", err)
	}

	return obj, nil
}

// Exists checks if file exists.
func (s *S3Storage) Exists(ctx context.Context, path string) (bool, error) {
	_, err := s.client.StatObject(ctx, s.bucket, s.fullPath(path), minio.StatObjectOptions{})
	if err != nil {
		errResponse := minio.ToErrorResponse(err)
		if errResponse.Code == "NoSuchKey" || errResponse.Code == "NotFound" {
			return false, nil
		}
		return false, fmt.Errorf("storage/s3: failed to check existence: %w", err)
	}
	return true, nil
}

// Missing checks if file doesn't exist.
func (s *S3Storage) Missing(ctx context.Context, path string) (bool, error) {
	exists, err := s.Exists(ctx, path)
	return !exists, err
}

// Size returns file size in bytes.
func (s *S3Storage) Size(ctx context.Context, path string) (int64, error) {
	info, err := s.client.StatObject(ctx, s.bucket, s.fullPath(path), minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return 0, storage.ErrFileNotFound
		}
		return 0, fmt.Errorf("storage/s3: failed to get size: %w", err)
	}
	return info.Size, nil
}

// LastModified returns file modification time.
func (s *S3Storage) LastModified(ctx context.Context, path string) (time.Time, error) {
	info, err := s.client.StatObject(ctx, s.bucket, s.fullPath(path), minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return time.Time{}, storage.ErrFileNotFound
		}
		return time.Time{}, fmt.Errorf("storage/s3: failed to get last modified: %w", err)
	}
	return info.LastModified, nil
}

// Delete removes a file.
func (s *S3Storage) Delete(ctx context.Context, path string) error {
	err := s.client.RemoveObject(ctx, s.bucket, s.fullPath(path), minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("storage/s3: failed to delete object: %w", err)
	}
	return nil
}

// DeleteDirectory removes a directory and all its contents.
func (s *S3Storage) DeleteDirectory(ctx context.Context, path string) error {
	prefix := s.fullPath(path)
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	// List all objects with prefix
	objectsCh := s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	// Delete objects
	errorCh := s.client.RemoveObjects(ctx, s.bucket, objectsCh, minio.RemoveObjectsOptions{})

	// Check for errors
	for err := range errorCh {
		if err.Err != nil {
			return fmt.Errorf("storage/s3: failed to delete directory: %w", err.Err)
		}
	}

	return nil
}

// Copy copies a file.
func (s *S3Storage) Copy(ctx context.Context, from, to string) error {
	src := minio.CopySrcOptions{
		Bucket: s.bucket,
		Object: s.fullPath(from),
	}

	dst := minio.CopyDestOptions{
		Bucket: s.bucket,
		Object: s.fullPath(to),
	}

	_, err := s.client.CopyObject(ctx, dst, src)
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return storage.ErrFileNotFound
		}
		return fmt.Errorf("storage/s3: failed to copy object: %w", err)
	}

	return nil
}

// Move moves a file.
func (s *S3Storage) Move(ctx context.Context, from, to string) error {
	// Copy then delete
	if err := s.Copy(ctx, from, to); err != nil {
		return err
	}
	return s.Delete(ctx, from)
}

// Files lists all files in a directory (non-recursive).
func (s *S3Storage) Files(ctx context.Context, directory string) ([]string, error) {
	prefix := s.fullPath(directory)
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	objectsCh := s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: false,
	})

	var files []string
	for obj := range objectsCh {
		if obj.Err != nil {
			return nil, fmt.Errorf("storage/s3: failed to list files: %w", obj.Err)
		}

		// Skip directories (keys ending with /)
		if !strings.HasSuffix(obj.Key, "/") {
			path := s.relativePath(obj.Key)
			if path != directory && path != directory+"/" {
				files = append(files, path)
			}
		}
	}

	return files, nil
}

// AllFiles lists all files recursively.
func (s *S3Storage) AllFiles(ctx context.Context, directory string) ([]string, error) {
	prefix := s.fullPath(directory)
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	objectsCh := s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	var files []string
	for obj := range objectsCh {
		if obj.Err != nil {
			return nil, fmt.Errorf("storage/s3: failed to list files: %w", obj.Err)
		}

		// Skip directories
		if !strings.HasSuffix(obj.Key, "/") {
			files = append(files, s.relativePath(obj.Key))
		}
	}

	return files, nil
}

// Directories lists all directories (non-recursive).
func (s *S3Storage) Directories(ctx context.Context, directory string) ([]string, error) {
	prefix := s.fullPath(directory)
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	objectsCh := s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: false,
	})

	var dirs []string
	for obj := range objectsCh {
		if obj.Err != nil {
			return nil, fmt.Errorf("storage/s3: failed to list directories: %w", obj.Err)
		}

		// Only include directories (keys ending with /)
		if strings.HasSuffix(obj.Key, "/") {
			path := s.relativePath(obj.Key)
			path = strings.TrimSuffix(path, "/")
			if path != directory {
				dirs = append(dirs, path)
			}
		}
	}

	return dirs, nil
}

// AllDirectories lists all directories recursively.
func (s *S3Storage) AllDirectories(ctx context.Context, directory string) ([]string, error) {
	// Get all files to extract directory structure
	files, err := s.AllFiles(ctx, directory)
	if err != nil {
		return nil, err
	}

	dirMap := make(map[string]bool)
	for _, file := range files {
		dir := filepath.Dir(file)
		for dir != "." && dir != directory {
			dirMap[dir] = true
			dir = filepath.Dir(dir)
		}
	}

	var dirs []string
	for dir := range dirMap {
		dirs = append(dirs, dir)
	}

	return dirs, nil
}

// MakeDirectory creates a directory (no-op for S3).
func (s *S3Storage) MakeDirectory(ctx context.Context, path string) error {
	// S3 doesn't have directories, they're implicit
	return nil
}

// URL gets the URL for a file.
func (s *S3Storage) URL(ctx context.Context, path string) (string, error) {
	if s.baseURL != "" {
		return fmt.Sprintf("%s/%s", s.baseURL, strings.TrimPrefix(path, "/")), nil
	}

	// Generate S3 URL
	endpoint := s.client.EndpointURL().String()
	return fmt.Sprintf("%s/%s/%s", endpoint, s.bucket, s.fullPath(path)), nil
}

// TemporaryURL generates a temporary URL with expiration.
func (s *S3Storage) TemporaryURL(ctx context.Context, path string, expiration time.Time) (string, error) {
	expires := time.Until(expiration)
	if expires <= 0 {
		return "", storage.ErrURLExpired
	}

	reqParams := make(url.Values)
	presignedURL, err := s.client.PresignedGetObject(ctx, s.bucket, s.fullPath(path), expires, reqParams)
	if err != nil {
		return "", fmt.Errorf("storage/s3: failed to generate presigned URL: %w", err)
	}

	return presignedURL.String(), nil
}

// SetVisibility sets file visibility (public/private).
func (s *S3Storage) SetVisibility(ctx context.Context, path string, visibility storage.Visibility) error {
	// MinIO doesn't have a direct SetACL method, so we need to copy the object with new ACL
	// For now, we'll return not implemented
	// In production, you'd need to use CopyObject with the new ACL
	return storage.ErrNotImplemented
}

// GetVisibility gets file visibility.
func (s *S3Storage) GetVisibility(ctx context.Context, path string) (storage.Visibility, error) {
	// MinIO doesn't expose ACL in StatObject
	// For now, return the default visibility
	return s.defaultVisibility, nil
}

// fullPath returns the full S3 key with prefix.
func (s *S3Storage) fullPath(path string) string {
	path = strings.TrimPrefix(path, "/")
	if s.prefix == "" {
		return path
	}
	return s.prefix + "/" + path
}

// relativePath removes the prefix from an S3 key.
func (s *S3Storage) relativePath(key string) string {
	if s.prefix == "" {
		return key
	}
	return strings.TrimPrefix(key, s.prefix+"/")
}
