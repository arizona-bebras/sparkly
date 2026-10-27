package interceptors

import (
	"context"
	"fmt"
	"net/http"
	"sparkly/backend/internal/domain"
	"sparkly/backend/internal/metadata"
	"time"

	"connectrpc.com/connect/v2"
)

func NewAuthInterceptor(refresher *TokenRefresher) connect.ServerInterceptor {
	return func(next connect.ServerFunc) connect.ServerFunc {
		return func(ctx context.Context, req connect.Spec, stream connect.ServerStream) error {
			info, ok := connect.CallInfoForServerContext(ctx)
			if !ok {
				return fmt.Errorf("call info not provided")
			}

			userID := info.RequestHeader().Get("X-User-Id")
			if userID != "" {
				tokenID := info.RequestHeader().Get("X-Token-Id")
				ctx = context.WithValue(ctx, metadata.UserIDContextKey, userID)
				ctx = context.WithValue(ctx, metadata.TokenIDContextKey, tokenID)
				return next(ctx, req, stream)
			}

			tokenID := info.RequestHeader().Get("X-Token-Id")
			if tokenID != "" {
				user, token, err := refresher.RefreshToken(ctx, tokenID)
				if err != nil {
					return fmt.Errorf("failed to refresh access token: %w", err)
				}
				accessCookie := &http.Cookie{
					Name:     "access_token",
					Value:    token,
					Path:     "/",
					HttpOnly: true,
					SameSite: http.SameSiteLaxMode,
					MaxAge:   int(domain.ACCESS_TOKEN_LIFETIME / time.Second),
					// Secure: true,
				}
				info.ResponseHeader().Set("Set-Cookie", accessCookie.String())
				ctx = context.WithValue(ctx, metadata.UserIDContextKey, user.ID.String())
				ctx = context.WithValue(ctx, metadata.TokenIDContextKey, tokenID)
				return next(ctx, req, stream)
			}

			return next(ctx, req, stream)
		}
	}
}
