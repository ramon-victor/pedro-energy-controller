package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/pedrohdcosta/projetoPortifolio/Portifolio_back/internal/auth"
	"github.com/pedrohdcosta/projetoPortifolio/Portifolio_back/internal/db"
)

func main() {
	_ = godotenv.Load(".env")
	ctx := context.Background()
	pool, err := db.NewPool(ctx)
	if err != nil {
		log.Fatal(err)
	}

	r := setupRouter(ctx, pool)
	port := getPort()

	log.Printf("Starting server on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

// setupRouter configures and returns the Gin router with all routes and middleware.
func setupRouter(ctx context.Context, pool *pgxpool.Pool) *gin.Engine {
	r := gin.Default()
	r.Use(gin.Logger(), gin.Recovery())

	// Health check endpoint
	r.GET("/health", healthCheckHandler)

	// Database schema initialization
	ensureSchema(ctx, pool)

	// API routes
	auth.RegisterRoutes(r, wrap(pool))

	// Configure static file serving for frontend SPA
	configureStaticFiles(r)

	return r
}

// healthCheckHandler returns a simple health check response.
func healthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// configureStaticFiles sets up static file serving and SPA routing.
// In production/Docker deployments, the frontend is built and served from ./static
func configureStaticFiles(r *gin.Engine) {
	const staticDir = "./static"

	stat, err := os.Stat(staticDir)
	if err != nil {
		log.Printf("Static directory not found (%v), serving API only", err)
		r.NoRoute(apiOnlyNoRouteHandler)
		return
	}

	if !stat.IsDir() {
		log.Printf("Static path exists but is not a directory, serving API only")
		r.NoRoute(apiOnlyNoRouteHandler)
		return
	}

	log.Printf("Serving static files from %s", staticDir)
	r.Static("/assets", staticDir+"/assets")
	r.StaticFile("/", staticDir+"/index.html")
	r.StaticFile("/favicon.ico", staticDir+"/favicon.ico")
	r.NoRoute(spaNoRouteHandler(staticDir))
}

// spaNoRouteHandler returns a handler for SPA client-side routing.
// API routes get JSON 404, all other routes serve index.html.
func spaNoRouteHandler(staticDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// API routes should return JSON 404, not the SPA
		if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
			log.Printf("API route not found: %s %s", c.Request.Method, c.Request.URL.Path)
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		// Serve index.html for all other routes (SPA client-side routing)
		c.File(staticDir + "/index.html")
	}
}

// apiOnlyNoRouteHandler returns JSON 404 for all routes when no static files are served.
func apiOnlyNoRouteHandler(c *gin.Context) {
	log.Printf("Route not found: %s %s", c.Request.Method, c.Request.URL.Path)
	c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
}

// getPort determines the port to listen on from environment variables.
// Priority: PORT (Azure) > APP_PORT > default "8080"
func getPort() string {
	if port := os.Getenv("PORT"); port != "" {
		return port
	}
	if port := os.Getenv("APP_PORT"); port != "" {
		return port
	}
	return "8080"
}

type pgxWrap struct{ *pgxpool.Pool }

func wrap(p *pgxpool.Pool) pgxWrap { return pgxWrap{p} }
func (w pgxWrap) Exec(ctx context.Context, sql string, args ...any) error {
	_, err := w.Pool.Exec(ctx, sql, args...)
	return err
}
func (w pgxWrap) QueryRow(ctx context.Context, sql string, args ...any) interface{ Scan(dest ...any) error } {
	return w.Pool.QueryRow(ctx, sql, args...)
}

func ensureSchema(ctx context.Context, p *pgxpool.Pool) {
	_, _ = p.Exec(ctx, `create table if not exists app_user(
	id bigserial primary key,
	name text not null,
	email text unique not null,
	password_hash text not null,
	created_at timestamptz default now()
	)`)
}
