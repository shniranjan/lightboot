package core

import "net/http"

// CSPMiddleware adds Content Security Policy headers to all HTTP responses.
// This restricts the Web UI to same origin and prevents XSS attacks.
func CSPMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Strict CSP for the SPA web UI:
        // - default-src 'self': only load resources from our domain
        // - script-src 'self': no inline scripts, no external scripts
        // - style-src 'self' 'unsafe-inline': allow inline styles (needed for Vue transitions)
        // - connect-src 'self': allow API calls and SSE to our domain
        // - img-src 'self' data: blob:: allow embedded images and blobs
        // - font-src 'self': only load fonts from our domain
        // - frame-src 'none': prevent clickjacking via frames
        // - object-src 'none': prevent plugins (Flash, etc.)
        // - base-uri 'self': prevent base tag injection
        // - form-action 'self': only allow forms to submit to our domain
        csp := "default-src 'self'; " +
            "script-src 'self'; " +
            "style-src 'self' 'unsafe-inline'; " +
            "connect-src 'self'; " +
            "img-src 'self' data: blob:; " +
            "font-src 'self'; " +
            "frame-src 'none'; " +
            "object-src 'none'; " +
            "base-uri 'self'; " +
            "form-action 'self'"

        w.Header().Set("Content-Security-Policy", csp)

        // Additional security headers
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        w.Header().Set("Referrer-Policy", "no-referrer")
        w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

        // HTTPS enforcement (when behind a TLS terminator)
        // w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

        next.ServeHTTP(w, r)
    })
}