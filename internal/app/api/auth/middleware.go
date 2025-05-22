package auth

import (
	"net/http"
	"strings"
	"sync"

	jwtAuth "github.com/422UR4H/HxH_RPG_System/pkg/auth"
	"github.com/danielgtaylor/huma/v2"
)

type contextKey string

const UserIDKey contextKey = "userID"

func AuthMiddlewareProvider(sessions *sync.Map) func(ctx huma.Context, next func(huma.Context)) {
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

		_, exists := sessions.Load(claims.UserID)
		if !exists {
			ctx.SetStatus(http.StatusUnauthorized)
			WriteAuthError(ctx.BodyWriter(), "Authentication failed", "Access Denied")
			return
		}

		ctx = huma.WithValue(ctx, UserIDKey, claims.UserID)
		next(ctx)
	}
}
