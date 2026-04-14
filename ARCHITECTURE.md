# ARCHITECTURE.md

## 시스템 구조

```
┌─────────────────────────────────────────────────────────┐
│                      main.go                            │
│  config.yaml 로드 → apis.yaml 파싱 → 스크립트 컴파일    │
│  → 라우트 등록 → HTTP 서버 기동                          │
└─────────────┬───────────────────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────────────────────┐
│                    HTTP Server                           │
│                 (net/http 표준 라이브러리)                 │
├─────────────────────────────────────────────────────────┤
│                    Middleware Chain                      │
│  ┌──────────┐  ┌──────────────┐  ┌───────────────────┐  │
│  │ Logger   │→ │ JWT Auth     │→ │ Route Dispatcher   │  │
│  │ (전체)   │  │ (조건부)     │  │                    │  │
│  └──────────┘  └──────────────┘  └───────────────────┘  │
└─────────────┬───────────────────────────────────────────┘
              │
     ┌────────┼──────────┐
     ▼        ▼          ▼
┌─────────┐ ┌────────┐ ┌──────────────┐
│ Dynamic │ │ Common │ │ API Explorer │
│ Routes  │ │ Routes │ │ (/_explorer) │
│(apis.yaml)│ │(/_auth/*│ │              │
│         │ │/_utils/*│ │              │
└────┬────┘ └────────┘ └──────────────┘
     │
     ▼
┌─────────────────────────────────────────────────────────┐
│                  Request Pipeline                        │
│  ┌─────────────────┐  ┌──────────────────────────────┐  │
│  │ Input Validation │→ │ Goja Script Engine           │  │
│  │ (JSON Schema)    │  │ ┌────────────┐ ┌──────────┐ │  │
│  └─────────────────┘  │ │ req (읽기)  │ │ res (쓰기)│ │  │
│                        │ │ .body       │ │ .json()   │ │  │
│                        │ │ .query      │ │ .file()   │ │  │
│                        │ │ .params     │ │.setHeader()│ │  │
│                        │ │ .headers    │ └──────────┘ │  │
│                        │ └────────────┘               │  │
│                        └──────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

## 디렉토리 구조 (계획)

```
dummy-web-server/
├── main.go                  # 엔트리포인트: 설정 로드, 서버 기동
├── config.yaml              # 서버 설정 (포트, JWT 등)
├── apis.yaml                # API 엔드포인트 정의
│
├── internal/
│   ├── config/              # config.yaml 로더 및 구조체
│   │   └── config.go
│   ├── router/              # HTTP 라우터, 미들웨어 체인
│   │   ├── router.go        # 라우트 등록 및 디스패치
│   │   └── middleware.go    # Logger, JWT Auth 미들웨어
│   ├── api/                 # apis.yaml 파서, 동적 라우트 생성
│   │   ├── loader.go        # YAML 파싱 및 라우트 등록
│   │   └── handler.go       # 요청 처리 (validation → script 실행)
│   ├── script/              # Goja 스크립트 엔진
│   │   ├── engine.go        # VM 생성, 사전 컴파일, Sandboxing
│   │   └── context.go       # req/res 컨텍스트 객체 정의
│   ├── auth/                # JWT 인증
│   │   ├── jwt.go           # 토큰 생성/검증/갱신
│   │   └── handler.go       # /_auth/* 핸들러
│   ├── validation/          # JSON Schema 검증
│   │   └── validator.go
│   └── utils/               # 유틸리티 엔드포인트
│       └── schema.go        # /_utils/schema 핸들러
│
├── explorer/                # API Explorer 웹 UI
│   ├── embed.go             # go:embed 선언
│   ├── handler.go           # /_explorer 핸들러
│   └── static/              # HTML/CSS/JS (embed 대상)
│       ├── index.html
│       ├── style.css
│       └── app.js
│
├── storage/                 # 파일 업로드/다운로드 저장소 (런타임)
├── scripts/                 # 외부 스크립트 파일 (사용자 정의)
│
├── go.mod
├── go.sum
├── CLAUDE.md
├── CONTEXT.md
├── PLAN.md
├── HISTORY.md
├── ARCHITECTURE.md
└── INSTALL.md
```

## 핵심 흐름

### 서버 기동
1. `config.yaml` 로드 → 서버 설정 구조체 생성
2. `apis.yaml` 파싱 → API 정의 목록 생성
3. 모든 스크립트(인라인 + 외부파일) 사전 컴파일 → 실패 시 즉시 종료
4. 라우트 등록: 공통 라우트(`/_auth/*`, `/_utils/*`, `/_explorer`) + 동적 라우트
5. HTTP 서버 기동

### 요청 처리
1. **Logger** — method, path 기록, 응답 완료 후 status, latency 출력
2. **JWT Auth** — JWT 활성 시 Bearer Token 검증. `/_auth/*`, `/_explorer`, `auth: false` 엔드포인트는 제외
3. **Route Dispatch** — 매칭된 핸들러로 전달
4. **Input Validation** — `validation.schema` 정의 시 Request Body를 JSON Schema로 검증
5. **Script Execution** — Goja VM에 `req`/`res` 주입 후 스크립트 실행
6. **Response** — `res.json()` 또는 `res.file()`로 응답 전송

## 패키지 의존 방향

```
main → config, router, api, auth, utils, explorer
router → auth (미들웨어)
api → script, validation
script → (외부 의존 없음, Goja만 사용)
auth → config (JWT 설정 참조)
explorer → api (API 목록 조회)
```

`internal/` 패키지 간 순환 의존 금지. 의존 방향은 항상 위에서 아래로.
