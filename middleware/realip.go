package middleware

import (
	"net"
	"net/http"
	"strings"

	"github.com/azizndao/grouter"
	"github.com/azizndao/grouter/util"
)

// RealIPConfig holds configuration for the RealIP middleware
type RealIPConfig struct {
	// TrustedProxies is a list of CIDR ranges for trusted proxies
	// Only these proxies are allowed to set X-Forwarded-For and X-Real-IP headers
	// If empty, all proxies are trusted (not recommended for production)
	TrustedProxies []string

	// Headers is the priority list of headers to check for the real IP
	// Default: ["X-Forwarded-For", "X-Real-IP", "X-Appengine-Remote-Addr"]
	Headers []string

	// trustedNets is the parsed list of trusted proxy networks
	trustedNets []*net.IPNet
}

// DefaultRealIPConfig returns default configuration for RealIP middleware
func DefaultRealIPConfig() RealIPConfig {
	return RealIPConfig{
		TrustedProxies: []string{
			"10.0.0.0/8",     // Private network
			"172.16.0.0/12",  // Private network
			"192.168.0.0/16", // Private network
			"127.0.0.0/8",    // Loopback
			"::1/128",        // IPv6 loopback
			"fc00::/7",       // IPv6 unique local addr
			"fe80::/10",      // IPv6 link-local addr
		},
		Headers: []string{
			"CF-Connecting-IP",        // Cloudflare
			"True-Client-IP",          // Cloudflare Enterprise / Akamai
			"X-Real-IP",               // Nginx
			"X-Forwarded-For",         // Standard
			"X-Appengine-Remote-Addr", // Google App Engine
		},
	}
}

// RealIP creates a middleware that sets the request's RemoteAddr to the real client IP.
// This middleware should be used when your application is behind a reverse proxy or load balancer.
//
// Security Note: Only use this middleware if you trust the proxy servers setting these headers.
// Configure TrustedProxies to limit which proxies can set the client IP.
//
// Example usage:
//
//	// Use defaults (trusts common private networks)
//	router.Use(middleware.RealIP())
//
//	// Custom configuration
//	router.Use(middleware.RealIP(middleware.RealIPConfig{
//	    TrustedProxies: []string{
//	        "10.0.0.0/8",  // Only trust this network
//	    },
//	    Headers: []string{"CF-Connecting-IP", "X-Forwarded-For"},
//	}))
func RealIP(config ...RealIPConfig) grouter.Middleware {
	cfg := util.FirstOrDefault(config, DefaultRealIPConfig)
	// Parse trusted proxy CIDRs
	if len(cfg.TrustedProxies) > 0 {
		cfg.trustedNets = make([]*net.IPNet, 0, len(cfg.TrustedProxies))
		for _, cidr := range cfg.TrustedProxies {
			_, network, err := net.ParseCIDR(cidr)
			if err != nil {
				// Skip invalid CIDR
				continue
			}
			cfg.trustedNets = append(cfg.trustedNets, network)
		}
	}

	// Set default headers if not provided
	if len(cfg.Headers) == 0 {
		cfg.Headers = DefaultRealIPConfig().Headers
	}

	return func(next grouter.Handler) grouter.Handler {
		return func(c *grouter.Ctx) error {
			// Get the remote address
			remoteAddr := c.Request.RemoteAddr
			host, _, err := net.SplitHostPort(remoteAddr)
			if err != nil {
				host = remoteAddr
			}

			// Check if the remote address is from a trusted proxy
			if len(cfg.trustedNets) > 0 && !isTrustedProxy(host, cfg.trustedNets) {
				// Not a trusted proxy, use RemoteAddr as-is
				return next(c)
			}

			// Try to get real IP from configured headers
			realIP := getRealIPFromHeaders(c.Request.Header, cfg.Headers)
			if realIP != "" {
				// Update the request's RemoteAddr
				c.Request.RemoteAddr = realIP
			}

			return next(c)
		}
	}
}

// isTrustedProxy checks if the given IP is in any of the trusted networks
func isTrustedProxy(ipStr string, trustedNets []*net.IPNet) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	for _, network := range trustedNets {
		if network.Contains(ip) {
			return true
		}
	}

	return false
}

// getRealIPFromHeaders extracts the real IP from HTTP headers in priority order
func getRealIPFromHeaders(h http.Header, headers []string) string {
	for _, header := range headers {
		if value := h.Get(header); value != "" {
			// X-Forwarded-For can contain multiple IPs: "client, proxy1, proxy2"
			// Extract the first (client) IP
			if idx := strings.Index(value, ","); idx != -1 {
				value = value[:idx]
			}
			value = strings.TrimSpace(value)

			// Validate IP
			if ip := net.ParseIP(value); ip != nil {
				return value
			}
		}
	}

	return ""
}
