# 🏗️ Architecture Overview - Uber Style

This project follows **Uber-style Architecture** - a pragmatic, battle-tested approach that simplifies Clean Architecture principles for real-world Go applications.

## 📁 Project Structure

```
.
├── cmd/
│   └── server/          # Application entry points
│       └── main.go      # Main application
├── internal/            # Private application code (cannot be imported)
│   ├── config/          # Configuration management
│   ├── handler/         # HTTP request handlers (presentation layer)
│   │   └── middleware/  # HTTP middleware (auth, cors, logging)
│   ├── service/         # Business logic layer (use cases)
│   ├── repository/      # Data access abstractions
│   ├── model/           # Domain models and entities
│   ├── router/          # HTTP routing setup
│   └── infrastructure/  # External dependencies
│       └── database/    # Database connections and migrations
├── pkg/                 # Public packages (can be imported by other projects)
│   └── server/          # Reusable server bootstrapping
└── bin/                 # Compiled binaries
```

## 🎯 Core Principles

### Dependency Flow
```
[HTTP Request] → handler → service → repository → database
                    ↓         ↓          ↓
                  model     model      model
```

**Key Rule**: Dependencies flow inward. Outer layers depend on inner layers, never the reverse.

## 📦 Layer Responsibilities

### 1. Handler Layer (`internal/handler/`)
**Purpose**: HTTP request/response handling, input validation, authentication

```go
// handler/user/create.go
type Handler struct {
    userService service.UserService
}

func (h *Handler) CreateUser(c *gin.Context) {
    var req CreateUserRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "Invalid request"})
        return
    }
    
    user, err := h.userService.Create(c.Request.Context(), req.ToModel())
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(201, NewUserResponse(user))
}
```

**Responsibilities**:
- Parse HTTP requests
- Validate input format
- Call appropriate service methods
- Format HTTP responses
- Handle HTTP-specific concerns (headers, status codes)

**NOT Allowed**:
- Business logic
- Direct database access
- Complex validations

### 2. Service Layer (`internal/service/`)
**Purpose**: Business logic, orchestration, transaction management

```go
// service/user/service.go
type UserService struct {
    userRepo   repository.UserRepository
    emailSvc   external.EmailService
    validator  validator.Validator
}

func (s *UserService) Create(ctx context.Context, user *model.User) (*model.User, error) {
    // Business validation
    if err := s.validator.ValidateUser(user); err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }
    
    // Check business rules
    exists, err := s.userRepo.ExistsByEmail(ctx, user.Email)
    if exists {
        return nil, ErrUserAlreadyExists
    }
    
    // Create user
    user.ID = uuid.New().String()
    user.CreatedAt = time.Now()
    
    if err := s.userRepo.Create(ctx, user); err != nil {
        return nil, fmt.Errorf("failed to create user: %w", err)
    }
    
    // Send welcome email (async)
    go s.emailSvc.SendWelcome(user.Email)
    
    return user, nil
}
```

**Responsibilities**:
- Implement business rules
- Orchestrate multiple repositories
- Handle transactions
- Business validations
- Integration with external services

**NOT Allowed**:
- HTTP concerns
- SQL queries
- Framework-specific code

### 3. Repository Layer (`internal/repository/`)
**Purpose**: Data access abstraction, CRUD operations

```go
// repository/user/interface.go
type UserRepository interface {
    Create(ctx context.Context, user *model.User) error
    GetByID(ctx context.Context, id string) (*model.User, error)
    GetByEmail(ctx context.Context, email string) (*model.User, error)
    Update(ctx context.Context, user *model.User) error
    Delete(ctx context.Context, id string) error
    ExistsByEmail(ctx context.Context, email string) (bool, error)
}

// repository/user/postgres.go
type postgresUserRepo struct {
    db *gorm.DB
}

func (r *postgresUserRepo) Create(ctx context.Context, user *model.User) error {
    return r.db.WithContext(ctx).Create(user).Error
}

func (r *postgresUserRepo) GetByID(ctx context.Context, id string) (*model.User, error) {
    var user model.User
    err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, ErrUserNotFound
        }
        return nil, err
    }
    return &user, nil
}
```

**Responsibilities**:
- Define data access interfaces
- Implement database operations
- Handle database-specific errors
- Query optimization

**NOT Allowed**:
- Business logic
- HTTP concerns
- Cross-repository transactions

### 4. Model Layer (`internal/model/`)
**Purpose**: Domain entities, value objects, domain errors

```go
// model/user.go
type User struct {
    ID        string    `json:"id" gorm:"primaryKey"`
    Email     string    `json:"email" gorm:"uniqueIndex"`
    Name      string    `json:"name"`
    Password  string    `json:"-" gorm:"column:password_hash"`
    Status    Status    `json:"status"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

// Domain methods
func (u *User) IsActive() bool {
    return u.Status == StatusActive
}

func (u *User) CanLogin() bool {
    return u.IsActive() && u.Password != ""
}

// model/errors.go
var (
    ErrUserNotFound      = errors.New("user not found")
    ErrUserAlreadyExists = errors.New("user already exists")
    ErrInvalidCredentials = errors.New("invalid credentials")
)
```

**Responsibilities**:
- Define core business entities
- Encapsulate domain logic
- Define domain-specific errors

**NOT Allowed**:
- Framework annotations (except for serialization)
- Database operations
- External service calls

### 5. Infrastructure Layer (`internal/infrastructure/`)
**Purpose**: Framework integration, external services

```go
// infrastructure/database/connection.go
func NewPostgresDB(cfg *config.DatabaseConfig) (*gorm.DB, error) {
    dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s",
        cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database)
    
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
        Logger: logger.Default.LogMode(logger.Info),
    })
    
    if err != nil {
        return nil, fmt.Errorf("failed to connect database: %w", err)
    }
    
    // Connection pool settings
    sqlDB, _ := db.DB()
    sqlDB.SetMaxIdleConns(10)
    sqlDB.SetMaxOpenConns(100)
    
    return db, nil
}
```

## 🔄 Request Flow Example

```
1. HTTP POST /users
   ↓
2. Router → UserHandler.CreateUser()
   ↓
3. Handler validates request body
   ↓
4. Handler calls UserService.Create()
   ↓
5. Service applies business rules
   ↓
6. Service calls UserRepository.Create()
   ↓
7. Repository saves to database
   ↓
8. Service returns User model
   ↓
9. Handler formats response
   ↓
10. HTTP 201 Created with user JSON
```

## 🚀 Why Uber Style?

### vs Traditional Clean Architecture

| Aspect | Clean Architecture | Uber Style | Benefit |
|--------|-------------------|------------|---------|
| Terminology | UseCase, Interactor, Presenter | Service, Handler, Repository | More intuitive |
| Layers | 4-5 layers | 3 main layers | Simpler to understand |
| Interfaces | Everything has interface | Pragmatic interfaces | Less boilerplate |
| Presentation | Complex presenter pattern | Direct JSON response | Faster development |

### Real-world Benefits

1. **Battle-tested at Scale**
   - Used at Uber handling billions of requests
   - Proven in high-traffic production environments

2. **Developer Friendly**
   - Intuitive naming (service vs usecase)
   - Less abstraction layers
   - Faster onboarding

3. **Go Idiomatic**
   - Follows Go community standards
   - Works well with Go's implicit interfaces
   - Minimal magic, maximum clarity

4. **Microservice Ready**
   - Easy to extract services
   - Clear boundaries
   - Consistent patterns

## 🧪 Testing Strategy

### Unit Tests
```go
// service/user/service_test.go
func TestUserService_Create(t *testing.T) {
    mockRepo := &MockUserRepository{}
    mockEmail := &MockEmailService{}
    
    service := NewUserService(mockRepo, mockEmail)
    
    mockRepo.On("ExistsByEmail", "test@example.com").Return(false, nil)
    mockRepo.On("Create", mock.Anything).Return(nil)
    mockEmail.On("SendWelcome", "test@example.com").Return(nil)
    
    user, err := service.Create(context.Background(), &model.User{
        Email: "test@example.com",
        Name:  "Test User",
    })
    
    assert.NoError(t, err)
    assert.NotEmpty(t, user.ID)
    mockRepo.AssertExpectations(t)
}
```

### Integration Tests
```go
// handler/user/handler_test.go
func TestCreateUser_Integration(t *testing.T) {
    // Setup test database
    db := setupTestDB()
    defer cleanupTestDB(db)
    
    // Setup dependencies
    repo := repository.NewUserRepository(db)
    svc := service.NewUserService(repo)
    handler := NewHandler(svc)
    
    // Setup router
    router := gin.New()
    router.POST("/users", handler.CreateUser)
    
    // Test request
    body := `{"email":"test@example.com","name":"Test User"}`
    req := httptest.NewRequest("POST", "/users", strings.NewReader(body))
    w := httptest.NewRecorder()
    
    router.ServeHTTP(w, req)
    
    assert.Equal(t, 201, w.Code)
}
```

## 📚 Best Practices

### Do's ✅
- Keep business logic in services
- Use dependency injection
- Return errors, don't panic
- Use context for cancellation
- Mock interfaces for testing
- Keep handlers thin

### Don'ts ❌
- Don't skip service layer for simple CRUD
- Don't put SQL in services
- Don't use ORM models as API responses
- Don't create circular dependencies
- Don't ignore error handling

## 🔮 Future Growth Path

As the application grows, transition from layer-based to domain-based organization:

```
internal/
├── user/                # User domain
│   ├── handler.go
│   ├── service.go
│   ├── repository.go
│   └── model.go
├── order/               # Order domain
│   ├── handler.go
│   ├── service.go
│   ├── repository.go
│   └── model.go
└── shared/              # Shared utilities
    ├── middleware/
    └── database/
```

## 📖 References

- [Uber Go Style Guide](https://github.com/uber-go/guide)
- [Uber's Go Proverbs](https://go-proverbs.github.io/)
- [Effective Go](https://golang.org/doc/effective_go.html)

---

*"Clear is better than clever" - Rob Pike*

*This architecture prioritizes maintainability, clarity, and real-world pragmatism over theoretical purity.*