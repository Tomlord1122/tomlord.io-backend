# 🧪 tomlord.io-backend 完整測試指南

這份指南將教你如何使用 **TablePlus** 和 **Postman** 來測試你的 Go 後端服務。

## 📋 目錄
- [環境準備](#環境準備)
- [啟動服務](#啟動服務)
- [TablePlus 數據庫連接](#tableplus-數據庫連接)
- [Postman API 測試](#postman-api-測試)
- [認證流程測試](#認證流程測試)
- [消息API測試](#消息api測試)
- [常見問題解決](#常見問題解決)

---

## 🚀 環境準備

### 1. 設置環境變數
首先複製環境變數範例文件：

```bash
# 在 tomlord.io-backend 目錄下執行
cp .env.example .env
```

### 2. 配置 Google OAuth (可選)
如果你想測試完整的OAuth流程，需要：

1. 去 [Google Cloud Console](https://console.cloud.google.com/)
2. 創建或選擇一個項目
3. 啟用 Google+ API
4. 創建 OAuth 2.0 憑證
5. 設置重定向 URI: `http://localhost:8080/auth/google/callback`
6. 將憑證填入 `.env` 文件：

```env
GOOGLE_CLIENT_ID=你的實際Google-Client-ID
GOOGLE_CLIENT_SECRET=你的實際Google-Client-Secret
```

> 💡 **提示**: 如果暫時不想設置Google OAuth，可以跳過這步，後面我會教你如何手動創建測試用戶。

---

## 🏃‍♂️ 啟動服務

### 一鍵啟動（推薦）
```bash
# 這個命令會：
# 1. 啟動 PostgreSQL 數據庫容器
# 2. 等待數據庫準備就緒
# 3. 運行所有 migrations
# 4. 啟動後端服務
make setup
```

### 分步啟動（調試用）
如果遇到問題，可以分步執行：

```bash
# 1. 啟動數據庫
make docker-up

# 2. 等待數據庫就緒
make wait-for-db

# 3. 運行 migrations
make migrateup

# 4. 啟動後端服務
make run
```

### 驗證服務狀態
```bash
# 檢查 Docker 容器狀態
docker ps

# 應該看到類似這樣的輸出：
# CONTAINER ID   IMAGE                      COMMAND                  CREATED         STATUS                   PORTS                    NAMES
# xxx            postgres:15-alpine         "docker-entrypoint.s…"  2 minutes ago   Up 2 minutes             0.0.0.0:5432->5432/tcp   psql_bp
# xxx            tomlordio-backend-app      "./main"                 1 minute ago    Up 1 minute              0.0.0.0:8080->8080/tcp   tomlordio-backend-app-1

# 測試後端是否啟動成功
curl http://localhost:8080/health
```

---

## 🗄️ TablePlus 數據庫連接

### 連接配置
1. 打開 TablePlus
2. 點擊 "Create a new connection"
3. 選擇 "PostgreSQL"
4. 填入以下連接信息：

| 欄位     | 值                    |
|----------|-----------------------|
| Name     | tomlord.io-backend    |
| Host     | localhost             |
| Port     | 5432                  |
| User     | melkey                |
| Password | password1234          |
| Database | blueprint             |

### 驗證數據庫結構
連接成功後，你應該會看到以下表：

```sql
-- 查看所有表
\dt

-- 應該有這些表：
-- users            - 用戶信息表
-- messages         - 消息/評論表
-- message_thumbs    - 點讚表
-- schema_migrations - 遷移歷史表
```

### 查看表結構
```sql
-- 查看用戶表結構
\d users;

-- 查看消息表結構
\d messages;

-- 查看點讚表結構
\d message_thumbs;
```

---

## 🔧 Postman API 測試

### 創建 Collection
1. 打開 Postman
2. 創建新的 Collection 叫 "tomlord.io-backend"
3. 在 Collection 的 Variables tab 設置：
   - Variable: `baseUrl`
   - Initial Value: `http://localhost:8080`
   - Current Value: `http://localhost:8080`

### 基礎API測試

#### 1. 健康檢查
```http
GET {{baseUrl}}/health
```

**預期回應:**
```json
{
  "status": "ok",
  "database": "connected",
  "timestamp": "2024-01-01T00:00:00Z"
}
```

#### 2. Hello World
```http
GET {{baseUrl}}/
```

**預期回應:**
```json
{
  "message": "Hello World from tomlord.io backend"
}
```

---

## 🔐 認證流程測試

由於你的系統使用 Google OAuth + JWT，我們有兩種測試方式：

### 方式一：完整 OAuth 流程（需要 Google 憑證）

#### 1. 啟動 OAuth 流程
在瀏覽器中訪問：
```
http://localhost:8080/auth/google
```

這會重定向到 Google 登錄頁面，登錄後會重定向回你的前端。

#### 2. 檢查用戶信息
登錄成功後，可以用瀏覽器的 cookie 或返回的 JWT token 測試：

```http
GET {{baseUrl}}/auth/me
Authorization: Bearer <你的JWT-token>
```

### 方式二：手動創建測試用戶（推薦用於API測試）

#### 1. 在 TablePlus 中創建測試用戶
```sql
-- 插入測試用戶
INSERT INTO users (id, google_id, email, name, picture_url) 
VALUES (
    gen_random_uuid(),
    'test-google-id-12345', 
    'test@example.com', 
    'Test User',
    'https://via.placeholder.com/150'
) RETURNING *;

-- 查看創建的用戶，記下 id
SELECT * FROM users WHERE email = 'test@example.com';
```

#### 2. 生成測試 JWT Token
創建一個臨時的測試腳本：

```bash
# 創建測試目錄
mkdir -p cmd/test-jwt
```

在 `cmd/test-jwt/main.go` 創建：

```go
package main

import (
    "fmt"
    "time"
    "github.com/golang-jwt/jwt/v5"
)

type Claims struct {
    UserID   string `json:"user_id"`
    Email    string `json:"email"`
    Name     string `json:"name"`
    GoogleID string `json:"google_id"`
    jwt.RegisteredClaims
}

func main() {
    // 使用與 .env 文件相同的 JWT_SECRET
    jwtSecret := "your-super-secret-jwt-key-change-in-production"
    
    // 使用你剛才插入的用戶信息（記得替換 UserID）
    claims := Claims{
        UserID:   "你從數據庫查到的UUID", // 替換成實際的UUID
        Email:    "test@example.com",
        Name:     "Test User",
        GoogleID: "test-google-id-12345",
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            Issuer:    "tomlord.io-backend",
        },
    }
    
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    tokenString, err := token.SignedString([]byte(jwtSecret))
    if err != nil {
        panic(err)
    }
    
    fmt.Println("JWT Token:")
    fmt.Println(tokenString)
    fmt.Println("\n你可以將這個token用於Postman測試！")
}
```

生成token：
```bash
cd cmd/test-jwt
go mod init test-jwt
go get github.com/golang-jwt/jwt/v5
go run main.go
```

#### 3. 在 Postman 中設置 Authorization
1. 在 Collection 的 Authorization tab 選擇 "Bearer Token"
2. 將生成的 JWT token 貼上
3. 或者在每個需要認證的請求中單獨設置

#### 4. 測試用戶信息
```http
GET {{baseUrl}}/auth/me
Authorization: Bearer <你的JWT-token>
```

**預期回應:**
```json
{
  "user": {
    "id": "uuid-string",
    "google_id": "test-google-id-12345",
    "email": "test@example.com", 
    "name": "Test User",
    "picture_url": "https://via.placeholder.com/150"
  }
}
```

---

## 💬 消息API測試

### 1. 創建消息（需要認證）
```http
POST {{baseUrl}}/api/messages
Authorization: Bearer <你的JWT-token>
Content-Type: application/json

{
  "post_slug": "test-post-1",
  "message": "這是我的第一條測試消息！🎉"
}
```

**預期回應:**
```json
{
  "message": {
    "id": "message-uuid",
    "post_slug": "test-post-1", 
    "message": "這是我的第一條測試消息！🎉",
    "user_id": "user-uuid",
    "user_name": "Test User",
    "user_picture_url": "https://via.placeholder.com/150",
    "thumb_count": 0,
    "user_thumbed": false,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

### 2. 獲取消息列表（公開API）
```http
GET {{baseUrl}}/api/messages/post/test-post-1
```

你也可以加上查詢參數：
```http
GET {{baseUrl}}/api/messages/post/test-post-1?limit=10&offset=0
```

### 3. 更新消息（需要認證，只能更新自己的消息）
```http
PUT {{baseUrl}}/api/messages/{{message_id}}
Authorization: Bearer <你的JWT-token>
Content-Type: application/json

{
  "message": "這是更新後的消息內容！✨"
}
```

### 4. 點讚/取消點讚（需要認證）
```http
POST {{baseUrl}}/api/messages/{{message_id}}/thumb
Authorization: Bearer <你的JWT-token>
```

**預期回應:**
```json
{
  "thumbed": true,
  "thumb_count": 1
}
```

### 5. 刪除消息（需要認證，只能刪除自己的消息）
```http
DELETE {{baseUrl}}/api/messages/{{message_id}}
Authorization: Bearer <你的JWT-token>
```

---

## 🔍 測試數據驗證

### 在 TablePlus 中查看數據

#### 查看所有用戶
```sql
SELECT * FROM users ORDER BY created_at DESC;
```

#### 查看所有消息（含用戶信息）
```sql
SELECT 
    m.id,
    m.post_slug,
    m.message,
    u.name as user_name,
    u.email as user_email,
    m.created_at,
    m.updated_at
FROM messages m
JOIN users u ON m.user_id = u.id
ORDER BY m.created_at DESC;
```

#### 查看點讚統計
```sql
SELECT 
    m.post_slug,
    m.message,
    COUNT(mt.id) as thumb_count,
    ARRAY_AGG(u.name) as thumbed_by_users
FROM messages m
LEFT JOIN message_thumbs mt ON m.id = mt.message_id
LEFT JOIN users u ON mt.user_id = u.id
GROUP BY m.id, m.post_slug, m.message
ORDER BY thumb_count DESC;
```

#### 查看特定用戶的所有點讚
```sql
SELECT 
    m.message,
    m.post_slug,
    mt.created_at as thumbed_at
FROM message_thumbs mt
JOIN messages m ON mt.message_id = m.id
WHERE mt.user_id = '你的用戶UUID'
ORDER BY mt.created_at DESC;
```

---

## 🐛 常見問題解決

### 1. 數據庫連接失敗
```bash
# 檢查 Docker 容器狀態
docker ps

# 重啟數據庫
make docker-down
make docker-up

# 等待數據庫就緒
make wait-for-db
```

### 2. JWT Token 驗證失敗
- 檢查 `.env` 文件中的 `JWT_SECRET` 是否與生成 token 時使用的一致
- 檢查 token 是否過期
- 確保 Authorization header 格式正確：`Bearer <token>`

### 3. OAuth 重定向錯誤
- 檢查 Google OAuth 設置中的重定向 URI
- 確保 `.env` 文件中的 `GOOGLE_CALLBACK_URL` 正確

### 4. 權限錯誤（403/401）
- 確保使用了正確的 JWT token
- 檢查你是否嘗試編輯/刪除別人的消息（只能操作自己的）

### 5. 查看詳細錯誤日誌
```bash
# 查看後端服務日誌
make docker-logs

# 或者直接運行後端看詳細輸出
make run
```

---

## 📚 完整的 Postman Collection 建議

建議創建以下 requests（按這個順序測試）：

### 基礎測試
1. **Health Check** - `GET /health`
2. **Hello World** - `GET /`

### 認證測試  
3. **Get Me** - `GET /auth/me` (需要 Bearer token)
4. **Logout** - `POST /auth/logout` (需要 Bearer token)

### 消息管理
5. **Create Message** - `POST /api/messages` (需要 Bearer token)
6. **Get Messages** - `GET /api/messages/post/:slug`
7. **Update Message** - `PUT /api/messages/:id` (需要 Bearer token)
8. **Delete Message** - `DELETE /api/messages/:id` (需要 Bearer token)

### 互動功能
9. **Toggle Thumb** - `POST /api/messages/:id/thumb` (需要 Bearer token)

### Collection Variables
在 Collection 設置中添加這些變量：
- `baseUrl`: `http://localhost:8080`
- `auth_token`: `<你的JWT-token>`
- `test_message_id`: `<測試消息的ID>`

---

## 🎯 測試完成確認清單

- [ ] ✅ 數據庫連接成功（TablePlus）
- [ ] ✅ 後端服務啟動成功
- [ ] ✅ Health check 通過
- [ ] ✅ 創建測試用戶成功
- [ ] ✅ JWT token 生成成功
- [ ] ✅ JWT 認證通過 (`/auth/me`)
- [ ] ✅ 創建消息成功
- [ ] ✅ 獲取消息列表成功
- [ ] ✅ 更新消息成功
- [ ] ✅ 點讚功能正常
- [ ] ✅ 刪除消息成功
- [ ] ✅ 權限控制正常（無法編輯他人消息）

當所有項目都打勾時，恭喜你！你的後端API已經可以正常工作了！🎉

---

## 📞 需要幫助？

如果測試過程中遇到任何問題，可以：

1. 檢查 Docker 容器狀態：`docker ps`
2. 查看後端日誌：`make docker-logs`
3. 重新啟動服務：`make docker-down && make setup`
4. 檢查數據庫連接：`make wait-for-db`

記住，如果你是編程新手，這些步驟看起來可能很複雜，但每一步都是為了確保你的後端服務能夠正確工作。一步一步來，不要急！ 🚀
