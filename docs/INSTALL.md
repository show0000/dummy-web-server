# INSTALL.md

## 사전 요구사항

* Go 1.23.3 이상

```bash
# macOS (Homebrew)
brew install go

# 설치 확인
go version
```

## 프로젝트 초기화

```bash
cd dummy-web-server
go mod init dummy-web-server
```

## 의존성 설치

```bash
# JavaScript 엔진
go get github.com/dop251/goja

# YAML 파서
go get gopkg.in/yaml.v3

# JWT
go get github.com/golang-jwt/jwt/v5

# JSON Schema 검증
go get github.com/santhosh-tekuri/jsonschema/v5
```

## 빌드 및 실행

```bash
# 빌드
go build -o dummy-web-server .

# 실행
./dummy-web-server
```
