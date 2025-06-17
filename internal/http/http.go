package http

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"vimai/ads-transcode/internal/config"
)

type HttpManagerCtx struct {
	logger zerolog.Logger
	config *config.Server
	router *chi.Mux
	http   *http.Server
}

func New(config *config.Server) *HttpManagerCtx {
	logger := log.With().Str("module", "http").Logger()

	router := chi.NewRouter()
	router.Use(middleware.RequestID) // Create a request ID for each request
	if config.Proxy {
		router.Use(middleware.RealIP)
	}
	router.Use(middleware.RequestLogger(&logformatter{logger}))
	router.Use(middleware.Recoverer) // Recover from panics without crashing server
	if config.CORS {
		router.Use(cors.Handler(cors.Options{
			AllowOriginFunc: func(r *http.Request, origin string) bool {
				return true
			},
			AllowedMethods:   []string{"GET", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Content-Type", "Content-Length", "Accept-Encoding"},
			ExposedHeaders:   []string{"Link"},
			AllowCredentials: true,
			MaxAge:           300, // Maximum value not ignored by any of major browsers
		}))
	}

	// serve static files
	if config.Static != "" {
		fs := http.FileServer(http.Dir(config.Static))
		router.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			if _, err := os.Stat(config.Static + r.RequestURI); os.IsNotExist(err) {
				http.StripPrefix(r.RequestURI, fs).ServeHTTP(w, r)
			} else {
				fs.ServeHTTP(w, r)
			}
		})
	}

	router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		//nolint
		_, _ = w.Write([]byte("404"))
	})

	return &HttpManagerCtx{
		logger: logger,
		config: config,
		router: router,
		http: &http.Server{
			Addr:    config.Bind,
			Handler: router,
		},
	}
}

func (s *HttpManagerCtx) Start() {
	if s.config.Cert != "" && s.config.Key != "" {
		s.logger.Warn().Msg("TLS support is provided for convenience, but you should never use it in production. Use a reverse proxy (apache nginx caddy) instead!")
		go func() {
			if err := s.http.ListenAndServeTLS(s.config.Cert, s.config.Key); err != http.ErrServerClosed {
				s.logger.Panic().Err(err).Msg("unable to start https server")
			}
		}()
		s.logger.Info().Msgf("https listening on %s", s.http.Addr)
	} else {
		go func() {
			if err := s.http.ListenAndServe(); err != http.ErrServerClosed {
				s.logger.Panic().Err(err).Msg("unable to start http server")
			}
		}()
		s.logger.Info().Msgf("http listening on %s", s.http.Addr)
	}
}

func (s *HttpManagerCtx) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return s.http.Shutdown(ctx)
}

func (s *HttpManagerCtx) WithProfiler() {
	s.router.Mount("/debug", middleware.Profiler())
}

func (s *HttpManagerCtx) Mount(fn func(r *chi.Mux)) {
	fn(s.router)
}
