# Internal Package Structure

Uber-style Architecture를 따르는 **도메인별 수직 분할 구조** (Modular Monolith)입니다.

```
internal/
├── member/           # Member 도메인
│   ├── handler/      # HTTP 요청/응답 처리
│   ├── service/      # 비즈니스 로직
│   ├── repository/   # 데이터 접근
│   └── constants.go  # 도메인 상수
│
├── room/             # Room 도메인
│   ├── handler/
│   ├── service/
│   ├── repository/
│   └── constants.go
│
├── prayer/           # Prayer 도메인 (향후 추가)
│   ├── handler/
│   ├── service/
│   └── repository/
│
├── model/            # 공유 도메인 Entity
│   ├── member.go
│   ├── room.go
│   ├── prayer.go
│   └── base.go
│
├── shared/           # 공통 인프라
│   ├── middleware/   # HTTP 미들웨어 (CORS, JWT, Timeout)
│   └── database/     # 데이터베이스 연결 관리
│
├── config/           # 설정 관리
└── router/           # 라우팅 설정 및 의존성 주입
```

**의존성 방향**: `Member ← Room ← Prayer` (단방향, 순환 참조 없음)

## 의존성 흐름

```
[HTTP Request] → handler → service → repository → database
                    ↓         ↓          ↓
                  model     model      model
```

**핵심 원칙**: 의존성은 안쪽으로만 흐릅니다. 외부 레이어는 내부 레이어에 의존하지만, 그 반대는 안 됩니다.

---

## 레이어별 설명

각 도메인은 독립적인 Handler → Service → Repository 레이어를 가집니다.

### 1. Handler Layer (각 도메인의 `/handler`)

**책임**: HTTP 요청/응답 처리, 입력 검증, 인증

```go
// internal/member/handler/create.go
type Handler struct {
    memberService *service.MemberService
}

func (h *Handler) CreateMember(c *gin.Context) {
    var req CreateMemberRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "Invalid request"})
        return
    }

    member, err := h.memberService.Create(c.Request.Context(), req.ToModel())
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    c.JSON(201, NewMemberResponse(member))
}
```

**할 수 있는 것**:
- HTTP 요청 파싱
- 입력 형식 검증 (JSON validation)
- Service 메서드 호출
- HTTP 응답 포맷팅
- HTTP 상태 코드 설정

**할 수 없는 것**:
- 비즈니스 로직 구현
- 직접 데이터베이스 접근
- 복잡한 검증 로직

---

### 2. Service Layer (각 도메인의 `/service`)

**책임**: 비즈니스 로직, 트랜잭션 관리, 유스케이스 구현

```go
// internal/member/service/service.go
type MemberService struct {
    memberRepo   repository.MemberRepository
    emailService external.EmailService
}

func (s *MemberService) Create(ctx context.Context, member *model.Member) (*model.Member, error) {
    // 비즈니스 검증
    exists, err := s.memberRepo.ExistsByEmail(ctx, member.Email)
    if err != nil {
        return nil, fmt.Errorf("failed to check email: %w", err)
    }
    if exists {
        return nil, errors.New("email already exists")
    }

    // 비밀번호 해싱
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(member.Password), bcrypt.DefaultCost)
    if err != nil {
        return nil, fmt.Errorf("failed to hash password: %w", err)
    }
    member.Password = string(hashedPassword)

    // 회원 생성
    if err := s.memberRepo.Create(ctx, member); err != nil {
        return nil, fmt.Errorf("failed to create member: %w", err)
    }

    // 비동기 이메일 발송
    go s.emailService.SendWelcome(member.Email)

    return member, nil
}
```

**할 수 있는 것**:
- 비즈니스 규칙 구현
- 여러 Repository 조합
- 트랜잭션 관리
- 비즈니스 검증
- 외부 서비스 호출

**할 수 없는 것**:
- HTTP 관련 코드
- SQL 쿼리 작성
- 프레임워크 의존성

---

#### Service 간 의존성 (Uber-style 특징)

Uber-style에서는 **Service가 다른 도메인의 Service/Repository를 의존할 수 있습니다** (순환 참조만 금지).

```go
// internal/room/service/service.go
type RoomService struct {
    roomRepo      repository.RoomRepository
    memberRepo    memberRepo.Repository     // Option 1: 데이터만 필요한 경우
    memberService *memberService.Service    // Option 2: 비즈니스 로직도 필요한 경우
}

func (s *RoomService) CreateRoom(ctx context.Context, req *CreateRoomRequest) (*model.Room, error) {
    // Option 1: Member Repository 직접 사용 (데이터 조회만)
    member, err := s.memberRepo.GetByID(ctx, req.OwnerID)
    if err != nil {
        return nil, fmt.Errorf("member not found: %w", err)
    }

    // Option 2: Member Service 사용 (비즈니스 로직 포함)
    if err := s.memberService.ValidateActive(ctx, req.OwnerID); err != nil {
        return nil, fmt.Errorf("member validation failed: %w", err)
    }

    room := &model.Room{
        Name:    req.Name,
        OwnerID: member.ID,
    }

    if err := s.roomRepo.Create(ctx, room); err != nil {
        return nil, err
    }

    return room, nil
}
```

**의존성 방향 규칙**:
- ✅ `Room Service` → `Member Service/Repository` (허용)
- ✅ `Prayer Service` → `Room Service`, `Member Service` (허용)
- ❌ `Member Service` → `Room Service` (순환 참조 금지)

---

### 3. Repository Layer (각 도메인의 `/repository`)

**책임**: 데이터 접근 추상화, CRUD 작업

```go
// internal/member/repository/interface.go
type MemberRepository interface {
    Create(ctx context.Context, member *model.Member) error
    GetByID(ctx context.Context, id int64) (*model.Member, error)
    GetByEmail(ctx context.Context, email string) (*model.Member, error)
    Update(ctx context.Context, member *model.Member) error
    Delete(ctx context.Context, id int64) error
    ExistsByEmail(ctx context.Context, email string) (bool, error)
}

// internal/member/repository/repository.go
type memberRepository struct {
    db *gorm.DB
}

func NewMemberRepository(db *gorm.DB) MemberRepository {
    return &memberRepository{db: db}
}

func (r *memberRepository) Create(ctx context.Context, member *model.Member) error {
    return r.db.WithContext(ctx).Create(member).Error
}

func (r *memberRepository) GetByEmail(ctx context.Context, email string) (*model.Member, error) {
    var member model.Member
    err := r.db.WithContext(ctx).Where("email = ?", email).First(&member).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, ErrMemberNotFound
        }
        return nil, err
    }
    return &member, nil
}

func (r *memberRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
    var count int64
    err := r.db.WithContext(ctx).Model(&model.Member{}).Where("email = ?", email).Count(&count).Error
    return count > 0, err
}
```

**할 수 있는 것**:
- 데이터베이스 CRUD 작업
- 쿼리 최적화
- DB 에러 변환
- 인터페이스 정의

**할 수 없는 것**:
- 비즈니스 로직
- HTTP 처리
- 트랜잭션 시작/종료 (Service 담당)

---

### 4. Model Layer (`/model` - 공유 Entity)

**책임**: 도메인 엔티티, 값 객체, 도메인 로직

모든 도메인이 공유하는 Entity를 중앙 집중식으로 관리합니다 (순환 참조 방지).

```go
// internal/model/member.go
type Member struct {
    ID        int64     `gorm:"primaryKey;default:MEMBER_SEQ.NEXTVAL"`
    Email     string    `gorm:"column:email;type:VARCHAR2(255);not null;uniqueIndex"`
    Name      string    `gorm:"column:name;type:VARCHAR2(100);not null"`
    Password  string    `gorm:"column:password;type:VARCHAR2(255);not null"`
    CreatedAt time.Time `gorm:"column:created_at;not null"`
    UpdatedAt time.Time `gorm:"column:updated_at;not null"`
}

// Factory 패턴 - 생성 로직 캡슐화
func NewMember(name, email, password string) (*Member, error) {
    name = strings.TrimSpace(name)
    email = strings.TrimSpace(strings.ToLower(email))

    if err := validateMemberFields(name, email, password); err != nil {
        return nil, err
    }

    return &Member{
        Name:     name,
        Email:    email,
        Password: password, // Service layer에서 해싱됨
    }, nil
}

// 도메인 메서드
func (m *Member) IsActive() bool {
    return !m.DeletedAt.Valid
}
```

**할 수 있는 것**:
- 도메인 엔티티 정의
- 도메인 메서드 (비즈니스 판단)
- Factory 메서드
- 기본 검증

**할 수 없는 것**:
- 데이터베이스 작업
- HTTP 처리
- 외부 서비스 호출

---

### 5. Router Layer (`/router`)

**책임**: 라우팅 설정 및 의존성 주입

```go
// internal/router/routes.go
func Setup(router *gin.Engine, cfg *config.Config, db *database.DB) {
    // Repository 초기화 (도메인별)
    memberRepo := memberRepo.NewMemberRepository(db.GetDB())
    roomRepo := roomRepo.NewRoomRepository(db.GetDB())

    // Service 초기화 (도메인 간 의존성 주입)
    memberService := memberService.NewMemberService(memberRepo)
    roomService := roomService.NewRoomService(roomRepo, memberRepo) // Room → Member 의존

    // Handler 초기화 (도메인별)
    memberHandler := memberHandler.NewMemberHandler(memberService)
    roomHandler := roomHandler.NewRoomHandler(roomService)

    // Routes (도메인별 그룹)
    v1 := router.Group("/api/v1")
    {
        members := v1.Group("/members")
        {
            members.POST("", memberHandler.Create)
            members.GET("/:id", memberHandler.GetByID)
        }

        rooms := v1.Group("/rooms")
        {
            rooms.POST("", roomHandler.Create)
            rooms.GET("/:id", roomHandler.GetByID)
        }
    }
}
```

**역할**:
- 의존성 주입 (DI)
- 라우트 그룹 설정
- 미들웨어 적용

---

### 6. Middleware (`/shared/middleware`)

**책임**: 횡단 관심사 (Cross-cutting concerns)

```go
// internal/shared/middleware/jwt.go
func JWT(secret string) gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        // JWT 검증 로직
        c.Next()
    }
}

// internal/shared/middleware/timeout.go
func Timeout(duration time.Duration) gin.HandlerFunc {
    return func(c *gin.Context) {
        ctx, cancel := context.WithTimeout(c.Request.Context(), duration)
        defer cancel()
        c.Request = c.Request.WithContext(ctx)
        c.Next()
    }
}
```

**제공 미들웨어**:
- `RequestID`: 요청 추적 ID 생성
- `CORS`: Cross-Origin Resource Sharing
- `JWT`: 인증 토큰 검증
- `Timeout`: 30초 글로벌 타임아웃
- `Recovery`: Panic 복구 및 로깅

---

### 7. Infrastructure Layer (`/shared/database`)

**책임**: 외부 시스템 연동

```go
// internal/shared/database/database.go
type DB struct {
    gormDB *gorm.DB
}

func New(cfg *config.Config) (*DB, error) {
    dsn := fmt.Sprintf("oracle://%s:%s@%s/%s",
        cfg.Database.Username,
        cfg.Database.Password,
        cfg.Database.Host,
        cfg.Database.Service,
    )

    db, err := gorm.Open(oracle.Open(dsn), &gorm.Config{
        Logger: NewGormLogger(),
    })

    if err != nil {
        return nil, fmt.Errorf("failed to connect database: %w", err)
    }

    return &DB{gormDB: db}, nil
}
```

**역할**:
- 데이터베이스 연결 관리
- 외부 API 클라이언트
- 캐시 연결
- 메시지 큐 연결

---

### 8. Constants (각 도메인의 `constants.go`)

**책임**: 도메인별 상수 및 에러 메시지

각 도메인은 자신의 상수를 독립적으로 관리합니다.

```go
// internal/member/constants.go
package member

const (
    EmailMaxLength    = 255
    NameMaxLength     = 100
    PasswordMinLength = 8
)

const (
    ErrEmailEmpty    = "member email cannot be empty"
    ErrEmailInvalid  = "member email format is invalid"
    ErrPasswordShort = "member password is too short"
)
```

```go
// internal/room/constants.go
package room

const (
    RoleOwner  = "OWNER"
    RoleAdmin  = "ADMIN"
    RoleMember = "MEMBER"
)

const (
    ErrRoomNotFound    = "room not found"
    ErrUnauthorized    = "unauthorized access"
    ErrInvalidRoomName = "invalid room name"
)
```

**사용처**:
- Model validation
- Service business rules
- 에러 메시지 표준화

---

## 주요 패턴

### 1. 의존성 주입 (Dependency Injection)

**올바른 방법** ✅
```go
// 생성자를 통한 명시적 DI
handler := NewMemberHandler(memberService)
service := NewMemberService(memberRepo)
repo := NewMemberRepository(db)
```

**잘못된 방법** ❌
```go
// Context에 DB 저장 (안티패턴)
c.Set("db", db)
db := c.MustGet("db").(*gorm.DB)
```

### 2. Context 전파

```go
// Handler → Service → Repository → Infrastructure
handler.Create(c *gin.Context)
  ctx := c.Request.Context()
  → service.Create(ctx context.Context, member)
    → repo.Create(ctx context.Context, member)
      → db.WithContext(ctx).Create(member)
```

### 3. 에러 처리

```go
// Repository: 구체적 에러 반환
if errors.Is(err, gorm.ErrRecordNotFound) {
    return nil, ErrMemberNotFound
}

// Service: 에러 래핑
if err != nil {
    return nil, fmt.Errorf("failed to create member: %w", err)
}

// Handler: HTTP 상태 코드 매핑
if errors.Is(err, ErrMemberNotFound) {
    c.JSON(404, gin.H{"error": "Member not found"})
    return
}
```

### 4. 트랜잭션 관리 (Service Layer)

```go
func (s *MemberService) CreateWithRoom(ctx context.Context, member *model.Member, room *model.Room) error {
    // Service에서 트랜잭션 시작
    return s.db.Transaction(func(tx *gorm.DB) error {
        // Repository에 트랜잭션 DB 전달
        if err := s.memberRepo.CreateWithTx(ctx, tx, member); err != nil {
            return err
        }
        if err := s.roomRepo.CreateWithTx(ctx, tx, room); err != nil {
            return err
        }
        return nil
    })
}
```

---

## 테스트 전략

### Unit Test (Service Layer)

```go
func TestMemberService_Create(t *testing.T) {
    // Mock repository
    mockRepo := &MockMemberRepository{}
    service := NewMemberService(mockRepo)

    mockRepo.On("ExistsByEmail", "test@example.com").Return(false, nil)
    mockRepo.On("Create", mock.Anything).Return(nil)

    member := &model.Member{
        Email: "test@example.com",
        Name:  "Test User",
    }

    result, err := service.Create(context.Background(), member)

    assert.NoError(t, err)
    assert.NotNil(t, result)
    mockRepo.AssertExpectations(t)
}
```

### Integration Test (Handler Layer)

```go
func TestMemberHandler_Create_Integration(t *testing.T) {
    // Setup test database
    db := setupTestDB()
    defer cleanupTestDB(db)

    // Setup dependencies
    repo := repository.NewMemberRepository(db)
    service := service.NewMemberService(repo)
    handler := handler.NewMemberHandler(service)

    // Setup router
    router := gin.New()
    router.POST("/members", handler.Create)

    // Test request
    body := `{"email":"test@example.com","name":"Test User","password":"password123"}`
    req := httptest.NewRequest("POST", "/members", strings.NewReader(body))
    w := httptest.NewRecorder()

    router.ServeHTTP(w, req)

    assert.Equal(t, 201, w.Code)
}
```

---

## Best Practices

### Do's ✅

1. **Service에 비즈니스 로직 배치**
   ```go
   // Service에서 비즈니스 규칙 검증
   if member.Age < 18 {
       return ErrMemberTooYoung
   }
   ```

2. **Repository는 단순하게 유지**
   ```go
   // Repository는 데이터 접근만
   func (r *memberRepository) GetByID(ctx context.Context, id int64) (*model.Member, error) {
       var member model.Member
       err := r.db.WithContext(ctx).First(&member, id).Error
       return &member, err
   }
   ```

3. **Handler는 얇게 유지**
   ```go
   // Handler는 요청/응답 처리만
   func (h *Handler) Create(c *gin.Context) {
       var req CreateRequest
       c.ShouldBindJSON(&req)
       result, err := h.service.Create(c.Request.Context(), req.ToModel())
       c.JSON(201, result)
   }
   ```

### Don'ts ❌

1. **Handler에서 직접 DB 접근 금지**
   ```go
   // ❌ 잘못됨
   func (h *Handler) Create(c *gin.Context) {
       db := c.MustGet("db").(*gorm.DB)
       db.Create(&member)
   }
   ```

2. **Service에서 HTTP 처리 금지**
   ```go
   // ❌ 잘못됨
   func (s *Service) Create(c *gin.Context) {
       c.JSON(200, member)
   }
   ```

3. **Repository에서 비즈니스 로직 금지**
   ```go
   // ❌ 잘못됨
   func (r *Repository) Create(member *Member) error {
       if member.Age < 18 {  // 비즈니스 로직은 Service에서!
           return errors.New("too young")
       }
       return r.db.Create(member).Error
   }
   ```

---

## 요청 플로우 예시

```
1. POST /api/v1/members
   ↓
2. internal/shared/middleware/timeout.go (30초 timeout 설정)
   ↓
3. internal/shared/middleware/request_id.go (요청 ID 생성)
   ↓
4. internal/router/routes.go (라우팅)
   ↓
5. internal/member/handler/create.go
   - JSON 파싱
   - 입력 검증
   ↓
6. internal/member/service/service.go
   - 비즈니스 규칙 검증
   - 이메일 중복 체크
   - 비밀번호 해싱
   ↓
7. internal/member/repository/repository.go
   - DB INSERT 쿼리 실행
   ↓
8. internal/shared/database/database.go
   - GORM → Oracle DB
   ↓
9. 응답 반환 (201 Created)
```

### 도메인 간 의존 플로우 예시 (Room 생성)

```
1. POST /api/v1/rooms
   ↓
2. internal/room/handler/create.go
   - JSON 파싱 및 검증
   ↓
3. internal/room/service/service.go
   ├─→ internal/member/repository/repository.go (Owner 조회)
   │   ↓
   │   internal/model/member.go (Member Entity)
   │
   └─→ internal/room/repository/repository.go (Room 생성)
       ↓
       internal/model/room.go (Room Entity)
   ↓
4. 응답 반환 (201 Created)
```

---

## 참고 자료

- 상세 아키텍처 가이드: [CLAUDE.md](../CLAUDE.md)
- Uber Go Style Guide: https://github.com/uber-go/guide
- Effective Go: https://golang.org/doc/effective_go.html

---

*"Clear is better than clever" - Go Proverb*
