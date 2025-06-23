# Coupon Issuance System

## Requirements

You need to develop a coupon issuance system that enables creating campaigns with configurable parameters. Each campaign specifies the number of available coupons and a specific start date and time when coupons can be issued on a first-come-first-served basis.

The expected traffic immediately after a campaign launch is approximately 500-1,000 requests per second. The system must meet the following requirements:

- Issue exactly the specified number of coupons per campaign (no excess issuance)
- Coupon issuance must automatically start at the exact specified date and time
- Data consistency must be guaranteed throughout the issuance process
- Each coupon must have a unique code across all campaigns (up to 10 characters, consisting of Korean characters and numbers).

## Challenges

- Implement a concurrency control mechanism to solve data consistency issues under high traffic conditions (500-1,000 requests per second).
- Implement horizontally scalable system (Scale-out)
- Explore and design solutions for various edge cases that might occur.
- Implement testing tools or scripts that can verify concurrency issues.

---

## 요구사항 충족 방법

- **쿠폰 코드 유일성**:  
  Redis의 전역 Set(`coupon:codes`)을 활용하여 모든 캠페인에 걸쳐 중복 없는 코드를 관리합니다.
  코드 생성기는 한글(가-힣)과 숫자(0-9)로 10자리 코드를 생성하며, 중복 시 재시도합니다.

- **정확한 발급 시작 시점**:  
  캠페인 생성 시 `start_time`을 지정하고, 발급 요청마다 현재 시간과 비교하여 자동으로 발급 가능 여부를 판단합니다.

- **동시성 안전성**:  
  쿠폰 발급 로직은 Redis Lua 스크립트로 원자적으로 처리되어 race condition, 초과 발급, 중복 코드 문제가 발생하지 않습니다.
  코드 생성기의 랜덤 생성기도 Mutex로 보호되어 데이터 레이스가 없습니다.

- **운영 에러 추적**:  
  logrus + Sentry hook을 통해 에러 레벨 이상의 로그가 Sentry로 자동 전송됩니다.
  모든 로그는 JSON 포맷으로 구조화되어 기록됩니다.

- **동시성 테스트 자동화**:  
  `scripts/concurrency_test.sh` 스크립트가 제공되어, 여러 서버 인스턴스를 자동으로 띄우고, 대량의 동시 요청을 분산하여 테스트하며, 결과를 자동으로 분석합니다.

---

## Redis 실행 방법 (Docker)

테스트 및 서버 실행을 위해 Redis 인스턴스가 필요합니다. Docker를 이용해 아래와 같이 Redis를 실행할 수 있습니다.

```bash
docker run --name coupon-redis -p 6379:6379 -d redis:7
```
- 위 명령어로 로컬 6379 포트에 Redis 7 컨테이너가 실행됩니다.
- 테스트 및 서버는 기본적으로 localhost:6379에 접속합니다.
- 필요시 컨테이너 중지는 `docker stop coupon-redis`, 삭제는 `docker rm coupon-redis`로 할 수 있습니다.

---

## 서버 실행 방법

1. **의존성 설치**
    ```bash
    go mod tidy
    ```

2. **서버 실행 (포트별로 여러 인스턴스)**
    ```bash
    # 예시: 5개 인스턴스 실행
    PORT=8080 go run cmd/server/main.go
    PORT=8081 go run cmd/server/main.go
    PORT=8082 go run cmd/server/main.go
    PORT=8083 go run cmd/server/main.go
    PORT=8084 go run cmd/server/main.go
    ```
    - 또는, 테스트 스크립트가 자동으로 서버를 띄웁니다.

3. **환경 변수**
    - `REDIS_ADDR` (옵션, 기본값: `localhost:6379`)
    - `SENTRY_DSN` (옵션, Sentry 연동 시)

---

## 부하 테스트 실행 방법

1. **부하 테스트 스크립트 실행 권한 부여**
    ```bash
    chmod +x scripts/concurrency_test.sh
    ```
    - (만약 실행 권한이 없다는 에러가 발생하면 위 명령어를 먼저 실행하세요.)

2. **부하 테스트 실행**
    ```bash
    ./scripts/concurrency_test.sh
    ```

    - 스크립트가 자동으로 5개의 서버 인스턴스를 백그라운드로 실행하고, 캠페인을 생성한 뒤, 200건의 동시 쿠폰 발급 요청을 여러 서버에 분산하여 보냅니다.
    - 부하 테스트가 끝나면 서버 인스턴스도 자동으로 종료됩니다.
    - 결과 파일(`results.txt`)과 콘솔 출력으로 성공/실패/중복 여부를 확인할 수 있습니다.

---

## 테스트 실행 방법

```bash
go test ./test
```