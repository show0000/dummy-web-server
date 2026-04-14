# PLAN.md

작업 계획 및 진행 상태. 완료 시 체크 표시로 업데이트.

## Phase 1: 프로젝트 기반 구조

- [x] 프로젝트 스펙 정의 (CONTEXT.md)
- [x] 개발 환경 셋업 (Go, 의존성 설치)
- [x] 프로젝트 디렉토리 구조 설계 및 생성
- [x] `config.yaml` 스키마 정의 및 로더 구현
- [x] `main.go` 엔트리포인트 (서버 기동, 설정 로드, Fail-Fast)

## Phase 2: 핵심 기능 — Dynamic Routing + Scripting

- [x] `apis.yaml` 파서 구현 (YAML → 라우트 등록)
- [x] HTTP 라우터 구현 (path variable `{name}` 지원)
- [x] Goja 스크립트 엔진 통합
  - [x] `req` 컨텍스트 주입 (body, query, params, headers)
  - [x] `res` 헬퍼 구현 (json, file, setHeader)
  - [x] 스크립트 사전 컴파일 (Fail-Fast)
  - [x] Sandboxing (시스템 리소스 접근 차단)
- [x] 외부 스크립트 파일 로드 (`scriptFile`)
- [x] Request Body JSON Schema 검증 (`validation`)

## Phase 3: 파일 처리

- [x] 파일 업로드 (POST, multipart/form-data)
- [x] 파일 다운로드 (`res.file(path)`, MIME 자동 추론)

## Phase 4: JWT 인증

- [x] JWT 토큰 발급/검증 로직
- [x] `POST /_auth/login` — Access/Refresh Token 발급
- [x] `POST /_auth/logout` — 토큰 무효화
- [x] `POST /_auth/refresh` — Refresh Token Rotation
- [x] 인증 미들웨어 (Bearer Token 검증, `auth: false` 제외)
- [x] `config.yaml`로 JWT Enable/Disable 및 만료 시간 설정

## Phase 5: 유틸리티

- [x] `POST /_utils/schema` — JSON → JSON Schema 생성기

## Phase 6: API Explorer (Built-in Web UI)

- [ ] 내장 HTML/CSS/JS 작성 (`go:embed`)
- [ ] `GET /_explorer` — API 목록 표시
- [ ] 테스트 콘솔 (params, body, headers 입력 → 요청 실행)
- [ ] 응답 뷰어 (status, headers, body 포맷팅)
- [ ] cURL 명령어 생성 및 클립보드 복사
- [ ] JWT 인증 연동 (토큰 자동 주입)

## Phase 7: 마무리

- [ ] 요청/응답 로깅 (Observability)
- [ ] 에러 핸들링 통합 (표준화된 에러 응답)
- [ ] 빌드 및 크로스 컴파일 검증
