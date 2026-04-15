# AGENT.md

AI Agent가 이 프로젝트를 개선, 수정, 사용하기 위한 인수인계 문서.

## 프로젝트 한줄 요약

YAML로 API를 정의하고 JavaScript로 응답을 제어하는 Mock 서버. 단일 바이너리로 실행.

## 첫 작업 전 반드시 읽을 파일

| 순서 | 파일 | 내용 |
|------|------|------|
| 1 | `CLAUDE.md` | 작업 규칙 (커밋, 테스트, 문서 갱신 의무) |
| 2 | `docs/CONTEXT.md` | 프로젝트 스펙, 요구사항, apis.yaml 스키마 |
| 3 | `docs/ARCHITECTURE.md` | 시스템 구조, 디렉토리, 패키지 의존 관계 |
| 4 | `docs/PLAN.md` | 작업 계획 및 진행 상태 |
| 5 | `docs/HISTORY.md` | 지금까지의 작업 이력과 맥락 |
| 6 | `docs/TEST.md` | 테스트 전략, 결과, 현재 98건 전체 PASS |
| 7 | `README.md` | 사용자 관점의 전체 사용법 |

## 빌드 / 테스트 / 실행

```bash
# 빌드
go build -o dummy-web-server ./src

# 단위 테스트 (특정 패키지)
go test ./src/internal/config/ -v

# 전체 테스트
go test ./... -v

# 실행
./dummy-web-server
./dummy-web-server --port 3000 --enable-login y
```

## 소스 구조와 역할

```
src/
├── main.go                     # 엔트리포인트. 아래 순서로 실행:
│                                #   config 로드 → apis.yaml 파싱 → 스크립트 컴파일
│                                #   → 라우트 등록 → 미들웨어 체인 → 서버 기동
│                                # CLI 플래그: --config, --port, --enable-login
├── main_test.go                # 통합 테스트 (httptest.NewServer로 실제 HTTP 검증)
│
├── internal/
│   ├── config/config.go        # Config 구조체, Load(), validate(), DefaultConfig()
│   ├── router/router.go        # 커스텀 라우터: {name} path variable, method 매칭
│   ├── router/middleware.go    # context 기반 path params 전달 (Params 함수)
│   ├── router/logger.go        # LoggerMiddleware: method, path, status, latency 로깅
│   ├── api/loader.go           # apis.yaml 파서: APIDefinition, LoadAPIs(), ResolveScript()
│   ├── api/handler.go          # 요청 파이프라인: body 파싱 → validation → script 실행 → 응답
│   │                           # multipart 업로드 처리, RegisterAPIs()
│   ├── script/engine.go        # Goja VM: Compile(Fail-Fast), Execute(Sandboxing)
│   ├── script/context.go       # req/res 객체, Response.WriteHTTP(), FileInfo
│   ├── auth/jwt.go             # JWTService: GenerateTokenPair, Validate, Refresh(Rotation), Logout
│   ├── auth/handler.go         # /_auth/* 핸들러, JWT Middleware (skipAuth 콜백)
│   ├── validation/validator.go # JSON Schema 검증 (santhosh-tekuri/jsonschema)
│   └── utils/schema.go         # JSON → JSON Schema 변환, /_utils/schema 핸들러
│
└── explorer/
    ├── embed.go                # go:embed static/*
    ├── handler.go              # /_explorer 라우팅, API 목록/config JSON 제공
    └── static/                 # HTML/CSS/JS (바이너리에 내장)
        ├── index.html
        ├── style.css
        └── app.js              # API 목록 렌더링, 테스트 콘솔, cURL 복사, JWT 연동
```

## 패키지 의존 방향

```
main → config, router, api, auth, utils, explorer
api → script, validation, router
auth → router (핸들러 등록)
explorer → (독립, embed만 사용)
script → (Goja만 사용, 내부 패키지 의존 없음)
```

순환 의존 금지. 새 패키지 추가 시 이 방향을 준수.

## 미들웨어 체인 순서

```
LoggerMiddleware → [JWT Middleware (조건부)] → Router → Handler
```

- Logger: 최외곽. 모든 요청의 method, path, status, latency를 로깅.
- JWT: `config.jwt.enabled: true`일 때만 적용. 아래 경로는 인증 제외:
  - `/_auth/*`, `/_explorer*`, `/_utils/*`, `/health`
  - `apis.yaml`에서 `auth: false`인 엔드포인트

## 요청 처리 파이프라인 (api/handler.go)

```
HTTP 요청
  → Content-Type 확인
    → multipart? → 파일 저장(storage) + req.files 구성
    → JSON? → body 파싱
  → validation.schema 있으면 → JSON Schema 검증
  → req 객체 구성 (body, query, params, headers, files)
  → Goja VM에서 script 실행
  → resp.IsMultipart → writeMultipartResponse (multipart/mixed)
  → resp.FilePath 있으면 → http.ServeFile
  → 아니면 → resp.WriteHTTP (JSON 응답)
```

## 핵심 설계 결정 (변경 시 주의)

| 결정 | 이유 |
|------|------|
| 커스텀 라우터 (net/http 표준 위) | path variable `{name}` 지원이 필요했으나 외부 라이브러리 배제 (Zero Dependency) |
| Goja VM을 매 요청마다 새로 생성 | Sandboxing 보장. VM 재사용 시 상태 오염 위험 |
| 스크립트 사전 컴파일 (`goja.Compile`) | Fail-Fast: 서버 시작 시 문법 오류 즉시 발견 |
| JWT 토큰 in-memory 저장 | Mock 서버 용도이므로 DB 불필요. 서버 재시작 시 초기화 |
| `go:embed`로 Explorer UI 내장 | 단일 바이너리 원칙 유지. 외부 CDN/파일 의존 없음 |
| `res.json`/`res.file` 호출 필수 | 스크립트가 응답을 생성하지 않으면 에러 반환 (침묵 실패 방지) |

## 새 기능 추가 시 체크리스트

1. `docs/PLAN.md`에 task 추가
2. 구현
3. 단위 테스트 작성 → `go test ./src/internal/<패키지>/ -v` 통과
4. 통합 테스트 → `go test ./... -v` 전체 통과
5. `docs/TEST.md` 업데이트
6. `docs/HISTORY.md` 업데이트
7. `docs/PLAN.md` 체크 표시
8. 구조 변경 시 `docs/ARCHITECTURE.md` 업데이트
9. `git add` → `git commit` → `git push`

## 자주 수정하게 될 파일

| 작업 | 파일 |
|------|------|
| 새 API 스크립트 기능 추가 (req/res 확장) | `script/context.go`, `script/engine.go` |
| 새 내장 엔드포인트 추가 | `main.go` (라우트 등록), 해당 핸들러 파일 |
| apis.yaml 스키마 필드 추가 | `api/loader.go` (구조체+파싱), `api/handler.go` (처리) |
| config.yaml 필드 추가 | `config/config.go` (구조체+validate) |
| Explorer UI 수정 | `explorer/static/` (HTML/CSS/JS) |
| JWT 미들웨어 경로 제외 추가 | `auth/handler.go` (Middleware 함수 내 skip 조건) |

## 외부 의존성

| 라이브러리 | 용도 | import |
|-----------|------|--------|
| `github.com/dop251/goja` | JavaScript 엔진 | `script/engine.go` |
| `gopkg.in/yaml.v3` | YAML 파싱 | `config/config.go`, `api/loader.go` |
| `github.com/golang-jwt/jwt/v5` | JWT 토큰 | `auth/jwt.go` |
| `github.com/santhosh-tekuri/jsonschema/v5` | JSON Schema 검증 | `validation/validator.go` |

이 4개 외에 외부 라이브러리를 추가하지 않는 것이 원칙 (Zero Dependency).

## 알려진 제약 / 향후 개선 가능 영역

- JWT 토큰 저장이 in-memory이므로 서버 재시작 시 모든 세션 초기화됨.
- `apis.yaml`의 `schemaFile` (외부 JSON Schema 파일)은 파서에 필드가 정의되어 있으나 handler에서 아직 로드 로직이 구현되지 않음.
- Explorer UI는 파일 업로드 테스트를 지원하지 않음 (JSON body만 가능).
- 라우터가 선형 탐색이므로 엔드포인트가 수백 개를 넘으면 성능 저하 가능.
- Hot reload (apis.yaml 변경 시 재시작 없이 반영) 미지원.
