# WebSocket 連接穩定性調試指南

## 🔧 **修復摘要**

我已經改進了 WebSocket 系統來解決連接不穩定的問題：

### **後端改進 (hub.go)**
1. **心跳機制**: 添加了 ping/pong 心跳檢測
2. **連接超時**: 設置了讀寫超時和最大消息大小限制
3. **錯誤處理**: 改進了錯誤處理和日誌記錄
4. **清理機制**: 定期清理失效連接
5. **線程安全**: 添加了更好的互斥鎖保護

### **前端改進 (websocket.svelte.ts)**  
1. **連接狀態管理**: 完整的連接狀態機
2. **重複初始化防護**: 防止多個連接實例
3. **指數退避重連**: 智能重連策略
4. **連接健康監控**: 定期檢查連接狀態
5. **優雅清理**: 更好的資源清理機制

## 🧪 **測試步驟**

### 1. 重啟後端服務
```bash
# 在 tomlord.io-backend/ 目錄下
make docker-down
make setup
```

### 2. 檢查後端日誌
```bash
# 啟動後端並觀察 WebSocket 日誌
make run

# 你應該看到類似的日誌：
# [GIN] GET /ws --> github.com/your-org/tomlord.io-backend/internal/server.(*Server).websocketHandler-fm
# Client user-id connected to rooms: map[room-name:true]
# Broadcasting new_comment message to room 'room-name'
```

### 3. 測試多用戶連接

#### 用戶 A (瀏覽器 1)
1. 打開 `http://localhost:5173/blog/your-post-slug`
2. 打開開發者工具 Console
3. 檢查 WebSocket 連接日誌：
```javascript
// 你應該看到：
// WebSocket manager already initialized (如果第二次調用)
// Connecting to WebSocket...  
// WebSocket connected successfully
// Subscribed to rooms: ["your-post-slug"]
```

#### 用戶 B (隱身模式/其他瀏覽器)
1. 打開相同的文章頁面
2. 檢查 Console 是否有相同的連接成功信息

### 4. 測試實時同步

#### A. 測試評論同步
1. **用戶 A**: 發表一條評論
2. **用戶 B**: 應該立即看到新評論出現（無需刷新）
3. **後端日誌**: 應該看到廣播消息

#### B. 測試點讚同步  
1. **用戶 A**: 對評論點讚
2. **用戶 B**: 應該看到點讚數更新
3. **用戶 A**: 再次點擊取消點讚
4. **用戶 B**: 應該看到點讚數減少

### 5. 測試連接穩定性

#### A. 網絡中斷模拟
```javascript
// 在瀏覽器 Console 中運行
// 模擬斷網
wsManager.disconnect();

// 等待 3 秒後重連
setTimeout(() => {
    wsManager.connect();
}, 3000);
```

#### B. 長時間連接測試
- 保持頁面打開 10+ 分鐘
- 檢查連接狀態是否仍然是 "Live updates active"
- 測試心跳是否正常工作

## 🔍 **調試命令**

### 檢查 WebSocket 連接狀態
```javascript
// 在瀏覽器 Console 中
console.log('Connection state:', wsManager.state);
console.log('Is connected:', wsManager.isConnected);
console.log('Subscribed rooms:', wsManager.rooms);
```

### 手動重置 WebSocket
```javascript
// 如果連接有問題，可以手動重置
wsManager.reset();
wsManager.init();
```

### 監控連接事件
```javascript
// 添加調試監聽器
wsManager.addEventListener('new_comment', (payload) => {
    console.log('🆕 New comment event:', payload);
});

wsManager.addEventListener('thumb_update', (payload) => {
    console.log('👍 Thumb update event:', payload);
});
```

## 📊 **連接狀態說明**

| 狀態 | 顏色 | 說明 |
|------|------|------|
| connected | 🟢 綠色 | 連接正常，實時更新可用 |
| connecting | 🟡 黃色 | 正在連接中 |
| reconnecting | 🟡 黃色 | 重新連接中 |
| failed | 🔴 紅色 | 連接失敗，已停止重試 |
| disconnected | ⚫ 灰色 | 未連接 |

## 🚨 **故障排除**

### 問題：只有一個用戶能收到實時更新

**可能原因：**
1. 房間訂閱失敗
2. 認證狀態不一致
3. 瀏覽器緩存問題

**解決方案：**
```bash
# 1. 清除瀏覽器緩存和 localStorage
localStorage.clear();
location.reload();

# 2. 檢查後端日誌中的房間訂閱
# 應該看到兩個不同的 Client connected 消息

# 3. 重啟後端服務
make docker-down && make setup
```

### 問題：連接頻繁斷開

**檢查項目：**
1. 網絡環境是否穩定
2. 防火牆/代理設置
3. 瀏覽器是否支持 WebSocket

**調試：**
```javascript
// 檢查瀏覽器支持
if (!window.WebSocket) {
    console.error('WebSocket not supported');
}

// 監控連接事件
wsManager.ws?.addEventListener('close', (event) => {
    console.log('Connection closed:', event.code, event.reason);
});
```

### 問題：心跳失敗

**後端日誌檢查：**
```
# 正常心跳日誌
Error sending ping to client user-id: <error>
Removing stale client: user-id
```

**前端調試：**
```javascript
// 檢查心跳響應
wsManager.addEventListener('ping', () => {
    console.log('Received ping from server');
});
```

## ✅ **成功指標**

連接穩定性修復成功的指標：

1. **兩個用戶都能穩定連接** ✅
2. **實時評論同步正常** ✅  
3. **實時點讚同步正常** ✅
4. **連接斷開後能自動重連** ✅
5. **長時間保持連接穩定** ✅
6. **心跳機制正常工作** ✅

## 📝 **性能監控**

```javascript
// 監控連接性能
const stats = {
    connectTime: 0,
    reconnectCount: 0,
    messageCount: 0
};

// 連接時間監控
const startTime = Date.now();
wsManager.addEventListener('connected', () => {
    stats.connectTime = Date.now() - startTime;
    console.log(`連接建立時間: ${stats.connectTime}ms`);
});

// 消息計數
wsManager.addEventListener('new_comment', () => stats.messageCount++);
wsManager.addEventListener('thumb_update', () => stats.messageCount++);

console.log('WebSocket 統計:', stats);
```

這些改進應該能夠解決你遇到的 WebSocket 連接不穩定問題。現在兩個用戶都應該能夠穩定連接並正常接收實時更新。 