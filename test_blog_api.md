# Blog API 測試指南

你的後端已經成功更新，現在包含完整的 Blog API 功能！🎉

## 📋 更新摘要

### 1. 數據庫 Schema 更新
- ✅ 新增 `blogs` 表存儲文章元數據
- ✅ 在 `messages` 表中添加 `blog_id` 外鍵
- ✅ 保持向後兼容性（仍支持 `post_slug`）

### 2. 新增 API 端點

#### Blog 管理 API
```
GET    /api/blogs                    # 獲取文章列表
GET    /api/blogs/:slug              # 獲取特定文章（含評論數）
POST   /api/blogs                    # 創建新文章（需認證）
PUT    /api/blogs/:slug              # 更新文章（需認證）  
DELETE /api/blogs/:slug              # 刪除文章（需認證）
GET    /api/blogs/:slug/messages     # 獲取文章的評論
```

#### 增強的 Message API
```
GET    /api/messages/blog/:slug      # 通過 blog slug 獲取評論
POST   /api/messages                 # 創建評論（現在支持 blog_id）
```

## 🧪 API 測試

### 1. 創建測試文章

**請求:**
```bash
curl -X POST http://localhost:8080/api/blogs \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "title": "百日計劃",
    "slug": "one-hundred-plan",
    "date": "2025-06-24",
    "lang": "zh-tw",
    "duration": "2min",
    "tags": ["Career"],
    "description": "我的百日學習計劃分享",
    "is_published": true
  }'
```

**回應:**
```json
{
  "blog": {
    "id": "uuid-string",
    "title": "百日計劃",
    "slug": "one-hundred-plan",
    "date": "2025-06-24",
    "lang": "zh-tw",
    "duration": "2min",
    "tags": ["Career"],
    "description": "我的百日學習計劃分享",
    "is_published": true,
    "created_at": "2025-01-16T...",
    "updated_at": "2025-01-16T..."
  }
}
```

### 2. 獲取文章列表

```bash
# 獲取所有已發布的文章
curl http://localhost:8080/api/blogs

# 按標籤篩選
curl "http://localhost:8080/api/blogs?tag=Career&limit=10"

# 按語言篩選
curl "http://localhost:8080/api/blogs?lang=zh-tw"
```

### 3. 獲取特定文章（含評論數）

```bash
curl http://localhost:8080/api/blogs/one-hundred-plan
```

**回應:**
```json
{
  "blog": {
    "id": "uuid-string",
    "title": "百日計劃",
    "slug": "one-hundred-plan",
    "date": "2025-06-24",
    "lang": "zh-tw",
    "duration": "2min",
    "tags": ["Career"],
    "description": "我的百日學習計劃分享",
    "is_published": true,
    "created_at": "2025-01-16T...",
    "updated_at": "2025-01-16T...",
    "message_count": 5
  }
}
```

### 4. 為文章創建評論

```bash
curl -X POST http://localhost:8080/api/messages \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "post_slug": "one-hundred-plan",
    "blog_id": "blog-uuid-from-step-1",
    "message": "很棒的計劃！我也想參與！"
  }'
```

### 5. 獲取文章的評論

```bash
# 通過 blog slug 獲取（推薦）
curl http://localhost:8080/api/blogs/one-hundred-plan/messages

# 或者使用新的 messages 端點
curl http://localhost:8080/api/messages/blog/one-hundred-plan
```

## 🔄 前端集成指南

### 1. 從 .svx 文件中提取元數據

你的前端可以這樣處理：

```javascript
// 解析 .svx 文件的 frontmatter
const frontmatterRegex = /^---\s*\n([\s\S]*?)\n---/;
const match = svxContent.match(frontmatterRegex);

if (match) {
  const frontmatter = yaml.parse(match[1]);
  
  // 創建或更新 blog entry
  const blogData = {
    title: frontmatter.title,
    slug: frontmatter.slug,
    date: frontmatter.date,
    lang: frontmatter.lang || 'zh-tw',
    duration: frontmatter.duration || '5min',
    tags: frontmatter.tags || [],
    description: frontmatter.description,
    is_published: true
  };
  
  // 發送到後端
  await fetch('/api/blogs', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(blogData)
  });
}
```

### 2. 渲染文章頁面時獲取評論

```javascript
// 在文章頁面組件中
async function loadBlogAndComments(slug) {
  // 獲取文章信息（含評論數）
  const blogResponse = await fetch(`/api/blogs/${slug}`);
  const blogData = await blogResponse.json();
  
  // 獲取評論
  const messagesResponse = await fetch(`/api/blogs/${slug}/messages`);
  const messagesData = await messagesResponse.json();
  
  return {
    blog: blogData.blog,
    messages: messagesData.messages
  };
}
```

### 3. 創建評論時使用 blog_id

```javascript
async function createComment(blogId, postSlug, message) {
  return await fetch('/api/messages', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${authToken}`
    },
    body: JSON.stringify({
      blog_id: blogId,      // 新字段：關聯到具體的 blog
      post_slug: postSlug,  // 保持向後兼容
      message: message
    })
  });
}
```

## 📊 數據庫關係

```
blogs (1) ←→ (N) messages
  ↓              ↓
  id ←─────── blog_id
  slug ────── post_slug (backup reference)
```

- **主要關係**: `messages.blog_id → blogs.id`
- **備用關係**: `messages.post_slug` （向後兼容）

## 🎯 建議的工作流程

1. **文章發布時**:
   - 前端解析 `.svx` frontmatter
   - 調用 `POST /api/blogs` 創建文章記錄
   - 存儲返回的 `blog_id` 用於評論關聯

2. **文章頁面渲染**:
   - 調用 `GET /api/blogs/:slug` 獲取文章信息
   - 調用 `GET /api/blogs/:slug/messages` 獲取評論
   - 同時顯示文章元數據和評論

3. **評論管理**:
   - 創建評論時使用 `blog_id` 和 `post_slug`
   - 查詢評論時優先使用 `blog_slug`

## 🚀 下一步

你的後端現在已經完全支援：
- ✅ 文章元數據管理
- ✅ 文章與評論的正確關聯
- ✅ 向後兼容性
- ✅ 完整的 CRUD 操作
- ✅ 標籤和語言篩選
- ✅ 評論計數

建議接下來：
1. 更新前端以使用新的 Blog API
2. 實現文章和評論的自動關聯
3. 考慮添加管理員權限控制
4. 添加文章搜索功能

恭喜！你的 database schema 和 API 架構現在更加強大和靈活了！🎉 