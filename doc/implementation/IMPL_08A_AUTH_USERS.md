# Implementation: Auth - Users & Sessions

## Agent: 08A - User Authentication

> **Phase 5.1** - Can run in parallel with 08B, 08C after Agent 07 completes.

---

## 1. Overview

Implement user registration, login, password management, and session handling.

---

## 2. Scope

| Feature | Included |
|---------|----------|
| Email + Password registration | ✅ |
| Email verification | ✅ |
| Login/Logout | ✅ |
| Password reset | ✅ |
| Session management | ✅ |
| OAuth | ❌ (Agent 08B) |
| API Keys | ❌ (Agent 08C) |

---

## 3. Models

**File:** `internal/models/user.go`

```go
package models

import (
    "time"

    "github.com/google/uuid"
)

type User struct {
    ID            uuid.UUID  `json:"id" db:"id"`
    Email         string     `json:"email" db:"email"`
    PasswordHash  string     `json:"-" db:"password_hash"`
    Name          string     `json:"name" db:"name"`
    AvatarURL     string     `json:"avatar_url,omitempty" db:"avatar_url"`
    EmailVerified bool       `json:"email_verified" db:"email_verified"`
    OAuthProvider string     `json:"-" db:"oauth_provider"`
    OAuthID       string     `json:"-" db:"oauth_provider_id"`
    LastLoginAt   *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
    CreatedAt     time.Time  `json:"created_at" db:"created_at"`
    UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

type Session struct {
    ID        string    `json:"id" db:"id"`
    UserID    uuid.UUID `json:"user_id" db:"user_id"`
    Data      []byte    `json:"-" db:"data"`
    ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
}
```

---

## 4. Repository

**File:** `internal/repository/user_repo.go`

```go
package repository

import (
    "context"
    "database/sql"

    "github.com/google/uuid"
    "github.com/Bidon15/banhbaoring/control-plane/internal/models"
)

type UserRepository interface {
    Create(ctx context.Context, user *models.User) error
    GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
    GetByEmail(ctx context.Context, email string) (*models.User, error)
    Update(ctx context.Context, user *models.User) error
    UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error
    SetEmailVerified(ctx context.Context, id uuid.UUID) error
    UpdateLastLogin(ctx context.Context, id uuid.UUID) error
}

type userRepo struct {
    db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
    return &userRepo{db: db}
}

func (r *userRepo) Create(ctx context.Context, user *models.User) error {
    query := `
        INSERT INTO users (id, email, password_hash, name)
        VALUES ($1, $2, $3, $4)
        RETURNING created_at, updated_at`
    
    user.ID = uuid.New()
    return r.db.QueryRowContext(ctx, query,
        user.ID, user.Email, user.PasswordHash, user.Name,
    ).Scan(&user.CreatedAt, &user.UpdatedAt)
}

func (r *userRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
    query := `
        SELECT id, email, password_hash, name, avatar_url, email_verified,
               oauth_provider, oauth_provider_id, last_login_at, created_at, updated_at
        FROM users WHERE id = $1`
    
    var user models.User
    err := r.db.QueryRowContext(ctx, query, id).Scan(
        &user.ID, &user.Email, &user.PasswordHash, &user.Name,
        &user.AvatarURL, &user.EmailVerified, &user.OAuthProvider,
        &user.OAuthID, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt,
    )
    if err == sql.ErrNoRows {
        return nil, nil
    }
    return &user, err
}

func (r *userRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
    query := `
        SELECT id, email, password_hash, name, avatar_url, email_verified,
               oauth_provider, oauth_provider_id, last_login_at, created_at, updated_at
        FROM users WHERE email = $1`
    
    var user models.User
    err := r.db.QueryRowContext(ctx, query, email).Scan(
        &user.ID, &user.Email, &user.PasswordHash, &user.Name,
        &user.AvatarURL, &user.EmailVerified, &user.OAuthProvider,
        &user.OAuthID, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt,
    )
    if err == sql.ErrNoRows {
        return nil, nil
    }
    return &user, err
}

func (r *userRepo) Update(ctx context.Context, user *models.User) error {
    query := `UPDATE users SET name = $2, avatar_url = $3 WHERE id = $1`
    _, err := r.db.ExecContext(ctx, query, user.ID, user.Name, user.AvatarURL)
    return err
}

func (r *userRepo) UpdatePassword(ctx context.Context, id uuid.UUID, hash string) error {
    query := `UPDATE users SET password_hash = $2 WHERE id = $1`
    _, err := r.db.ExecContext(ctx, query, id, hash)
    return err
}

func (r *userRepo) SetEmailVerified(ctx context.Context, id uuid.UUID) error {
    query := `UPDATE users SET email_verified = true WHERE id = $1`
    _, err := r.db.ExecContext(ctx, query, id)
    return err
}

func (r *userRepo) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
    query := `UPDATE users SET last_login_at = NOW() WHERE id = $1`
    _, err := r.db.ExecContext(ctx, query, id)
    return err
}
```

---

## 5. Service

**File:** `internal/service/auth_service.go`

```go
package service

import (
    "context"
    "crypto/rand"
    "encoding/base64"
    "time"

    "github.com/google/uuid"
    "golang.org/x/crypto/bcrypt"

    "github.com/Bidon15/banhbaoring/control-plane/internal/models"
    "github.com/Bidon15/banhbaoring/control-plane/internal/repository"
    apierrors "github.com/Bidon15/banhbaoring/control-plane/internal/pkg/errors"
)

type AuthService interface {
    Register(ctx context.Context, req RegisterRequest) (*models.User, error)
    Login(ctx context.Context, email, password string) (*models.User, string, error)
    Logout(ctx context.Context, sessionID string) error
    ValidateSession(ctx context.Context, sessionID string) (*models.User, error)
    RequestPasswordReset(ctx context.Context, email string) (string, error)
    ResetPassword(ctx context.Context, token, newPassword string) error
    VerifyEmail(ctx context.Context, token string) error
}

type RegisterRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
    Name     string `json:"name" validate:"required,min=2"`
}

type authService struct {
    userRepo    repository.UserRepository
    sessionRepo repository.SessionRepository
    bcryptCost  int
}

func NewAuthService(
    userRepo repository.UserRepository,
    sessionRepo repository.SessionRepository,
    bcryptCost int,
) AuthService {
    return &authService{
        userRepo:    userRepo,
        sessionRepo: sessionRepo,
        bcryptCost:  bcryptCost,
    }
}

func (s *authService) Register(ctx context.Context, req RegisterRequest) (*models.User, error) {
    // Check if email exists
    existing, err := s.userRepo.GetByEmail(ctx, req.Email)
    if err != nil {
        return nil, err
    }
    if existing != nil {
        return nil, apierrors.NewConflictError("Email already registered")
    }

    // Hash password
    hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), s.bcryptCost)
    if err != nil {
        return nil, err
    }

    user := &models.User{
        Email:        req.Email,
        PasswordHash: string(hash),
        Name:         req.Name,
    }

    if err := s.userRepo.Create(ctx, user); err != nil {
        return nil, err
    }

    // TODO: Send verification email (Agent 08A enhancement)

    return user, nil
}

func (s *authService) Login(ctx context.Context, email, password string) (*models.User, string, error) {
    user, err := s.userRepo.GetByEmail(ctx, email)
    if err != nil {
        return nil, "", err
    }
    if user == nil {
        return nil, "", apierrors.ErrUnauthorized
    }

    if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
        return nil, "", apierrors.ErrUnauthorized
    }

    // Create session
    sessionID, err := s.createSession(ctx, user.ID)
    if err != nil {
        return nil, "", err
    }

    // Update last login
    _ = s.userRepo.UpdateLastLogin(ctx, user.ID)

    return user, sessionID, nil
}

func (s *authService) Logout(ctx context.Context, sessionID string) error {
    return s.sessionRepo.Delete(ctx, sessionID)
}

func (s *authService) ValidateSession(ctx context.Context, sessionID string) (*models.User, error) {
    session, err := s.sessionRepo.Get(ctx, sessionID)
    if err != nil {
        return nil, err
    }
    if session == nil || session.ExpiresAt.Before(time.Now()) {
        return nil, apierrors.ErrUnauthorized
    }

    return s.userRepo.GetByID(ctx, session.UserID)
}

func (s *authService) createSession(ctx context.Context, userID uuid.UUID) (string, error) {
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    sessionID := base64.URLEncoding.EncodeToString(b)

    session := &models.Session{
        ID:        sessionID,
        UserID:    userID,
        ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // 7 days
    }

    if err := s.sessionRepo.Create(ctx, session); err != nil {
        return "", err
    }

    return sessionID, nil
}

func (s *authService) RequestPasswordReset(ctx context.Context, email string) (string, error) {
    // Implementation: generate token, store in Redis, send email
    panic("TODO(08A): implement password reset")
}

func (s *authService) ResetPassword(ctx context.Context, token, newPassword string) error {
    panic("TODO(08A): implement password reset")
}

func (s *authService) VerifyEmail(ctx context.Context, token string) error {
    panic("TODO(08A): implement email verification")
}
```

---

## 6. Handler

**File:** `internal/handler/auth_handler.go`

```go
package handler

import (
    "encoding/json"
    "net/http"
    "time"

    "github.com/go-chi/chi/v5"
    "github.com/go-playground/validator/v10"

    "github.com/Bidon15/banhbaoring/control-plane/internal/service"
    "github.com/Bidon15/banhbaoring/control-plane/internal/pkg/response"
    apierrors "github.com/Bidon15/banhbaoring/control-plane/internal/pkg/errors"
)

type AuthHandler struct {
    authService service.AuthService
    validate    *validator.Validate
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
    return &AuthHandler{
        authService: authService,
        validate:    validator.New(),
    }
}

func (h *AuthHandler) Routes() chi.Router {
    r := chi.NewRouter()

    r.Post("/register", h.Register)
    r.Post("/login", h.Login)
    r.Post("/logout", h.Logout)
    r.Get("/me", h.Me)
    r.Post("/password/forgot", h.ForgotPassword)
    r.Post("/password/reset", h.ResetPassword)
    r.Post("/email/verify", h.VerifyEmail)

    return r
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
    var req service.RegisterRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        response.Error(w, apierrors.ErrBadRequest)
        return
    }

    if err := h.validate.Struct(req); err != nil {
        response.Error(w, apierrors.NewValidationError("", err.Error()))
        return
    }

    user, err := h.authService.Register(r.Context(), req)
    if err != nil {
        response.Error(w, err)
        return
    }

    response.Created(w, user)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Email    string `json:"email"`
        Password string `json:"password"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        response.Error(w, apierrors.ErrBadRequest)
        return
    }

    user, sessionID, err := h.authService.Login(r.Context(), req.Email, req.Password)
    if err != nil {
        response.Error(w, err)
        return
    }

    // Set session cookie
    http.SetCookie(w, &http.Cookie{
        Name:     "session",
        Value:    sessionID,
        Path:     "/",
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteLaxMode,
        MaxAge:   int(7 * 24 * time.Hour / time.Second),
    })

    response.OK(w, map[string]any{
        "user":       user,
        "session_id": sessionID,
    })
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
    cookie, err := r.Cookie("session")
    if err != nil {
        response.NoContent(w)
        return
    }

    _ = h.authService.Logout(r.Context(), cookie.Value)

    // Clear cookie
    http.SetCookie(w, &http.Cookie{
        Name:   "session",
        Value:  "",
        Path:   "/",
        MaxAge: -1,
    })

    response.NoContent(w)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
    cookie, err := r.Cookie("session")
    if err != nil {
        response.Error(w, apierrors.ErrUnauthorized)
        return
    }

    user, err := h.authService.ValidateSession(r.Context(), cookie.Value)
    if err != nil {
        response.Error(w, err)
        return
    }

    response.OK(w, user)
}

func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
    panic("TODO(08A): implement forgot password handler")
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
    panic("TODO(08A): implement reset password handler")
}

func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
    panic("TODO(08A): implement verify email handler")
}
```

---

## 7. Deliverables

| File | Description |
|------|-------------|
| `internal/models/user.go` | User & Session models |
| `internal/repository/user_repo.go` | User database operations |
| `internal/repository/session_repo.go` | Session database operations |
| `internal/service/auth_service.go` | Auth business logic |
| `internal/handler/auth_handler.go` | HTTP handlers |
| `internal/handler/auth_handler_test.go` | Tests |

---

## 8. API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/auth/register` | Create account |
| POST | `/v1/auth/login` | Login |
| POST | `/v1/auth/logout` | Logout |
| GET | `/v1/auth/me` | Current user |
| POST | `/v1/auth/password/forgot` | Request reset |
| POST | `/v1/auth/password/reset` | Reset password |
| POST | `/v1/auth/email/verify` | Verify email |

---

## 9. Success Criteria

- [ ] User registration works
- [ ] Login returns session
- [ ] Session validation works
- [ ] Password hashing with bcrypt
- [ ] All tests pass

---

## 10. Agent Prompt

```
You are Agent 08A - User Authentication. Implement user registration, login, and sessions.

Read the spec: doc/implementation/IMPL_08A_AUTH_USERS.md

Deliverables:
1. User model and repository
2. Session model and repository  
3. Auth service (register, login, logout, validate)
4. Auth HTTP handlers
5. Unit tests

Dependencies: Agent 07 (Foundation) must complete first.

Test: go test ./internal/... -v
```

