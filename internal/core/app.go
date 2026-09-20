package core

import (
	"context"
	"net/http"

	"github.com/Arondy/url-shortener/internal/args"
	"github.com/Arondy/url-shortener/internal/config"
	"github.com/Arondy/url-shortener/internal/core/repository/in_memory"
	"github.com/Arondy/url-shortener/internal/core/repository/postgres"
	"github.com/Arondy/url-shortener/internal/core/service/url_shortener"
	core_http "github.com/Arondy/url-shortener/internal/core/transport/http"
	"github.com/Arondy/url-shortener/internal/core/transport/http/handlers/shorten"
	"github.com/Arondy/url-shortener/internal/core/transport/http/middleware"
	"go.uber.org/zap"
)

type RepoWithClear interface {
	url_shortener.URLShortenerRepo
	ClearExpired(ctx context.Context)
}

func Run(ctx context.Context, cfg *config.Config, args args.Args, logger *zap.SugaredLogger) error {
	var shortenerRepo RepoWithClear

	if args.InMemory {
		shortenerRepo = in_memory.NewURLShortenerRepository(cfg.DB.URLsTTL, cfg.DB.ClearFrequency)
	} else {
		db, err := postgres.NewDB(ctx, cfg.DB, logger)
		if err != nil {
			return err
		}
		defer db.Close()

		shortenerRepo = postgres.NewURLShortenerRepository(db)
	}

	go shortenerRepo.ClearExpired(ctx)

	shortenerService := url_shortener.NewService(shortenerRepo)
	shortenHander := shorten.NewShortenHandler(cfg.DomainURL, shortenerService)

	var router http.Handler = core_http.NewRouter(shortenHander)
	router = middleware.WrapInMiddleware(router, logger)
	server := core_http.NewServer(router, cfg.HTTPServer, logger.Named("Server"))

	if err := server.Run(ctx); err != nil {
		return err
	}
	return nil
}
