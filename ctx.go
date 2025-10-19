package grouter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/azizndao/grouter/errors"
)

// Ctx provides easy access to request data and response helpers
type Ctx struct {
	Request    *http.Request
	Response   http.ResponseWriter
	statusCode int
	body       []byte // Cached request body
	bodyRead   bool   // Track if body has been read
}

// NewCtx creates a new Context from request and response
func NewCtx(w http.ResponseWriter, r *http.Request) *Ctx {
	return &Ctx{
		Request:    r,
		Response:   w,
		statusCode: http.StatusOK, // Default to 200
	}
}

func (c *Ctx) Context() context.Context {
	return c.Request.Context()
}

// SetValue sets a custom value in the request context
func (c *Ctx) SetValue(key string, value any) *http.Request {
	ctx := context.WithValue(c.Context(), key, value)
	return c.Request.WithContext(ctx)
}

func (c *Ctx) GetValue(key string) any {
	return c.Context().Value(key)
}

// ParseBody parses the request body into the given struct
func (c *Ctx) ParseBody(out any) error {
	body, err := c.Body()
	if err != nil {
		return err
	}

	return json.Unmarshal(body, out)
}

// ValidateBody parses and validates the request body in one call
func (c *Ctx) ValidateBody(out any) error {
	if err := c.ParseBody(out); err != nil {
		return errors.BadRequest("Invalid request body", err)
	}
	validator := c.getValidator()
	if validator == nil {
		return errors.InternalServerError("Validator not configured", nil)
	}

	// Get locale from Accept-Language header
	locale := c.getLocaleFromHeader()
	return validator.Validate(out, locale)
}

type Validator interface {
	Validate(out any, locale ...string) error
}

// getValidator retrieves the validator from the request context
func (c *Ctx) getValidator() Validator {
	if v := c.GetValue("validator"); v != nil {
		if validator, ok := v.(Validator); ok {
			return validator
		}
	}
	return nil
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

func (c *Ctx) PathValue(key string) string {
	return c.Request.PathValue(key)
}

// Query gets a query parameter by key
func (c *Ctx) Query(key string) string {
	return c.Request.URL.Query().Get(key)
}

// QueryInt gets a query parameter as int
func (c *Ctx) QueryInt(key string) (int, error) {
	value := c.Query(key)
	if value == "" {
		return 0, nil
	}
	return strconv.Atoi(value)
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
		return 0, nil
	}
	return strconv.ParseFloat(value, 64)
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

// ContentType gets the Content-Type header
func (c *Ctx) ContentType() string {
	return c.Get("Content-Type")
}

// IP returns the client's IP address
// When behind a proxy, it extracts the first IP from X-Forwarded-For header
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
	return c.Request.RemoteAddr
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

// Status sets the response status code (stored until response is sent)
func (c *Ctx) Status(code int) *Ctx {
	c.statusCode = code
	return c
}

// JSON sends a JSON response
func (c *Ctx) JSON(data any) error {
	c.Set("Content-Type", "application/json")
	c.Response.WriteHeader(c.statusCode)
	return json.NewEncoder(c.Response).Encode(data)
}

// SendString sends a plain text response
func (c *Ctx) SendString(text string) error {
	c.Set("Content-Type", "text/plain")
	c.Response.WriteHeader(c.statusCode)
	_, err := c.Response.Write([]byte(text))
	return err
}

func (c *Ctx) HTML(data []byte) error {
	c.Set("Content-Type", "text/html")
	c.Response.WriteHeader(c.statusCode)
	_, err := c.Response.Write(data)
	return err
}

func (c *Ctx) File(file string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return err
	}

	http.ServeContent(c.Response, c.Request, file, stat.ModTime(), f)
	return nil
}

func (c *Ctx) Redirect(status int, url string) error {
	http.Redirect(c.Response, c.Request, url, status)
	return nil
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
