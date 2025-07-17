#!/bin/bash

echo "🚀 開始 Minikube 部署..."

# 清理舊的資源（如果存在）
echo "🧹 清理舊的資源..."
kubectl delete namespace tomlord-io --ignore-not-found=true
sleep 5

# 構建 Docker 映像
echo "📦 構建 Docker 映像..."
docker build -f Dockerfile.production -t tomlord-io-backend:latest .

# 部署到 Kubernetes
echo "⚙️ 部署到 Kubernetes..."
kubectl apply -f k8s/minikube/namespace.yaml
kubectl apply -f k8s/minikube/configmap.yaml
kubectl apply -f k8s/minikube/secret.yaml
kubectl apply -f k8s/minikube/postgres.yaml

# 等待 PostgreSQL 就緒
echo "⏳ 等待 PostgreSQL 就緒..."
kubectl wait --for=condition=ready pod -l app=postgres -n tomlord-io --timeout=120s

# 等待 PostgreSQL 完全啟動
echo "⏳ 等待 PostgreSQL 完全啟動..."
sleep 15

# 創建 migrations ConfigMap
echo "🗄️ 創建 migrations ConfigMap..."
kubectl create configmap tomlord-io-migrations --from-file=migrations/ -n tomlord-io --dry-run=client -o yaml | kubectl apply -f -

# 運行數據庫遷移
echo "🔄 運行數據庫遷移..."
kubectl apply -f k8s/minikube/migration-job.yaml

# 檢查 migration job 狀態
echo "📊 檢查 migration job 狀態..."
kubectl wait --for=condition=complete job/tomlord-io-db-migration -n tomlord-io --timeout=300s

# 顯示 migration 日誌
echo "📝 Migration 日誌:"
kubectl logs job/tomlord-io-db-migration -n tomlord-io

# 檢查 migration 是否成功
MIGRATION_STATUS=$(kubectl get job tomlord-io-db-migration -n tomlord-io -o jsonpath='{.status.succeeded}')
if [ "$MIGRATION_STATUS" != "1" ]; then
    echo "❌ Migration 失敗！"
    kubectl logs job/tomlord-io-db-migration -n tomlord-io
    exit 1
fi

echo "✅ Migration 成功！"

# 部署應用
echo "🚀 部署應用..."
kubectl apply -f k8s/minikube/deployment.yaml
kubectl apply -f k8s/minikube/service.yaml

# 等待應用就緒
echo "⏳ 等待應用就緒..."
kubectl wait --for=condition=ready pod -l app=tomlord-io-backend -n tomlord-io --timeout=120s

# 獲取服務 URL
echo "🌐 獲取服務 URL..."
NODE_PORT=$(kubectl get service tomlord-io-backend-service -n tomlord-io -o jsonpath='{.spec.ports[0].nodePort}')
MINIKUBE_IP=$(minikube ip)

echo "✅ 部署完成！"
echo "📱 服務地址: http://$MINIKUBE_IP:$NODE_PORT"
echo "🔍 健康檢查: http://$MINIKUBE_IP:$NODE_PORT/health"
echo "🔌 WebSocket: ws://$MINIKUBE_IP:$NODE_PORT/ws"

# 顯示 Pod 狀態
echo "📊 Pod 狀態:"
kubectl get pods -n tomlord-io

# 顯示服務狀態
echo "🌐 服務狀態:"
kubectl get services -n tomlord-io

# 測試健康檢查
echo "🔍 測試健康檢查..."
sleep 5
curl -f http://$MINIKUBE_IP:$NODE_PORT/health || echo "❌ 健康檢查失敗"