package api

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

// RegisterStaticHandler registers the SPA static file handler on the given engine.
// sub should be an fs.FS rooted at the directory containing index.html.
func RegisterStaticHandler(r *gin.Engine, sub fs.FS) {
	fileServer := http.FileServer(http.FS(sub))

	r.NoRoute(func(c *gin.Context) {
		requestPath := strings.TrimPrefix(c.Request.URL.Path, "/")
		if requestPath == "" {
			requestPath = "index.html"
		}

		// Never serve SPA fallback for unknown API routes.
		if strings.HasPrefix(requestPath, "api/") {
			c.Status(http.StatusNotFound)
			return
		}

		// Try to serve exact file
		if f, err := sub.Open(requestPath); err == nil {
			f.Close()
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}

		// Missing static assets should be 404, not index.html.
		if ext := path.Ext(requestPath); ext != "" || strings.HasPrefix(requestPath, "_app/") {
			c.Status(http.StatusNotFound)
			return
		}

		// SPA fallback: serve index.html
		data, err := fs.ReadFile(sub, "index.html")
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.Header("Cache-Control", "no-store")
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.Status(http.StatusOK)
		_, _ = c.Writer.Write(data)
	})
}
