package middleware

import (
	"net"
	"net/http"
	"strings"
)

// RealIPMiddleware resolves the real client IP when the server runs behind a
// trusted reverse proxy such as Cloudflare.
//
// When trustProxy is true the request RemoteAddr is replaced with the IP from
// CF-Connecting-IP / X-Forwarded-For / X-Real-IP. This must run BEFORE the
// rate limiter and logging middleware so they see the real client.
//
// When trustProxy is false (server is directly exposed) proxy headers are
// ignored because they can be spoofed by any client, and only the socket
// peer address is used.
func RealIPMiddleware(trustProxy bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.RemoteAddr = resolveClientIP(r, trustProxy)
			next.ServeHTTP(w, r)
		})
	}
}

// ClientIP returns the best-effort client IP, honouring Cloudflare's
// CF-Connecting-IP header first. Only safe to call when the server is behind
// a trusted proxy and header spoofing is not a concern.
func ClientIP(r *http.Request) string {
	return resolveClientIP(r, true)
}

func resolveClientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if ip := firstForwardedIP(r.Header.Get("CF-Connecting-IP")); ip != "" {
			return ip
		}
		if ip := firstForwardedIP(r.Header.Get("X-Forwarded-For")); ip != "" {
			return ip
		}
		if ip := firstForwardedIP(r.Header.Get("X-Real-IP")); ip != "" {
			return ip
		}
	}
	return socketIP(r.RemoteAddr)
}

// firstForwardedIP returns the first entry of a possibly comma-separated
// forwarded header value, trimmed of whitespace.
func firstForwardedIP(value string) string {
	if value == "" {
		return ""
	}
	if i := strings.IndexByte(value, ','); i != -1 {
		value = value[:i]
	}
	return strings.TrimSpace(value)
}

// socketIP strips the port from a "host:port" remote address, returning the
// bare IP (IPv4 or IPv6).
func socketIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		// No port present; return as-is.
		return remoteAddr
	}
	return host
}
