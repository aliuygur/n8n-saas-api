package appctx

import (
	"log/slog"
	"net/http"
)

func Handler(h http.Handler, logger *slog.Logger) http.Handler {
	return LoggerMiddleware(logger)(h)
}
