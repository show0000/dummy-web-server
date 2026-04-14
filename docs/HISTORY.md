# HISTORY.md

작업 이력. 최신 항목이 상단.

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
