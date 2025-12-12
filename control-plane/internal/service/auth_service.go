// Package service provides business logic implementations.
package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/Bidon15/popsigner/control-plane/internal/models"
	apierrors "github.com/Bidon15/popsigner/control-plane/internal/pkg/errors"
	"github.com/Bidon15/popsigner/control-plane/internal/repository"
)

// AuthService defines the interface for authentication operations.
type AuthService interface {
	Register(ctx context.Context, req RegisterRequest) (*models.User, error)
	Login(ctx context.Context, email, password string) (*models.User, string, error)
	Logout(ctx context.Context, sessionID string) error
	LogoutAll(ctx context.Context, userID uuid.UUID) error
	ValidateSession(ctx context.Context, sessionID string) (*models.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, req UpdateProfileRequest) (*models.User, error)
	ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error
	RequestPasswordReset(ctx context.Context, email string) (string, error)
	ResetPassword(ctx context.Context, token, newPassword string) error
	VerifyEmail(ctx context.Context, token string) error
}

// RegisterRequest contains the data needed to register a new user.
type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Name     string `json:"name" validate:"required,min=2"`
}

// UpdateProfileRequest contains the data for updating a user profile.
type UpdateProfileRequest struct {
	Name      *string `json:"name,omitempty" validate:"omitempty,min=2"`
	AvatarURL *string `json:"avatar_url,omitempty" validate:"omitempty,url"`
}

// AuthServiceConfig holds configuration for the auth service.
type AuthServiceConfig struct {
	BCryptCost     int
	SessionExpiry  time.Duration
	PasswordResetExpiry time.Duration
	EmailVerifyExpiry   time.Duration
}

// DefaultAuthServiceConfig returns sensible default configuration.
func DefaultAuthServiceConfig() AuthServiceConfig {
	return AuthServiceConfig{
		BCryptCost:          12,
		SessionExpiry:       7 * 24 * time.Hour, // 7 days
		PasswordResetExpiry: 1 * time.Hour,
		EmailVerifyExpiry:   24 * time.Hour,
	}
}

type authService struct {
	userRepo    repository.UserRepository
	sessionRepo repository.SessionRepository
	config      AuthServiceConfig
}

// NewAuthService creates a new authentication service.
func NewAuthService(
	userRepo repository.UserRepository,
	sessionRepo repository.SessionRepository,
	config AuthServiceConfig,
) AuthService {
	return &authService{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		config:      config,
	}
}

// Register creates a new user account.
func (s *authService) Register(ctx context.Context, req RegisterRequest) (*models.User, error) {
	// Check if email already exists
	existing, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existing != nil {
		return nil, apierrors.NewConflictError("Email already registered")
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), s.config.BCryptCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	hashStr := string(hash)
	user := &models.User{
		Email:         req.Email,
		PasswordHash:  &hashStr,
		Name:          &req.Name,
		EmailVerified: false,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// TODO: Send verification email (would be implemented with email service)

	return user, nil
}

// Login authenticates a user and creates a session.
func (s *authService) Login(ctx context.Context, email, password string) (*models.User, string, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, "", apierrors.ErrUnauthorized.WithMessage("Invalid email or password")
	}

	// Check if user has a password (could be OAuth only)
	if user.PasswordHash == nil {
		return nil, "", apierrors.ErrUnauthorized.WithMessage("Invalid email or password")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(password)); err != nil {
		return nil, "", apierrors.ErrUnauthorized.WithMessage("Invalid email or password")
	}

	// Create session
	sessionID, err := s.createSession(ctx, user.ID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create session: %w", err)
	}

	// Update last login timestamp (fire and forget)
	_ = s.userRepo.UpdateLastLogin(ctx, user.ID)

	return user, sessionID, nil
}

// Logout invalidates a session.
func (s *authService) Logout(ctx context.Context, sessionID string) error {
	return s.sessionRepo.Delete(ctx, sessionID)
}

// LogoutAll invalidates all sessions for a user.
func (s *authService) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	return s.sessionRepo.DeleteAllForUser(ctx, userID)
}

// ValidateSession validates a session and returns the associated user.
func (s *authService) ValidateSession(ctx context.Context, sessionID string) (*models.User, error) {
	session, err := s.sessionRepo.Get(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	if session == nil {
		return nil, apierrors.ErrUnauthorized.WithMessage("Invalid session")
	}

	// Check if session is expired
	if session.ExpiresAt.Before(time.Now()) {
		// Clean up expired session
		_ = s.sessionRepo.Delete(ctx, sessionID)
		return nil, apierrors.ErrUnauthorized.WithMessage("Session expired")
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		// User was deleted, clean up session
		_ = s.sessionRepo.Delete(ctx, sessionID)
		return nil, apierrors.ErrUnauthorized.WithMessage("User not found")
	}

	return user, nil
}

// GetUserByID retrieves a user by their ID.
func (s *authService) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, apierrors.NewNotFoundError("User")
	}
	return user, nil
}

// UpdateProfile updates a user's profile information.
func (s *authService) UpdateProfile(ctx context.Context, userID uuid.UUID, req UpdateProfileRequest) (*models.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, apierrors.NewNotFoundError("User")
	}

	// Update fields if provided
	if req.Name != nil {
		user.Name = req.Name
	}
	if req.AvatarURL != nil {
		user.AvatarURL = req.AvatarURL
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return user, nil
}

// ChangePassword changes a user's password after verifying the old one.
func (s *authService) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return apierrors.NewNotFoundError("User")
	}

	// Verify old password
	if user.PasswordHash == nil {
		return apierrors.ErrBadRequest.WithMessage("Cannot change password for OAuth accounts")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(oldPassword)); err != nil {
		return apierrors.ErrUnauthorized.WithMessage("Current password is incorrect")
	}

	// Hash new password
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), s.config.BCryptCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	if err := s.userRepo.UpdatePassword(ctx, userID, string(hash)); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Invalidate all other sessions for security
	_ = s.sessionRepo.DeleteAllForUser(ctx, userID)

	return nil
}

// RequestPasswordReset generates a password reset token.
// In a real implementation, this would send an email with the token.
func (s *authService) RequestPasswordReset(ctx context.Context, email string) (string, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}

	// Always return success to prevent email enumeration
	if user == nil {
		return "", nil
	}

	// Generate reset token
	token, err := generateSecureToken(32)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	// TODO: Store token in Redis with expiry and send email
	// For now, return the token (in production, this would only be sent via email)

	return token, nil
}

// ResetPassword resets a user's password using a reset token.
func (s *authService) ResetPassword(ctx context.Context, token, newPassword string) error {
	// TODO: Implement token lookup from Redis
	// For now, return not implemented
	return apierrors.ErrBadRequest.WithMessage("Password reset not yet implemented")
}

// VerifyEmail verifies a user's email using a verification token.
func (s *authService) VerifyEmail(ctx context.Context, token string) error {
	// TODO: Implement token lookup from Redis and call SetEmailVerified
	return apierrors.ErrBadRequest.WithMessage("Email verification not yet implemented")
}

// createSession creates a new session for a user.
func (s *authService) createSession(ctx context.Context, userID uuid.UUID) (string, error) {
	sessionID, err := generateSecureToken(32)
	if err != nil {
		return "", fmt.Errorf("failed to generate session ID: %w", err)
	}

	session := &models.Session{
		ID:        sessionID,
		UserID:    userID,
		Data:      make(map[string]interface{}),
		ExpiresAt: time.Now().Add(s.config.SessionExpiry),
	}

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	return sessionID, nil
}

// generateSecureToken generates a cryptographically secure random token.
func generateSecureToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

