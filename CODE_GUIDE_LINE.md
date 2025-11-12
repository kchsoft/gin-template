# Go + Gin Uber-style 코드 가이드라인

> **대상**: Java Spring Boot 개발자를 위한 Go + Gin 아키텍처 가이드
> **목적**: 코드 리뷰 시 Uber-style 준수 여부와 Best Practice 체크

## 🏗️ 프로젝트 아키텍처: 도메인별 수직 분할

이 프로젝트는 **Uber-style + 도메인별 수직 분할** 구조를 따릅니다:

```
internal/
├── member/                    # Member 도메인
│   ├── handler/               # HTTP Layer
│   ├── service/               # Business Logic Layer
│   ├── repository/            # Data Access Layer
│   └── constants.go           # 도메인 상수
├── room/                      # Room 도메인
│   ├── handler/
│   ├── service/
│   └── repository/
├── model/                     # 공유 Entity (Member, Room, Prayer 등)
└── shared/                    # 공통 인프라 (middleware, database)
```

**핵심 원칙:**
- ✅ 도메인별 독립적인 모듈 구성
- ✅ Service 간 의존 허용 (순환 참조만 금지)
- ✅ 공유 Model로 도메인 간 Entity 참조
- ✅ 의존성 방향: `Member ← Room ← Prayer` (단방향)

---

## 📋 빠른 체크리스트

코드 리뷰 시 다음 항목들을 확인하세요:

### ✅ 아키텍처 체크리스트

- [ ] **올바른 레이어에 위치**하는가?
- [ ] **의존성 방향**이 올바른가? (Handler → Service → Repository → DB)
- [ ] **레이어 간 책임 분리**가 명확한가?
- [ ] **순환 참조**가 없는가?

### ✅ Go Best Practice

- [ ] **에러 처리**를 명시적으로 하는가?
- [ ] **Context 전파**가 올바른가?
- [ ] **nil 체크**를 하는가?
- [ ] **defer** 사용이 적절한가?
- [ ] **gofmt/goimports**를 통과하는가?

### ✅ Gin Best Practice

- [ ] **c.Request.Context()** 사용하는가?
- [ ] **gin.H vs struct** 선택이 적절한가?
- [ ] **HTTP 상태 코드**가 올바른가?
- [ ] **ShouldBindJSON** 에러 처리가 있는가?

---

## 🏗️ 레이어별 아키텍처 가이드

### 1️⃣ Handler Layer (≈ Spring Controller)

#### ✅ 올바른 예시

```go
// internal/member/handler/create.go
package handler

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "your-project/internal/member/service"  // 같은 도메인의 service
)

type Handler struct {
    memberService *service.Service  // 같은 도메인 Service
}

func NewHandler(memberService *service.Service) *Handler {
    return &Handler{
        memberService: memberService,
    }
}

func (h *Handler) Create(c *gin.Context) {
    // 1. Request DTO 파싱
    var req CreateMemberRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
        return
    }

    // 2. Context 추출
    ctx := c.Request.Context()

    // 3. Service 호출 (DTO → Model 변환)
    member, err := h.memberService.Create(ctx, req.ToModel())
    if err != nil {
        // 4. 에러 타입에 따른 HTTP 상태 코드 매핑
        switch {
        case errors.Is(err, service.ErrEmailAlreadyExists):
            c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
        default:
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
        }
        return
    }

    // 5. Response DTO 변환 및 반환
    c.JSON(http.StatusCreated, NewMemberResponse(member))
}
```

#### ❌ 잘못된 예시

```go
// ❌ 비즈니스 로직이 Handler에 있음
func (h *Handler) Create(c *gin.Context) {
    var req CreateMemberRequest
    c.ShouldBindJSON(&req)

    // ❌ 비즈니스 검증이 Handler에 있음
    if len(req.Password) < 8 {
        c.JSON(400, gin.H{"error": "password too short"})
        return
    }

    // ❌ Repository 직접 호출
    if err := h.memberRepo.Create(req.ToModel()); err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
}
```

#### 🔍 체크포인트

| 항목 | 올바른 방법 | 잘못된 방법 |
|-----|-----------|-----------|
| **의존성** | Service만 의존 | Repository 직접 의존 |
| **Context** | `c.Request.Context()` 사용 | Context 무시 |
| **에러 처리** | 에러 타입 구분 + HTTP 상태 매핑 | 모든 에러 500 |
| **검증** | 형식 검증만 (JSON validation) | 비즈니스 검증 포함 |
| **응답** | DTO 변환 후 반환 | Model 직접 반환 |

#### 🆚 Spring Boot vs Go

| Spring Boot | Go + Gin |
|-------------|----------|
| `@RestController` | `handler` 패키지 |
| `@Autowired` | 생성자 DI |
| `@PostMapping` | `router.POST("/path", handler.Method)` |
| `@RequestBody` | `c.ShouldBindJSON(&req)` |
| `ResponseEntity<T>` | `c.JSON(status, data)` |
| Exception → `@ExceptionHandler` | `err → switch/if → HTTP status` |

---

### 2️⃣ Service Layer (≈ Spring Service)

#### ✅ 올바른 예시

```go
// internal/member/service/service.go
package service

import (
    "context"
    "fmt"
    "errors"
    "golang.org/x/crypto/bcrypt"
    "your-project/internal/model"
    "your-project/internal/member/repository"  // 같은 도메인의 repository
)

type Service struct {
    memberRepo repository.Repository  // 같은 도메인 Repository 인터페이스
    // 필요시 다른 도메인 repository나 service 의존 가능 (순환 참조만 금지)
}

func NewService(memberRepo repository.Repository) *Service {
    return &Service{
        memberRepo: memberRepo,
    }
}

func (s *Service) Create(ctx context.Context, member *model.Member) (*model.Member, error) {
    // 1. 비즈니스 규칙 검증
    if member.Email == "" {
        return nil, errors.New("email is required")
    }

    // 2. 중복 체크 (비즈니스 로직)
    exists, err := s.memberRepo.ExistsByEmail(ctx, member.Email)
    if err != nil {
        return nil, fmt.Errorf("failed to check email existence: %w", err)
    }
    if exists {
        return nil, ErrEmailAlreadyExists
    }

    // 3. 비밀번호 해싱 (비즈니스 로직)
    hashedPassword, err := bcrypt.GenerateFromPassword(
        []byte(member.Password),
        bcrypt.DefaultCost,
    )
    if err != nil {
        return nil, fmt.Errorf("failed to hash password: %w", err)
    }
    member.Password = string(hashedPassword)

    // 4. Repository 호출
    if err := s.memberRepo.Create(ctx, member); err != nil {
        return nil, fmt.Errorf("failed to create member: %w", err)
    }

    return member, nil
}

// 도메인 에러 정의
var (
    ErrEmailAlreadyExists = errors.New("email already exists")
    ErrMemberNotFound     = errors.New("member not found")
)
```

#### ❌ 잘못된 예시

```go
// ❌ HTTP 처리가 Service에 있음
func (s *Service) Create(c *gin.Context) {  // ❌ gin.Context 사용
    var member model.Member
    c.ShouldBindJSON(&member)

    s.memberRepo.Create(&member)

    c.JSON(200, member)  // ❌ HTTP 응답이 Service에
}

// ❌ SQL 쿼리가 Service에 있음
func (s *Service) GetByEmail(email string) (*model.Member, error) {
    var member model.Member
    // ❌ 직접 SQL 실행
    s.db.Where("email = ?", email).First(&member)
    return &member, nil
}

// ❌ Context 무시
func (s *Service) Create(member *model.Member) error {  // ❌ Context 없음
    return s.memberRepo.Create(member)  // ❌ Context 전달 안 함
}
```

#### 🔍 체크포인트

| 항목 | 올바른 방법 | 잘못된 방법 |
|-----|-----------|-----------|
| **Context** | 첫 번째 파라미터 `ctx context.Context` | Context 없음 |
| **의존성** | Repository 인터페이스 | DB 직접 접근 |
| **에러 처리** | `fmt.Errorf("...: %w", err)` 래핑 | `err` 그대로 반환 |
| **트랜잭션** | Service에서 시작/관리 | Repository에서 시작 |
| **비즈니스 로직** | Service에 집중 | Handler나 Repository에 분산 |

#### 🔗 도메인 간 의존 (Uber-style 핵심)

**Room Service가 Member Repository/Service를 의존하는 경우:**

```go
// internal/room/service/service.go
package service

import (
    "context"
    "your-project/internal/model"
    roomRepo "your-project/internal/room/repository"
    memberRepo "your-project/internal/member/repository"  // ✅ 다른 도메인 Repository
    // 또는
    memberService "your-project/internal/member/service"  // ✅ 다른 도메인 Service
)

type Service struct {
    roomRepo      roomRepo.Repository
    memberRepo    memberRepo.Repository      // ✅ 옵션 1: Repository 의존 (데이터만)
    // 또는
    memberService *memberService.Service     // ✅ 옵션 2: Service 의존 (로직 포함)
}

func (s *Service) AddMember(ctx context.Context, roomID, memberID int64) error {
    // 옵션 1: Member Repository 사용 (데이터만 필요)
    member, err := s.memberRepo.GetByID(ctx, memberID)
    if err != nil {
        return err
    }

    // 옵션 2: Member Service 사용 (비즈니스 로직 필요)
    // if err := s.memberService.ValidateForRoom(ctx, memberID); err != nil {
    //     return err
    // }

    return s.roomRepo.AddMember(ctx, roomID, memberID)
}
```

**⚠️ 주의: 순환 참조 금지**
```go
// ❌ 절대 금지
// internal/member/service/service.go
type Service struct {
    roomService *roomService.Service  // ❌
}

// internal/room/service/service.go
type Service struct {
    memberService *memberService.Service  // ❌
}
// → import cycle error!
```

#### 🆚 Spring Boot vs Go

| Spring Boot | Go + Gin |
|-------------|----------|
| `@Service` | `service` 패키지 |
| `@Transactional` | 수동 트랜잭션 (`db.Transaction(...)`) |
| Custom Exception | `var ErrXXX = errors.New("...")` |
| `Optional<T>` | `*T, error` 반환 |
| `@Async` | `go func() { ... }()` |
| Service → Service 의존 | ✅ 허용 (순환 참조만 금지) |

---

### 3️⃣ Repository Layer (≈ Spring Repository)

#### ✅ 올바른 예시

```go
// internal/member/repository/interface.go
package repository

import (
    "context"
    "your-project/internal/model"
)

// 인터페이스 정의 (Spring의 Repository 인터페이스와 유사)
type Repository interface {
    Create(ctx context.Context, member *model.Member) error
    GetByID(ctx context.Context, id int64) (*model.Member, error)
    GetByEmail(ctx context.Context, email string) (*model.Member, error)
    Update(ctx context.Context, member *model.Member) error
    Delete(ctx context.Context, id int64) error
    ExistsByEmail(ctx context.Context, email string) (bool, error)
}

// internal/member/repository/repository.go
package repository

import (
    "context"
    "errors"
    "gorm.io/gorm"
    "your-project/internal/model"
)

type repository struct {
    db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
    return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, member *model.Member) error {
    // Context 전달
    return r.db.WithContext(ctx).Create(member).Error
}

func (r *repository) GetByID(ctx context.Context, id int64) (*model.Member, error) {
    var member model.Member
    err := r.db.WithContext(ctx).First(&member, id).Error

    // DB 에러를 도메인 에러로 변환
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, ErrMemberNotFound
        }
        return nil, err
    }

    return &member, nil
}

func (r *repository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
    var count int64
    err := r.db.WithContext(ctx).
        Model(&model.Member{}).
        Where("email = ?", email).
        Count(&count).Error

    return count > 0, err
}

// 도메인 에러
var ErrMemberNotFound = errors.New("member not found")
```

#### ❌ 잘못된 예시

```go
// ❌ 비즈니스 로직이 Repository에 있음
func (r *repository) Create(ctx context.Context, member *model.Member) error {
    // ❌ 비즈니스 검증이 Repository에
    if member.Age < 18 {
        return errors.New("too young")
    }

    // ❌ 비밀번호 해싱이 Repository에
    hashedPassword, _ := bcrypt.GenerateFromPassword(...)
    member.Password = string(hashedPassword)

    return r.db.Create(member).Error
}

// ❌ Context 무시
func (r *repository) GetByID(id int64) (*model.Member, error) {
    var member model.Member
    // ❌ WithContext 없음
    r.db.First(&member, id)
    return &member, nil
}

// ❌ 트랜잭션 시작
func (r *repository) CreateWithRoom(member *model.Member, room *model.Room) error {
    // ❌ Repository에서 트랜잭션 시작 (Service에서 해야 함)
    return r.db.Transaction(func(tx *gorm.DB) error {
        tx.Create(member)
        tx.Create(room)
        return nil
    })
}
```

#### 🔍 체크포인트

| 항목 | 올바른 방법 | 잘못된 방법 |
|-----|-----------|-----------|
| **인터페이스** | 별도 파일로 정의 | 인터페이스 없음 |
| **Context** | 모든 메서드에 `ctx` 전달 | Context 무시 |
| **에러 변환** | DB 에러 → 도메인 에러 | DB 에러 그대로 반환 |
| **책임** | 데이터 접근만 | 비즈니스 로직 포함 |
| **트랜잭션** | 전달받은 tx 사용 | Repository에서 시작 |

#### 🆚 Spring Boot vs Go

| Spring Boot | Go + Gin |
|-------------|----------|
| `extends JpaRepository<T, ID>` | Interface + 구현체 분리 |
| `findById(id)` | `GetByID(ctx, id)` |
| `existsByEmail(email)` | `ExistsByEmail(ctx, email)` |
| `@Query("SELECT ...")` | GORM 체이닝 |
| `Optional<T>` | `*T, error` |

---

### 4️⃣ Model Layer (≈ Spring Entity)

#### ✅ 올바른 예시

```go
// model/member.go
package model

import (
    "errors"
    "regexp"
    "strings"
)

// Entity 정의 (GORM 태그 사용)
type Member struct {
    ID        int64  `gorm:"primaryKey;default:MEMBER_SEQ.NEXTVAL"`
    Email     string `gorm:"column:email;type:VARCHAR2(255);not null;uniqueIndex"`
    Name      string `gorm:"column:name;type:VARCHAR2(100);not null"`
    Password  string `gorm:"column:password;type:VARCHAR2(255);not null"`

    BaseEntity  // 공통 필드 (CreatedAt, UpdatedAt 등)
}

// TableName 메서드 (GORM 테이블명 매핑)
func (*Member) TableName() string {
    return "member"
}

// Factory 메서드 (생성자 패턴)
func NewMember(name, email, password string) (*Member, error) {
    // 정규화
    name = strings.TrimSpace(name)
    email = strings.TrimSpace(strings.ToLower(email))

    // 기본 검증
    if err := validateMemberFields(name, email, password); err != nil {
        return nil, err
    }

    return &Member{
        Name:     name,
        Email:    email,
        Password: password,
    }, nil
}

// 도메인 메서드
func (m *Member) IsActive() bool {
    return !m.DeletedAt.Valid
}

func (m *Member) CanLogin() bool {
    return m.IsActive() && m.Password != ""
}

// private validation 함수
func validateMemberFields(name, email, password string) error {
    if name == "" {
        return errors.New("name is required")
    }
    if !emailRegex.MatchString(email) {
        return errors.New("invalid email format")
    }
    if len(password) < 8 {
        return errors.New("password too short")
    }
    return nil
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
```

#### ❌ 잘못된 예시

```go
// ❌ 비즈니스 로직이 Model에 과도하게 있음
func (m *Member) Save() error {  // ❌ 저장 로직이 Model에
    db := getDB()
    return db.Create(m).Error
}

func (m *Member) SendWelcomeEmail() error {  // ❌ 외부 서비스 호출이 Model에
    emailService.Send(m.Email, "Welcome!")
    return nil
}

// ❌ 복잡한 비즈니스 로직이 Model에
func (m *Member) CheckDuplicateEmail() (bool, error) {  // ❌ Repository 역할이 Model에
    var count int64
    db := getDB()
    db.Model(&Member{}).Where("email = ?", m.Email).Count(&count)
    return count > 0, nil
}
```

#### 🔍 체크포인트

| 항목 | 올바른 방법 | 잘못된 방법 |
|-----|-----------|-----------|
| **책임** | 데이터 구조 + 간단한 도메인 로직 | 복잡한 비즈니스 로직 |
| **Factory** | `NewXXX()` 생성자 함수 | 직접 구조체 생성 |
| **검증** | 기본 형식 검증만 | 복잡한 비즈니스 검증 |
| **메서드** | 판단 메서드 (`IsXXX()`, `CanXXX()`) | 동작 메서드 (`Save()`, `Delete()`) |
| **의존성** | 다른 Model만 참조 | Repository, Service 참조 |

#### 🆚 Spring Boot vs Go

| Spring Boot | Go + Gin |
|-------------|----------|
| `@Entity` | struct + GORM 태그 |
| `@Table(name = "...")` | `TableName()` 메서드 |
| `@Column(name = "...")` | `gorm:"column:..."` |
| `@Id @GeneratedValue` | `gorm:"primaryKey;autoIncrement"` |
| `@CreatedDate` | `gorm:"autoCreateTime"` or BaseEntity |
| Constructor | `NewXXX()` 함수 |

---

## 🎯 Go/Gin 특화 Best Practices

### 1. 에러 처리

#### ✅ Go 스타일

```go
// 1. 에러 정의 (package level)
var (
    ErrNotFound      = errors.New("resource not found")
    ErrAlreadyExists = errors.New("resource already exists")
)

// 2. 에러 래핑 (context 추가)
if err != nil {
    return fmt.Errorf("failed to create member: %w", err)  // %w로 원본 에러 보존
}

// 3. 에러 체크
if errors.Is(err, gorm.ErrRecordNotFound) {
    return nil, ErrNotFound
}

// 4. Custom 에러 (필요시)
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("%s: %s", e.Field, e.Message)
}
```

#### ❌ Java 스타일 (안티패턴)

```go
// ❌ Exception 던지기 (Go에서는 panic 사용 지양)
func Create(member *Member) {
    if member == nil {
        panic("member is nil")  // ❌ 일반 에러에 panic 사용
    }
}

// ❌ try-catch 패턴 흉내
func Create() {
    defer func() {  // ❌ 일반 에러 처리를 defer/recover로
        if r := recover(); r != nil {
            log.Println("recovered:", r)
        }
    }()
}
```

### 2. Context 사용

#### ✅ 올바른 Context 사용

```go
// Handler에서 추출
func (h *Handler) Create(c *gin.Context) {
    ctx := c.Request.Context()  // ✅ Gin Context에서 추출
    result, err := h.service.Create(ctx, data)
}

// Service에서 전파
func (s *Service) Create(ctx context.Context, data *Model) error {
    // Context timeout/cancel 체크
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }

    return s.repo.Create(ctx, data)  // ✅ Repository로 전파
}

// Repository에서 사용
func (r *Repository) Create(ctx context.Context, data *Model) error {
    return r.db.WithContext(ctx).Create(data).Error  // ✅ DB에 전달
}
```

#### ❌ 잘못된 Context 사용

```go
// ❌ Context 무시
func (s *Service) Create(data *Model) error {
    return s.repo.Create(data)  // ❌ Context 없음
}

// ❌ background context 남발
func (s *Service) Create(data *Model) error {
    ctx := context.Background()  // ❌ 요청 Context 무시
    return s.repo.Create(ctx, data)
}

// ❌ Gin Context를 Service에 전달
func (h *Handler) Create(c *gin.Context) {
    h.service.Create(c, data)  // ❌ gin.Context 전달 (c.Request.Context() 사용해야)
}
```

### 3. Nil 체크

#### ✅ 올바른 Nil 체크

```go
func (s *Service) Create(ctx context.Context, member *model.Member) (*model.Member, error) {
    // 1. 포인터 nil 체크
    if member == nil {
        return nil, errors.New("member is nil")
    }

    // 2. Repository 호출 후 nil 체크
    result, err := s.repo.GetByEmail(ctx, member.Email)
    if err != nil {
        return nil, err
    }
    if result != nil {  // ✅ nil 체크
        return nil, ErrAlreadyExists
    }

    return member, nil
}
```

### 4. Pointer vs Value

#### 📌 일반 가이드라인

```go
// 구조체가 작고 불변: Value
type Point struct {
    X, Y int
}
func Distance(p Point) float64 { ... }  // ✅ Value

// 구조체가 크거나 수정 필요: Pointer
type Member struct {
    ID    int64
    Email string
    Name  string
    // ... 많은 필드
}
func Update(m *Member) error { ... }  // ✅ Pointer

// Repository 반환
func (r *Repo) GetByID(ctx context.Context, id int64) (*Member, error) {
    // ✅ 포인터 반환 (nil 가능, 수정 가능)
    return &member, nil
}
```

### 5. Defer 활용

#### ✅ Defer 올바른 사용

```go
// 1. 리소스 정리
func ProcessFile(filename string) error {
    file, err := os.Open(filename)
    if err != nil {
        return err
    }
    defer file.Close()  // ✅ 함수 종료 시 자동 close

    // 파일 처리...
    return nil
}

// 2. Lock 해제
func (s *Service) UpdateSafely() {
    s.mu.Lock()
    defer s.mu.Unlock()  // ✅ 함수 종료 시 자동 unlock

    // Critical section...
}

// 3. 트랜잭션 롤백 (GORM 예시는 자동이지만 수동 시)
func (s *Service) CreateWithTransaction(ctx context.Context) error {
    tx := s.db.Begin()
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
        }
    }()

    // 작업...
    return tx.Commit().Error
}
```

---

## 🚨 일반적인 실수 (Anti-patterns)

### 1. ❌ Handler에서 DB 직접 접근

```go
// ❌ 잘못됨
func (h *Handler) Create(c *gin.Context) {
    db := c.MustGet("db").(*gorm.DB)  // ❌
    var member model.Member
    c.ShouldBindJSON(&member)
    db.Create(&member)
    c.JSON(200, member)
}

// ✅ 올바름
func (h *Handler) Create(c *gin.Context) {
    var req CreateRequest
    c.ShouldBindJSON(&req)

    ctx := c.Request.Context()
    member, err := h.memberService.Create(ctx, req.ToModel())
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    c.JSON(201, NewMemberResponse(member))
}
```

### 2. ❌ Context에 값 저장 (DI 대신)

```go
// ❌ 잘못됨 - Middleware에서
func DBMiddleware(db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Set("db", db)  // ❌ Context에 DB 저장
        c.Next()
    }
}

// Handler에서
func (h *Handler) Create(c *gin.Context) {
    db := c.MustGet("db").(*gorm.DB)  // ❌ Context에서 DB 꺼내기
}

// ✅ 올바름 - 생성자 DI
type Handler struct {
    memberService *service.MemberService  // ✅ 구조체 필드로 의존성
}

func NewHandler(memberService *service.MemberService) *Handler {
    return &Handler{memberService: memberService}
}
```

### 3. ❌ 모든 에러를 500으로 반환

```go
// ❌ 잘못됨
func (h *Handler) GetByID(c *gin.Context) {
    member, err := h.service.GetByID(c.Request.Context(), id)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})  // ❌ 모두 500
        return
    }
}

// ✅ 올바름
func (h *Handler) GetByID(c *gin.Context) {
    member, err := h.service.GetByID(c.Request.Context(), id)
    if err != nil {
        switch {
        case errors.Is(err, service.ErrNotFound):
            c.JSON(404, gin.H{"error": "Member not found"})
        case errors.Is(err, service.ErrInvalidID):
            c.JSON(400, gin.H{"error": "Invalid ID format"})
        default:
            c.JSON(500, gin.H{"error": "Internal server error"})
        }
        return
    }
    c.JSON(200, member)
}
```

### 4. ❌ Panic 남발

```go
// ❌ 잘못됨
func (s *Service) Create(member *Member) {
    if member == nil {
        panic("member is nil")  // ❌ 일반 에러에 panic
    }
}

// ✅ 올바름
func (s *Service) Create(ctx context.Context, member *Member) error {
    if member == nil {
        return errors.New("member is nil")  // ✅ error 반환
    }
    return nil
}

// ✅ Panic은 복구 불가능한 상황에만
func init() {
    if os.Getenv("REQUIRED_ENV") == "" {
        panic("REQUIRED_ENV is not set")  // ✅ 초기화 실패
    }
}
```

### 5. ❌ ORM Model을 API Response로 직접 사용

```go
// ❌ 잘못됨
func (h *Handler) GetByID(c *gin.Context) {
    member, _ := h.service.GetByID(c.Request.Context(), id)
    c.JSON(200, member)  // ❌ Password 같은 민감 정보 노출
}

// ✅ 올바름
type MemberResponse struct {
    ID    int64  `json:"id"`
    Email string `json:"email"`
    Name  string `json:"name"`
    // Password 제외
}

func NewMemberResponse(m *model.Member) *MemberResponse {
    return &MemberResponse{
        ID:    m.ID,
        Email: m.Email,
        Name:  m.Name,
    }
}

func (h *Handler) GetByID(c *gin.Context) {
    member, _ := h.service.GetByID(c.Request.Context(), id)
    c.JSON(200, NewMemberResponse(member))  // ✅ DTO 변환
}
```

---

## 📝 코드 리뷰 체크리스트 (요약)

### 전체 구조 (도메인별 수직 분할)

```
✅ 파일이 올바른 패키지에 위치하는가?
   - Handler → internal/{domain}/handler/      예) internal/member/handler/
   - Service → internal/{domain}/service/      예) internal/member/service/
   - Repository → internal/{domain}/repository/ 예) internal/member/repository/
   - Model → internal/model/                    (공유 Entity)
   - Shared → internal/shared/                  (공통 인프라: middleware, database)

✅ 의존성 방향이 올바른가?
   Handler → Service → Repository → Database
   도메인 간: Member ← Room ← Prayer (단방향, 순환 참조 금지)

✅ Service 간 의존이 올바른가?
   - 같은 도메인 Repository: ✅ 허용
   - 다른 도메인 Repository: ✅ 허용 (데이터만 필요)
   - 다른 도메인 Service: ✅ 허용 (비즈니스 로직 필요)
   - 순환 참조: ❌ 절대 금지

✅ 순환 참조가 없는가?
   import cycle 체크
```

### Handler

```
✅ Service만 의존하는가? (Repository 직접 접근 X)
✅ c.Request.Context() 사용하는가?
✅ ShouldBindJSON 에러 처리가 있는가?
✅ 에러 타입별 HTTP 상태 코드 매핑하는가?
✅ Response DTO로 변환하는가? (Model 직접 반환 X)
✅ 비즈니스 로직이 없는가?
```

### Service

```
✅ 첫 번째 파라미터가 context.Context인가?
✅ Repository 인터페이스를 의존하는가?
✅ 에러를 fmt.Errorf("...: %w", err)로 래핑하는가?
✅ 비즈니스 로직이 집중되어 있는가?
✅ HTTP 관련 코드가 없는가? (gin.Context 사용 X)
✅ SQL 쿼리가 없는가? (Repository 사용)
```

### Repository

```
✅ 인터페이스가 정의되어 있는가?
✅ 모든 메서드가 context.Context를 받는가?
✅ db.WithContext(ctx) 사용하는가?
✅ DB 에러를 도메인 에러로 변환하는가?
✅ 비즈니스 로직이 없는가?
✅ 트랜잭션을 시작하지 않는가? (Service에서 시작)
```

### Model

```
✅ GORM 태그가 올바른가?
✅ TableName() 메서드가 있는가?
✅ Factory 함수 (NewXXX)가 있는가?
✅ 기본 검증만 하는가? (복잡한 비즈니스 로직 X)
✅ 외부 의존성이 없는가? (DB, Service 호출 X)
```

### Go Best Practice

```
✅ gofmt/goimports를 통과하는가?
✅ 에러를 명시적으로 처리하는가? (err 무시 X)
✅ nil 체크를 하는가?
✅ defer를 적절히 사용하는가?
✅ panic을 남발하지 않는가?
✅ Context를 전파하는가?
```

---

## 🎓 Spring Boot 개발자를 위한 용어 매핑

| Spring Boot | Go + Gin | 설명 |
|-------------|----------|------|
| `@RestController` | Handler struct | HTTP 요청 처리 |
| `@Service` | Service struct | 비즈니스 로직 |
| `@Repository` | Repository interface | 데이터 접근 |
| `@Entity` | Model struct + GORM | 도메인 엔티티 |
| `@Autowired` | Constructor DI | 의존성 주입 |
| `@RequestBody` | `ShouldBindJSON(&req)` | Request body 파싱 |
| `ResponseEntity<T>` | `c.JSON(status, data)` | HTTP 응답 |
| `@Transactional` | `db.Transaction(func(tx) {...})` | 트랜잭션 |
| `Optional<T>` | `*T, error` | Nullable 타입 |
| Exception | `error` interface | 에러 처리 |
| `throw new XXXException()` | `return errors.New("...")` | 에러 반환 |
| `@ExceptionHandler` | Handler에서 switch/if | 에러 처리 |
| Lombok `@Data` | struct + tags | DTO/Entity 정의 |
| `@Valid` | `ShouldBindJSON` + validation | 입력 검증 |

---

## 🔗 참고 자료

- **프로젝트 아키텍처**: [CLAUDE.md](CLAUDE.md)
- **상세 레이어 가이드**: [internal/README.md](internal/README.md)
- **Uber Go Style Guide**: https://github.com/uber-go/guide
- **Effective Go**: https://golang.org/doc/effective_go.html
- **GORM 문서**: https://gorm.io/docs/
- **Gin 문서**: https://gin-gonic.com/docs/

---

## 💬 코드 리뷰 요청 템플릿

코드 리뷰를 요청할 때 다음과 같이 물어보세요:

```
@CODE_GUIDE_LINE.md 를 참고해서 내 코드를 리뷰해줘.

[리뷰 받고 싶은 파일 경로]
1. internal/member/handler/create.go
2. internal/member/service/service.go
3. internal/room/service/service.go

[확인하고 싶은 사항]
1. Uber-style 아키텍처를 잘 따르고 있나?
2. 도메인별 수직 분할 구조가 올바른가?
3. Service 간 의존성이 적절한가? (순환 참조 없는가?)
4. Best Practice를 준수하고 있나?
5. Go 관용적 표현(idiomatic)을 사용하고 있나?
```

---

*Happy Coding! 🚀*
