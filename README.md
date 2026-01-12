# TQQQ Infinite Trader

**무한매수법 자동화 시스템**

한국투자증권 KIS API를 활용한 레버리지 ETF 무한매수 전략 자동화 웹 애플리케이션입니다.

---

## 📋 목차

1. [프로젝트 개요](#프로젝트-개요)
2. [기술 스택](#기술-스택)
3. [전략 설명](#전략-설명)
4. [설치 및 실행](#설치-및-실행)
5. [환경 변수 설정](#환경-변수-설정)
6. [API 테스트](#api-테스트)
7. [사용 방법](#사용-방법)
8. [프로젝트 구조](#프로젝트-구조)

---

## 프로젝트 개요

이 시스템은 TQQQ, SOXL 등 레버리지 ETF에 대한 **무한매수법(Infinite Buying Strategy)**을 자동화합니다.

### 핵심 기능
- ✅ 매일 자동으로 일정 금액만큼 매수
- ✅ 평균 매수가 대비 10% 수익 시 전량 매도
- ✅ 실시간 포트폴리오 동기화
- ✅ 웹 대시보드를 통한 모니터링
- ✅ Docker 기반 배포

---

## 기술 스택

### Backend
- **언어**: Go 1.23
- **프레임워크**: Gin (웹 서버)
- **데이터베이스**: SQLite (GORM)
- **스케줄러**: robfig/cron
- **API**: 한국투자증권 KIS API

### Frontend
- **프레임워크**: SvelteKit 5
- **스타일링**: Tailwind CSS, DaisyUI, Flowbite
- **빌드 도구**: Vite

### Infrastructure
- **프록시**: Nginx
- **컨테이너**: Docker, Docker Compose

---

## 전략 설명

### 무한매수법이란?

레버리지 ETF의 장기 상승 추세를 가정하고, 변동성을 활용하여 기계적으로 매수/매도를 반복하는 전략입니다.

### 매수 규칙
1. 총 투자금을 **40분할** (기본값)
2. 매일 **1~2분할**씩 매수 (LOC 주문 권장)
3. 하락 시 평균 단가 낮추기 효과

### 매도 규칙
1. 평균 매수가 대비 **10% 수익** 도달 시 전량 매도
2. 매도 후 사이클 재시작 (1/40부터 다시 매수)

### 주문 유형
- **LOC (Limit On Close)**: 장 마감 직전 지정가 주문
- **Market**: 시장가 즉시 체결

---

## 설치 및 실행

### 사전 요구사항
- Docker & Docker Compose 설치
- 한국투자증권 KIS API 계정 (실전 또는 모의투자)

### 1. 프로젝트 클론
```bash
git clone <repository-url>
cd TQQQ-InfiniteTrader
```

### 2. 환경 변수 설정
`.env` 파일을 생성하거나 `docker-compose.yml`의 환경변수를 수정합니다:

```env
KIS_APP_KEY=your_app_key_here
KIS_APP_SECRET=your_app_secret_here
KIS_ACCOUNT_NUM=12345678901  # 계좌번호 (하이픈 제거)
KIS_BASE_URL=https://openapi.koreainvestment.com:9443
# 모의투자: https://openapivts.koreainvestment.com:29443
```

### 3. Docker Compose 실행
```bash
docker-compose up -d --build
```

### 4. 접속
- **웹 UI**: http://localhost:8081
- **백엔드 API**: http://localhost:8082
- **Nginx 프록시**: http://localhost:8081/api

---

## 환경 변수 설정

| 변수명 | 설명 | 필수 여부 |
|--------|------|-----------|
| `KIS_APP_KEY` | KIS API 앱 키 | ✅ 필수 |
| `KIS_APP_SECRET` | KIS API 시크릿 키 | ✅ 필수 |
| `KIS_ACCOUNT_NUM` | 계좌번호 (하이픈 제외) | ✅ 필수 |
| `KIS_BASE_URL` | API 베이스 URL | ✅ 필수 |

> **주의**: 모의투자와 실전투자의 Base URL이 다릅니다. 반드시 확인하세요!

---

## API 테스트

백엔드 API의 모든 기능을 `curl` 명령어로 테스트할 수 있습니다.

### 1. 대시보드 데이터 조회
```bash
curl -v http://localhost:8081/api/dashboard
```

**예상 응답 (200 OK)**:
```json
{
  "cycles": [
    {
      "ID": 1,
      "Symbol": "TQQQ",
      "CurrentCycleDay": 5,
      "TotalBoughtQty": 10,
      "AvgPrice": 50.0,
      "TotalInvested": 500.0
    }
  ]
}
```

---

### 2. 설정 조회 (GET)
```bash
curl -v http://localhost:8081/api/settings
```

**예상 응답 (200 OK)**:
```json
{
  "ID": 1,
  "CreatedAt": "2025-12-24T13:51:45Z",
  "UpdatedAt": "2025-12-24T13:51:45Z",
  "DeletedAt": null,
  "Principal": 10000,
  "SplitCount": 40,
  "TargetRate": 0.1,
  "Symbols": "TQQQ",
  "IsActive": true
}
```

---

### 3. 설정 업데이트 (POST)
```bash
curl -v -X POST http://localhost:8081/api/settings \
  -H "Content-Type: application/json" \
  -d '{
    "Principal": 20000,
    "SplitCount": 40,
    "TargetRate": 0.15,
    "Symbols": "TQQQ,SOXL",
    "IsActive": true
  }'
```

**예상 응답 (200 OK)**:
```json
{
  "status": "ok"
}
```

---

### 4. 포트폴리오 동기화 (POST)
실제 KIS 계좌 잔고와 로컬 DB를 동기화합니다.

```bash
curl -v -X POST http://localhost:8081/api/sync
```

**예상 응답 (200 OK)**:
```json
{
  "status": "synced"
}
```

> **주의**: 이 요청은 실제 KIS API를 호출하므로, 환경 변수가 올바르게 설정되어 있어야 합니다.

---

### 5. 전체 테스트 스크립트
모든 엔드포인트를 한 번에 테스트:

```bash
#!/bin/bash

echo "1. Dashboard Test"
curl -s http://localhost:8081/api/dashboard | jq

echo -e "\n2. Get Settings"
curl -s http://localhost:8081/api/settings | jq

echo -e "\n3. Update Settings"
curl -s -X POST http://localhost:8081/api/settings \
  -H "Content-Type: application/json" \
  -d '{"Principal": 15000, "SplitCount": 40, "TargetRate": 0.12, "Symbols": "TQQQ", "IsActive": true}' | jq

echo -e "\n4. Sync Portfolio"
curl -s -X POST http://localhost:8081/api/sync | jq
```

---

## 사용 방법

### 1. 초기 설정
1. 웹 UI (`http://localhost:8081`)에 접속
2. **Settings** 탭 이동
3. 다음 항목 설정:
   - **Principal Amount**: 총 투자 금액 (예: $10,000)
   - **Split Count**: 분할 횟수 (기본 40)
   - **Target Profit Rate**: 목표 수익률 (0.10 = 10%)
   - **Trading Symbols**: 거래 종목 (예: `TQQQ,SOXL`)
   - **Strategy Activation**: 전략 활성화 ON

### 2. 대시보드 모니터링
- **Dashboard** 탭에서 실시간 사이클 상태 확인
- Cycle Day: 현재 매수 진행 상황 (예: 5/40)
- Holdings: 보유 수량
- Avg Price: 평균 매수가
- Total Invested: 총 투자 금액

### 3. 수동 동기화
- "Sync Now" 버튼 클릭 → KIS 계좌 잔고와 로컬 DB 동기화

### 4. 자동 실행
- 스케줄러가 매일 **15:50 ET (월~금)**에 자동으로 전략 실행 (미국 장 마감 10분 전)
- 로그는 Docker 컨테이너 로그에서 확인:
  ```bash
  docker-compose logs -f backend
  ```

---

## 프로젝트 구조

```
TQQQ-InfiniteTrader/
├── backend/                 # Go 백엔드
│   ├── cmd/server/         # 메인 엔트리포인트
│   │   └── main.go
│   ├── internal/
│   │   ├── api/            # HTTP 핸들러
│   │   │   └── handler.go
│   │   ├── config/         # 환경 변수 로드
│   │   │   └── config.go
│   │   ├── kis/            # KIS API 클라이언트
│   │   │   └── client.go
│   │   ├── model/          # GORM 모델
│   │   │   └── model.go
│   │   ├── repository/     # SQLite DB
│   │   │   └── sqlite.go
│   │   ├── service/        # 비즈니스 로직
│   │   │   └── strategy.go
│   │   └── worker/         # 스케줄러
│   │       └── scheduler.go
│   ├── Dockerfile
│   └── go.mod
│
├── frontend/               # Svelte 프론트엔드
│   ├── src/
│   │   ├── routes/        # 페이지
│   │   │   ├── +layout.svelte
│   │   │   ├── +page.svelte       # Dashboard
│   │   │   ├── settings/
│   │   │   │   └── +page.svelte   # Settings
│   │   │   └── logs/
│   │   │       └── +page.svelte   # Logs
│   │   ├── lib/
│   │   │   └── api.ts     # API 클라이언트
│   │   └── app.css        # 글로벌 스타일
│   ├── Dockerfile
│   └── package.json
│
├── nginx/
│   └── nginx.conf          # 리버스 프록시 설정
│
├── docker-compose.yml      # 전체 스택 오케스트레이션
└── README.md
```

---

## 주의 사항

### 보안
- ⚠️ **절대로** `.env` 파일을 Git에 커밋하지 마세요
- KIS API 키는 안전하게 보관하세요

### 리스크
- 레버리지 ETF는 변동성이 크므로 원금 손실 가능성이 있습니다
- **반드시 모의투자에서 먼저 테스트**하세요
- 투자 결정은 본인 책임입니다

### 운영
- 매도 체결 후 사이클이 자동으로 리셋됩니다
- 주말/공휴일에는 미국 증시 휴장으로 거래 불가
- 스케줄러는 **월~금 15:50 ET** 실행 (미국 장 마감 10분 전)

---

## 클라우드 서버 배포

### 사전 요구사항
- Docker 및 Docker Compose 설치
- 서버 시간대를 **America/New_York (ET)**으로 설정 권장

### 1. 서버 시간대 설정 (선택)
```bash
# Ubuntu/Debian
sudo timedatectl set-timezone America/New_York

# 확인
date
```

> **참고**: 컨테이너 내부는 `docker-compose.yml`의 `TZ=America/New_York` 환경변수로 자동 설정됩니다.
> 호스트 시간대 설정은 선택사항이며, 로그 확인 시 편의를 위한 것입니다.

### 2. 프로젝트 배포
```bash
git clone <repository-url>
cd TQQQ-InfiniteTrader

# 환경 변수 설정
cp .env.example .env
nano .env  # KIS API 키 입력

# 컨테이너 실행
docker-compose up -d --build
```

### 3. 스케줄러 확인
```bash
# 로그에서 스케줄러 시작 메시지 확인
docker logs tqqq-backend | grep "Scheduler"
# 예상 출력: Scheduler started (15:50 ET Mon-Fri)
```

### 4. 미국 공휴일 처리
현재 스케줄러는 미국 공휴일을 자동으로 감지하지 않습니다. 공휴일에는 KIS API에서 주문이 거부될 수 있으며, 이는 정상 동작입니다.

---

## 트러블슈팅

### 1. Docker 빌드 실패
```bash
docker-compose down
docker-compose build --no-cache
docker-compose up -d
```

### 2. 백엔드 502 에러
```bash
# 백엔드 로그 확인
docker-compose logs backend

# 환경 변수 확인
docker exec tqqq-backend env | grep KIS
```

### 3. 프론트엔드 연결 실패
```bash
# 프론트엔드 재시작
docker-compose restart frontend nginx
```

---

## 라이선스

이 프로젝트는 개인적인 학습 및 투자 목적으로만 사용하세요.
