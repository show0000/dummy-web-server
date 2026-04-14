# dummy-web-server

YAML 기반 동적 Mock API 서버. 외부 시스템 연동 테스트 시 실제 서버 없이 API 동작을 재현합니다.

- YAML 한 장으로 API 정의 (코드 작성 불필요)
- JavaScript 스크립트로 동적 응답 제어
- 의존성 없는 단일 실행 바이너리

## 빌드

```bash
go build -o dummy-web-server ./src
```

크로스 컴파일:
```bash
# Linux
GOOS=linux   GOARCH=amd64 go build -o dummy-web-server         ./src
GOOS=linux   GOARCH=arm64 go build -o dummy-web-server-arm64   ./src

# Windows
GOOS=windows GOARCH=amd64 go build -o dummy-web-server.exe     ./src
GOOS=windows GOARCH=386   go build -o dummy-web-server-x86.exe ./src
GOOS=windows GOARCH=arm64 go build -o dummy-web-server-arm64.exe ./src

# macOS
GOOS=darwin  GOARCH=amd64 go build -o dummy-web-server-intel   ./src
GOOS=darwin  GOARCH=arm64 go build -o dummy-web-server-arm64   ./src
```

## 실행

```bash
./dummy-web-server
```

### CLI 옵션

```
--config string        config.yaml 경로 (default "config.yaml")
--port int             서버 포트 (config.yaml 오버라이드)
--enable-login y|n     JWT 로그인 활성화 (config.yaml 오버라이드)
--help                 도움말 표시
```

```bash
# 포트 변경
./dummy-web-server --port 3000

# JWT 로그인 활성화
./dummy-web-server --enable-login y

# 조합
./dummy-web-server --port 9090 --enable-login y --config ./my-config.yaml
```

## 설정 파일

### config.yaml

서버 전체 설정을 정의합니다.

```yaml
server:
  port: 8080                          # 서버 포트

jwt:
  enabled: false                      # JWT 인증 활성화 여부
  secret: "change-me-to-a-secure-secret"  # JWT 서명 시크릿
  accessTokenExpiry: "15m"            # Access Token 만료 시간
  refreshTokenExpiry: "168h"          # Refresh Token 만료 시간 (7일)

paths:
  apis: "./apis.yaml"                 # API 정의 파일 경로
  storage: "./storage"                # 파일 업로드 저장 디렉토리
  scripts: "./scripts"                # 외부 스크립트 파일 디렉토리
```

### apis.yaml

API 엔드포인트를 정의합니다. 서버 시작 시 로드되어 동적으로 라우트에 등록됩니다.

```yaml
apis:
  - entrypoint: string     # (필수) 라우트 경로. Path Variable은 {name} 형식
    method: string          # (필수) HTTP 메서드 (GET, POST, PUT, DELETE, PATCH)
    description: string     # (선택) 엔드포인트 설명
    auth: bool              # (선택) JWT 인증 적용 여부. 기본값: true
    validation:             # (선택) Request Body JSON Schema 검증
      schema: object        #   JSON Schema 인라인 정의
      schemaFile: string    #   또는 외부 JSON Schema 파일 경로
    script: string          # (택1) 인라인 JavaScript 코드
    scriptFile: string      # (택1) 외부 JavaScript 파일 경로
```

## 스크립트 API

각 엔드포인트의 스크립트에는 `req`(요청)와 `res`(응답) 객체가 주입됩니다.

### req (읽기 전용)

| 속성 | 타입 | 설명 |
|------|------|------|
| `req.body` | Object | JSON 파싱된 요청 바디 |
| `req.query` | Object | URL 쿼리 파라미터 (`?key=value`) |
| `req.params` | Object | Path Variable (`/users/{id}` → `req.params.id`) |
| `req.headers` | Object | 요청 헤더 (키는 소문자 정규화) |
| `req.files` | Array | 업로드된 파일 목록 (fieldName, fileName, size, savedPath) |

### res (응답 제어)

| 메서드 | 설명 |
|--------|------|
| `res.json(status, body)` | JSON 응답 전송 |
| `res.file(path)` | 파일 다운로드 응답 (MIME 자동 추론) |
| `res.setHeader(key, value)` | 응답 헤더 설정 (`res.json`/`res.file` 전에 호출) |

### 예시

```yaml
apis:
  # GET - Path Variable + 조건 분기
  - entrypoint: /api/users/{id}
    method: GET
    description: 사용자 조회
    script: |
      if (req.params.id === "0") {
        res.json(404, { error: "not found" });
      } else {
        res.json(200, { id: req.params.id, name: "User " + req.params.id });
      }

  # POST - Request Body 검증 + 응답
  - entrypoint: /api/users
    method: POST
    description: 사용자 생성
    script: |
      res.json(201, { created: req.body.name });
    validation:
      schema:
        type: object
        required: [name, email]
        properties:
          name:
            type: string
          email:
            type: string

  # GET - 커스텀 헤더 + 쿼리 파라미터
  - entrypoint: /api/search
    method: GET
    description: 검색 API
    script: |
      res.setHeader("X-Total-Count", "100");
      res.json(200, { query: req.query.q, page: req.query.page });

  # GET - 파일 다운로드 (인증 불필요)
  - entrypoint: /api/download/{fileName}
    method: GET
    description: 파일 다운로드
    auth: false
    script: |
      res.file("./storage/" + req.params.fileName);

  # POST - 파일 업로드
  - entrypoint: /api/upload
    method: POST
    description: 파일 업로드
    script: |
      if (req.files.length === 0) {
        res.json(400, { error: "no files" });
      } else {
        res.json(200, {
          fileName: req.files[0].fileName,
          size: req.files[0].size,
          savedPath: req.files[0].savedPath
        });
      }

  # 외부 스크립트 파일 사용
  - entrypoint: /api/complex
    method: POST
    description: 복잡한 로직
    scriptFile: ./scripts/complex-handler.js
```

## 내장 엔드포인트

### /health

서버 상태 확인.

```bash
curl http://localhost:8080/health
# {"status":"ok"}
```

### /_explorer

브라우저에서 `http://localhost:8080/_explorer` 접속 시 Swagger UI와 유사한 웹 UI를 제공합니다.

- API 목록 (method, path, description)
- 테스트 콘솔 (params, body, headers 입력 후 요청 실행)
- 응답 뷰어 (status, headers, body 포맷팅)
- cURL 명령어 복사
- JWT 활성 시 로그인 후 토큰 자동 주입

### /_utils/schema

JSON을 전송하면 JSON Schema를 생성하여 반환합니다.

```bash
curl -X POST http://localhost:8080/_utils/schema \
  -H "Content-Type: application/json" \
  -d '{"name": "Alice", "age": 30, "tags": ["go", "mock"]}'
```

응답:
```json
{
  "type": "object",
  "required": ["age", "name", "tags"],
  "properties": {
    "name": { "type": "string" },
    "age": { "type": "integer" },
    "tags": { "type": "array", "items": { "type": "string" } }
  }
}
```

### /_auth/* (JWT 활성 시)

JWT를 활성화하면(`--enable-login y` 또는 `config.yaml`에서 `jwt.enabled: true`) 인증 엔드포인트가 활성화됩니다.

**로그인:**

Request:
```http
POST /_auth/login
Content-Type: application/json

{
  "username": "string",
  "password": "string"
}
```

Response (200 OK):
```json
{
  "accessToken": "eyJhbGciOiJIUzI1NiIs...",
  "refreshToken": "eyJhbGciOiJIUzI1NiIs..."
}
```

Error (400 Bad Request):
```json
{ "error": "username and password are required" }
```

```bash
curl -X POST http://localhost:8080/_auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "testuser", "password": "testpass"}'
```

**동작 상세:**
- Mock 서버 특성상 **모든 non-empty username/password 조합이 허용**됩니다 (실제 인증 검증 없음).
- 토큰의 `sub` claim에 username이 저장됩니다.
- Access Token 만료: 기본 15분 (`jwt.accessTokenExpiry`로 조정).
- Refresh Token 만료: 기본 168시간/7일 (`jwt.refreshTokenExpiry`로 조정).
- 토큰 저장소는 in-memory. 서버 재시작 시 모든 세션 초기화.

**인증이 필요한 API 호출:**
```bash
curl http://localhost:8080/api/users/1 \
  -H "Authorization: Bearer eyJhbG..."
```

**토큰 갱신 (Refresh Token Rotation):**
```bash
curl -X POST http://localhost:8080/_auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refreshToken": "eyJhbG..."}'
```

**로그아웃:**
```bash
curl -X POST http://localhost:8080/_auth/logout \
  -H "Authorization: Bearer eyJhbG..." \
  -H "Content-Type: application/json" \
  -d '{"refreshToken": "eyJhbG..."}'
```

> Mock 서버 특성상 로그인 시 모든 username/password 조합이 허용됩니다.
> `auth: false`로 설정된 엔드포인트는 토큰 없이 접근 가능합니다.

## 디렉토리 구조

```
dummy-web-server/
├── config.yaml          # 서버 설정
├── apis.yaml            # API 엔드포인트 정의
├── storage/             # 파일 업로드 저장소
├── scripts/             # 외부 스크립트 파일
└── src/                 # Go 소스 코드
```
