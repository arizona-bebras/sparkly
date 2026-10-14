package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"

	"connectrpc.com/connect/v2"

	"sparkly/backend/internal/domain"
	"sparkly/backend/internal/services"
	pb "sparkly/backend/proto"
	"sparkly/backend/proto/backendv1connect"
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
		log.Printf("failed reg new user: %v", err)
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
		MaxAge:   24 * 60 * 60,
		// Secure: true,
	}
	refreshCookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    token.Refresh,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   7 * 24 * 60 * 60,
		// Secure: true,
	}
	callInfo.ResponseHeader().Add("Set-Cookie", accessCookie.String())
	callInfo.ResponseHeader().Add("Set-Cookie", refreshCookie.String())

	return &pb.RegResponse{
		Email:        user.Email,
		Token:        token.Access,
		RefreshToken: token.Refresh,
	}, nil
}
