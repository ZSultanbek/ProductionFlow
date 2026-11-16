package http

import (
	"net/http"
	"net/url"
)

// CORS says which web origins may call this API from a browser.
//
// It only matters when the web app and the API are on different origins: in
// production (Cloudflare Pages → Render) and during `flutter run -d chrome`,
// which serves the app from a random localhost port. Behind the compose nginx
// everything is one origin and the browser never asks. The Android app is not
// a browser and is unaffected either way.
type CORS struct {
	// AllowedOrigins are matched exactly: scheme, host and port, no trailing
	// slash. Example: "https://productionflow.pages.dev".
	AllowedOrigins []string
	// AllowLocalhost also admits http://localhost and http://127.0.0.1 on any
	// port. For development only — never set it in production.
	AllowLocalhost bool
}

func (c CORS) allows(origin string) bool {
	for _, allowed := range c.AllowedOrigins {
		if origin == allowed {
			return true
		}
	}
	if !c.AllowLocalhost {
		return false
	}
	u, err := url.Parse(origin)
	if err != nil || u.Scheme != "http" {
		return false
	}
	// Hostname() strips the port, so "localhost.evil.com" cannot sneak through
	// a prefix check: it is simply a different hostname.
	return u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1"
}

// cors adds the response headers a browser needs before it will hand a
// cross-origin response to the web app, and answers preflight requests.
//
// No cookies are involved — the token travels in the Authorization header — so
// there is no Allow-Credentials here and no CSRF surface to worry about.
func cors(cfg CORS) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// The response differs by Origin, so caches must key on it — even
			// when this particular request has none.
			w.Header().Add("Vary", "Origin")

			if origin == "" || !cfg.allows(origin) {
				// Not a browser, or an origin we do not serve: no CORS headers,
				// and the browser blocks the page from reading the response.
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("Access-Control-Allow-Origin", origin)

			// A preflight is the browser asking permission before the real
			// request. It carries no Authorization header, so it must be
			// answered here, before requireAuth can turn it away.
			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
				w.Header().Set("Access-Control-Max-Age", "600")
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
