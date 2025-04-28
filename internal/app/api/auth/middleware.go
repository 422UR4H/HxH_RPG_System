package auth

import (
	"net/http"
	"sync"

	jwtAuth "github.com/422UR4H/HxH_RPG_System/pkg/auth"
	"github.com/danielgtaylor/huma/v2"
)

type contextKey string

const UserIDKey contextKey = "userID"

func AuthMiddlewareProvider(sessions *sync.Map) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		tokenStr := ctx.Header("Authorization")
		if tokenStr == "" {
			ctx.SetStatus(http.StatusUnauthorized)
			w := ctx.BodyWriter()
			w.Write([]byte(
				`{"error":"Authentication required","message":"Missing authorization token"}`,
			))
			return
		}

		claims, err := jwtAuth.ValidateToken(tokenStr)
		if err != nil {
			ctx.SetStatus(http.StatusUnauthorized)
			w := ctx.BodyWriter()
			w.Write([]byte(
				`{"error":"Authentication failed","message":"Invalid authorization token"}`,
			))
			return
		}

		_, exists := sessions.Load(claims.UserID)
		if !exists {
			ctx.SetStatus(http.StatusUnauthorized)
			w := ctx.BodyWriter()
			w.Write([]byte(
				`{"error":"Authentication failed","message":"Access Denied"}`,
			))
			return
		}

		ctx.AppendHeader(string(UserIDKey), claims.UserID.String())
		next(ctx)
	}
}
