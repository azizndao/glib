# Phase 4: Authentication & Authorization

**Timeline**: Weeks 9-13  
**Priority**: Critical - Core security functionality  
**Dependencies**: Phase 1 (Foundation), Phase 2 (Database)

## Overview

Build a comprehensive authentication and authorization system inspired by Laravel's Auth system with:

- Multiple authentication guards (JWT, Session, API Token)
- User providers (flexible user storage)
- Password hashing and verification
- OAuth2 social authentication
- Authorization with policies and gates
- Password reset flows
- Email verification

## Architecture

```
auth/
├── auth.go              # Main auth manager
├── guard.go            # Guard interface & implementations
├── provider.go         # User provider interface
├── authenticatable.go  # User interface contract
├── password.go         # Password hashing
└── events.go           # Auth events

auth/guards/
├── jwt_guard.go        # JWT token authentication
├── session_guard.go    # Session-based authentication
└── token_guard.go      # API token authentication

auth/providers/
├── gorm_provider.go    # GORM-based user provider
└── custom_provider.go  # Custom user provider

auth/middleware/
├── authenticate.go     # Require authentication
├── guest.go           # Require guest (not authenticated)
└── authorize.go       # Check permissions

auth/jwt/
├── token.go           # Token generation/validation
├── claims.go          # JWT claims structure
├── blacklist.go       # Token blacklist
└── refresh.go         # Refresh token logic

auth/oauth/
├── manager.go         # OAuth manager
├── provider.go        # OAuth provider interface
└── providers/
    ├── google.go
    ├── github.go
    └── facebook.go

auth/policies/
├── gate.go            # Gate manager
├── policy.go          # Policy interface
└── response.go        # Authorization response
```

## 1. Core Authentication

### Authenticatable Interface

Any model that can be authenticated must implement this interface:

```go
package auth

type Authenticatable interface {
    GetAuthIdentifier() interface{}        // Primary key
    GetAuthPassword() string               // Hashed password
    GetRememberToken() string              // Remember me token
    SetRememberToken(token string)
}
```

### User Model Example

```go
package models

type User struct {
    orm.Model
    Name          string `gorm:"type:varchar(100);not null"`
    Email         string `gorm:"uniqueIndex;not null"`
    Password      string `gorm:"not null" json:"-"`
    RememberToken string `gorm:"type:varchar(100)" json:"-"`
    EmailVerified bool   `gorm:"default:false"`
}

// Implement Authenticatable
func (u *User) GetAuthIdentifier() interface{} {
    return u.ID
}

func (u *User) GetAuthPassword() string {
    return u.Password
}

func (u *User) GetRememberToken() string {
    return u.RememberToken
}

func (u *User) SetRememberToken(token string) {
    u.RememberToken = token
}

// Hash password before create
func (u *User) BeforeCreate(tx *gorm.DB) error {
    if u.Password != "" {
        hashed, err := auth.HashPassword(u.Password)
        if err != nil {
            return err
        }
        u.Password = hashed
    }
    return nil
}
```

### Auth Manager

Central authentication manager:

```go
type Manager struct {
    guards    map[string]Guard
    providers map[string]UserProvider
    config    *config.Repository
    default   string
}

func NewManager(config *config.Repository) *Manager {
    return &Manager{
        guards:    make(map[string]Guard),
        providers: make(map[string]UserProvider),
        config:    config,
        default:   config.GetString("auth.default", "jwt"),
    }
}

// Get default guard
func (m *Manager) Guard() Guard {
    return m.GuardNamed(m.default)
}

// Get named guard
func (m *Manager) GuardNamed(name string) Guard {
    if guard, exists := m.guards[name]; exists {
        return guard
    }
    panic(fmt.Sprintf("Auth guard '%s' not found", name))
}

// Attempt authentication
func (m *Manager) Attempt(credentials map[string]string) (Authenticatable, string, error) {
    return m.Guard().Attempt(credentials)
}

// Get authenticated user
func (m *Manager) User() Authenticatable {
    return m.Guard().User()
}

// Check if authenticated
func (m *Manager) Check() bool {
    return m.Guard().Check()
}

// Logout
func (m *Manager) Logout() error {
    return m.Guard().Logout()
}
```

## 2. JWT Authentication

### JWT Guard

```go
package guards

type JWTGuard struct {
    provider  UserProvider
    secret    string
    ttl       time.Duration
    blacklist *jwt.Blacklist
    user      Authenticatable
}

func NewJWTGuard(provider UserProvider, config *config.Repository) *JWTGuard {
    return &JWTGuard{
        provider:  provider,
        secret:    config.GetString("jwt.secret"),
        ttl:       config.GetDuration("jwt.ttl", 60*time.Minute),
        blacklist: jwt.NewBlacklist(),
    }
}

// Attempt login with credentials
func (g *JWTGuard) Attempt(credentials map[string]string) (Authenticatable, string, error) {
    user, err := g.provider.RetrieveByCredentials(credentials)
    if err != nil {
        return nil, "", ErrInvalidCredentials
    }

    if !g.provider.ValidateCredentials(user, credentials) {
        return nil, "", ErrInvalidCredentials
    }

    token, err := g.generateToken(user)
    if err != nil {
        return nil, "", err
    }

    g.user = user
    return user, token, nil
}

// Authenticate request
func (g *JWTGuard) Authenticate(c *glib.Ctx) error {
    tokenString := g.extractToken(c)
    if tokenString == "" {
        return ErrUnauthorized
    }

    // Check blacklist
    if g.blacklist.IsBlacklisted(tokenString) {
        return ErrTokenBlacklisted
    }

    // Validate token
    claims, err := g.validateToken(tokenString)
    if err != nil {
        return err
    }

    // Retrieve user
    user, err := g.provider.RetrieveByID(claims.Subject)
    if err != nil {
        return ErrUnauthorized
    }

    g.user = user
    return nil
}

// Generate JWT token
func (g *JWTGuard) generateToken(user Authenticatable) (string, error) {
    claims := jwt.NewClaims(
        user.GetAuthIdentifier(),
        time.Now().Add(g.ttl),
    )

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(g.secret))
}

// Refresh token
func (g *JWTGuard) Refresh(oldToken string) (string, error) {
    // Blacklist old token
    g.blacklist.Add(oldToken, g.ttl)

    // Generate new token
    return g.generateToken(g.user)
}

// Logout (blacklist token)
func (g *JWTGuard) Logout() error {
    // Blacklist current token
    // Implementation depends on storing token in request context
    return nil
}
```

### JWT Claims

```go
package jwt

type Claims struct {
    jwt.RegisteredClaims
    UserID interface{} `json:"uid"`
}

func NewClaims(userID interface{}, expiresAt time.Time) *Claims {
    return &Claims{
        UserID: userID,
        RegisteredClaims: jwt.RegisteredClaims{
            Subject:   fmt.Sprintf("%v", userID),
            ExpiresAt: jwt.NewNumericDate(expiresAt),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
    }
}
```

### Token Blacklist

```go
package jwt

type Blacklist struct {
    cache cache.Cache
}

func NewBlacklist(cache cache.Cache) *Blacklist {
    return &Blacklist{cache: cache}
}

func (b *Blacklist) Add(token string, ttl time.Duration) {
    b.cache.Set("blacklist:"+token, true, ttl)
}

func (b *Blacklist) IsBlacklisted(token string) bool {
    _, exists := b.cache.Get("blacklist:" + token)
    return exists
}
```

## 3. Session-Based Authentication

### Session Guard

```go
package guards

type SessionGuard struct {
    provider UserProvider
    session  session.Store
    user     Authenticatable
}

func NewSessionGuard(provider UserProvider, session session.Store) *SessionGuard {
    return &SessionGuard{
        provider: provider,
        session:  session,
    }
}

// Attempt login
func (g *SessionGuard) Attempt(c *glib.Ctx, credentials map[string]string, remember bool) (Authenticatable, error) {
    user, err := g.provider.RetrieveByCredentials(credentials)
    if err != nil {
        return nil, ErrInvalidCredentials
    }

    if !g.provider.ValidateCredentials(user, credentials) {
        return nil, ErrInvalidCredentials
    }

    // Store in session
    g.session.Set(c, "user_id", user.GetAuthIdentifier())

    // Remember me
    if remember {
        token := generateRememberToken()
        user.SetRememberToken(token)
        // Save user

        c.SetCookie(&http.Cookie{
            Name:     "remember_token",
            Value:    token,
            Expires:  time.Now().Add(30 * 24 * time.Hour),
            HttpOnly: true,
            Secure:   c.IsSecure(),
        })
    }

    g.user = user
    return user, nil
}

// Authenticate request
func (g *SessionGuard) Authenticate(c *glib.Ctx) error {
    // Check session
    if userID, exists := g.session.Get(c, "user_id"); exists {
        user, err := g.provider.RetrieveByID(userID)
        if err == nil {
            g.user = user
            return nil
        }
    }

    // Check remember me cookie
    if cookie, err := c.GetCookie("remember_token"); err == nil {
        user, err := g.provider.RetrieveByToken(cookie.Value)
        if err == nil {
            g.user = user
            g.session.Set(c, "user_id", user.GetAuthIdentifier())
            return nil
        }
    }

    return ErrUnauthorized
}

// Logout
func (g *SessionGuard) Logout(c *glib.Ctx) error {
    g.session.Delete(c, "user_id")
    c.ClearCookie("remember_token")
    g.user = nil
    return nil
}
```

## 4. User Providers

### Provider Interface

```go
type UserProvider interface {
    RetrieveByID(id interface{}) (Authenticatable, error)
    RetrieveByCredentials(credentials map[string]string) (Authenticatable, error)
    RetrieveByToken(token string) (Authenticatable, error)
    ValidateCredentials(user Authenticatable, credentials map[string]string) bool
}
```

### GORM Provider

```go
package providers

type GormUserProvider struct {
    db        *gorm.DB
    model     interface{}
    hashCheck func(hashed, plain string) bool
}

func NewGormUserProvider(db *gorm.DB, model interface{}) *GormUserProvider {
    return &GormUserProvider{
        db:        db,
        model:     model,
        hashCheck: auth.CheckPasswordHash,
    }
}

func (p *GormUserProvider) RetrieveByID(id interface{}) (Authenticatable, error) {
    user := reflect.New(reflect.TypeOf(p.model).Elem()).Interface()
    err := p.db.First(user, id).Error
    if err != nil {
        return nil, err
    }
    return user.(Authenticatable), nil
}

func (p *GormUserProvider) RetrieveByCredentials(credentials map[string]string) (Authenticatable, error) {
    user := reflect.New(reflect.TypeOf(p.model).Elem()).Interface()

    query := p.db
    for key, value := range credentials {
        if key != "password" {
            query = query.Where(fmt.Sprintf("%s = ?", key), value)
        }
    }

    err := query.First(user).Error
    if err != nil {
        return nil, err
    }

    return user.(Authenticatable), nil
}

func (p *GormUserProvider) ValidateCredentials(user Authenticatable, credentials map[string]string) bool {
    password, exists := credentials["password"]
    if !exists {
        return false
    }

    return p.hashCheck(user.GetAuthPassword(), password)
}
```

## 5. Authorization (Policies & Gates)

### Gate Manager

```go
package auth

type GateManager struct {
    gates    map[string]GateFunc
    policies map[reflect.Type]Policy
    before   []BeforeFunc
    after    []AfterFunc
}

type GateFunc func(user Authenticatable, args ...interface{}) AuthResponse
type BeforeFunc func(user Authenticatable, ability string) *AuthResponse
type AfterFunc func(user Authenticatable, ability string, result AuthResponse) AuthResponse

// Define gate
func (gm *GateManager) Define(ability string, callback GateFunc) {
    gm.gates[ability] = callback
}

// Check ability
func (gm *GateManager) Check(user Authenticatable, ability string, args ...interface{}) bool {
    response := gm.Authorize(user, ability, args...)
    return response.Allowed()
}

// Authorize (with response)
func (gm *GateManager) Authorize(user Authenticatable, ability string, args ...interface{}) AuthResponse {
    // Before callbacks
    for _, before := range gm.before {
        if response := before(user, ability); response != nil {
            return *response
        }
    }

    // Check gates
    if gate, exists := gm.gates[ability]; exists {
        response := gate(user, args...)
        return gm.runAfter(user, ability, response)
    }

    // Check policies
    if len(args) > 0 {
        model := args[0]
        if policy, exists := gm.policies[reflect.TypeOf(model)]; exists {
            response := policy.Authorize(user, ability, model)
            return gm.runAfter(user, ability, response)
        }
    }

    return gm.runAfter(user, ability, Deny("Unauthorized"))
}

func (gm *GateManager) runAfter(user Authenticatable, ability string, response AuthResponse) AuthResponse {
    for _, after := range gm.after {
        response = after(user, ability, response)
    }
    return response
}
```

### Policies

```go
package policies

type PostPolicy struct{}

func (p *PostPolicy) View(user auth.Authenticatable, post *models.Post) auth.AuthResponse {
    // Anyone can view published posts
    if post.Published {
        return auth.Allow()
    }

    // Only author can view unpublished
    if user.GetAuthIdentifier() == post.UserID {
        return auth.Allow()
    }

    return auth.Deny("You don't have permission to view this post")
}

func (p *PostPolicy) Update(user auth.Authenticatable, post *models.Post) auth.AuthResponse {
    if user.GetAuthIdentifier() == post.UserID {
        return auth.Allow()
    }

    return auth.Deny("You don't own this post")
}

func (p *PostPolicy) Delete(user auth.Authenticatable, post *models.Post) auth.AuthResponse {
    userModel := user.(*models.User)

    // Admin can delete any post
    if userModel.IsAdmin() {
        return auth.Allow()
    }

    // Owner can delete
    if user.GetAuthIdentifier() == post.UserID {
        return auth.Allow()
    }

    return auth.Deny("Insufficient permissions")
}
```

### Authorization Response

```go
package auth

type AuthResponse struct {
    allowed bool
    message string
}

func Allow() AuthResponse {
    return AuthResponse{allowed: true}
}

func Deny(message string) AuthResponse {
    return AuthResponse{allowed: false, message: message}
}

func (r AuthResponse) Allowed() bool {
    return r.allowed
}

func (r AuthResponse) Denied() bool {
    return !r.allowed
}

func (r AuthResponse) Message() string {
    return r.message
}
```

## 6. Middleware

### Auth Middleware

```go
package middleware

func Authenticate(guards ...string) glib.Middleware {
    return func(next glib.Handler) glib.Handler {
        return func(c *glib.Ctx) error {
            authManager := container.Resolve[*auth.Manager](app.Container())

            guardName := "jwt"
            if len(guards) > 0 {
                guardName = guards[0]
            }

            guard := authManager.GuardNamed(guardName)

            if err := guard.Authenticate(c); err != nil {
                return errors.Unauthorized("Unauthenticated", err)
            }

            // Store user in context
            c.SetValue("auth.user", guard.User())

            return next(c)
        }
    }
}

// Shorthand
func Auth(guards ...string) glib.Middleware {
    return Authenticate(guards...)
}
```

### Guest Middleware

```go
func Guest() glib.Middleware {
    return func(next glib.Handler) glib.Handler {
        return func(c *glib.Ctx) error {
            authManager := container.Resolve[*auth.Manager](app.Container())

            if authManager.Check() {
                return errors.Forbidden("Already authenticated", nil)
            }

            return next(c)
        }
    }
}
```

### Can Middleware

```go
func Can(ability string) glib.Middleware {
    return func(next glib.Handler) glib.Handler {
        return func(c *glib.Ctx) error {
            user := auth.User(c)

            gateManager := container.Resolve[*auth.GateManager](app.Container())

            if !gateManager.Check(user, ability) {
                return errors.Forbidden("Unauthorized action", nil)
            }

            return next(c)
        }
    }
}
```

## 7. Password Reset

### Reset Token

```go
type PasswordReset struct {
    Email     string    `gorm:"index"`
    Token     string    `gorm:"index"`
    CreatedAt time.Time
}

func (PasswordReset) TableName() string {
    return "password_resets"
}
```

### Reset Service

```go
type PasswordResetService struct {
    db     *gorm.DB
    mailer mail.Mailer
}

func (s *PasswordResetService) SendResetLink(email string) error {
    // Generate token
    token := generateResetToken()

    // Store token
    reset := &PasswordReset{
        Email: email,
        Token: hashToken(token),
    }
    s.db.Create(reset)

    // Send email
    return s.mailer.Send(email, "Reset Password", token)
}

func (s *PasswordResetService) Reset(token, password string) error {
    // Find token
    reset := &PasswordReset{}
    err := s.db.Where("token = ?", hashToken(token)).
        Where("created_at > ?", time.Now().Add(-1*time.Hour)).
        First(reset).Error

    if err != nil {
        return ErrInvalidToken
    }

    // Update user password
    user := &models.User{}
    s.db.Where("email = ?", reset.Email).First(user)

    hashed, _ := auth.HashPassword(password)
    user.Password = hashed
    s.db.Save(user)

    // Delete token
    s.db.Delete(reset)

    return nil
}
```

## 8. Example Usage

### Login Controller

```go
func (ctrl *AuthController) Login(c *glib.Ctx) error {
    var req LoginRequest
    if err := c.ValidateBody(&req); err != nil {
        return err
    }

    user, token, err := auth.Attempt(map[string]string{
        "email":    req.Email,
        "password": req.Password,
    })

    if err != nil {
        return errors.Unauthorized("Invalid credentials", err)
    }

    return c.JSON(map[string]interface{}{
        "user":  user,
        "token": token,
    })
}

func (ctrl *AuthController) Me(c *glib.Ctx) error {
    user := auth.User(c)
    return c.JSON(user)
}

func (ctrl *AuthController) Logout(c *glib.Ctx) error {
    if err := auth.Logout(); err != nil {
        return err
    }
    return c.NoContent()
}
```

### Protected Routes

```go
// routes/api.go
func API(r glib.Router) {
    // Public routes
    r.Post("/login", authController.Login)
    r.Post("/register", authController.Register)

    // Protected routes
    protected := r.Group("/", middleware.Auth())
    {
        protected.Get("/me", authController.Me)
        protected.Post("/logout", authController.Logout)

        // Posts
        protected.Post("/posts", postController.Store)
        protected.Put("/posts/{id}", postController.Update).
            Middleware(middleware.Can("update"))
        protected.Delete("/posts/{id}", postController.Destroy).
            Middleware(middleware.Can("delete"))
    }
}
```

## Success Metrics

### Phase 4 Complete When

- ✅ JWT authentication works
- ✅ Session authentication works
- ✅ Password hashing is secure
- ✅ User providers are flexible
- ✅ Policies and gates authorize actions
- ✅ Middleware protects routes
- ✅ Password reset flow works
- ✅ OAuth2 providers integrate
- ✅ All tests pass with >90% coverage
- ✅ Documentation complete
- ✅ CLI command `glib make:auth` generates scaffolding
