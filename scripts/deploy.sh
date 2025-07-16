#!/bin/bash

# Deploy script for tomlord.io backend to AWS EKS
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
CLUSTER_NAME="tomlord-io-cluster"
REGION="us-west-2"
NAMESPACE="tomlord-io"
ECR_REPO="tomlord-io-backend"

echo -e "${GREEN}🚀 Starting deployment to AWS EKS...${NC}"

# Check prerequisites
check_prerequisites() {
    echo -e "${YELLOW}📋 Checking prerequisites...${NC}"
    
    if ! command -v kubectl &> /dev/null; then
        echo -e "${RED}❌ kubectl not found. Please install kubectl.${NC}"
        exit 1
    fi
    
    if ! command -v aws &> /dev/null; then
        echo -e "${RED}❌ AWS CLI not found. Please install AWS CLI.${NC}"
        exit 1
    fi
    
    if ! command -v docker &> /dev/null; then
        echo -e "${RED}❌ Docker not found. Please install Docker.${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}✅ Prerequisites check passed${NC}"
}

# Update kubeconfig
update_kubeconfig() {
    echo -e "${YELLOW}🔧 Updating kubeconfig...${NC}"
    aws eks update-kubeconfig --region $REGION --name $CLUSTER_NAME
    echo -e "${GREEN}✅ Kubeconfig updated${NC}"
}

# Create namespace if it doesn't exist
create_namespace() {
    echo -e "${YELLOW}📁 Creating namespace if needed...${NC}"
    kubectl create namespace $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -
    echo -e "${GREEN}✅ Namespace ready${NC}"
}

# Apply Kubernetes manifests
apply_manifests() {
    echo -e "${YELLOW}📦 Applying Kubernetes manifests...${NC}"
    
    # Apply in order
    kubectl apply -f k8s/configmap.yaml
    kubectl apply -f k8s/secret.yaml
    kubectl apply -f k8s/deployment.yaml
    kubectl apply -f k8s/service.yaml
    
    echo -e "${GREEN}✅ Manifests applied${NC}"
}

# Run database migrations
run_migrations() {
    echo -e "${YELLOW}🗄️ Running database migrations...${NC}"
    kubectl apply -f k8s/migration-job.yaml
    
    # Wait for migration job to complete
    echo "Waiting for migration job to complete..."
    kubectl wait --for=condition=complete --timeout=300s job/tomlord-io-db-migration -n $NAMESPACE
    
    echo -e "${GREEN}✅ Database migrations completed${NC}"
}

# Wait for deployment
wait_for_deployment() {
    echo -e "${YELLOW}⏳ Waiting for deployment to be ready...${NC}"
    kubectl rollout status deployment/tomlord-io-backend -n $NAMESPACE --timeout=300s
    echo -e "${GREEN}✅ Deployment is ready${NC}"
}

# Verify deployment
verify_deployment() {
    echo -e "${YELLOW}🔍 Verifying deployment...${NC}"
    
    # Check pods
    echo "Pods status:"
    kubectl get pods -n $NAMESPACE -l app=tomlord-io-backend
    
    # Check services
    echo "Services status:"
    kubectl get services -n $NAMESPACE
    
    # Test health endpoint
    echo "Testing health endpoint..."
    SERVICE_URL=$(kubectl get service tomlord-io-backend-lb -n $NAMESPACE -o jsonpath='{.status.loadBalancer.ingress[0].hostname}')
    if [ ! -z "$SERVICE_URL" ]; then
        curl -f "http://$SERVICE_URL/health" || echo "Health check failed"
    fi
    
    echo -e "${GREEN}✅ Deployment verified${NC}"
}

# Main deployment flow
main() {
    echo -e "${GREEN}🎯 Deploying tomlord.io backend to EKS${NC}"
    
    check_prerequisites
    update_kubeconfig
    create_namespace
    apply_manifests
    run_migrations
    wait_for_deployment
    verify_deployment
    
    echo -e "${GREEN}🎉 Deployment completed successfully!${NC}"
    echo -e "${YELLOW}📝 Next steps:${NC}"
    echo "1. Update your frontend environment variables with the new backend URL"
    echo "2. Update your OAuth callback URLs"
    echo "3. Monitor the deployment with: kubectl logs -f deployment/tomlord-io-backend -n $NAMESPACE"
}

# Run main function
main "$@" 