package glib

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/azizndao/glib/errors"
	"github.com/azizndao/glib/slog"
	"github.com/azizndao/glib/validation"
	"github.com/go-chi/chi/v5"
)

// Ctx provides easy access to request data and response helpers
type Ctx struct {
	Request    *http.Request
	Response   http.ResponseWriter
	statusCode int
	body       []byte                // Cached request body
	bodyRead   bool                  // Track if body has been read
	logger     *slog.Logger          // Logger instance for logging within routes and middleware
	validator  *validation.Validator // Validator instance for request validation
}

// newCtx creates a new Context from request and response
func newCtx(w http.ResponseWriter, r *http.Request, logger *slog.Logger, validator *validation.Validator) *Ctx {
	return &Ctx{
		Request:    r,
		Response:   w,
		statusCode: http.StatusOK, // Default to 200
		logger:     logger,
		validator:  validator,
	}
}

func (c *Ctx) Context() context.Context {
	return c.Request.Context()
}

// Deadline returns the time when work done on behalf of this context should be canceled
func (c *Ctx) Deadline() (deadline time.Time, ok bool) {
	return c.Request.Context().Deadline()
}

// Done returns a channel that's closed when work done on behalf of this context should be canceled
func (c *Ctx) Done() <-chan struct{} {
	return c.Request.Context().Done()
}

// Err returns the error if the context is canceled or has exceeded its deadline
func (c *Ctx) Err() error {
	return c.Request.Context().Err()
}

// Value returns the value associated with this context for key, or nil if no value is associated with key
func (c *Ctx) Value(key any) any {
	return c.Request.Context().Value(key)
}

// Logger returns the logger instance for logging within routes and middleware
func (c *Ctx) Logger() *slog.Logger {
	return c.logger
}

// SetValue sets a custom value in the request context
func (c *Ctx) SetValue(key any, value any) {
	c.Request = c.Request.WithContext(context.WithValue(c.Context(), key, value))
}

// GetValue gets a value from the request context
func (c *Ctx) GetValue(key any) any {
	return c.Context().Value(key)
}

// ParseBody parses the request body into the given struct
// Validates that Content-Type is application/json before parsing
func (c *Ctx) ParseBody(out any) error {
	// Validate Content-Type
	contentType := c.ContentType()
	if contentType != "" && !strings.HasPrefix(strings.ToLower(contentType), "application/json") {
		return errors.BadRequest("Invalid Content-Type", fmt.Errorf("expected application/json, got %s", contentType))
	}

	body, err := c.Body()
	if err != nil {
		return err
	}

	if len(body) == 0 {
		return errors.BadRequest("Empty request body", nil)
	}

	if err := json.Unmarshal(body, out); err != nil {
		return errors.BadRequest("Invalid JSON", err)
	}

	return nil
}

// ValidateBody parses and validates the request body in one call
func (c *Ctx) ValidateBody(out any) error {
	if err := c.ParseBody(out); err != nil {
		return errors.BadRequest("Invalid request body", err)
	}

	// Get locale from Accept-Language header
	locale := c.getLocaleFromHeader()
	return c.validator.Validate(out, locale)
}

// ValidateBody is a generic helper to parse and validate the request body
func ValidateBody[T any](c *Ctx) (*T, error) {
	var out T
	if err := c.ValidateBody(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

// getLocaleFromHeader extracts the locale from Accept-Language header
// Returns the first supported locale or "en" as default
func (c *Ctx) getLocaleFromHeader() string {
	acceptLang := c.Get("Accept-Language")
	if acceptLang == "" {
		return "en"
	}

	// Parse Accept-Language header (e.g., "en-US,en;q=0.9,fr;q=0.8")
	// Extract first language code
	parts := strings.Split(acceptLang, ",")
	if len(parts) > 0 {
		lang := strings.TrimSpace(parts[0])
		// Extract language code before any quality value or variant
		if idx := strings.Index(lang, ";"); idx != -1 {
			lang = lang[:idx]
		}
		if idx := strings.Index(lang, "-"); idx != -1 {
			lang = lang[:idx]
		}
		return strings.ToLower(strings.TrimSpace(lang))
	}

	return "en"
}

// Body gets the raw request body as bytes
// The body is cached after the first read, so this method can be called multiple times
func (c *Ctx) Body() ([]byte, error) {
	if c.bodyRead {
		return c.body, nil
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, err
	}
	defer c.Request.Body.Close()

	c.body = body
	c.bodyRead = true
	return body, nil
}

// FormValue gets a form value by key
func (c *Ctx) FormValue(key string) string {
	return c.Request.FormValue(key)
}

// FormFile gets a file from multipart form
func (c *Ctx) FormFile(key string) (multipart.File, *multipart.FileHeader, error) {
	return c.Request.FormFile(key)
}

// PathValue gets a path parameter by key
// Uses Chi's URL parameter extraction from request context
func (c *Ctx) PathValue(key string) string {
	return chi.URLParam(c.Request, key)
}

// Query gets a query parameter by key
func (c *Ctx) Query(key string) string {
	return c.Request.URL.Query().Get(key)
}

// QueryInt gets a query parameter as int
func (c *Ctx) QueryInt(key string) (int, error) {
	value := c.Query(key)
	if value == "" {
		return 0, errors.New("Query parameter not found")
	}
	return strconv.Atoi(value)
}

// QueryIntDefault gets a query parameter as int with a default value
func (c *Ctx) QueryIntDefault(key string, defaultValue int) int {
	intValue, err := c.QueryInt(key)
	if err != nil {
		return defaultValue
	}
	return intValue
}

// QueryBool gets a query parameter as bool
func (c *Ctx) QueryBool(key string) bool {
	value := strings.ToLower(c.Query(key))
	return value == "true" || value == "1" || value == "yes" || value == "on"
}

// QueryFloat gets a query parameter as float64
func (c *Ctx) QueryFloat(key string) (float64, error) {
	value := c.Query(key)
	if value == "" {
		return 0, errors.New("Query parameter not found")
	}
	return strconv.ParseFloat(value, 64)
}

// QueryFloatDefault gets a query parameter as float64 with a default value
func (c *Ctx) QueryFloatDefault(key string, defaultValue float64) float64 {
	floatValue, err := c.QueryFloat(key)
	if err != nil {
		return defaultValue
	}
	return floatValue
}

// QueryDefault gets a query parameter with a default value
func (c *Ctx) QueryDefault(key, defaultValue string) string {
	value := c.Query(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// QueryAll gets all values for a query parameter key
func (c *Ctx) QueryAll(key string) []string {
	return c.Request.URL.Query()[key]
}

// QueryArray is an alias for QueryAll for convenience
func (c *Ctx) QueryArray(key string) []string {
	return c.QueryAll(key)
}

// PathInt gets a path parameter as int
func (c *Ctx) PathInt(key string) (int, error) {
	value := c.PathValue(key)
	if value == "" {
		return 0, errors.New("Path parameter not found")
	}
	return strconv.Atoi(value)
}

// PathIntDefault gets a path parameter as int with a default value
func (c *Ctx) PathIntDefault(key string, defaultValue int) int {
	intValue, err := c.PathInt(key)
	if err != nil {
		return defaultValue
	}
	return intValue
}

// PathFloat gets a path parameter as float64
func (c *Ctx) PathFloat(key string) (float64, error) {
	value := c.PathValue(key)
	if value == "" {
		return 0, errors.New("Path parameter not found")
	}
	return strconv.ParseFloat(value, 64)
}

// Get gets a request header by key
func (c *Ctx) Get(key string) string {
	return c.Request.Header.Get(key)
}

// SetHeaders sets multiple headers at once
func (c *Ctx) SetHeaders(headers map[string]string) *Ctx {
	for key, value := range headers {
		c.Response.Header().Set(key, value)
	}
	return c
}

// GetHeaders gets all request headers
func (c *Ctx) GetHeaders() map[string][]string {
	return c.Request.Header
}

// Authorization gets the Authorization header
func (c *Ctx) Authorization() string {
	return c.Get("Authorization")
}

// BearerToken extracts the bearer token from the Authorization header
// Returns empty string if no bearer token is present
func (c *Ctx) BearerToken() string {
	auth := c.Authorization()
	if auth == "" {
		return ""
	}

	// Check if it starts with "Bearer "
	const prefix = "Bearer "
	if len(auth) > len(prefix) && strings.HasPrefix(auth, prefix) {
		return auth[len(prefix):]
	}

	return ""
}

// ContentType gets the Content-Type header
func (c *Ctx) ContentType() string {
	return c.Get("Content-Type")
}

// IP returns the client's IP address
// When behind a proxy, it extracts the first IP from X-Forwarded-For header
// Properly handles IPv6 addresses and strips port information
func (c *Ctx) IP() string {
	if xff := c.Request.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs: "client, proxy1, proxy2"
		// Extract the first (client) IP
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	if ip := c.Request.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}

	// RemoteAddr includes port, strip it
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		// If splitting fails, return as-is (might be just IP without port)
		return c.Request.RemoteAddr
	}
	return host
}

func (c *Ctx) UserAgent() string {
	return c.Request.UserAgent()
}

func (c *Ctx) Method() string {
	return c.Request.Method
}

func (c *Ctx) Path() string {
	return c.Request.URL.Path
}

// BaseURL gets the base URL (scheme + host)
func (c *Ctx) BaseURL() string {
	return fmt.Sprintf("%s://%s", c.Scheme(), c.Host())
}

// URL gets the full request URL
func (c *Ctx) URL() *url.URL {
	return c.Request.URL
}

// Scheme gets the request scheme (http or https)
func (c *Ctx) Scheme() string {
	if c.Request.TLS != nil {
		return "https"
	}
	if scheme := c.Get("X-Forwarded-Proto"); scheme != "" {
		return scheme
	}
	return "http"
}

// Host gets the request host
func (c *Ctx) Host() string {
	if host := c.Get("X-Forwarded-Host"); host != "" {
		return host
	}
	return c.Request.Host
}

func (c *Ctx) Set(key, value string) *Ctx {
	c.Response.Header().Set(key, value)
	return c
}

func (c *Ctx) GetCookie(name string) (*http.Cookie, error) {
	return c.Request.Cookie(name)
}

// GetCookieDefault gets a cookie value with a default fallback
func (c *Ctx) GetCookieDefault(name, defaultValue string) string {
	cookie, err := c.Request.Cookie(name)
	if err != nil {
		return defaultValue
	}
	return cookie.Value
}

func (c *Ctx) SetCookie(cookie *http.Cookie) *Ctx {
	http.SetCookie(c.Response, cookie)
	return c
}

// ClearCookie clears a cookie by setting it to expire
func (c *Ctx) ClearCookie(name string) *Ctx {
	cookie := &http.Cookie{
		Name:     name,
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		HttpOnly: true,
		Secure:   c.IsSecure(),
		SameSite: http.SameSiteLaxMode,
	}
	return c.SetCookie(cookie)
}

// NoContent sends a 204 No Content response
func (c *Ctx) NoContent() error {
	c.Response.WriteHeader(http.StatusNoContent)
	return nil
}

func (c *Ctx) End() error {
	c.Response.WriteHeader(c.statusCode)
	return nil
}

// Status sets the response status code (stored until response is sent)
func (c *Ctx) Status(code int) *Ctx {
	c.statusCode = code
	return c
}

// Created sends a 201 Created response with optional data
func (c *Ctx) Created(data any) error {
	c.statusCode = http.StatusCreated
	if data != nil {
		return c.JSON(data)
	}
	return c.End()
}

// Accepted sends a 202 Accepted response with optional data
func (c *Ctx) Accepted(data any) error {
	c.statusCode = http.StatusAccepted
	if data != nil {
		return c.JSON(data)
	}
	return c.End()
}

// JSON sends a JSON response
func (c *Ctx) JSON(data any) error {
	c.Set("Content-Type", "application/json; charset=utf-8")
	c.Response.WriteHeader(c.statusCode)
	return json.NewEncoder(c.Response).Encode(data)
}

// XML sends an XML response
func (c *Ctx) XML(data any) error {
	c.Set("Content-Type", "application/xml; charset=utf-8")
	c.Response.WriteHeader(c.statusCode)
	_, err := c.Response.Write([]byte(fmt.Sprintf("%v", data)))
	return err
}

// SendString sends a plain text response
func (c *Ctx) SendString(text string) error {
	c.Set("Content-Type", "text/plain; charset=utf-8")
	c.Response.WriteHeader(c.statusCode)
	_, err := c.Response.Write([]byte(text))
	return err
}

func (c *Ctx) HTML(data []byte) error {
	c.Set("Content-Type", "text/html; charset=utf-8")
	c.Response.WriteHeader(c.statusCode)
	_, err := c.Response.Write(data)
	return err
}

// Stream sends a streaming response with a custom writer function
func (c *Ctx) Stream(callback func(w io.Writer) error) error {
	c.Response.WriteHeader(c.statusCode)
	return callback(c.Response)
}

// SSE sends a Server-Sent Event
func (c *Ctx) SSE(event, data string) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")

	if event != "" {
		if _, err := fmt.Fprintf(c.Response, "event: %s\n", event); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(c.Response, "data: %s\n\n", data); err != nil {
		return err
	}

	if flusher, ok := c.Response.(http.Flusher); ok {
		flusher.Flush()
	}

	return nil
}

func (c *Ctx) File(file string) error {
	return c.SendFile(file, false)
}

// SendFile sends a file as response with optional download (Content-Disposition: attachment)
func (c *Ctx) SendFile(file string, download bool) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return err
	}

	// Set Content-Disposition header if download is true
	if download {
		c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", stat.Name()))
	}

	// Note: ServeContent handles its own status code
	http.ServeContent(c.Response, c.Request, file, stat.ModTime(), f)
	return nil
}

// Download sends a file with Content-Disposition: attachment
func (c *Ctx) Download(file string, filename ...string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return err
	}

	// Use custom filename if provided, otherwise use the file's name
	name := stat.Name()
	if len(filename) > 0 && filename[0] != "" {
		name = filename[0]
	}

	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", name))
	http.ServeContent(c.Response, c.Request, name, stat.ModTime(), f)
	return nil
}

func (c *Ctx) Redirect(status int, url string) error {
	http.Redirect(c.Response, c.Request, url, status)
	return nil
}

// ParseMultipartForm parses a multipart form with the given max memory
func (c *Ctx) ParseMultipartForm(maxMemory int64) error {
	return c.Request.ParseMultipartForm(maxMemory)
}

// MultipartForm returns the parsed multipart form
func (c *Ctx) MultipartForm() (*multipart.Form, error) {
	if c.Request.MultipartForm == nil {
		if err := c.ParseMultipartForm(32 << 20); err != nil { // 32 MB default
			return nil, err
		}
	}
	return c.Request.MultipartForm, nil
}

// Bind parses request data into the provided struct based on Content-Type
// Supports JSON, form data, and query parameters
func (c *Ctx) Bind(out any) error {
	contentType := strings.ToLower(c.ContentType())

	switch {
	case strings.HasPrefix(contentType, "application/json"):
		return c.ParseBody(out)
	case strings.HasPrefix(contentType, "application/x-www-form-urlencoded"),
		strings.HasPrefix(contentType, "multipart/form-data"):
		if err := c.Request.ParseForm(); err != nil {
			return errors.BadRequest("Invalid form data", err)
		}
		// Note: This is a basic implementation
		// For production, consider using a struct tag-based form decoder library
		return errors.New("Form binding not fully implemented - use ParseBody for JSON")
	default:
		// Try JSON as fallback
		return c.ParseBody(out)
	}
}

// IsSecure checks if the request is using HTTPS
func (c *Ctx) IsSecure() bool {
	return c.Request.TLS != nil || c.Get("X-Forwarded-Proto") == "https"
}

// AcceptsJSON checks if the client accepts JSON responses
func (c *Ctx) AcceptsJSON() bool {
	accept := strings.ToLower(c.Get("Accept"))
	return strings.Contains(accept, "application/json") || strings.Contains(accept, "*/*")
}

// AcceptsHTML checks if the client accepts HTML responses
func (c *Ctx) AcceptsHTML() bool {
	accept := strings.ToLower(c.Get("Accept"))
	return strings.Contains(accept, "text/html") || strings.Contains(accept, "*/*")
}

// Accepts checks if the client accepts a specific content type
func (c *Ctx) Accepts(contentType string) bool {
	accept := strings.ToLower(c.Get("Accept"))
	contentType = strings.ToLower(contentType)
	return strings.Contains(accept, contentType) || strings.Contains(accept, "*/*")
}

// IsSuccess checks if the status code is in the 2xx range
func (c *Ctx) IsSuccess() bool {
	return c.statusCode >= 200 && c.statusCode < 300
}

// IsClientError checks if the status code is in the 4xx range
func (c *Ctx) IsClientError() bool {
	return c.statusCode >= 400 && c.statusCode < 500
}

// IsServerError checks if the status code is in the 5xx range
func (c *Ctx) IsServerError() bool {
	return c.statusCode >= 500 && c.statusCode < 600
}

// GetRequestID gets the request ID from X-Request-ID header
func (c *Ctx) GetRequestID() string {
	return c.Get("X-Request-ID")
}

// SetRequestID sets the X-Request-ID header
func (c *Ctx) SetRequestID(id string) *Ctx {
	return c.Set("X-Request-ID", id)
}
