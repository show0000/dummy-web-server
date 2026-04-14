# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 필수 규칙

**작업 완료 시 반드시 `git add` → `git commit` → `git push` 할 것. 커밋 없이 다음 task로 넘어가지 않는다.**

**아래 파일들은 작업 중 항상 최신 상태로 유지할 것. Agent가 작업을 이어받을 때 이 파일들만으로 전체 맥락을 파악할 수 있어야 한다.**

| 파일 | 용도 | 갱신 시점 |
|------|------|-----------|
| [CONTEXT.md](./docs/CONTEXT.md) | 프로젝트 스펙, 요구사항 | 스펙 변경 시 |
| [ARCHITECTURE.md](./docs/ARCHITECTURE.md) | 시스템 구조, 디렉토리, 패키지 의존 관계 | 구조 변경 시 |
| [PLAN.md](./docs/PLAN.md) | 작업 계획 및 진행 상태 (체크리스트) | 작업 시작/완료 시 |
| [HISTORY.md](./docs/HISTORY.md) | 작업 이력 (무엇을, 왜, 어떻게) | 매 작업 완료 후 |
| [INSTALL.md](./docs/INSTALL.md) | 개발 환경 셋업 가이드 | 의존성 변경 시 |

## 테스트 규칙

1. **단위 테스트:** PLAN.md의 task를 하나 완료할 때마다 해당 기능의 단위 테스트를 작성하고 통과시킬 것.
2. **통합 테스트:** 완료된 task가 누적될 때마다, Phase 1의 첫 task부터 방금 완료한 task까지 전체 통합 테스트를 실행하여 기존 기능이 깨지지 않았는지 검증할 것.
3. **테스트 실행:**
   ```bash
   # 단위 테스트 (특정 패키지)
   go test ./src/internal/config/ -v
   # 통합 테스트 (전체)
   go test ./... -v
   ```
4. **테스트 미통과 시 다음 task로 넘어가지 말 것.**

## Overview

YAML 기반 동적 Mock API 서버. 외부 시스템 연동 테스트 시 실제 서버 없이 API 동작을 재현하는 것이 목적.

- **Language:** Go | **Scripting:** Goja (JS) | **Config:** YAML | **Auth:** JWT
- **목표:** 의존성 없는 단일 실행 바이너리 (`go build`로 정적 컴파일)

## Directory Layout

```
./              — 프로젝트 루트 (CLAUDE.md, README.md, go.mod, config.yaml)
./src/          — Go 소스 코드 (main.go, internal/, explorer/)
./docs/         — 프로젝트 문서 (CONTEXT, ARCHITECTURE, PLAN, HISTORY, INSTALL)
./storage/      — 파일 업로드/다운로드 저장소 (런타임)
./scripts/      — 외부 스크립트 파일 (사용자 정의)
```

## Build & Run

```bash
go build -o dummy-web-server ./src
./dummy-web-server
```
