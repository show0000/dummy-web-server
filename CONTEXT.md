# CONTEXT.md

## 1. Project Overview
* **Project Name:** dummy-web-server
* **Objective:** 외부 시스템 인터페이스 테스트를 위한 고성능, 저지연 Mock 서버. 문서(YAML) 기반의 동적 API 생성과 스크립트를 통한 비즈니스 로직 시뮬레이션을 지원함.
* **Key Deliverable:** 의존성 없는 단일 실행 파일 (Single Binary).

## 2. Tech Stack & Constraints
* **Language:** Go (Golang)
* **Scripting Engine:** Goja (JavaScript Engine) - JSON 친화적 로직 실행.
* **Config Format:** YAML (API 정의 및 서버 전체 설정).
* **Auth:** JWT (Access/Refresh Token).
* **Target:** 운영체제 독립적 정적 컴파일 바이너리.

## 3. Functional Requirements

### 3.1. Document-based API Management
* **Dynamic Routing:** `apis.yaml`을 로드하여 런타임에 엔드포인트 등록.
* **Input Validation:**
    * 기본 타입(String, Float, Int, Bool) 및 Array(JSON Array) 지원.
    * JSON Schema를 이용한 Request Body 유효성 검증. Query, Path Parameter는 검증 대상이 아님.
* **File Operations:**
    * **Upload:** POST 요청을 통한 파일 저장.
    * **Download:** GET 요청 시 `res.file(path)`를 통한 파일 스트리밍 전송.

### 3.2. Scripting Logic (Goja)
* **Control Flow:** JS 문법(if/else, for/while, 변수 할당) 지원.
* **Context Injection:**
    * `req`: 요청 정보를 담은 읽기 전용 객체.
        * `req.body`: JSON 파싱된 요청 바디 (Object).
        * `req.query`: URL 쿼리 파라미터 (Object).
        * `req.params`: Path Variable (Object).
        * `req.headers`: 요청 헤더 (Object). 키는 소문자 정규화.
    * `res`: 응답 제어 헬퍼 객체.
        * `res.json(status, body)`: JSON 응답 전송.
        * `res.file(path)`: 파일 다운로드 응답 전송 (MIME 타입 자동 추론).
        * `res.setHeader(key, value)`: 응답 헤더 설정. `res.json()` 또는 `res.file()` 호출 전에 사용.

### 3.3. Common Features (Global)
* **JWT System:**
    * 서버 설정(`config.yaml`)으로 Enable/Disable 제어.
    * **고정 엔드포인트:**
        * `POST /_auth/login`: 사용자 인증 및 Access/Refresh Token 발급.
        * `POST /_auth/logout`: 토큰 무효화.
        * `POST /_auth/refresh`: Refresh Token으로 Access Token 재발급.
    * **토큰 정책:**
        * Access Token: 단시간 유효 (기본 15분, 설정 가능).
        * Refresh Token: 장시간 유효 (기본 7일, 설정 가능).
        * Refresh 요청 시 기존 Refresh Token은 폐기하고 새로 발급 (Rotation).
    * **검증:** JWT가 활성화된 경우, `/_auth/*` 및 `auth: false`로 표시된 엔드포인트를 제외한 모든 요청에 대해 Authorization 헤더의 Bearer Token을 검증.
* **JSON Schema Generator:** `POST /_utils/schema` 호출 시 전송된 JSON을 분석하여 JSON Schema 반환.
* **API Explorer (Built-in):**
    * `GET /_explorer` 접속 시 내장 웹 UI 제공. Swagger UI와 유사한 역할.
    * **API 목록:** `apis.yaml`에 등록된 전체 엔드포인트를 method, path, description과 함께 표시.
    * **테스트 콘솔:** 각 API에 대해 path params, query params, request body, headers를 입력하고 요청을 실행할 수 있는 인터랙티브 폼 제공.
    * **응답 뷰어:** 상태 코드, 응답 헤더, 응답 바디를 포맷팅하여 표시.
    * **cURL 복사:** 테스트 콘솔에서 입력한 요청 정보를 cURL 명령어로 생성하여 클립보드에 복사. 터미널 실행 및 팀원 공유 용도.
    * **인증 연동:** JWT가 활성화된 경우, 로그인 후 발급받은 토큰을 자동으로 후속 요청의 Authorization 헤더에 포함.
    * **구현:** HTML/CSS/JS를 `go:embed`로 바이너리에 내장. 외부 CDN 의존성 없음.

## 4. Architecture & Implementation Principles
* **Fail-Fast:** 서버 시작 시 모든 YAML 설정 파싱 및 스크립트 사전 컴파일을 수행. 오류 발견 시 원인을 표준 출력에 로깅하고 즉시 종료.
* **Sandboxing:** Goja VM에서 시스템 리소스(os, net, fs 등) 접근을 차단. 스크립트는 주입된 `req`/`res` 컨텍스트만 사용 가능.
* **Zero Dependency:** `go:embed`를 활용하여 내장 리소스(기본 에러 응답 템플릿, 기본 config 등)를 바이너리에 포함. 단, `apis.yaml`, `config.yaml`, 스크립트 파일, 업로드 저장소 등 사용자 정의 파일은 외부 파일시스템에서 로드.
* **Observability:** 모든 인입 요청(method, path, status, latency)과 스크립트 실행 결과를 구조화된 형식으로 표준 출력에 로깅.

## 5. apis.yaml Schema

### 5.1. 구조 정의
```yaml
apis:
  - entrypoint: string     # (필수) 라우트 경로. Path Variable은 {name} 형식.
    method: string          # (필수) HTTP 메서드 (GET, POST, PUT, DELETE, PATCH).
    description: string     # (선택) 엔드포인트 설명.
    auth: bool              # (선택) JWT 인증 적용 여부. 기본값: true (JWT 활성 시).
    validation:             # (선택) Request Body JSON Schema 검증.
      schema: object        # JSON Schema 객체 (인라인).
      schemaFile: string    # 또는 외부 JSON Schema 파일 경로. schema와 중복 시 schema 우선.
    script: string          # (택1) 인라인 JavaScript 코드.
    scriptFile: string      # (택1) 외부 JavaScript 파일 경로. script와 중복 시 script 우선.
```

### 5.2. 예시
```yaml
apis:
  # 파일 다운로드 (인증 불필요)
  - entrypoint: /api/v1/download/{fileName}
    method: GET
    description: 파일 다운로드 API
    auth: false
    script: |
      const fileName = req.params.fileName;
      const filePath = `./storage/${fileName}`;

      if (fileName.includes("private")) {
          return res.json(403, { message: "접근 권한이 없습니다." });
      }

      return res.file(filePath);

  # 사용자 생성 (Request Body 검증 + 외부 스크립트)
  - entrypoint: /api/v1/users
    method: POST
    description: 사용자 생성 API
    scriptFile: ./scripts/create-user.js
    validation:
      schema:
        type: object
        required: [name, email]
        properties:
          name:
            type: string
          email:
            type: string
            format: email
          age:
            type: integer
```
