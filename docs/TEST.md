# TEST.md

테스트 전략 및 결과 기록.

## 테스트 전략

### 단위 테스트
- Go 표준 `testing` 패키지 사용. 외부 프레임워크 없음.
- 각 패키지별 `_test.go` 파일에서 개별 함수/메서드를 검증.
- `httptest.NewRecorder`로 핸들러 단위의 인메모리 HTTP 검증.

### 통합 테스트
- `httptest.NewServer`로 실제 TCP 서버를 기동하고 `http.Client`로 요청/응답을 검증.
- config.yaml + apis.yaml 로드 → 서버 기동 → HTTP 요청 → status code, headers, body 전체 검증.
- 스크립트 실행 결과가 응답에 정확히 반영되는지 확인.

### 실행 방법
```bash
# 단위 테스트 (특정 패키지)
go test ./src/internal/config/ -v

# 통합 테스트 (전체)
go test ./... -v
```

---

## 테스트 결과

### Phase 1: 프로젝트 기반 구조

| 패키지 | 테스트 | 건수 | 결과 |
|--------|--------|------|------|
| `src/internal/config` | DefaultConfig, LoadValidConfig, LoadPartialConfigUsesDefaults, LoadInvalidPort, LoadInvalidPortZero, LoadJWTEnabledWithoutSecret, LoadInvalidTokenExpiry, LoadFileNotFound, LoadInvalidYAML, JWTTokenDurations | 10 | PASS |
| `src` (main) | RunFailsWithMissingConfig, RunFailsWithInvalidConfig, HealthEndpoint | 3 | PASS |

### Phase 2: Dynamic Routing + Scripting

| 패키지 | 테스트 | 건수 | 결과 |
|--------|--------|------|------|
| `src/internal/api` | LoadValidAPIs, LoadMethodUppercase, LoadMissingEntrypoint, LoadMissingMethod, LoadInvalidMethod, LoadMissingScript, LoadEntrypointNoSlash, LoadFileNotFound, LoadInvalidYAML, AuthEnabled, ResolveScriptInline, ResolveScriptFile, ResolveScriptFileMissing, ResolveScriptInlinePriority, LoadValidation | 15 | PASS |
| `src/internal/router` | ExactMatch, PathVariable, MultiplePathVariables, MethodMismatch, PathMismatch, SegmentCountMismatch, TrailingSlash, SamePathDifferentMethods, RootPath, ParamsFromContextEmpty | 10 | PASS |
| `src/internal/script` | CompileValid, CompileInvalid, ExecuteResJson, ExecuteResFile, ExecuteSetHeader, ExecuteReqBody, ExecuteReqQuery, ExecuteReqParams, ExecuteReqHeaders, ExecuteConditionalLogic, ExecuteNoResponse, ExecuteRuntimeError, SandboxingRequireBlocked, ResponseWriteHTTPJson, ResponseWriteHTTPFileHeaders | 15 | PASS |
| `src/internal/validation` | ValidateValid, ValidateMissingRequired, ValidateWrongType, ValidateArray, ValidateEmptySchema | 5 | PASS |
| `src` (통합) | RunFailsWithMissingConfig, RunFailsWithInvalidConfig, HealthEndpoint, DynamicAPIJsonResponse, DynamicAPIWithQueryParams, DynamicAPIWithRequestBody, DynamicAPIWithValidation, DynamicAPIWithSetHeader, DynamicAPIConditionalResponse, DynamicAPIExternalScript, NotFoundRoute | 11 | PASS |

> `src` (통합) 테스트는 `httptest.NewServer`로 실제 TCP 서버를 기동하여 HTTP 클라이언트로 요청/응답을 검증.

### Phase 3: 파일 처리

| 패키지 | 테스트 | 건수 | 결과 |
|--------|--------|------|------|
| `src` (통합) | FileUpload (multipart 업로드 → 저장 → req.files 검증), FileDownload (실제 파일 → http.ServeFile → 콘텐츠 검증) | 2 | PASS |

### Phase 4: JWT 인증

| 패키지 | 테스트 | 건수 | 결과 |
|--------|--------|------|------|
| `src/internal/auth` | GenerateTokenPair, ValidateAccessToken, ValidateInvalidToken, ValidateExpiredToken, RefreshTokenRotation, RefreshInvalidToken, LogoutBlacklistsAccessToken, DifferentSecretsReject | 8 | PASS |
| `src` (JWT 통합) | JWTLoginAndAccessProtectedAPI, JWTAuthFalseSkipsAuth, JWTRefreshTokenRotation, JWTLogout, JWTHealthSkipsAuth | 5 | PASS |

---

**총 테스트: 81건 / 전체 PASS**
