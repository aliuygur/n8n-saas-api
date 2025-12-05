package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"encore.dev/beta/auth"
	"encore.dev/beta/errs"
	"github.com/aliuygur/n8n-saas-api/internal/db"
	"github.com/samber/lo"
)

// UserData represents the authenticated user's data
type UserData struct {
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

// AuthHandler validates session tokens and returns user information
//
//encore:authhandler
func (s *Service) AuthHandler(ctx context.Context, token string) (auth.UID, *UserData, error) {
	queries := db.New(s.db)

	// Get session and user using the generated query
	result, err := queries.GetSessionByToken(ctx, token)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil, &errs.Error{
				Code:    errs.Unauthenticated,
				Message: "invalid or expired session",
			}
		}
		return "", nil, fmt.Errorf("failed to validate session: %w", err)
	}

	userData := &UserData{
		Email:   result.User.Email,
		Name:    result.User.Name,
		Picture: result.User.Picture,
	}

	// Use email as the unique identifier
	return auth.UID(result.User.Email), userData, nil
}

// Auth endpoints

type GoogleLoginResponse struct {
	AuthURL string `json:"auth_url"`
}

// GoogleLogin initiates the Google OAuth flow
//
//encore:api public method=GET path=/auth/google/login
func (s *Service) GoogleLogin(ctx context.Context) (*GoogleLoginResponse, error) {
	// Generate a random state token for CSRF protection
	state := lo.RandomString(32, lo.LettersCharset)

	// TODO: Store state in cache/session for validation in callback
	// For now, we'll just generate the URL
	authURL := s.oauth2Config.AuthCodeURL(state)

	return &GoogleLoginResponse{
		AuthURL: authURL,
	}, nil
}

type GoogleCallbackRequest struct {
	Code  string `json:"code"`
	State string `json:"state"`
}

type GoogleCallbackResponse struct {
	SessionToken string    `json:"session_token"`
	User         *UserInfo `json:"user"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type UserInfo struct {
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

// GoogleCallback handles the OAuth callback from Google
//
//encore:api public method=POST path=/auth/google/callback
func (s *Service) GoogleCallback(ctx context.Context, req *GoogleCallbackRequest) (*GoogleCallbackResponse, error) {
	// TODO: Validate state token to prevent CSRF

	// Exchange code for token
	token, err := s.oauth2Config.Exchange(ctx, req.Code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Get user info from Google
	client := s.oauth2Config.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user info: status=%d body=%s", resp.StatusCode, string(body))
	}

	var googleUser GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&googleUser); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// Create or get user in database
	queries := db.New(s.db)

	var user db.User
	existingUser, err := queries.GetUserByEmail(ctx, googleUser.Email)
	if err == sql.ErrNoRows {
		// Create new user
		user, err = queries.CreateUser(ctx, db.CreateUserParams{
			Email:   googleUser.Email,
			Name:    googleUser.Name,
			Picture: googleUser.Picture,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	} else {
		user = existingUser
	}

	// Update last login
	user, err = queries.UpdateUserLastLogin(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to update last login: %w", err)
	}

	// Create session
	sessionToken := lo.RandomString(64, lo.LettersCharset)

	expiresAt := time.Now().Add(7 * 24 * time.Hour) // 7 days
	session, err := queries.CreateSession(ctx, db.CreateSessionParams{
		UserID:    user.ID,
		Token:     sessionToken,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &GoogleCallbackResponse{
		SessionToken: session.Token,
		User: &UserInfo{
			Email:   user.Email,
			Name:    user.Name,
			Picture: user.Picture,
		},
		ExpiresAt: session.ExpiresAt,
	}, nil
}

type GetMeRequest struct {
	SessionToken string `header:"Authorization"`
}

type GetMeResponse struct {
	User *UserInfo `json:"user"`
}

// GetMe returns the current user's information
// This uses Encore's auth handler, so the user data is automatically available
//
//encore:api auth method=GET path=/auth/me
func (s *Service) GetMe(ctx context.Context) (*UserInfo, error) {
	// Get user data from Encore's auth context
	userData, _ := auth.Data().(*UserData)
	if userData == nil {
		return nil, fmt.Errorf("user data not found")
	}

	return &UserInfo{
		Email:   userData.Email,
		Name:    userData.Name,
		Picture: userData.Picture,
	}, nil
}

type LogoutRequest struct {
	SessionToken string `header:"Authorization"`
}

type LogoutResponse struct {
	Success bool `json:"success"`
}

// Logout invalidates the current session
// Uses auth level so we can access the session token from context
//
//encore:api auth method=POST path=/auth/logout
func (s *Service) Logout(ctx context.Context) (*LogoutResponse, error) {
	// Get authenticated user
	userEmail, ok := auth.UserID()
	if !ok {
		return &LogoutResponse{Success: false}, nil
	}

	// Get user ID from email
	queries := db.New(s.db)
	user, err := queries.GetUserByEmail(ctx, string(userEmail))
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Delete all sessions for this user
	if err := queries.DeleteUserSessions(ctx, user.ID); err != nil {
		return nil, fmt.Errorf("failed to delete sessions: %w", err)
	}

	return &LogoutResponse{Success: true}, nil
}
