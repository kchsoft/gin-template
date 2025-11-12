# Pray Together API Server

Go 기반 RESTful API 서버 (Uber-style Architecture)

## 기술 스택

- **Go 1.24.5** - 최신 안정 버전
- **Gin** - HTTP 웹 프레임워크
- **GORM** - ORM 라이브러리
- **Oracle Database** - 메인 데이터베이스
- **Viper** - 설정 관리
- **JWT** - 인증 토큰
- **slog** - 구조화된 로깅

## 프로젝트 구조

**도메인별 수직 분할 구조** (Modular Monolith)

```
.
├── cmd/
│   └── server/         # 애플리케이션 진입점
│       └── main.go
│
├── internal/           # 비공개 애플리케이션 코드
│   ├── member/         # Member 도메인 (향후 추가)
│   │   ├── handler/    # HTTP 요청/응답 처리
│   │   ├── service/    # 비즈니스 로직
│   │   ├── repository/ # 데이터 접근
│   │   └── constants.go # 도메인 상수
│   │
│   ├── room/           # Room 도메인
│   │   ├── handler/    # HTTP 요청/응답 처리
│   │   ├── service/    # 비즈니스 로직
│   │   ├── repository/ # 데이터 접근
│   │   └── constants.go # 도메인 상수
│   │
│   ├── prayer/         # Prayer 도메인 (향후 추가)
│   │   ├── handler/
│   │   ├── service/
│   │   └── repository/
│   │
│   ├── model/          # 공유 도메인 Entity
│   │   ├── member.go
│   │   ├── room.go
│   │   ├── prayer.go
│   │   └── base.go
│   │
│   ├── shared/         # 공통 인프라
│   │   ├── middleware/ # HTTP 미들웨어 (CORS, JWT, Timeout)
│   │   └── database/   # 데이터베이스 연결 관리
│   │
│   ├── config/         # 설정 관리
│   └── router/         # 라우팅 설정 및 의존성 주입
│
├── pkg/                # 공개 라이브러리
│   └── server/         # 서버 Bootstrap 패턴
│
└── bin/                # 컴파일된 바이너리
```

**의존성 방향**: `Member ← Room ← Prayer` (단방향, 순환 참조 없음)

자세한 아키텍처는 [CLAUDE.md](CLAUDE.md) 참조

## 시작하기

### 필수 요구사항

- Go 1.24.5 이상
- Oracle Database 접속 정보

### 환경 설정

1. 환경 변수 파일 생성:
```bash
cp .env.example .env.local
```

2. `.env.local` 파일 수정:
```env
APP_ENV=local
APP_PORT=8080

DB_HOST=your-oracle-host
DB_SERVICE=your-service-name
DB_USERNAME=your-username
DB_PASSWORD=your-password

JWT_SECRET=your-jwt-secret-min-32-chars
JWT_EXPIRES_IN=24h
```

### 실행

```bash
# 개발 환경 (Hot Reload)
air

# Air 없이 실행
go run cmd/server/main.go -env=local

# 프로덕션 빌드
go build -o bin/server cmd/server/main.go
./bin/server -env=production
```

### Hot Reload (Air)

개발 시 파일 변경을 감지하여 자동으로 재시작:

```bash
# Air 설치 (처음 한 번만)
go install github.com/air-verse/air@latest

# 실행
air

# 또는 설정 파일 지정
air -c .air.toml
```

Air 설정은 `.air.toml` 파일에서 관리됩니다.

## 주요 기능

### 미들웨어

- **Request ID**: 모든 요청에 고유 ID 부여
- **CORS**: Cross-Origin Resource Sharing 설정
- **JWT**: 토큰 기반 인증
- **Timeout**: 30초 글로벌 timeout (Context 기반)
- **Recovery**: Panic 복구 및 로깅

### API 엔드포인트

```
GET    /api/v1/ping     # 헬스 체크
```

## 아키텍처 특징

### Uber-style Architecture
- **도메인별 수직 분할**: 각 도메인(Member, Room, Prayer)이 독립적인 모듈
- **실용적인 레이어 분리**: Handler → Service → Repository → Database
- **Service 간 의존 허용**: Room Service가 Member Repository/Service 의존 가능 (순환 참조만 금지)
- **공유 Model**: 도메인 간 Entity 참조를 위한 중앙 집중식 모델
- Go 커뮤니티 표준을 따르는 구조
- 의존성 역전 원칙 준수
- 테스트 용이성 및 빠른 개발 속도

### Context-based Timeout
- 고루틴 누수 없는 안전한 구현
- 30초 글로벌 timeout
- 각 레이어별 적절한 context 전파

### Bootstrap Pattern
- 공통 서버 설정과 애플리케이션 로직 분리
- 재사용 가능한 서버 초기화

### Dependency Injection
- 생성자를 통한 명시적 의존성 주입
- Context가 아닌 구조체 필드로 의존성 관리

### Graceful Shutdown
- 안전한 서버 종료
- 진행 중인 요청 완료 대기

## 개발 가이드

### 코드 스타일
- `gofmt` 및 `goimports` 사용
- 구조화된 로깅 (slog)
- 명시적 에러 처리

### 테스트
```bash
# 전체 테스트
go test ./...

# 커버리지 포함
go test -cover ./...
```

### 로깅
```go
slog.Info("Server started", 
    "port", cfg.Port,
    "env", cfg.Env,
)
```

## 환경별 설정

- `.env.local` - 로컬 개발
- `.env.test` - 테스트 환경
- `.env.production` - 프로덕션 환경

## 라이센스

Private