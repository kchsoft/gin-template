# Oracle Cloud DB 최소 설정 가이드 (Wallet 없이)

## 1. 필요한 패키지 설치

```bash
go get github.com/godoes/gorm-oracle
go get gorm.io/gorm
```

이 두 패키지만 있으면 됩니다. `sijms/go-ora/v2`는 자동으로 의존성으로 설치됩니다.

## 2. Oracle Cloud Console에서 연결 정보 확인

Oracle Cloud Console → Autonomous Database → DB Connection에서:
- **Host**: `adb.ap-chuncheon-1.oraclecloud.com` (예시)
- **Port**: `1522` (보통 1522)
- **Service Name**: `g0524ab680e3e6c_z5f5ees1n47gddba_high.adb.oraclecloud.com` (예시)

## 3. 환경변수 설정 (.env)

```env
# Oracle Cloud ATP 연결 정보
DB_TYPE=oracle
DB_HOST=adb.ap-chuncheon-1.oraclecloud.com
DB_PORT=1522
DB_USER=ADMIN
DB_PASSWORD=YourPassword123!
DB_SERVICE=g0524ab680e3e6c_z5f5ees1n47gddba_high.adb.oraclecloud.com
DB_SSL=true
```

## 4. 데이터베이스 연결 코드

```go
package main

import (
    "fmt"
    "log"
    "net/url"
    "os"
    
    "github.com/godoes/gorm-oracle"
    "gorm.io/gorm"
    "github.com/joho/godotenv"
)

func InitDB() (*gorm.DB, error) {
    // .env 파일 로드
    godotenv.Load()
    
    // 환경변수 읽기
    host := os.Getenv("DB_HOST")
    port := os.Getenv("DB_PORT")
    user := os.Getenv("DB_USER")
    password := os.Getenv("DB_PASSWORD")
    service := os.Getenv("DB_SERVICE")
    sslEnabled := os.Getenv("DB_SSL")
    
    // 패스워드 URL 인코딩 (특수문자 처리)
    encodedPassword := url.QueryEscape(password)
    
    // DSN 생성 (Wallet 없이 TLS만 사용)
    var dsn string
    if sslEnabled == "true" {
        dsn = fmt.Sprintf("oracle://%s:%s@%s:%s/%s?SSL=true",
            user, encodedPassword, host, port, service)
    } else {
        dsn = fmt.Sprintf("oracle://%s:%s@%s:%s/%s",
            user, encodedPassword, host, port, service)
    }
    
    log.Printf("Connecting to Oracle Cloud DB (SSL=%s)", sslEnabled)
    
    // GORM으로 연결
    db, err := gorm.Open(oracle.Open(dsn), &gorm.Config{})
    if err != nil {
        return nil, fmt.Errorf("failed to connect to database: %w", err)
    }
    
    // 연결 테스트
    sqlDB, err := db.DB()
    if err != nil {
        return nil, err
    }
    
    if err := sqlDB.Ping(); err != nil {
        return nil, fmt.Errorf("failed to ping database: %w", err)
    }
    
    log.Println("✅ Successfully connected to Oracle Cloud DB")
    return db, nil
}
```

## 5. Gin과 통합

```go
package main

import (
    "github.com/gin-gonic/gin"
)

func main() {
    // DB 초기화
    db, err := InitDB()
    if err != nil {
        log.Fatal("Failed to connect to database:", err)
    }
    
    // Gin 설정
    r := gin.Default()
    
    // DB를 핸들러에서 사용할 수 있도록 미들웨어 설정
    r.Use(func(c *gin.Context) {
        c.Set("db", db)
        c.Next()
    })
    
    // 라우트 설정
    r.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "ok", "database": "connected"})
    })
    
    r.Run(":8080")
}
```

## 6. 주의사항

### 필수 확인사항:
1. **Oracle Cloud ATP 설정**: Network ACL에서 본인 IP 허용 필요
2. **패스워드 복잡도**: 대문자, 소문자, 숫자, 특수문자 포함 필수
3. **SSL=true**: Oracle Cloud는 기본적으로 TLS 연결 필요

### 장점:
- ✅ Wallet 파일 관리 불필요
- ✅ 최소한의 의존성 (2개 패키지)
- ✅ Docker/컨테이너 배포 간편
- ✅ Oracle Client 설치 불필요

### 단점:
- ⚠️ mTLS보다 보안 수준 낮음 (단방향 인증)
- ⚠️ 일부 Oracle 고급 기능 제한

## 7. 트러블슈팅

### 연결 실패시 확인:
1. Oracle Cloud Console에서 DB 상태 확인 (Available?)
2. Network ACL 설정 확인 (본인 IP 허용?)
3. 패스워드 특수문자 URL 인코딩 확인
4. Service Name 전체 경로 확인 (_high.adb.oraclecloud.com 포함)

### 자주 발생하는 오류:
- `ORA-12541`: 호스트/포트 확인
- `ORA-01017`: 사용자명/패스워드 확인
- `ORA-12154`: Service Name 확인
- `timeout`: Network ACL 또는 방화벽 확인

## 전달 메시지 예시

다른 프로젝트의 Claude에게:

"Oracle Cloud ATP 연결을 Wallet 없이 TLS 방식으로 설정해줘. 
패키지는 `github.com/godoes/gorm-oracle`만 사용하고, 
DSN은 `oracle://USER:PASS@HOST:1522/SERVICE?SSL=true` 형식으로.
Oracle Client 설치 없이 순수 Go 드라이버로 연결하면 돼."