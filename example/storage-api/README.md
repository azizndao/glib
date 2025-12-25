# Storage API Example

A comprehensive HTTP API demonstrating the Glib Storage module features.

## Overview

This example showcases a production-ready file storage API built with Glib's modular storage system. It demonstrates:

- **File uploads** via multipart forms
- **File downloads** with streaming support
- **Temporary signed URLs** with expiration
- **File management** (copy, move, delete)
- **Directory operations** (list, create, delete)
- **Multiple storage disks** (local filesystem and S3-compatible)
- **Visibility control** (public/private files)
- **CORS support** for browser-based uploads

## Quick Start

### 1. Install Dependencies

```bash
cd example/storage-api
go mod download
```

### 2. Run the Server

```bash
# Using default configuration (local storage only)
go run main.go

# Or with custom configuration
STORAGE_ROOT=./my-storage PORT=3000 go run main.go
```

The server will start on `http://localhost:8080` (or your configured port).

Visit `http://localhost:8080/` for interactive API documentation.

## Configuration

Configure the server using environment variables:

### Local Storage (Always Available)

```bash
# Storage directory (created automatically)
STORAGE_ROOT=./storage

# Base URL for file access
BASE_URL=http://localhost:8080/files

# Secret key for signing temporary URLs
URL_SECRET=change-me-in-production

# Server port
PORT=8080
```

### S3 Storage (Optional)

```bash
# S3 endpoint (required to enable S3)
S3_ENDPOINT=s3.amazonaws.com

# S3 credentials
S3_ACCESS_KEY=your-access-key
S3_SECRET_KEY=your-secret-key

# Bucket configuration
S3_BUCKET=my-bucket
S3_REGION=us-east-1
S3_USE_SSL=true

# Optional path prefix for multi-tenancy
S3_PREFIX=uploads/
```

#### S3-Compatible Services

**AWS S3**:
```bash
S3_ENDPOINT=s3.us-east-1.amazonaws.com
S3_USE_SSL=true
S3_REGION=us-east-1
```

**MinIO** (local or self-hosted):
```bash
S3_ENDPOINT=localhost:9000
S3_USE_SSL=false
```

**DigitalOcean Spaces**:
```bash
S3_ENDPOINT=nyc3.digitaloceanspaces.com
S3_USE_SSL=true
S3_REGION=nyc3
```

**Backblaze B2**:
```bash
S3_ENDPOINT=s3.us-west-000.backblazeb2.com
S3_USE_SSL=true
S3_REGION=us-west-000
```

## API Endpoints

### Upload File

Upload a file via multipart form data.

```bash
POST /upload

# Basic upload
curl -F "file=@document.pdf" http://localhost:8080/upload

# With custom path and disk
curl -F "file=@photo.jpg" \
     -F "path=images/2024/photo.jpg" \
     -F "disk=local" \
     -F "visibility=public" \
     http://localhost:8080/upload

# Upload to S3
curl -F "file=@video.mp4" \
     -F "path=videos/demo.mp4" \
     -F "disk=s3" \
     http://localhost:8080/upload
```

**Response**:
```json
{
  "success": true,
  "path": "uploads/document.pdf",
  "url": "http://localhost:8080/files/uploads/document.pdf",
  "size": 1024,
  "disk": "local",
  "modified": "2024-12-24T10:30:00Z"
}
```

### Download File

Download a file from storage.

```bash
GET /files/{path}

# Download from local disk
curl http://localhost:8080/files/uploads/document.pdf

# Download from S3
curl http://localhost:8080/files/videos/demo.mp4?disk=s3

# Download with output
curl -O http://localhost:8080/files/uploads/document.pdf
```

### List Files

List all files in a directory.

```bash
GET /files?dir={directory}&recursive={true|false}&disk={disk}

# List files in uploads directory
curl "http://localhost:8080/files?dir=uploads"

# List all files recursively
curl "http://localhost:8080/files?recursive=true"

# List files on S3
curl "http://localhost:8080/files?disk=s3&dir=images"
```

**Response**:
```json
{
  "success": true,
  "disk": "local",
  "dir": "uploads",
  "count": 2,
  "files": [
    {
      "path": "uploads/document.pdf",
      "url": "http://localhost:8080/files/uploads/document.pdf",
      "size": 1024,
      "modified": "2024-12-24T10:30:00Z",
      "visibility": "public"
    },
    {
      "path": "uploads/photo.jpg",
      "url": "http://localhost:8080/files/uploads/photo.jpg",
      "size": 2048,
      "modified": "2024-12-24T10:35:00Z",
      "visibility": "private"
    }
  ]
}
```

### Delete File

Delete a file or directory.

```bash
DELETE /files/{path}?disk={disk}&directory={true|false}

# Delete a file
curl -X DELETE http://localhost:8080/files/uploads/document.pdf

# Delete a directory
curl -X DELETE "http://localhost:8080/files/uploads?directory=true"

# Delete from S3
curl -X DELETE http://localhost:8080/files/videos/demo.mp4?disk=s3
```

**Response**:
```json
{
  "success": true,
  "path": "uploads/document.pdf",
  "message": "Deleted successfully"
}
```

### Copy File

Copy a file to a new location.

```bash
POST /files/{path}/copy

# Copy file
curl -X POST \
     -H "Content-Type: application/json" \
     -d '{"destination":"backup/document.pdf"}' \
     http://localhost:8080/files/uploads/document.pdf/copy

# Copy on S3
curl -X POST \
     -H "Content-Type: application/json" \
     -d '{"destination":"archive/demo.mp4"}' \
     http://localhost:8080/files/videos/demo.mp4/copy?disk=s3
```

**Response**:
```json
{
  "success": true,
  "source": "uploads/document.pdf",
  "destination": "backup/document.pdf",
  "url": "http://localhost:8080/files/backup/document.pdf"
}
```

### Move File

Move a file to a new location.

```bash
POST /files/{path}/move

# Move file
curl -X POST \
     -H "Content-Type: application/json" \
     -d '{"destination":"archive/document.pdf"}' \
     http://localhost:8080/files/uploads/document.pdf/move
```

**Response**:
```json
{
  "success": true,
  "source": "uploads/document.pdf",
  "destination": "archive/document.pdf",
  "url": "http://localhost:8080/files/archive/document.pdf"
}
```

### Generate Temporary URL

Generate a temporary signed URL with expiration.

```bash
GET /temp-url/{path}?expires={seconds}&disk={disk}

# Generate URL valid for 1 hour (3600 seconds)
curl "http://localhost:8080/temp-url/uploads/document.pdf?expires=3600"

# Generate URL valid for 10 minutes (600 seconds)
curl "http://localhost:8080/temp-url/uploads/private-file.pdf?expires=600"

# Generate URL for S3 file
curl "http://localhost:8080/temp-url/videos/demo.mp4?expires=3600&disk=s3"
```

**Response**:
```json
{
  "success": true,
  "path": "uploads/document.pdf",
  "url": "http://localhost:8080/files/uploads/document.pdf?expires=1703419800&signature=abc123...",
  "expires_at": "2024-12-24T11:30:00Z",
  "expires_in": 3600
}
```

The generated URL can be shared and will work until the expiration time. After expiration, the URL becomes invalid.

### Set File Visibility

Control whether a file is public or private.

```bash
POST /visibility/{path}

# Make file public
curl -X POST \
     -H "Content-Type: application/json" \
     -d '{"visibility":"public"}' \
     http://localhost:8080/visibility/uploads/document.pdf

# Make file private (requires signed URL to access)
curl -X POST \
     -H "Content-Type: application/json" \
     -d '{"visibility":"private"}' \
     http://localhost:8080/visibility/uploads/secret.pdf
```

**Response**:
```json
{
  "success": true,
  "path": "uploads/document.pdf",
  "visibility": "public"
}
```

**Visibility behavior**:
- **Public**: File can be accessed directly via URL
- **Private**: File requires a temporary signed URL to access

### Get File Info

Retrieve file metadata.

```bash
GET /info/{path}?disk={disk}

# Get file info
curl http://localhost:8080/info/uploads/document.pdf

# Get S3 file info
curl http://localhost:8080/info/videos/demo.mp4?disk=s3
```

**Response**:
```json
{
  "success": true,
  "path": "uploads/document.pdf",
  "disk": "local",
  "exists": true,
  "size": 1024,
  "modified": "2024-12-24T10:30:00Z",
  "visibility": "public",
  "url": "http://localhost:8080/files/uploads/document.pdf"
}
```

### Health Check

Check server and storage status.

```bash
GET /health

curl http://localhost:8080/health
```

**Response**:
```json
{
  "status": "healthy",
  "disks": ["local", "s3"],
  "time": "2024-12-24T10:00:00Z"
}
```

## Complete Usage Example

Here's a complete workflow demonstrating all features:

```bash
# 1. Upload a file
curl -F "file=@test.txt" \
     -F "path=documents/test.txt" \
     -F "visibility=private" \
     http://localhost:8080/upload

# 2. Get file info
curl http://localhost:8080/info/documents/test.txt

# 3. Generate temporary URL (valid for 10 minutes)
TEMP_URL=$(curl -s "http://localhost:8080/temp-url/documents/test.txt?expires=600" | jq -r .url)
echo "Temporary URL: $TEMP_URL"

# 4. Download using temporary URL
curl "$TEMP_URL"

# 5. Copy file to backup
curl -X POST \
     -H "Content-Type: application/json" \
     -d '{"destination":"backup/test.txt"}' \
     http://localhost:8080/files/documents/test.txt/copy

# 6. List all files
curl "http://localhost:8080/files?recursive=true"

# 7. Make file public
curl -X POST \
     -H "Content-Type: application/json" \
     -d '{"visibility":"public"}' \
     http://localhost:8080/visibility/documents/test.txt

# 8. Now file is accessible without signature
curl http://localhost:8080/files/documents/test.txt

# 9. Delete the file
curl -X DELETE http://localhost:8080/files/documents/test.txt
```

## Testing with Frontend

Example HTML page for testing file uploads:

```html
<!DOCTYPE html>
<html>
<head>
    <title>File Upload Test</title>
</head>
<body>
    <h1>File Upload</h1>
    <form id="uploadForm">
        <input type="file" name="file" required>
        <select name="disk">
            <option value="local">Local</option>
            <option value="s3">S3</option>
        </select>
        <button type="submit">Upload</button>
    </form>

    <div id="result"></div>

    <script>
        document.getElementById('uploadForm').onsubmit = async (e) => {
            e.preventDefault();
            const formData = new FormData(e.target);
            
            const response = await fetch('http://localhost:8080/upload', {
                method: 'POST',
                body: formData
            });
            
            const result = await response.json();
            document.getElementById('result').innerHTML = 
                `<pre>${JSON.stringify(result, null, 2)}</pre>`;
        };
    </script>
</body>
</html>
```

## Error Handling

The API returns consistent JSON error responses:

```json
{
  "success": false,
  "error": "Error message describing what went wrong"
}
```

Common HTTP status codes:
- `200 OK` - Request succeeded
- `201 Created` - File uploaded successfully
- `400 Bad Request` - Invalid parameters or missing data
- `404 Not Found` - File does not exist
- `500 Internal Server Error` - Server-side error

## Production Deployment

### Security Considerations

1. **Change the URL secret**:
   ```bash
   URL_SECRET=$(openssl rand -base64 32)
   ```

2. **Use HTTPS** in production (set `BASE_URL` to HTTPS endpoint)

3. **Secure S3 credentials** (use IAM roles or secrets manager)

4. **Add authentication middleware** for upload/delete operations

5. **Implement rate limiting** to prevent abuse

6. **Validate file types** and sizes before upload

### Environment Variables Template

Create a `.env` file (not committed to git):

```bash
# Local Storage
STORAGE_ROOT=/var/storage/glib
BASE_URL=https://api.example.com/files
URL_SECRET=your-secure-random-secret-here

# S3 Storage (optional)
S3_ENDPOINT=s3.amazonaws.com
S3_ACCESS_KEY=AKIAIOSFODNN7EXAMPLE
S3_SECRET_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
S3_BUCKET=my-production-bucket
S3_REGION=us-east-1
S3_USE_SSL=true
S3_PREFIX=production/

# Server
PORT=8080
```

### Docker Deployment

Example `Dockerfile`:

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o storage-api main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/storage-api .
EXPOSE 8080
CMD ["./storage-api"]
```

Build and run:

```bash
docker build -t glib-storage-api .
docker run -p 8080:8080 \
  -e STORAGE_ROOT=/storage \
  -e URL_SECRET=your-secret \
  -v $(pwd)/storage:/storage \
  glib-storage-api
```

## Architecture

This example demonstrates:

- **Modular disk management** - Easy to switch between local and S3
- **Streaming support** - Efficient handling of large files
- **Signed URLs** - Secure temporary access to private files
- **CORS support** - Works with browser-based uploads
- **Clean error handling** - Consistent JSON responses
- **Environment configuration** - Easy deployment customization

## Further Reading

- [Storage Module Documentation](../../storage/README.md)
- [Local Driver Documentation](../../storage/local/README.md)
- [S3 Driver Documentation](../../storage/s3/README.md)
- [Glib Framework Documentation](../../README.md)

## License

This example is part of the Glib framework and is released under the same license.
