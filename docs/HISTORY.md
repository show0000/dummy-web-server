# HISTORY.md

작업 이력. 최신 항목이 상단.

---

## 2026-04-14 — Phase 7: 마무리 (로깅, 빌드 검증)

- `router/logger.go`: LoggerMiddleware 구현 (method, path, status, latency).
  - statusWriter로 응답 status code 캡처.
  - main.go에서 최외곽 미들웨어로 적용 (JWT 미들웨어 위).
- 에러 핸들링: 이미 api/handler.go의 writeError로 표준화된 JSON 에러 응답 적용 완료.
- 크로스 컴파일 검증: linux/amd64, darwin/arm64, windows/amd64 모두 성공.
- 단위 테스트 2건, 전체 98건 통과.

---

## 2026-04-14 — Phase 6: API Explorer (Built-in Web UI)

- `explorer/embed.go`: `go:embed static/*`로 HTML/CSS/JS 바이너리 내장.
- `explorer/handler.go`: /_explorer (index.html), /_explorer/apis (JSON), 정적 파일 서빙.
- `explorer/static/index.html`: 2-패널 레이아웃 (API 목록 + 테스트 콘솔).
- `explorer/static/style.css`: method 뱃지 색상, 모달, 반응형 레이아웃.
- `explorer/static/app.js`: 
  - API 목록 렌더링 (method, path, description, auth 표시).
  - 테스트 콘솔: path params, query params, headers, body 입력 → fetch → 응답 뷰어.
  - cURL 명령어 생성 → 클립보드 복사.
  - JWT 연동: 로그인 모달, 토큰 자동 주입.
- main.go에서 explorer 라우트 등록 (APIInfo 변환).
- 통합 테스트 3건 (HTML 페이지, API 목록 JSON, 정적 파일), 전체 96건 통과.

---

## 2026-04-14 — Phase 5: JSON Schema 생성기

- `utils/schema.go`: GenerateSchema 재귀 함수 (object, array, string, integer, number, boolean, null).
  - Object: properties + required 자동 생성, 키 정렬로 결정적 출력.
  - Array: 첫 번째 요소로 items 타입 추론.
  - float64 중 정수값은 integer로 분류.
- SchemaHandler: POST /_utils/schema 핸들러.
- JWT 미들웨어에서 /_utils/* 경로 제외 추가.
- 단위 테스트 11건, 통합 테스트 1건, 전체 93건 통과.

---

## 2026-04-14 — Phase 4: JWT 인증

- `auth/jwt.go`: JWTService 구현.
  - GenerateTokenPair (jti로 유일성 보장), ValidateAccessToken, Refresh (Token Rotation), Logout (blacklist).
  - In-memory 토큰 저장소 (refreshTokens map, blacklist).
- `auth/handler.go`: /_auth/login, /_auth/logout, /_auth/refresh 핸들러.
  - Mock 서버 특성상 모든 username/password 조합 허용.
  - Middleware: Bearer Token 검증, /_auth/*, /_explorer, /health, auth:false 제외.
- `main.go`: JWT 활성 시 미들웨어 래핑, buildSkipAuthFunc로 auth:false 엔드포인트 매칭 (path variable 포함).
- 단위 테스트 8건 (발급, 검증, 만료, Rotation, 무효토큰, 로그아웃, 다른 시크릿).
- 통합 테스트 5건 (로그인→보호API 접근, auth:false 스킵, Refresh Rotation, 로그아웃, /health 스킵).
- 전체 테스트 81건 통과.

---

## 2026-04-14 — Phase 3: 파일 업로드/다운로드

- `script/context.go`: `FileInfo` 구조체 추가 (fieldName, fileName, size, savedPath).
- `script/engine.go`: `req.files` 배열을 Goja VM에 주입.
- `api/handler.go`: multipart/form-data 파싱 추가.
  - 업로드 파일은 storagePath에 자동 저장, 메타데이터를 `req.files`로 스크립트에 전달.
  - form field 값은 `req.body`에 매핑.
  - `RegisterAPIs`에 storagePath 파라미터 추가.
- 다운로드: `res.file(path)` → `http.ServeFile`로 실제 파일 스트리밍 동작 확인.
- 통합 테스트: 실제 multipart 업로드 → 파일 저장 확인, 실제 파일 다운로드 → 콘텐츠 검증.
- 전체 테스트 68건 통과.

---

## 2026-04-14 — 요청 파이프라인 완성 (validation + handler + 통합)

- `src/internal/validation/validator.go`: JSON Schema 검증기 구현 (santhosh-tekuri/jsonschema 활용).
- `src/internal/api/handler.go`: 전체 요청 파이프라인 연결.
  - Body 파싱 → JSON Schema 검증 → req/res 컨텍스트 구성 → 스크립트 실행 → 응답.
  - `RegisterAPIs()`: apis.yaml 로드 → 스크립트 사전 컴파일 → 라우터 등록 (Fail-Fast).
  - 파일 응답 시 `http.ServeFile` 사용, 커스텀 헤더 지원.
- `src/main.go`: `buildRouterFromConfig()` 분리. /health + 동적 API 라우트 등록.
- 통합 테스트를 실제 TCP 서버(`httptest.NewServer`) 방식으로 전환.
  - JSON 응답, query params, request body, validation 성공/실패, setHeader, 조건분기, 외부 스크립트, 404.
- 단위 테스트 5건(validation), 통합 테스트 66건 전체 통과.

---

## 2026-04-14 — Goja 스크립트 엔진 통합

- `src/internal/script/context.go`: Request(읽기전용), Response, ResHelper 구조체.
  - `res.json(status, body)`, `res.file(path)`, `res.setHeader(key, value)` 구현.
  - `Response.WriteHTTP()`: JSON 응답 직렬화, 파일 응답 시 MIME 추론 + Content-Disposition 헤더.
- `src/internal/script/engine.go`: Compile(사전 컴파일), Execute(VM 실행) 구현.
  - Sandboxing: require, process, global, globalThis를 undefined로 차단.
  - 스크립트가 res.json/res.file 미호출 시 에러 반환.
- 단위 테스트 15건 (컴파일, req 4종 주입, res 3종, 조건분기, 미응답, 런타임에러, Sandboxing, WriteHTTP).
- 통합 테스트 53건 전체 통과.

---

## 2026-04-14 — HTTP 라우터 구현

- `src/internal/router/router.go`: Router 구조체, Handle(), ServeHTTP() 구현.
- path variable `{name}` 매칭: splitPath로 세그먼트 분리 후 패턴 비교.
- method별 라우트 분리, trailing slash 정규화, 미매칭 시 404 반환.
- `src/internal/router/middleware.go`: context 기반 path params 전달 (Params 함수).
- 단위 테스트 10건, 통합 테스트 38건 전체 통과.

---

## 2026-04-14 — apis.yaml 파서 구현

- `src/internal/api/loader.go`: APIDefinition, Validation 구조체 정의.
- `LoadAPIs(path)`: YAML 파싱 + 검증 (entrypoint 필수/슬래시, method 필수/유효값, script/scriptFile 택1).
- `AuthEnabled()`: auth 필드 nil이면 기본 true.
- `ResolveScript(basePath)`: 인라인 우선, scriptFile은 basePath 기준 상대경로 해석.
- method 소문자 입력 시 자동 대문자 변환.
- 단위 테스트 15건, 통합 테스트 28건 전체 통과.

---

## 2026-04-14 — 디렉토리 구조 정리

- 소스 파일 → `./src/` (main.go, internal/, explorer/)
- 문서 파일 → `./docs/` (CONTEXT, ARCHITECTURE, PLAN, HISTORY, INSTALL)
- 루트 유지: CLAUDE.md, README.md, go.mod, go.sum, config.yaml
- `test.md` 삭제.
- import 경로 `dummy-web-server/internal/` → `dummy-web-server/src/internal/` 변경.
- CLAUDE.md, ARCHITECTURE.md 경로 참조 업데이트.
- `go test ./... -v` 전체 13건 통과 확인.

---

## 2026-04-14 — main.go 엔트리포인트 구현

- `main.go`: `run(configPath)` 함수로 서버 로직 분리. config 로드 실패 시 FATAL 출력 후 즉시 종료 (Fail-Fast).
- `-config` 플래그로 config.yaml 경로 지정 (기본값: `config.yaml`).
- `/health` 엔드포인트 추가 (서버 상태 확인용).
- 단위 테스트 3건 (config 미존재, config 무효, /health 응답), 통합 테스트 13건 전체 통과.

---

## 2026-04-14 — config.yaml 스키마 정의 및 로더 구현

- `config.yaml` 작성: server(port), jwt(enabled, secret, expiry), paths(apis, storage, scripts).
- `internal/config/config.go`: Config 구조체, DefaultConfig(), Load(path), validate() 구현.
- `JWTConfig.AccessTokenDuration()`, `RefreshTokenDuration()` 헬퍼 메서드 추가.
- 부분 설정 시 기본값 유지 (DefaultConfig 위에 Unmarshal).
- 단위 테스트 10건 통과 (정상 로드, 부분 설정, 포트 범위, JWT 검증, 파일 미존재, 잘못된 YAML, 토큰 duration).
- 통합 테스트 `go test ./...` 통과.

---

## 2026-04-14 — 프로젝트 디렉토리 구조 생성

- ARCHITECTURE.md 기준으로 패키지 뼈대 생성.
- `internal/` 하위: config, router, api, script, auth, validation, utils.
- `explorer/` (static 포함), `storage/`, `scripts/`.
- 각 패키지에 최소 .go 파일 배치, `main.go` 엔트리포인트 생성.
- `go build ./...` 통과 확인.

---

## 2026-04-14 — 프로젝트 초기 설정

### 스펙 정의
- `CONTEXT.md` 작성: 프로젝트 개요, 기술 스택, 기능 요구사항, 아키텍처 원칙 정의.
- 사용자 피드백 반영:
  - 3.1: List → JSON Array 명확화, JSON Schema 검증 범위를 Request Body로 한정.
  - 3.2: `req.headers`, `res.setHeader(key, value)` API Reference에 추가.
  - 3.3: JWT 고정 엔드포인트 3개 정의 (`login`, `logout`, `refresh`), Refresh Token Rotation 방식, 기본 만료 시간 명시.
  - 4: `go:embed` 대상(내장 리소스)과 외부 로드 대상의 경계 명시.
- `apis.yaml` 스키마 정의 (Section 5): `description`, `auth`, `validation`, `scriptFile` 필드 추가.

### 공통 기능 추가
- API Explorer (`GET /_explorer`): Swagger UI 유사 내장 웹 UI 스펙 추가.
  - API 목록, 테스트 콘솔, 응답 뷰어, cURL 복사, JWT 인증 연동.
  - `go:embed`로 바이너리 내장, 외부 CDN 의존성 없음.

### 환경 셋업
- Go 1.26.2 설치 (`go.mod`: go 1.23.3).
- 의존성 설치: `goja`, `yaml.v3`, `jwt/v5`, `jsonschema/v5`.
- `INSTALL.md` 작성.

### 프로젝트 관리 파일 구성
- `CLAUDE.md`: Agent 작업 가이드.
- `PLAN.md`: 작업 계획 및 진행 상태 추적.
- `HISTORY.md`: 작업 이력 기록.
