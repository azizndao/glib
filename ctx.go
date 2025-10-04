package grouter

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
)

// Ctx provides easy access to request data and response helpers
type Ctx struct {
	Request  *http.Request
	Response http.ResponseWriter
}

// NewCtx creates a new Context from request and response
func NewCtx(w http.ResponseWriter, r *http.Request) *Ctx {
	return &Ctx{
		Request:  r,
		Response: w,
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

// BodyParser parses the request body into the given struct
func (c *Ctx) BodyParser(out any) error {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return err
	}
	defer c.Request.Body.Close()

	return json.Unmarshal(body, out)
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

func (c *Ctx) Get(key string) string {
	return c.Request.Header.Get(key)
}

// IP returns the client's IP address
func (c *Ctx) IP() string {
	if ip := c.Request.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	if ip := c.Request.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	return c.Request.RemoteAddr
}

// Method returns the request method
func (c *Ctx) Method() string {
	return c.Request.Method
}

// Path returns the request path
func (c *Ctx) Path() string {
	return c.Request.URL.Path
}

// Response helpers

// Status sets the response status code
func (c *Ctx) Status(code int) *Ctx {
	c.Response.WriteHeader(code)
	return c
}

// Set sets a response header
func (c *Ctx) Set(key, value string) *Ctx {
	c.Response.Header().Set(key, value)
	return c
}

// JSON sends a JSON response
func (c *Ctx) JSON(status int, data any) error {
	c.Set("Content-Type", "application/json")
	c.Status(status)
	return json.NewEncoder(c.Response).Encode(data)
}

// SendString sends a plain text response
func (c *Ctx) SendString(status int, text string) error {
	c.Set("Content-Type", "text/plain")
	c.Status(status)
	_, err := c.Response.Write([]byte(text))
	return err
}

func (c *Ctx) HTML(status int, data []byte) error {
	c.Set("Content-Type", "text/html")
	_, err := c.Response.Write(data)
	c.Status(status)
	return err
}

func (c *Ctx) File(status int, file string) error {
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
	c.Status(status)
	return nil
}

func (c *Ctx) Redirect(status int, url string) error {
	http.Redirect(c.Response, c.Request, url, status)
	c.Status(status)
	return nil
}
