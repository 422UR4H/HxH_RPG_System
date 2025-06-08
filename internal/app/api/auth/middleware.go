package auth

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/session"
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
			WriteAuthError(ctx.BodyWriter(), "Authentication required", "Missing authorization token")
			return
		}

		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			ctx.SetStatus(http.StatusUnauthorized)
			WriteAuthError(ctx.BodyWriter(), "Authentication failed", "Invalid token format, expected 'Bearer <token>'")
			return
		}
		tokenStr := authHeader[len(bearerPrefix):]

		claims, err := jwtAuth.ValidateToken(tokenStr)
		if err != nil {
			ctx.SetStatus(http.StatusUnauthorized)
			WriteAuthError(ctx.BodyWriter(), "Authentication failed", "Invalid authorization token")
			return
		}

		// find token in memory cache first
		token, exists := sessions.Load(claims.UserID)
		if !exists {
			// if not found, verify in sessions db table
			dbToken, err := sessionRepo.GetSessionTokenByUserUUID(ctx.Context(), claims.UserID)
			if err == nil && dbToken == tokenStr {
				// if found, add session into memory cache
				sessions.Store(claims.UserID, tokenStr)
				exists = true
				token = tokenStr
			}
		}
		if !exists {
			ctx.SetStatus(http.StatusUnauthorized)
			WriteAuthError(ctx.BodyWriter(), "Authentication failed", "Access Denied")
			return
		}
		if token != tokenStr {
			// check directly with the bank as a last resort
			valid, err := sessionRepo.ValidateSession(ctx.Context(), claims.UserID, tokenStr)
			errMsg := "Access Denied!"
			if err != nil {
				errMsg = fmt.Sprintf("Access Denied: %s", err.Error())
			}
			if !valid {
				ctx.SetStatus(http.StatusUnauthorized)
				WriteAuthError(ctx.BodyWriter(), "Authentication failed", errMsg)
				return
			}
			// update cache
			sessions.Store(claims.UserID, tokenStr)
		}

		ctx = huma.WithValue(ctx, UserIDKey, claims.UserID)
		next(ctx)
	}
}
