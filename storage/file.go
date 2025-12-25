package storage

import (
	"io"
	"mime"
	"mime/multipart"
	"path/filepath"
)

// File represents an uploaded file with metadata.
type File struct {
	// Name is the original filename
	Name string

	// Size is the file size in bytes
	Size int64

	// MimeType is the MIME type (auto-detected if empty)
	MimeType string

	// Extension is the file extension (without dot)
	Extension string

	// Reader provides the file contents
	Reader io.Reader

	// Visibility determines if file is public or private
	Visibility Visibility

	// Metadata contains custom key-value pairs
	Metadata map[string]string
}

// NewFile creates a new File from an io.Reader.
func NewFile(name string, size int64, reader io.Reader) *File {
	f := &File{
		Name:       name,
		Size:       size,
		Reader:     reader,
		Extension:  filepath.Ext(name),
		Visibility: VisibilityPrivate,
		Metadata:   make(map[string]string),
	}

	// Auto-detect MIME type if not set
	if f.MimeType == "" {
		f.MimeType = DetectMimeType(name)
	}

	// Remove leading dot from extension
	if len(f.Extension) > 0 && f.Extension[0] == '.' {
		f.Extension = f.Extension[1:]
	}

	return f
}

// NewFileFromMultipart creates a File from multipart.FileHeader.
// This is a convenience helper for HTTP file uploads.
func NewFileFromMultipart(header *multipart.FileHeader) (*File, error) {
	reader, err := header.Open()
	if err != nil {
		return nil, err
	}

	f := NewFile(header.Filename, header.Size, reader)

	// Use MIME type from header if available
	if contentType := header.Header.Get("Content-Type"); contentType != "" {
		f.MimeType = contentType
	}

	return f, nil
}

// DetectMimeType detects MIME type from filename extension.
func DetectMimeType(filename string) string {
	ext := filepath.Ext(filename)
	if ext == "" {
		return "application/octet-stream"
	}

	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		return "application/octet-stream"
	}

	return mimeType
}

// WithVisibility sets the file visibility.
func (f *File) WithVisibility(v Visibility) *File {
	f.Visibility = v
	return f
}

// WithMimeType sets the MIME type.
func (f *File) WithMimeType(mimeType string) *File {
	f.MimeType = mimeType
	return f
}

// WithMetadata adds custom metadata.
func (f *File) WithMetadata(key, value string) *File {
	if f.Metadata == nil {
		f.Metadata = make(map[string]string)
	}
	f.Metadata[key] = value
	return f
}
