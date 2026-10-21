package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"connectrpc.com/connect/v2"

	"sparkly/backend/internal/domain"
	"sparkly/backend/internal/services"
	pb "sparkly/backend/proto/v1"
	"sparkly/backend/proto/v1/backendv1connect"
)

type UserHandler struct {
	backendv1connect.UnimplementedUserServiceHandler

	service *services.UserService
}

func NewUserHandler(service *services.UserService) *UserHandler {
	return &UserHandler{
		service: service,
	}
}

func (h *UserHandler) Reg(ctx context.Context, r *pb.RegRequest) (*pb.RegResponse, error) {
	email, password := r.GetEmail(), r.GetPassword()

	user, token, err := h.service.Reg(ctx, email, password)
	if err != nil {
		if errors.Is(err, domain.ErrDuplicatedUser) {
			return nil, connect.NewError(connect.CodeAlreadyExists, "user with this email already registered")
		}
		log.Printf("UserService/Reg error: %v", err)
		return nil, connect.NewError(connect.CodeInternal, "internal error")
	}

	callInfo, ok := connect.CallInfoForServerContext(ctx)
	if !ok {
		log.Printf("failed to get call info, req: %s", r.String())
		return nil, connect.NewError(connect.CodeInternal, "internal error")
	}

	accessCookie := &http.Cookie{
		Name:     "access_token",
		Value:    token.Access,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(domain.ACCESS_TOKEN_LIFETIME / time.Second),
		// Secure: true,
	}
	refreshCookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    token.Refresh,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(domain.REFRESH_TOKEN_LIFETIME / time.Second),
		// Secure: true,
	}
	callInfo.ResponseHeader().Add("Set-Cookie", accessCookie.String())
	callInfo.ResponseHeader().Add("Set-Cookie", refreshCookie.String())

	return &pb.RegResponse{
		Email:        user.Email,
		Token:        token.Access,
		RefreshToken: token.Refresh,
		ExpiresIn:    time.Now().Add(time.Duration(accessCookie.MaxAge * int(time.Second))).Unix(),
	}, nil
}

func (h *UserHandler) GetToken(ctx context.Context, r *pb.GetTokenRequest) (*pb.GetTokenResponse, error) {
	email, password := r.GetEmail(), r.GetPassword()

	user, token, err := h.service.GetTokens(ctx, email, password)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, "this user is not registered")
		}
		if errors.Is(err, domain.ErrWrongPassword) {
			return nil, connect.NewError(connect.CodeUnauthenticated, "wrong password")
		}
		log.Printf("UserService/GetToken error: %v", err)
		return nil, connect.NewError(connect.CodeInternal, "internal error")
	}

	callInfo, ok := connect.CallInfoForServerContext(ctx)
	if !ok {
		log.Printf("failed to get call info, req: %s", r.String())
		return nil, connect.NewError(connect.CodeInternal, "internal error")
	}

	accessCookie := &http.Cookie{
		Name:     "access_token",
		Value:    token.Access,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(domain.ACCESS_TOKEN_LIFETIME / time.Second),
		// Secure: true,
	}
	refreshCookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    token.Refresh,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(domain.REFRESH_TOKEN_LIFETIME / time.Second),
		// Secure: true,
	}
	callInfo.ResponseHeader().Add("Set-Cookie", accessCookie.String())
	callInfo.ResponseHeader().Add("Set-Cookie", refreshCookie.String())

	return &pb.GetTokenResponse{
		Email:        user.Email,
		Token:        token.Access,
		RefreshToken: token.Refresh,
		ExpiresIn:    time.Now().Add(time.Duration(accessCookie.MaxAge * int(time.Second))).Unix(),
	}, nil
}

func (h *UserHandler) RefreshToken(ctx context.Context, r *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	tokenID := r.GetTokenId()

	user, token, err := h.service.RefreshToken(ctx, tokenID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, "this user is not registered")
		}
		if errors.Is(err, domain.ErrTokenNotFound) {
			return nil, connect.NewError(connect.CodeUnauthenticated, "invalid token")
		}
		if errors.Is(err, domain.ErrRevokedRefreshToken) {
			return nil, connect.NewError(connect.CodeInvalidArgument, "token has been revoked")
		}
		if errors.Is(err, domain.ErrExpiredRefreshToken) {
			return nil, connect.NewError(connect.CodeUnauthenticated, "token expired")
		}
		log.Printf("UserService/RefreshToken error: %v", err)
		return nil, connect.NewError(connect.CodeInternal, "internal error")
	}

	callInfo, ok := connect.CallInfoForServerContext(ctx)
	if !ok {
		log.Printf("failed to get call info, req: %s", r.String())
		return nil, connect.NewError(connect.CodeInternal, "internal error")
	}

	accessCookie := &http.Cookie{
		Name:     "access_token",
		Value:    token.Access,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(domain.ACCESS_TOKEN_LIFETIME / time.Second),
		// Secure: true,
	}

	callInfo.ResponseHeader().Add("Set-Cookie", accessCookie.String())

	return &pb.RefreshTokenResponse{
		Email:     user.Email,
		Token:     token.Access,
		ExpiresIn: time.Now().Add(time.Duration(accessCookie.MaxAge * int(time.Second))).Unix(),
	}, nil
}

func (h *UserHandler) RevokeToken(ctx context.Context, r *pb.RevokeTokenRequest) (*pb.RevokeTokenResponse, error) {
	tokenID := r.GetTokenId()

	err := h.service.RevokeToken(ctx, tokenID)
	if err != nil {
		if errors.Is(err, domain.ErrTokenNotFound) {
			return nil, connect.NewError(connect.CodeUnauthenticated, "invalid token")
		}

		if errors.Is(err, domain.ErrRevokedRefreshToken) {
			return nil, connect.NewError(connect.CodeInvalidArgument, "this token is already revoked")
		}
		log.Printf("UserService/RevokeToken error: %v", err)
		return nil, connect.NewError(connect.CodeInternal, "internal error")
	}

	return &pb.RevokeTokenResponse{
		Success: true,
	}, nil
}
