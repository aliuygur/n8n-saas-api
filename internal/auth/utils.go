package auth

import (
	"context"
	"net/http"

	"encore.dev/rlog"
)

func AuthOnly(ctx context.Context, w http.ResponseWriter, r *http.Request) bool {
	_, ok := GetUserID()

	if !ok {
		rlog.Info("unauthenticated access attempt")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return false
	}

	return true
}
