# Velocity Bulletin Backend

Go, Gin, GORM, PostgreSQL로 만든 의도적으로 작고 단순한 게시판 API입니다. ECS Fargate 위에서 상태 없는(stateless) 컨테이너로 동작하도록 설계되었으며, PostgreSQL은 Neon을, 이미지 업로드는 선택적으로 S3를 사용합니다.

## 주요 기능

- bcrypt와 단기 만료 HS256 JWT를 사용한 이메일/비밀번호 회원가입 및 로그인
- 사용자 프로필 수정 및 소프트 삭제
- 카테고리(`GENERAL`, `QUESTION`, `NOTICE`), 이미지 메타데이터, 검색, 정렬, 페이지네이션, 조회수, 좋아요를 지원하는 게시글 CRUD
- 댓글 CRUD
- `USER`/`ADMIN` 권한 분리 — 공지 발행 및 사용자 비활성화는 관리자만 가능
- JPEG, PNG, WebP(최대 5MB) 파일에 대한 10분짜리 S3 presigned PUT URL(선택 사항)
- JSON 접속 로그, 요청 ID, liveness/readiness 프로브, 정상 종료(graceful shutdown)
- 버전 관리되는 SQL 마이그레이션과 멱등성(idempotent)을 보장하는 관리자 시드 명령

## 로컬 실행

요구 사항: Go 1.26+, Docker, Docker Compose

```bash
cp .env.example .env
docker compose up -d postgres
set -a; source .env; set +a
go run ./cmd/migrate -action up
go run ./cmd/seed
go run ./cmd/server
```

API는 `http://localhost:8080`에서 대기하며, OpenAPI 문서는 `http://localhost:8080/openapi.yaml`에서 제공됩니다.

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"user@example.com","displayName":"User","password":"password123"}'
```

## 환경 설정

| 변수 | 용도 | 기본값 |
| --- | --- | --- |
| `HTTP_ADDR` | 서버 리스닝 주소 | `:8080` |
| `DATABASE_URL` | 런타임 PostgreSQL URL, Neon의 pooled URL 사용 | 필수 |
| `MIGRATION_DATABASE_URL` | 마이그레이션용 URL, Neon의 direct URL 사용 | `DATABASE_URL` |
| `JWT_SECRET` | HS256 시크릿, 32자 이상 | 필수 |
| `JWT_TTL` | 액세스 토큰 유효 기간 | `1h` |
| `CORS_ORIGINS` | 콤마로 구분된 프론트엔드 origin 목록 | `http://localhost:3000` |
| `ADMIN_EMAIL`, `ADMIN_PASSWORD` | `cmd/seed`에서 사용하는 값 | 서버 실행 시 선택 |
| `AWS_REGION` | S3 리전 | `ap-northeast-2` |
| `S3_BUCKET` | 업로드 버킷, 비워두면 업로드 기능 비활성화 | 비어 있음 |
| `S3_PUBLIC_BASE_URL` | CloudFront 또는 공개 S3 오브젝트 base URL | 비어 있음 |

ECS에서는 데이터베이스 URL, JWT 시크릿 등의 민감한 값을 태스크 정의의 Secrets Manager 참조를 통해 주입하세요. 애플리케이션은 오직 환경 변수만 읽기 때문에 시크릿 제공 방식에 종속되지 않습니다.

업로드 기능을 사용하려면 Fargate 태스크 역할에 `arn:aws:s3:::<bucket>/posts/*`에 대한 `s3:PutObject` 권한이 필요합니다. AWS SDK의 기본 자격 증명 탐색이 태스크 역할을 자동으로 사용하므로, 환경 변수 파일에 AWS 액세스 키를 넣지 마세요.

## 주요 명령어

```bash
make build
make test
make migrate-up
make migrate-down
make seed
```

통합 테스트는 격리된 PostgreSQL 데이터베이스가 필요합니다.

```bash
TEST_DATABASE_URL='postgresql://postgres:postgres@localhost:5432/velocity_test?sslmode=disable' \
ALLOW_INTEGRATION_DB_RESET=true \
go test ./internal/integration -v
```

통합 테스트는 애플리케이션 테이블을 truncate하므로, 공유 중이거나 운영 중인 데이터베이스를 절대 대상으로 지정하지 마세요.

## 배포 규약

- 컨테이너 포트: `8080`
- ALB 헬스 체크: `/health/ready`
- Liveness 신호: `/health/live`
- 종료 처리: `SIGTERM` 수신, 10초의 정상 종료(graceful shutdown) 타임아웃
- 데이터베이스 변경: 애플리케이션 태스크를 배포하기 전에 Neon의 direct URL을 사용해 `cmd/migrate`를 일회성 ECS 태스크로 실행

## 라이선스

이 프로젝트는 [MIT 라이선스](./LICENSE)를 따릅니다.
