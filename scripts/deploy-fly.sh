#!/bin/bash

# Fly.io 自動化部署腳本
# 使用方法: ./scripts/deploy-fly.sh

set -e  # 遇到錯誤時退出

# 顏色定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印帶顏色的消息
print_message() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# 檢查必要工具
check_requirements() {
    print_step "檢查必要工具..."
    
    if ! command -v flyctl &> /dev/null; then
        print_error "Fly CLI 未安裝。請先安裝: https://fly.io/docs/hands-on/install-flyctl/"
        exit 1
    fi
    
    if ! command -v docker &> /dev/null; then
        print_error "Docker 未安裝。請先安裝 Docker。"
        exit 1
    fi
    
    print_message "所有必要工具已安裝 ✓"
}

# 檢查登錄狀態
check_auth() {
    print_step "檢查 Fly.io 登錄狀態..."
    
    if ! flyctl auth whoami &> /dev/null; then
        print_warning "未登錄 Fly.io，正在登錄..."
        flyctl auth login
    else
        print_message "已登錄 Fly.io ✓"
    fi
}

# 檢查應用程序是否存在
check_app_exists() {
    if flyctl apps list | grep -q "tomlord-io-backend"; then
        return 0
    else
        return 1
    fi
}

# 創建應用程序
create_app() {
    print_step "創建 Fly.io 應用程序..."
    
    if check_app_exists; then
        print_message "應用程序已存在，跳過創建"
        return
    fi
    
    flyctl launch \
        --name tomlord-io-backend \
        --region hkg \
        --dockerfile Dockerfile.production \
        --no-deploy \
        --yes
    
    print_message "應用程序創建成功 ✓"
}

# 設置環境變數
setup_secrets() {
    print_step "設置環境變數..."
    
    # 檢查是否已經設置了必要的環境變數
    if flyctl secrets list | grep -q "JWT_SECRET"; then
        print_message "環境變數已設置，跳過..."
        return
    fi
    
    print_warning "請手動設置以下環境變數："
    echo ""
    echo "1. 設置 Google OAuth 配置："
    echo "   flyctl secrets set GOOGLE_CLIENT_ID=\"your-google-client-id\""
    echo "   flyctl secrets set GOOGLE_CLIENT_SECRET=\"your-google-client-secret\""
    echo "   flyctl secrets set GOOGLE_CALLBACK_URL=\"https://tomlord-io-backend.fly.dev/auth/google/callback\""
    echo ""
    echo "2. 設置 JWT 和 Session 密鑰："
    echo "   flyctl secrets set JWT_SECRET=\"\$(openssl rand -base64 32)\""
    echo "   flyctl secrets set SESSION_SECRET=\"\$(openssl rand -base64 32)\""
    echo ""
    echo "3. 設置前端 URL："
    echo "   flyctl secrets set FRONTEND_URL=\"https://tomlord.fyi\""
    echo "   flyctl secrets set ALLOWED_ORIGINS=\"https://tomlord.fyi,https://www.tomlord.fyi\""
    echo ""
    echo "4. 如果使用外部數據庫，設置數據庫配置："
    echo "   flyctl secrets set BLUEPRINT_DB_HOST=\"your-db-host\""
    echo "   flyctl secrets set BLUEPRINT_DB_PORT=\"5432\""
    echo "   flyctl secrets set BLUEPRINT_DB_DATABASE=\"tomlord_production\""
    echo "   flyctl secrets set BLUEPRINT_DB_USERNAME=\"your-username\""
    echo "   flyctl secrets set BLUEPRINT_DB_PASSWORD=\"your-password\""
    echo "   flyctl secrets set BLUEPRINT_DB_SCHEMA=\"public\""
    echo ""
    
    read -p "設置完成後按 Enter 繼續..."
}

# 創建數據庫（可選）
setup_database() {
    print_step "設置數據庫..."
    
    read -p "是否使用 Fly.io PostgreSQL 數據庫？(y/n): " -n 1 -r
    echo
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        print_message "創建 Fly.io PostgreSQL 數據庫..."
        
        # 檢查數據庫是否已存在
        if ! flyctl postgres list | grep -q "tomlord-db"; then
            flyctl postgres create --name tomlord-db --region hkg
            print_message "PostgreSQL 數據庫創建成功 ✓"
        else
            print_message "PostgreSQL 數據庫已存在 ✓"
        fi
        
        # 附加到應用程序
        if ! flyctl postgres list | grep -q "tomlord-io-backend"; then
            flyctl postgres attach --postgres-app tomlord-db --app tomlord-io-backend
            print_message "數據庫附加成功 ✓"
        else
            print_message "數據庫已附加 ✓"
        fi
    else
        print_message "跳過數據庫創建，請確保已設置外部數據庫環境變數"
    fi
}

# TODO:[PRODUCTION] Verify database migration setup
verify_migration_setup() {
    print_step "驗證數據庫遷移設置..."
    
    # 檢查 migrations 目錄是否存在
    if [ ! -d "migrations" ]; then
        print_error "migrations 目錄不存在！"
        exit 1
    fi
    
    # 檢查是否有遷移文件
    if [ -z "$(ls -A migrations/*.up.sql 2>/dev/null)" ]; then
        print_error "沒有找到遷移文件！"
        exit 1
    fi
    
    print_message "遷移設置驗證通過 ✓"
    print_message "找到以下遷移文件："
    ls -la migrations/*.up.sql | while read line; do
        echo "   $line"
    done
}

# 部署應用程序
deploy_app() {
    print_step "部署應用程序..."
    
    print_message "開始構建和部署（包含數據庫遷移）..."
    flyctl deploy
    
    print_message "部署完成 ✓"
}

# TODO:[PRODUCTION] Check migration status after deployment
check_migration_status() {
    print_step "檢查數據庫遷移狀態..."
    
    print_message "等待應用程序完全啟動..."
    sleep 15
    
    # 檢查遷移是否成功執行
    print_message "檢查數據庫連接和表結構..."
    
    # 可以通過 flyctl ssh 連接檢查表是否存在
    print_message "可以通過以下命令檢查數據庫表："
    echo "   flyctl ssh console"
    echo "   然後在容器內執行:"
    echo "   migrate -path /migrations -database \$DATABASE_URL version"
}

# 檢查部署狀態
check_deployment() {
    print_step "檢查部署狀態..."
    
    print_message "等待應用程序啟動..."
    sleep 10
    
    # 檢查應用程序狀態
    if flyctl status | grep -q "running"; then
        print_message "應用程序運行正常 ✓"
    else
        print_error "應用程序未正常運行"
        flyctl status
        exit 1
    fi
    
    # 檢查健康檢查
    print_message "檢查健康檢查..."
    if flyctl health check | grep -q "passing"; then
        print_message "健康檢查通過 ✓"
    else
        print_warning "健康檢查失敗，請檢查日誌"
        flyctl logs --since 5m
    fi
}

# 顯示部署信息
show_deployment_info() {
    print_step "部署信息"
    
    echo ""
    echo "🎉 部署完成！"
    echo ""
    echo "📱 應用程序信息："
    flyctl info
    echo ""
    echo "🌐 訪問地址："
    echo "   API 端點: https://tomlord-io-backend.fly.dev"
    echo "   健康檢查: https://tomlord-io-backend.fly.dev/health"
    echo "   OAuth 登錄: https://tomlord-io-backend.fly.dev/auth/google"
    echo ""
    echo "📊 監控命令："
    echo "   查看狀態: flyctl status"
    echo "   查看日誌: flyctl logs"
    echo "   查看儀表板: flyctl dashboard"
    echo ""
    echo "🔄 更新部署："
    echo "   flyctl deploy"
    echo ""
    echo "🗄️ 數據庫遷移："
    echo "   遷移會在每次部署時自動執行"
    echo "   手動檢查遷移狀態: flyctl ssh console"
    echo "   在容器內執行: migrate -path /migrations -database \$DATABASE_URL version"
    echo ""
}

# 主函數
main() {
    echo "🚀 Fly.io 自動化部署腳本（含數據庫遷移）"
    echo "============================================"
    echo ""
    
    check_requirements
    check_auth
    verify_migration_setup
    create_app
    setup_secrets
    setup_database
    deploy_app
    check_migration_status
    check_deployment
    show_deployment_info
    
    print_message "部署腳本執行完成！"
}

# 執行主函數
main "$@" 