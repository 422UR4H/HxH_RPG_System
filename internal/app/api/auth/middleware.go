package auth

import (
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/422UR4H/HxH_RPG_System/internal/application/session"
	jwtAuth "github.com/422UR4H/HxH_RPG_System/pkg/auth"
	"github.com/danielgtaylor/huma/v2"
)

type contextKey string

const UserIDKey contextKey = "userID"

func AuthMiddlewareProvider(
	sessions *sync.Map, sessionRepo session.IRepository,
) func(ctx huma.Context, next func(huma.Context)) {

	return func(ctx huma.Context, next func(huma.Context)) {
		authHeader := ctx.Header("Authorization")
		if authHeader == "" {
			ctx.SetStatus(http.StatusUnauthorized)
			WriteAuthError(ctx.BodyWriter(), "Autenticação necessária.", "Entre na sua conta para continuar.")
			return
		}

		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			ctx.SetStatus(http.StatusUnauthorized)
			WriteAuthError(ctx.BodyWriter(), "Falha na autenticação.", "Entre de novo na sua conta.")
			return
		}
		tokenStr := authHeader[len(bearerPrefix):]

		claims, err := jwtAuth.ValidateToken(tokenStr)
		if err != nil {
			ctx.SetStatus(http.StatusUnauthorized)
			WriteAuthError(ctx.BodyWriter(), "Falha na autenticação.", "Entre de novo na sua conta.")
			return
		}

		// find token in memory cache first. A miss (the process just restarted) falls to the
		// same database check a cache mismatch does: every login is a session row, and a token
		// from another tab or device is as valid as the user's latest login — comparing it to
		// the latest alone logged everyone else out on each restart.
		token, exists := sessions.Load(claims.UserID)
		if !exists || token != tokenStr {
			// check directly with the bank as a last resort
			valid, err := sessionRepo.ValidateSession(ctx.Context(), claims.UserID, tokenStr)
			errMsg := "Entre de novo na sua conta."
			if err != nil {
				log.Printf("[ERROR] session validation: %v", err)
			}
			if !valid {
				ctx.SetStatus(http.StatusUnauthorized)
				WriteAuthError(ctx.BodyWriter(), "Falha na autenticação.", errMsg)
				return
			}
			// update cache
			sessions.Store(claims.UserID, tokenStr)
		}

		ctx = huma.WithValue(ctx, UserIDKey, claims.UserID)
		next(ctx)
	}
}
