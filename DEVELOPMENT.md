# 個人部落格後端資料庫開發指南

## 📋 目錄
1. [專案概述](#專案概述)
2. [資料庫設計思路](#資料庫設計思路)
3. [技術架構](#技術架構)
4. [資料庫結構設計](#資料庫結構設計)
5. [實作步驟](#實作步驟)
6. [最佳實踐與設計模式](#最佳實踐與設計模式)
7. [效能優化考量](#效能優化考量)
8. [安全性考量](#安全性考量)
9. [測試策略](#測試策略)
10. [部署與維護](#部署與維護)

---

## 🎯 專案概述

本專案是一個完整的個人部落格後端系統，採用現代化的技術棧和架構設計。系統支援多語言部落格文章、用戶評論、點讚功能，以及 OAuth 身份驗證。

### 核心功能
- **用戶管理**: OAuth 登入 (Google)、用戶資料管理
- **部落格系統**: 多語言文章、標籤分類、SEO 優化
- **互動功能**: 評論系統、點讚機制
- **即時通訊**: WebSocket 支援即時更新

---

## 🧠 資料庫設計思路

### 設計原則

#### 1. **正規化設計**
我們採用第三正規化 (3NF) 設計，避免資料冗餘，確保資料一致性：
- 用戶資料獨立存儲
- 部落格文章與評論分離
- 點讚記錄通過關聯表實現

#### 2. **擴展性考量**
- 使用 UUID 作為主鍵，支援分散式部署
- 設計彈性的標籤系統 (PostgreSQL 陣列類型)
- 支援多語言內容

#### 3. **效能優化導向**
- 針對常用查詢建立複合索引
- 使用 GIN 索引優化陣列搜尋
- 設計高效的關聯查詢

### 實體關係分析

```
Users (用戶)
  ↓ (1:N)
Messages (評論)
  ↓ (N:1)
Blogs (部落格文章)
  ↓ (N:N)
MessageThumbs (點讚記錄)
```

**為什麼這樣設計？**
- **用戶與評論**: 一個用戶可以發表多個評論，但每個評論只能屬於一個用戶
- **評論與文章**: 一篇文章可以有多個評論，但每個評論只能屬於一篇文章
- **點讚記錄**: 用戶和評論之間是多對多關係，需要中間表來記錄

---

## 🏗️ 技術架構

### 技術棧選擇

| 組件 | 技術選擇 | 選擇理由 |
|------|----------|----------|
| **資料庫** | PostgreSQL | 支援 JSON、陣列類型，效能優異 |
| **查詢生成** | sqlc | 類型安全的 SQL 查詢，避免 ORM 複雜性 |
| **後端框架** | Gin | 輕量級、高效能的 HTTP 框架 |
| **身份驗證** | JWT + OAuth | 無狀態認證，支援第三方登入 |
| **即時通訊** | WebSocket | 低延遲的雙向通訊 |

### 為什麼選擇這些技術？

1. **PostgreSQL**: 
   - 支援陣列類型 (`tags TEXT[]`)，適合標籤系統
   - 強大的全文搜尋能力
   - 優秀的並發處理能力

2. **sqlc**:
   - 生成類型安全的 Go 程式碼
   - 避免 ORM 的 N+1 查詢問題
   - 更好的查詢效能控制

3. **Gin**:
   - 輕量級，適合微服務架構
   - 優秀的中間件支援
   - 活躍的社群和豐富的文件

---

## 🗄️ 資料庫結構設計

### 1. 用戶表 (users)

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    google_id VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    picture_url TEXT,  
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

**設計考量**:
- **UUID 主鍵**: 避免自增 ID 的資訊洩露風險
- **google_id**: 支援 OAuth 登入，確保唯一性
- **email**: 作為備用識別方式
- **timestamps**: 追蹤記錄生命週期

**索引策略**:
```sql
CREATE INDEX idx_users_google_id ON users(google_id);
CREATE INDEX idx_users_email ON users(email);
```

### 2. 部落格表 (blogs)

```sql
CREATE TABLE blogs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(500) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    date DATE NOT NULL,
    lang VARCHAR(10) NOT NULL DEFAULT 'zh-tw',
    duration VARCHAR(20) NOT NULL DEFAULT '5min',
    tags TEXT[] DEFAULT '{}',
    description TEXT,
    is_published BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

**設計考量**:
- **slug**: SEO 友善的 URL，必須唯一
- **tags**: 使用 PostgreSQL 陣列類型，支援標籤搜尋
- **is_published**: 支援草稿和發布狀態
- **lang**: 多語言支援

**索引策略**:
```sql
CREATE INDEX idx_blogs_slug ON blogs(slug);
CREATE INDEX idx_blogs_date ON blogs(date DESC);
CREATE INDEX idx_blogs_published ON blogs(is_published);
CREATE INDEX idx_blogs_lang ON blogs(lang);
CREATE INDEX idx_blogs_tags ON blogs USING GIN(tags);
```

**GIN 索引的重要性**: 對於陣列類型的搜尋，GIN 索引能大幅提升 `@>` 和 `&&` 運算子的效能。

### 3. 評論表 (messages)

```sql
CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    blog_id UUID NOT NULL REFERENCES blogs(id) ON DELETE CASCADE,
    post_slug VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    thumb_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

**設計考量**:
- **外鍵約束**: 確保資料完整性
- **CASCADE 刪除**: 當用戶或文章被刪除時，相關評論自動清理
- **thumb_count**: 冗餘欄位，提升查詢效能

**索引策略**:
```sql
CREATE INDEX idx_messages_user_id ON messages(user_id);
CREATE INDEX idx_messages_blog_id ON messages(blog_id);
CREATE INDEX idx_messages_post_slug ON messages(post_slug);
CREATE INDEX idx_messages_created_at ON messages(created_at DESC);
CREATE INDEX idx_messages_blog_created ON messages(blog_id, created_at DESC);
```

### 4. 點讚表 (message_thumbs)

```sql
CREATE TABLE message_thumbs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(message_id, user_id)
);
```

**設計考量**:
- **複合唯一約束**: 防止用戶重複點讚同一評論
- **CASCADE 刪除**: 當評論或用戶被刪除時，點讚記錄自動清理

**索引策略**:
```sql
CREATE INDEX idx_message_thumbs_message_id ON message_thumbs(message_id);
CREATE INDEX idx_message_thumbs_user_id ON message_thumbs(user_id);
```

---

## 🚀 實作步驟

### 步驟 1: 環境準備

```bash
# 安裝必要工具
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# 建立專案結構
mkdir -p migrations sqlc/queries internal/db
```

### 步驟 2: 資料庫初始化

```bash
# 啟動 PostgreSQL (使用 Docker)
docker run --name postgres-blog \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=blog \
  -p 5432:5432 \
  -d postgres:15

# 建立資料庫
createdb -h localhost -U postgres blog
```

### 步驟 3: 建立遷移檔案

建立 `migrations/001_create_users_table.up.sql`:

```sql
-- 啟用 UUID 擴展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    google_id VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    picture_url TEXT,  
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 建立索引
CREATE INDEX idx_users_google_id ON users(google_id);
CREATE INDEX idx_users_email ON users(email);
```

**為什麼要啟用 uuid-ossp 擴展？**
PostgreSQL 預設不包含 UUID 生成函數，需要額外啟用。`uuid_generate_v4()` 函數能生成隨機的 UUID，確保全域唯一性。

### 步驟 4: 配置 sqlc

建立 `sqlc.yaml`:

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "sqlc/queries/"
    schema: "migrations/"
    gen:
      go:
        package: "db"
        out: "internal/db"
        emit_json_tags: true
        emit_prepared_queries: false
        emit_interface: true
        emit_exact_table_names: false
        emit_empty_slices: true
```

**配置說明**:
- `emit_json_tags`: 為結構體欄位生成 JSON 標籤
- `emit_interface`: 生成查詢介面，便於測試和模擬
- `emit_empty_slices`: 查詢無結果時返回空切片而非 nil

### 步驟 5: 撰寫 SQL 查詢

建立 `sqlc/queries/blogs.sql`:

```sql
-- name: CreateBlog :one
INSERT INTO blogs (title, slug, date, lang, duration, tags, description, is_published)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetBlogBySlug :one
SELECT * FROM blogs
WHERE slug = $1;

-- name: GetBlogs :many
SELECT * FROM blogs
WHERE is_published = true
ORDER BY date DESC
LIMIT $1 OFFSET $2;
```

**命名慣例**:
- `:one`: 返回單一記錄
- `:many`: 返回多筆記錄
- `:exec`: 不返回資料 (INSERT/UPDATE/DELETE)

### 步驟 6: 生成 Go 程式碼

```bash
# 執行遷移
migrate -path migrations -database "postgres://postgres:postgres@localhost:5432/blog?sslmode=disable" up

# 生成 sqlc 程式碼
sqlc generate
```

### 步驟 7: 實作資料庫服務

建立 `internal/database/service.go`:

```go
package database

import (
    "context"
    "database/sql"
    "fmt"
    
    "github.com/jackc/pgx/v5/pgxpool"
    "your-project/internal/db"
)

type Service struct {
    *db.Queries
    db *pgxpool.Pool
}

func NewService(db *pgxpool.Pool) *Service {
    return &Service{
        Queries: db.New(db),
        db:      db,
    }
}

// 事務支援
func (s *Service) WithTx(ctx context.Context, fn func(*db.Queries) error) error {
    tx, err := s.db.Begin(ctx)
    if err != nil {
        return fmt.Errorf("begin transaction: %w", err)
    }
    
    defer func() {
        if p := recover(); p != nil {
            tx.Rollback(ctx)
            panic(p)
        }
    }()
    
    if err := fn(db.New(tx)); err != nil {
        if rbErr := tx.Rollback(ctx); rbErr != nil {
            return fmt.Errorf("rollback: %w", rbErr)
        }
        return err
    }
    
    return tx.Commit(ctx)
}
```

**為什麼需要事務支援？**
某些操作需要原子性，例如創建評論時同時更新點讚計數。事務確保這些操作要麼全部成功，要麼全部失敗。

---

## 🎨 最佳實踐與設計模式

### 1. Repository Pattern

```go
type BlogRepository interface {
    Create(ctx context.Context, blog *CreateBlogParams) (*Blog, error)
    GetBySlug(ctx context.Context, slug string) (*Blog, error)
    List(ctx context.Context, limit, offset int32) ([]*Blog, error)
}

type blogRepository struct {
    db *db.Queries
}

func NewBlogRepository(db *db.Queries) BlogRepository {
    return &blogRepository{db: db}
}
```

**優點**:
- 抽象資料存取邏輯
- 便於單元測試
- 支援多種資料來源

### 2. Service Layer Pattern

```go
type BlogService struct {
    repo BlogRepository
    cache Cache
}

func (s *BlogService) CreateBlog(ctx context.Context, params CreateBlogParams) (*Blog, error) {
    // 業務邏輯驗證
    if err := s.validateBlog(params); err != nil {
        return nil, err
    }
    
    // 建立部落格
    blog, err := s.repo.Create(ctx, params)
    if err != nil {
        return nil, err
    }
    
    // 清除快取
    s.cache.Invalidate("blogs")
    
    return blog, nil
}
```

**職責分離**:
- **Repository**: 資料存取
- **Service**: 業務邏輯
- **Handler**: HTTP 請求處理

### 3. 錯誤處理策略

```go
type AppError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Details string `json:"details,omitempty"`
}

func (e *AppError) Error() string {
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

var (
    ErrBlogNotFound = &AppError{Code: "BLOG_NOT_FOUND", Message: "部落格文章不存在"}
    ErrInvalidInput = &AppError{Code: "INVALID_INPUT", Message: "輸入資料無效"}
)
```

**錯誤處理原則**:
- 使用結構化錯誤
- 提供錯誤代碼和詳細訊息
- 支援國際化

---

## ⚡ 效能優化考量

### 1. 索引策略

```sql
-- 複合索引優化常用查詢
CREATE INDEX idx_blogs_published_date_lang ON blogs(is_published, date DESC, lang);

-- 部分索引減少索引大小
CREATE INDEX idx_blogs_published_only ON blogs(date DESC, lang) 
WHERE is_published = true;
```

**索引設計原則**:
- 針對 WHERE 子句建立索引
- 考慮 ORDER BY 和 GROUP BY 需求
- 避免過度索引

### 2. 查詢優化

```sql
-- 使用 JOIN 避免 N+1 查詢
SELECT 
    b.*,
    COUNT(m.id) as comment_count,
    COUNT(DISTINCT mt.user_id) as like_count
FROM blogs b
LEFT JOIN messages m ON b.id = m.blog_id
LEFT JOIN message_thumbs mt ON m.id = mt.message_id
WHERE b.is_published = true
GROUP BY b.id
ORDER BY b.date DESC;
```

**查詢優化技巧**:
- 使用 JOIN 減少資料庫往返
- 適當使用子查詢
- 避免 SELECT *

### 3. 快取策略

```go
type Cache interface {
    Get(key string) (interface{}, bool)
    Set(key string, value interface{}, ttl time.Duration)
    Invalidate(pattern string)
}

func (s *BlogService) GetBlogBySlug(ctx context.Context, slug string) (*Blog, error) {
    // 嘗試從快取取得
    if cached, found := s.cache.Get(fmt.Sprintf("blog:%s", slug)); found {
        return cached.(*Blog), nil
    }
    
    // 從資料庫查詢
    blog, err := s.repo.GetBySlug(ctx, slug)
    if err != nil {
        return nil, err
    }
    
    // 存入快取
    s.cache.Set(fmt.Sprintf("blog:%s", slug), blog, time.Hour)
    
    return blog, nil
}
```

---

## 🔒 安全性考量

### 1. SQL 注入防護

```go
// sqlc 自動處理參數化查詢，防止 SQL 注入
func (s *BlogService) SearchBlogs(ctx context.Context, query string) ([]*Blog, error) {
    // 使用參數化查詢，query 參數會被安全處理
    return s.repo.SearchBlogs(ctx, query)
}
```

### 2. 輸入驗證

```go
func (s *BlogService) validateBlog(params CreateBlogParams) error {
    if params.Title == "" {
        return ErrInvalidInput
    }
    
    if params.Slug == "" {
        return ErrInvalidInput
    }
    
    // 驗證 slug 格式
    if !slugRegex.MatchString(params.Slug) {
        return ErrInvalidInput
    }
    
    return nil
}
```

### 3. 權限控制

```go
func (s *BlogService) UpdateBlog(ctx context.Context, id uuid.UUID, params UpdateBlogParams, userID uuid.UUID) (*Blog, error) {
    // 檢查用戶權限
    blog, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return nil, err
    }
    
    // 只有作者或管理員可以編輯
    if blog.AuthorID != userID && !s.isAdmin(userID) {
        return nil, ErrUnauthorized
    }
    
    return s.repo.Update(ctx, id, params)
}
```

---

## 🧪 測試策略

### 1. 單元測試

```go
func TestBlogService_CreateBlog(t *testing.T) {
    // 建立 mock repository
    mockRepo := &MockBlogRepository{}
    service := NewBlogService(mockRepo)
    
    // 設定期望行為
    mockRepo.On("Create", mock.Anything, mock.Anything).Return(&Blog{ID: uuid.New()}, nil)
    
    // 執行測試
    blog, err := service.CreateBlog(context.Background(), CreateBlogParams{
        Title: "測試文章",
        Slug:  "test-article",
    })
    
    // 驗證結果
    assert.NoError(t, err)
    assert.NotNil(t, blog)
    mockRepo.AssertExpectations(t)
}
```

### 2. 整合測試

```go
func TestBlogIntegration(t *testing.T) {
    // 建立測試資料庫
    db := setupTestDB(t)
    defer cleanupTestDB(t, db)
    
    service := NewBlogService(db)
    
    // 執行實際的資料庫操作
    blog, err := service.CreateBlog(context.Background(), CreateBlogParams{
        Title: "整合測試",
        Slug:  "integration-test",
    })
    
    assert.NoError(t, err)
    assert.NotNil(t, blog)
}
```

### 3. 效能測試

```go
func BenchmarkBlogService_GetBlogs(b *testing.B) {
    service := setupBenchmarkService(b)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := service.GetBlogs(context.Background(), 10, 0)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

---

## 🚀 部署與維護

### 1. 資料庫遷移

```bash
# 建立新的遷移檔案
migrate create -ext sql -dir migrations -seq add_blog_views

# 應用遷移
migrate -path migrations -database "$DATABASE_URL" up

# 回滾遷移
migrate -path migrations -database "$DATABASE_URL" down 1
```

### 2. 監控與日誌

```go
func (s *BlogService) GetBlogs(ctx context.Context, limit, offset int32) ([]*Blog, error) {
    start := time.Now()
    defer func() {
        metrics.ObserveQueryDuration("get_blogs", time.Since(start))
    }()
    
    blogs, err := s.repo.GetBlogs(ctx, limit, offset)
    if err != nil {
        log.Error().Err(err).Msg("failed to get blogs")
        return nil, err
    }
    
    metrics.IncrementCounter("blogs_retrieved", int64(len(blogs)))
    return blogs, nil
}
```

### 3. 備份策略

```bash
#!/bin/bash
# 資料庫備份腳本
BACKUP_DIR="/backups"
DATE=$(date +%Y%m%d_%H%M%S)
DB_NAME="blog"

pg_dump $DB_NAME | gzip > "$BACKUP_DIR/backup_$DATE.sql.gz"

# 保留最近 30 天的備份
find $BACKUP_DIR -name "backup_*.sql.gz" -mtime +30 -delete
```

---

## 📚 學習資源

### 推薦書籍
- **《PostgreSQL 實戰》** - 深入理解 PostgreSQL 特性
- **《資料庫設計實戰》** - 學習資料庫設計原則
- **《Go 語言實戰》** - 掌握 Go 語言最佳實踐

### 線上資源
- [PostgreSQL 官方文件](https://www.postgresql.org/docs/)
- [sqlc 文件](https://docs.sqlc.dev/)
- [Gin 框架文件](https://gin-gonic.com/docs/)

### 實作練習
1. 為部落格系統添加文章分類功能
2. 實作全文搜尋功能
3. 添加用戶角色和權限管理
4. 實作文章版本控制

---

## 🎉 總結

本開發指南涵蓋了構建個人部落格後端資料庫的完整流程，從設計思路到實作細節，再到部署維護。關鍵要點包括：

1. **設計優先**: 良好的資料庫設計是系統成功的基礎
2. **效能考量**: 索引、查詢優化和快取策略缺一不可
3. **安全性**: 輸入驗證、權限控制和 SQL 注入防護
4. **可維護性**: 清晰的架構、完整的測試和監控

記住，資料庫設計是一個迭代的過程，需要根據實際使用情況不斷優化。保持學習的心態，持續改進你的系統！

---

*最後更新: 2024年*
*作者: 後端工程師團隊*