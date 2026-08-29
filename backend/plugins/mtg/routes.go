package mtg

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hay-kot/httpkit/server"
	"github.com/sysadminsmedia/homebox/backend/pkgs/plugins"
)

// RegisterRoutes registers HTTP endpoints for the MTG plugin.
func (p *Plugin) RegisterRoutes(r chi.Router, env *plugins.PluginEnv) {
	r.Get("/image-proxy", p.handleImageProxy())
	r.Get("/search", p.handleSearch())
}

func (p *Plugin) handleSearch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		results, err := p.SearchProducts(r.Context(), q)
		if err != nil {
			_ = server.JSON(w, http.StatusInternalServerError, err.Error())
			return
		}
		if results == nil {
			results = []plugins.ProductSearchResult{}
		}
		_ = server.JSON(w, http.StatusOK, results)
	}
}

func (p *Plugin) handleImageProxy() http.HandlerFunc {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	return func(w http.ResponseWriter, r *http.Request) {
		rawURL := r.URL.Query().Get("url")
		if rawURL == "" {
			_ = server.JSON(w, http.StatusBadRequest, "url parameter is required")
			return
		}

		parsed, err := url.Parse(rawURL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			_ = server.JSON(w, http.StatusBadRequest, "invalid image url")
			return
		}

		host := strings.ToLower(parsed.Hostname())
		if host == "localhost" || strings.HasPrefix(host, "127.") || host == "::1" {
			_ = server.JSON(w, http.StatusForbidden, "invalid host")
			return
		}

		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, rawURL, nil)
		if err != nil {
			_ = server.JSON(w, http.StatusInternalServerError, "failed to build request")
			return
		}
		req.Header.Set("User-Agent", "Homebox/1.0 (+https://github.com/sysadminsmedia/homebox)")

		resp, err := client.Do(req)
		if err != nil {
			_ = server.JSON(w, http.StatusBadGateway, "failed to fetch remote image")
			return
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusOK {
			_ = server.JSON(w, resp.StatusCode, fmt.Sprintf("remote returned status %d", resp.StatusCode))
			return
		}

		cType := resp.Header.Get("Content-Type")
		if !strings.HasPrefix(cType, "image/") {
			cType = "image/jpeg"
		}

		w.Header().Set("Content-Type", cType)
		w.Header().Set("Cache-Control", "public, max-age=604800, immutable")
		w.Header().Set("Cross-Origin-Resource-Policy", "cross-origin")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		limitedReader := io.LimitReader(resp.Body, 10*1024*1024)
		_, _ = io.Copy(w, limitedReader)
	}
}
