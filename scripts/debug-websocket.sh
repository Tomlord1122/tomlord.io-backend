#!/bin/bash

# WebSocket Debug Script for Minikube
# This script helps debug WebSocket connections in Minikube environment

echo "🔍 WebSocket Debug Script for Minikube"
echo "======================================"

# Get Minikube IP
MINIKUBE_IP=$(minikube ip)
echo "📍 Minikube IP: $MINIKUBE_IP"

# Get NodePort
NODEPORT=$(kubectl get svc tomlord-io-backend-service -n tomlord-io -o jsonpath='{.spec.ports[0].nodePort}')
echo "🔌 NodePort: $NODEPORT"

# Construct WebSocket URL
WS_URL="ws://$MINIKUBE_IP:$NODEPORT/ws"
echo "🔗 WebSocket URL: $WS_URL"

# Test HTTP connectivity first
HTTP_URL="http://$MINIKUBE_IP:$NODEPORT/health"
echo "🌐 Testing HTTP connectivity..."
curl -v "$HTTP_URL"

echo ""
echo "📋 Debug Information:"
echo "===================="
echo "1. Check if pods are running:"
kubectl get pods -n tomlord-io

echo ""
echo "2. Check service configuration:"
kubectl get svc -n tomlord-io

echo ""
echo "3. Check pod logs for WebSocket errors:"
kubectl logs -n tomlord-io -l app=tomlord-io-backend --tail=50 | grep -i websocket

echo ""
echo "4. Test WebSocket connection with wscat (if installed):"
if command -v wscat &> /dev/null; then
    echo "wscat -c '$WS_URL'"
    echo "Then send: {\"action\":\"subscribe\",\"rooms\":[\"test-room\"]}"
else
    echo "Install wscat: npm install -g wscat"
fi

echo ""
echo "5. Environment variables in pod:"
kubectl exec -n tomlord-io -l app=tomlord-io-backend -- env | grep -E "(APP_ENV|ALLOWED_ORIGINS)"

echo ""
echo "🎯 Next Steps:"
echo "1. Make sure your frontend connects to: $WS_URL"
echo "2. Check browser console for CORS errors"
echo "3. Verify the origin header in WebSocket upgrade request"
echo "4. Check pod logs for detailed error messages"

echo ""
echo "🚀 Production Notes:"
echo "==================="
echo "TODO:[PRODUCTION] For production deployment:"
echo "- Update ALLOWED_ORIGINS in k8s/configmap.yaml"
echo "- Update domains in internal/websocket/hub.go"
echo "- Update domains in internal/server/cors.go"
echo "- Update GOOGLE_CALLBACK_URL in k8s/secret.yaml"
echo "- Use HTTPS for WebSocket connections (wss://)"
echo "- Configure proper SSL/TLS certificates" 