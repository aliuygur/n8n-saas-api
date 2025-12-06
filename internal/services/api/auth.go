package api

import (
	"context"
	"fmt"

	eauth "encore.dev/beta/auth"
	"encore.dev/beta/errs"
	"github.com/aliuygur/n8n-saas-api/internal/auth"
	"github.com/golang-jwt/jwt/v5"
)

type JWTClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// AuthHandler validates JWT tokens and returns user information
//
//encore:authhandler
func (s *Service) AuthHandler(ctx context.Context, tokenString string) (eauth.UID, *auth.User, error) {
	if tokenString == "" {
		return "", nil, &errs.Error{
			Code:    errs.Unauthenticated,
			Message: "no authorization header provided",
		}
	}

	// Parse and validate JWT token
	jwtToken, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return "", nil, &errs.Error{
			Code:    errs.Unauthenticated,
			Message: "invalid or expired token",
		}
	}

	claims, ok := jwtToken.Claims.(*JWTClaims)
	if !ok || !jwtToken.Valid {
		return "", nil, &errs.Error{
			Code:    errs.Unauthenticated,
			Message: "invalid token claims",
		}
	}

	userData := &auth.User{
		ID:    claims.UserID,
		Email: claims.Email,
	}

	return eauth.UID(claims.UserID), userData, nil
}

// LogoutResponse represents the logout response
type LogoutResponse struct {
	Success bool `json:"success"`
}

// Logout logs out the user (client should discard the JWT token)
//
//encore:api public method=POST path=/api/auth/logout
func (s *Service) Logout(ctx context.Context) (*LogoutResponse, error) {
	// Since JWT is stateless and stored on client side,
	// logout is handled by the client discarding the token
	return &LogoutResponse{Success: true}, nil
}

// MeResponse represents the current user response
type MeResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// Me returns the current authenticated user
//
//encore:api auth method=GET path=/api/auth/me
func (s *Service) Me(ctx context.Context) (*MeResponse, error) {
	user := auth.MustGetUser()
	return &MeResponse{
		ID:    user.ID,
		Email: user.Email,
	}, nil
}
