# Production Deployment Checklist

## 🔧 Environment Configuration

### [ ] Environment Variables
- [ ] `APP_ENV=production`
- [ ] `FRONTEND_URL` - Set to your production frontend URL
- [ ] `ALLOWED_ORIGINS` - Set to your production domains (comma-separated)
- [ ] `JWT_SECRET` - Use a strong, unique secret
- [ ] `SESSION_SECRET` - Use a strong, unique secret
- [ ] `GOOGLE_CLIENT_ID` - Production Google OAuth client ID
- [ ] `GOOGLE_CLIENT_SECRET` - Production Google OAuth client secret
- [ ] `GOOGLE_CALLBACK_URL` - Production callback URL

### [ ] Domain Configuration
- [ ] ✅ **COMPLETED** - All hardcoded production values have been replaced with environment variables
- [ ] Set `FRONTEND_URL` environment variable to your production frontend URL
- [ ] Set `ALLOWED_ORIGINS` environment variable to your production domains
- [ ] Update `k8s/configmap.yaml` with production values

### [ ] Google OAuth Setup
- [ ] Create production OAuth client in Google Console
- [ ] Add production callback URL: `https://your-domain.com/auth/google/callback`
- [ ] Update authorized origins in Google Console
- [ ] Update authorized redirect URIs in Google Console

## 🚀 Deployment Configuration

### [ ] Docker Configuration
- [ ] Review `Dockerfile.production` - Update port if needed
- [ ] ✅ **COMPLETED** - `docker-compose.yml` updated with environment variables
- [ ] Test production build locally

### [ ] Kubernetes Configuration
- [ ] Update `k8s/configmap.yaml` with production values
- [ ] Update `k8s/secret.yaml` with production secrets
- [ ] Review resource limits and requests
- [ ] Configure production ingress/load balancer
- [ ] Set up SSL/TLS certificates

### [ ] Database Configuration
- [ ] Production PostgreSQL instance
- [ ] Database migrations
- [ ] Connection pooling settings
- [ ] Backup strategy

## 🔒 Security Configuration

### [ ] Secrets Management
- [ ] Store sensitive data in Kubernetes secrets
- [ ] Use strong, unique secrets for JWT and sessions
- [ ] Rotate secrets regularly
- [ ] Never commit secrets to version control

### [ ] Network Security
- [ ] Configure firewall rules
- [ ] Set up SSL/TLS termination
- [ ] Enable HTTPS only
- [ ] Configure CORS properly

### [ ] Application Security
- [ ] Enable rate limiting
- [ ] Configure proper logging
- [ ] Set up monitoring and alerting
- [ ] Regular security updates

## 📊 Monitoring & Logging

### [ ] Application Monitoring
- [ ] Set up health checks
- [ ] Configure metrics collection
- [ ] Set up alerting
- [ ] Monitor WebSocket connections

### [ ] Logging
- [ ] Configure structured logging
- [ ] Set up log aggregation
- [ ] Configure log retention
- [ ] Monitor error rates

## 🧪 Testing

### [ ] Pre-deployment Testing
- [ ] Test WebSocket connections
- [ ] Test OAuth flow
- [ ] Test database connections
- [ ] Load testing
- [ ] Security testing

### [ ] Post-deployment Testing
- [ ] Verify all endpoints work
- [ ] Test WebSocket real-time features
- [ ] Verify OAuth authentication
- [ ] Check CORS configuration
- [ ] Monitor application performance

## 📝 Documentation

### [ ] Update Documentation
- [ ] Update README with production setup
- [ ] Document deployment process
- [ ] Update troubleshooting guide
- [ ] Document monitoring and alerting

## 🔄 Maintenance

### [ ] Regular Maintenance
- [ ] Schedule regular security updates
- [ ] Monitor resource usage
- [ ] Review and rotate secrets
- [ ] Update dependencies
- [ ] Backup verification

---

## Quick Reference

### Production Environment Variables to Set:
1. `APP_ENV=production`
2. `FRONTEND_URL=https://your-domain.com`
3. `ALLOWED_ORIGINS=https://your-domain.com,https://www.your-domain.com`
4. `GOOGLE_CALLBACK_URL=https://your-domain.com/auth/google/callback`
5. `JWT_SECRET=<strong-secret>`
6. `SESSION_SECRET=<strong-secret>`

### Files Updated:
- ✅ `internal/websocket/hub.go` - Now uses environment variables
- ✅ `internal/server/cors.go` - Now uses environment variables  
- ✅ `internal/server/routes.go` - Now uses environment variables
- ✅ `k8s/configmap.yaml` - Updated with new environment variables
- ✅ `.env.example` - Updated with new environment variables
- ✅ `docker-compose.yml` - Updated with new environment variables 