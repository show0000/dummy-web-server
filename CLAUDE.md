# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 필수 규칙

**아래 파일들은 작업 중 항상 최신 상태로 유지할 것. Agent가 작업을 이어받을 때 이 파일들만으로 전체 맥락을 파악할 수 있어야 한다.**

| 파일 | 용도 | 갱신 시점 |
|------|------|-----------|
| [CONTEXT.md](./CONTEXT.md) | 프로젝트 스펙, 요구사항 | 스펙 변경 시 |
| [ARCHITECTURE.md](./ARCHITECTURE.md) | 시스템 구조, 디렉토리, 패키지 의존 관계 | 구조 변경 시 |
| [PLAN.md](./PLAN.md) | 작업 계획 및 진행 상태 (체크리스트) | 작업 시작/완료 시 |
| [HISTORY.md](./HISTORY.md) | 작업 이력 (무엇을, 왜, 어떻게) | 매 작업 완료 후 |
| [INSTALL.md](./INSTALL.md) | 개발 환경 셋업 가이드 | 의존성 변경 시 |

## 테스트 규칙

1. **단위 테스트:** PLAN.md의 task를 하나 완료할 때마다 해당 기능의 단위 테스트를 작성하고 통과시킬 것.
2. **통합 테스트:** 완료된 task가 누적될 때마다, Phase 1의 첫 task부터 방금 완료한 task까지 전체 통합 테스트를 실행하여 기존 기능이 깨지지 않았는지 검증할 것.
3. **테스트 실행:**
   ```bash
   # 단위 테스트 (특정 패키지)
   go test ./internal/config/ -v
   # 통합 테스트 (전체)
   go test ./... -v
   ```
4. **테스트 미통과 시 다음 task로 넘어가지 말 것.**

## Overview

YAML 기반 동적 Mock API 서버. 외부 시스템 연동 테스트 시 실제 서버 없이 API 동작을 재현하는 것이 목적.

- **Language:** Go | **Scripting:** Goja (JS) | **Config:** YAML | **Auth:** JWT
- **목표:** 의존성 없는 단일 실행 바이너리 (`go build`로 정적 컴파일)

## Build & Run

```bash
go build -o dummy-web-server .
./dummy-web-server
```

## Architecture

- `apis.yaml` — 엔드포인트 정의 (라우트, 메서드, 스크립트, validation)
- `config.yaml` — 서버 전체 설정 (포트, JWT 활성화/토큰 만료 등)
- Goja VM으로 스크립트 실행. `req`/`res` 컨텍스트만 노출, 시스템 리소스 접근 차단 (Sandboxing)
- 서버 시작 시 YAML 파싱 + 스크립트 사전 컴파일 → 실패 시 즉시 종료 (Fail-Fast)
- `go:embed`로 내장 리소스 포함. 사용자 정의 파일(`apis.yaml`, 스크립트 등)은 외부 로드
