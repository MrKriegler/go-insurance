package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/MrKriegler/go-insurance/internal/core"
	"github.com/MrKriegler/go-insurance/pkg/problem"
)

func writeError(ctx context.Context, log *slog.Logger, w http.ResponseWriter, err error, detail string) {
	switch {
	case errors.Is(err, core.ErrNotFound):
		log.WarnContext(ctx, "resource not found", "err", err)
		problem.Write(w, http.StatusNotFound, "Not Found", detail)

	case errors.Is(err, core.ErrValidation):
		log.WarnContext(ctx, "validation failed", "err", err)
		problem.Write(w, http.StatusBadRequest, "Validation Error", detail)

	case errors.Is(err, core.ErrConflict):
		log.WarnContext(ctx, "resource conflict", "err", err)
		problem.Write(w, http.StatusConflict, "Conflict", detail)

	case errors.Is(err, core.ErrUnauthorized):
		log.WarnContext(ctx, "unauthorized request", "err", err)
		problem.Write(w, http.StatusUnauthorized, "Unauthorized", detail)

	case errors.Is(err, core.ErrForbidden):
		log.WarnContext(ctx, "forbidden operation", "err", err)
		problem.Write(w, http.StatusForbidden, "Forbidden", detail)

	case errors.Is(err, context.DeadlineExceeded):
		log.ErrorContext(ctx, "operation timeout", "err", err)
		problem.Write(w, http.StatusGatewayTimeout, "Timeout", "Operation took too long.")

	default:
		log.ErrorContext(ctx, "internal server error", "err", err)
		problem.Write(w, http.StatusInternalServerError, "Internal Server Error", detail)
	}
}
