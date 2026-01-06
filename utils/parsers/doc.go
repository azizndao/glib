// Package parsers provides helper functions for parsing request parameters,
// environment variables, and common type conversions.
//
// These utilities are used by glib's code generation and are also available
// for users to simplify custom handler implementations.
//
// # Core Parsing Functions
//
// The Parse* functions convert string values to typed values with consistent
// error handling:
//
//	age, err := parsers.ParseInt("25", "age")
//	timeout, err := parsers.ParseDuration("30s", "timeout")
//	id, err := parsers.ParseUUID("123e4567-e89b-12d3-a456-426614174000", "id")
//
// # Request Helpers
//
// Extract values from HTTP requests:
//
//	id := r.PathValue("id")                    // Path parameters (Go 1.22+)
//	query := r.URL.Query().Get("q")            // Query parameters
//	tags := parsers.GetQuerySlice(r, "tags")     // Multiple query values
//	header := r.Header.Get("Authorization")    // Headers
//
// # Environment Variable Helpers
//
// Parse environment variables with defaults:
//
//	port, err := parsers.GetEnvInt("PORT", 8080)
//	timeout, err := parsers.GetEnvDuration("TIMEOUT", 30*time.Second)
//	dbURL := parsers.GetEnvOr("DATABASE_URL", "localhost:5432")
//
// # Body Parsing
//
// Type-safe JSON body parsing:
//
//	body, err := parsers.ParseJSONBody[CreateUserRequest](r)
//
// # Language Detection
//
// Detect user language from Accept-Language header:
//
//	lang := parsers.DetectLanguage(r.Header.Get("Accept-Language"), "en")
package parsers
