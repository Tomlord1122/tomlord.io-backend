[English](./README.md)

# tomlord.io-backend 開發指南

本指南詳細介紹了如何使用 Gin (HTTP)、Viper (設定)、sqlc + pgx (PostgreSQL)、Gorilla WebSocket、Goth (OAuth)、Docker/Compose 和 Fly.io 來開發、執行和部署一個生產等級的 Golang 後端專案。

## 技術棧

### 前置需求

開始前，請確保已安裝以下工具：

```bash
# 安裝 sqlc，用於從 SQL 產生 Go 程式碼
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# (選用) 安裝 go-blueprint，用於專案初始化
go install github.com/melkeydev/go-blueprint@latest

# 安裝 golang-migrate，用於資料庫遷移
brew install golang-migrate
```

### Go Modules

本專案依賴以下 Go Modules：

```go
require (
	github.com/gin-contrib/cors v1.7.6
	github.com/gin-gonic/gin v1.10.1
	github.com/golang-jwt/jwt/v5 v5.3.0
	github.com/google/uuid v1.6.0
	github.com/gorilla/sessions v1.4.0
	github.com/gorilla/websocket v1.5.3
	github.com/jackc/pgx/v5 v5.7.5
	github.com/markbates/goth v1.81.0
	github.com/spf13/viper v1.20.1
)
```

## Makefile 與 Docker

我們使用 `Makefile` 來簡化常見的開發任務，並透過 `Docker` 進行容器化。

### Makefile

`Makefile` 將常用指令包裝起來以加速開發流程。它會動態地從 `.env` 檔案載入本地開發環境變數，並在 `APP_ENV` 設為 `production` 時使用生產環境的變數。

主要指令包括：
- `make build`：建置應用程式。
- `make run`：在本地執行應用程式。
- `make test`：執行所有測試。
- `make docker-up`：使用 Docker Compose 啟動服務。
- `make migrateup`：套用所有資料庫遷移。
- `make migratedown`：還原所有資料庫遷移。
- `make sqlc`：從 SQL 查詢產生 Go 程式碼。
- `make setup`：一個方便的指令，用來啟動 Docker 容器、等待資料庫就緒後執行遷移。
- `make watch`：使用 `air` 啟用熱重載。

**Makefile 片段:**
```makefile
# Go 專案的簡易 Makefile
ENV_FILE ?= .env
ifneq (,$(wildcard $(ENV_FILE)))
include $(ENV_FILE)
endif

# Docker 操作的容器名稱
DB_CONTAINER_NAME=psql_bp

# 資料庫遷移 URL
DB_URL_LOCAL=postgresql://${BLUEPRINT_DB_USERNAME}:${BLUEPRINT_DB_PASSWORD}@localhost:${BLUEPRINT_DB_PORT}/${BLUEPRINT_DB_DATABASE}?sslmode=disable
DB_URL_PROD=postgresql://${BLUEPRINT_DB_USERNAME}:${BLUEPRINT_DB_PASSWORD}@${BLUEPRINT_DB_HOST}:${BLUEPRINT_DB_PORT}/${BLUEPRINT_DB_DATABASE}?sslmode=require
DB_URL=$(if $(filter production,$(APP_ENV)),$(DB_URL_PROD),$(DB_URL_LOCAL))

# ... 其他指令

setup: docker-up wait-for-db migrateup
	@echo "後端服務已就緒!"

wait-for-db:
	@echo "正在等待資料庫就緒..."
	# ... 檢查資料庫是否就緒的邏輯
```

### Dockerfile

`Dockerfile` 採用多階段建置 (multi-stage build) 來建立一個輕量且最佳化的生產環境映像檔。

- **Builder Stage**: 在 `golang:1.24.4-alpine` 映像檔中編譯 Go 應用程式。
- **Production Stage**: 將編譯好的二進位檔和必要的憑證複製到一個輕量的 `alpine:3.20.1` 映像檔中，並為了安全性建立一個非 root 使用者。
- **Health Check**: 包含一個 `HEALTHCHECK` 來確保容器正常運作。

**Dockerfile 片段:**
```dockerfile
FROM golang:1.24.4-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -o main cmd/api/main.go

# Production stage
FROM alpine:3.20.1 AS prod

COPY --from=builder /app/main /main
USER appuser
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -fsS "http://localhost:${PORT:-8080}/health" || exit 1
ENTRYPOINT ["/main"]
```

### docker-compose.yml

`docker-compose.yml` 檔案用於在本地建立一個 PostgreSQL 資料庫，方便開發與測試。

```yaml
services:
  psql_bp:
    container_name: psql_bp
    image: postgres:17.5-alpine3.22
    restart: unless-stopped
    environment:
      POSTGRES_DB: ${BLUEPRINT_DB_DATABASE}
      POSTGRES_USER: ${BLUEPRINT_DB_USERNAME}
      POSTGRES_PASSWORD: ${BLUEPRINT_DB_PASSWORD}
    ports:
      - "${BLUEPRINT_DB_PORT}:5432"
    volumes:
      - psql_volume_tomlord:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "sh -c 'pg_isready -U ${BLUEPRINT_DB_USERNAME} -d ${BLUEPRINT_DB_DATABASE}'"]
      # ... healthcheck 參數
networks:
  tomlord_network:
```

## PostgreSQL 資料庫結構與遷移

我們使用 `golang-migrate` 進行資料庫結構遷移，並用 `sqlc` 從 SQL 產生型別安全的 Go 程式碼。

### sqlc 設定

執行 `sqlc init` 後，設定 `sqlc.yaml` 來定義來源目錄和輸出套件。

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "./sqlc/queries"
    schema: "./sqlc/migrations"
    gen:
      go:
        package: "db"
        out: "./internal/db_sqlc"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_prepared_queries: false
        emit_interface: true
        emit_exact_table_names: false
        emit_empty_slices: true 
```

- `emit_json_tags`: 讓 struct 序列化時使用 `json` 標籤，方便 API 回應。
- `emit_empty_slices`: 確保回傳多筆紀錄的查詢在沒有結果時回傳空 slice 而非 `nil`。
- `emit_interface`: 為所有查詢產生一個 `Querier` interface，有利於後續的 mock 與測試。

### 遷移檔案

遷移檔案存放於 `./sqlc/migrations`。每個遷移都包含一個 `up` 和一個 `down` 檔案。

- **`001_create_users_table.up.sql`**: 建立 `users` 表，用來儲存 Google OAuth 的使用者資訊。
- **`002_create_messages_table.up.sql`**: 建立 `messages` 表，並設定指向 `users` 的 foreign key。
- **`003_create_message_thumbs_table.up.sql`**: 用於儲存留言按讚的關聯表，對 `(message_id, user_id)` 設有唯一約束，並使用 `ON DELETE CASCADE` 維護資料完整性。
- **`004_create_blogs_table.up.sql`**: 建立 `blogs` 表，支援標籤 (text 陣列) 並使用 GIN 索引來提升標籤搜尋效率。
- **`005_add_foreign_key_post_slug.up.sql`**: 為 `messages.post_slug` 加上指向 `blogs.slug` 的 foreign key 約束。

## SQLC CRUD 程式碼

`sqlc` 的 SQL 查詢檔案位於 `./sqlc/queries`。每個查詢都必須有一個特殊的註解開頭，作為 `sqlc` 的進入點。

- **`-- name: <QueryName> :one`**: 表示查詢應回傳單筆紀錄。
- **`-- name: <QueryName> :many`**: 表示查詢應回傳多筆紀錄。
- **`-- name: <QueryName> :exec`**: 表示執行一個操作而不回傳任何紀錄。

### 查詢檔案

- **`users.sql`**: 包含建立和讀取使用者的查詢。
- **`messages.sql`**: 包含對留言的 CRUD 操作、依部落格 slug 讀取留言以及更新按讚數的查詢。使用 `LEFT JOIN` 和 `COALESCE` 來正確處理沒有按讚的留言。
- **`message_thumbs.sql`**: 管理留言按讚。使用 `ON CONFLICT DO NOTHING` 來實現冪等的按讚操作，並用 `EXISTS()` 檢查使用者是否已按讚。
- **`blogs.sql`**: 處理部落格文章的讀取，包含依標籤或語言篩選。同時也包含一個查詢，能在讀取文章時一併取得留言數量。

### 產生的 Go 程式碼

執行 `sqlc generate` 後，會在 `./internal/db_sqlc` 中產生以下檔案：

- **`models.go`**: 包含每個資料庫 table 對應的 Go struct 定義 (例如 `Blog`, `User`)。欄位使用 `pgtype` 來正確處理 nullable 的值。
- **`querier.go`**: 匯出一個 `Querier` interface，其中列出了所有產生的查詢方法。Service 層應依賴此 interface。
- **`db.go`**: 提供一個 `New()` 建構子來建立 `*Queries` struct，以及一個 `WithTx()` 方法讓我們在 transaction 中安全地執行查詢。

Service 層 (`internal/services/*`) 只透過我們封裝的 `DBService` 所提供的 `Querier` interface 與資料庫互動，該 `DBService` 透過 `WithTx` 提供 transaction 支援。

## 建立 Server

### 入口點: `cmd/api/main.go`

應用程式的啟動流程如下：
1.  `config.Load()`: 載入環境變數。
2.  `server.NewServer()`: 初始化所有 services 和依賴。
3.  `server.ListenAndServe()`: 啟動 HTTP server。
4.  設定 graceful shutdown 機制來處理終止信號。

### Server 組裝: `internal/server/server.go`

`NewServer` 函式負責依賴注入：
- 初始化 `DBService`, `AuthService`, `MessageService`, `BlogService`。
- 使用 `JWT_SECRET` 建立 `AuthMiddleware` 實例。
- 在獨立的 goroutine 中啟動 `WebSocket Hub`。
- 設定並回傳一個掛載了 Gin router 和適當 timeout 的 `*http.Server`。

### Routes/Handlers: `internal/server/routes.go`

Router 的設定如下：
- **Global middleware**: `SetupCORS()` 用於處理跨域請求。
- **公開路由**: `/`, `/health`, `/debug/jwt`。
- **OAuth 路由**: `/auth/:provider` 和 `/auth/:provider/callback`，透過 Goth 處理 Google 登入。
- **API 路由**:
    - `/api/blogs`: 用於讀取部落格文章。在非生產環境下，開放新增與更新的端點。
    - `/api/messages`: 提供完整的留言 CRUD 功能和一個按讚的切換端點。
    - `/api/sync-blogs`: 一個一次性的端點，用於同步前端 build 後的部落格文章內容。
- **WebSocket 路由**: `/ws` 處理即時連線。

## 建立 Service 層

### BlogService: `internal/services/blog.go`

- **職責**: 處理部落格文章的業務邏輯。
- **DTOs**: 定義 `CreateBlogRequest` 和 `BlogInfo` 等 request/response struct，將 API 與資料庫模型解耦。
- **邏輯**:
    - 轉換輸入型別 (例如，string 轉為 `pgtype.Date`)。
    - 根據請求參數 (如 `tag`, `lang`) 選擇對應的 `sqlc` 查詢。
    - 將資料庫模型映射回 DTOs 以便回傳 JSON，過程中將 `pgtype` 值轉為基本型別。

### MessageService: `internal/services/message.go`

- **職責**: 管理留言的建立、讀取和互動。
- **邏輯**:
    - **新增**: 新增一則留言後，會重新讀取該留言以填充使用者資訊 (`user_name`, `user_picture_url`)。
    - **刪除**: 實現了兩種刪除路徑：一種給留言作者本人，另一種給超級使用者。
    - **切換按讚**: 使用資料庫 transaction (`WithTx`) 來原子化地檢查讚是否存在，然後決定要新增還是刪除。留言的 `thumb_count` 會在同一個 transaction 內更新，以避免競爭條件 (race condition)。

## 設定 Configuration 和 CORS

### 設定載入: `internal/config/config.go`

- **Viper**: 用於從環境變數載入設定。
- **`.env` 支援**: 當 `APP_ENV` 為 `local` 或未設定時，會自動載入 `.env` 檔案。
- **預設值**: 為 `PORT` 和 `FRONTEND_URL` 設定預設值。

### CORS: `internal/server/cors.go` & `internal/originpolicy/origins.go`

- **集中化策略**: `originpolicy.AllowedOrigins()` 是 HTTP 和 WebSocket 連線所允許來源的單一真實來源 (Single Source of Truth)。
- **區分環境**:
    - **Production**: 使用 `ALLOWED_ORIGINS` 環境變數，若無則退回 `FRONTEND_URL`，最終預設為 `https://tomlord.fyi`。
    - **Development**: 預設為 `http://localhost:5173`。
- **Gin Middleware**: `SetupCORS()` 使用 `gin-contrib/cors` middleware，並根據上述策略設定允許的來源。

## 建立 Middleware

### JWT Middleware: `internal/middleware/auth.go`

- **`GenerateJWT`**: 簽發一個包含使用者 claims 且效期為 1 小時的 JWT。
- **`ValidateJWT`**: 驗證 token 簽章並解析 claims。
- **`RequireAuth`**: 一個 middleware，若請求中沒有合法的 JWT 則中止請求。它會將使用者資訊注入 Gin context。
- **`OptionalAuth`**: 一個 middleware，若請求中有 token 則注入使用者資訊，但即使沒有 token 也不會失敗。用於那些對已登入使用者顯示額外資訊的公開端點。
- **`RequireSuperUserOrOwner`**: 檢查是否具有超級使用者權限 (基於一個寫死的 email) 或是否為擁有者。
- **`RequireSyncToken`**: `/api/sync-blogs` 端點的獨立驗證機制。

### OAuth 與使用者 Upsert: `internal/auth/oauth.go`

- **Goth**: 處理與 Google 的 OAuth2 流程。
- **User Upsert**: 在 callback handler 中，呼叫 `AuthService.CreateOrUpdateUser`。它會根據 `google_id` 檢查使用者是否存在，然後決定是建立新使用者還是更新現有使用者。
- **Redirect**: 成功登入後，它會產生一個 JWT，將其設定在 cookie 中，並將使用者重導向到前端，同時在 query 參數中附上 token。

## 建立 WebSocket

### 核心: `internal/websocket/hub.go`

- **`Hub`**: 管理 clients、rooms 和訊息廣播。
- **`Client`**: 代表一個 WebSocket 連線，持有其訂閱的 rooms 和一個用於發送訊息的 buffered channel。
- **Origin Check**: upgrader 的 `CheckOrigin` 函式使用 `originpolicy.AllowedOrigins()` 來強制執行與 HTTP 端點相同的 CORS 策略。
- **Ping/Pong**: 實作心跳機制以偵測並清理無效連線。
- **動態訂閱**: Client 可以發送 JSON 訊息來訂閱或取消訂閱 rooms (例如 `{"action": "subscribe", "rooms": ["post-slug-1"]}`)。

### Server 整合

- **路由**: `GET /ws` 受 `OptionalAuth()` 保護，以便在可能的情況下將連線與 `userID` 關聯。
- **廣播**: 當相關事件發生時，Service 會向 hub 廣播事件：
    - `MessageTypeNewComment`：有新留言時。
    - `MessageTypeThumbUpdate`：留言被按讚或取消讚時。
    - `MessageTypeCommentDelete`：留言被刪除時。
- **房間命名**: 房間以 `post_slug` 命名，以便只將事件傳遞給正在觀看該特定文章的 clients。

## 部署

此專案已設定好在 Fly.io 上部署。

### Fly.io 指令

```bash
# 安裝 Fly CLI
brew install flyctl

# 登入 Fly
fly auth login

# 初始化應用程式
fly init

# 設定生產環境密鑰
fly secrets set \
  PORT=8080 \
  APP_ENV=production \
  BLUEPRINT_DB_HOST=[your_production_db_host] \
  # ... 其他密鑰
  FRONTEND_URL=[your_frontend_url] \
  ALLOWED_ORIGINS=[your_allowed_origins] \
  SYNC_SESSION_SECRET=[your_session_secret]

# 列出密鑰以供確認
fly secrets list

# 部署應用程式
fly deploy
```
