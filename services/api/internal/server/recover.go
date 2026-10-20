package server

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
)

// recoverPanics turns a panic in a handler into a logged 500 so one bad
// request cannot take the process down. The panic value is logged by type only,
// because it can hold request data.
func recoverPanics(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
				panic(rec)
			}
			logger.Error("panic",
				"requestId", requestIDFrom(r.Context()),
				"panicType", fmt.Sprintf("%T", rec),
				"stack", string(debug.Stack()),
			)
			writeError(w, r, http.StatusInternalServerError, codeInternal, "Something went wrong.")
		}()
		next.ServeHTTP(w, r)
	})
}
