# JSON Samples

`POST /_utils/schema` 엔드포인트 테스트용 샘플 JSON 파일.

## 사용법

```bash
# 서버 실행 후
./dummy-web-server

# 다른 터미널에서 스키마 생성 요청
curl -X POST http://localhost:8080/_utils/schema \
  -H "Content-Type: application/json" \
  -d @samples/simple.json

curl -X POST http://localhost:8080/_utils/schema \
  -H "Content-Type: application/json" \
  -d @samples/user.json

curl -X POST http://localhost:8080/_utils/schema \
  -H "Content-Type: application/json" \
  -d @samples/order.json
```

## 샘플 목록

| 파일 | 용도 | 포함 타입 |
|------|------|-----------|
| `simple.json` | 기본 | string, integer, boolean |
| `user.json` | 사용자 정보 | 중첩 object, array, 모든 primitive |
| `order.json` | 주문 정보 | 복잡한 중첩, object 배열 |
| `nested.json` | 깊은 중첩 | 3단계 중첩 object/array |
| `mixed-types.json` | 모든 타입 | integer/float 구분, null, 빈 배열 포함 |
