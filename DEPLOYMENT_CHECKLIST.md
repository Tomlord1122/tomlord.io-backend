# 後端部署檢查清單 (AWS + Kubernetes)

## 🏗️ AWS 基礎設施準備

### 1. AWS 服務設置
- [ ] **EKS Cluster**: 創建 Kubernetes 集群
- [ ] **RDS PostgreSQL**: 設置託管數據庫
- [ ] **ECR**: 創建容器映像倉庫
- [ ] **VPC**: 配置網絡和安全組
- [ ] **IAM**: 設置適當的權限和角色
- [ ] **ALB/NLB**: 配置負載均衡器
- [ ] **Route 53**: DNS 配置 (可選)
- [ ] **Certificate Manager**: SSL 證書 (可選)

### 2. 數據庫設置 (RDS PostgreSQL)
```sql
-- 創建數據庫和用戶
CREATE DATABASE tomlord_production;
CREATE USER tomlord_user WITH PASSWORD 'your-secure-password';
GRANT ALL PRIVILEGES ON DATABASE tomlord_production TO tomlord_user;
```

### 3. ECR 倉庫創建
```bash
# 創建 ECR 倉庫
aws ecr create-repository --repository-name tomlord-io-backend --region us-west-2

# 獲取登入令牌
aws ecr get-login-password --region us-west-2 | docker login --username AWS --password-stdin <account-id>.dkr.ecr.us-west-2.amazonaws.com
```

## 🔧 Kubernetes 配置

### 1. 環境變數和機密
- [ ] 更新 `k8s/secret.yaml` 中的 base64 編碼值:
```bash
# 生成 base64 編碼
echo -n "your-db-host.rds.amazonaws.com" | base64
echo -n "production_database" | base64
echo -n "secure-jwt-secret-256-bits" | base64
echo -n "secure-session-secret" | base64
echo -n "your-google-client-id" | base64
echo -n "your-google-client-secret" | base64
echo -n "https://your-backend-domain.com/auth/google/callback" | base64
```

### 2. 網絡配置
- [ ] 確保 EKS 節點可以訪問 RDS
- [ ] 配置安全組規則
- [ ] 設置適當的子網

### 3. 存儲配置
- [ ] 為持久化數據配置 EBS 卷 (如果需要)
- [ ] 配置備份策略

## 🚀 部署流程

### 1. 構建和推送映像
```bash
# 構建生產映像
docker build -f Dockerfile.production -t tomlord-io-backend:latest .

# 標記並推送到 ECR
docker tag tomlord-io-backend:latest <account-id>.dkr.ecr.us-west-2.amazonaws.com/tomlord-io-backend:latest
docker push <account-id>.dkr.ecr.us-west-2.amazonaws.com/tomlord-io-backend:latest
```

### 2. 部署到 Kubernetes
```bash
# 使用部署腳本
chmod +x scripts/deploy.sh
./scripts/deploy.sh

# 或手動部署
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/secret.yaml
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
```

### 3. 數據庫遷移
```bash
# 運行遷移 Job
kubectl apply -f k8s/migration-job.yaml

# 檢查遷移狀態
kubectl logs job/tomlord-io-db-migration -n tomlord-io
```

## ✅ 部署後驗證

### 1. 服務健康檢查
- [ ] Pod 狀態: `kubectl get pods -n tomlord-io`
- [ ] 服務狀態: `kubectl get services -n tomlord-io`
- [ ] 健康端點: `curl https://your-backend-url/health`

### 2. 功能測試
- [ ] API 端點響應正常
- [ ] 數據庫連接正常
- [ ] OAuth 認證流程
- [ ] WebSocket 連接
- [ ] CORS 配置正確

### 3. 監控設置
- [ ] 設置 CloudWatch 日誌
- [ ] 配置性能監控
- [ ] 設置告警規則
- [ ] 檢查資源使用情況

## 🔐 安全配置

### 1. 網絡安全
- [ ] 配置適當的安全組
- [ ] 啟用 VPC Flow Logs
- [ ] 限制數據庫訪問

### 2. 應用安全
- [ ] 使用強密碼和機密
- [ ] 啟用 HTTPS/TLS
- [ ] 配置適當的 CORS
- [ ] 設置 Session 安全選項

### 3. 運營安全
- [ ] 定期備份數據庫
- [ ] 更新系統補丁
- [ ] 監控安全事件
- [ ] 實施最小權限原則

## 🔄 CI/CD 設置

### 1. GitHub Secrets 配置
在 GitHub Repository Settings > Secrets 中添加:
- [ ] `AWS_ACCESS_KEY_ID`
- [ ] `AWS_SECRET_ACCESS_KEY`
- [ ] `AWS_REGION`
- [ ] `EKS_CLUSTER_NAME`
- [ ] `ECR_REPOSITORY`

### 2. 自動化部署
- [ ] 推送到 main branch 觸發部署
- [ ] 測試通過才能部署
- [ ] 自動回滾機制

## 🚨 常見問題排解

### Pod 無法啟動
```bash
# 檢查 Pod 狀態
kubectl describe pod <pod-name> -n tomlord-io

# 檢查日誌
kubectl logs <pod-name> -n tomlord-io
```

### 數據庫連接失敗
- 檢查 RDS 安全組配置
- 確認數據庫憑據正確
- 檢查 VPC 和子網配置

### 負載均衡器問題
```bash
# 檢查服務狀態
kubectl get services -n tomlord-io

# 檢查 AWS Load Balancer Controller
kubectl logs -n kube-system deployment/aws-load-balancer-controller
```

### SSL/TLS 問題
- 檢查證書配置
- 確認域名解析正確
- 檢查安全組 443 端口

## 📊 監控和維護

### 1. 性能監控
- [ ] CPU 和記憶體使用率
- [ ] 網絡流量
- [ ] 數據庫性能
- [ ] 響應時間

### 2. 日誌管理
- [ ] 應用日誌收集
- [ ] 錯誤追蹤
- [ ] 審計日誌

### 3. 備份策略
- [ ] 數據庫自動備份
- [ ] 配置文件備份
- [ ] 災難恢復計劃 