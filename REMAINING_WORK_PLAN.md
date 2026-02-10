# Remaining Work Plan - K8s Cluster Support

## Overview
This document outlines the remaining work items that should be completed in future PRs to make the K8s cluster deployment feature production-ready.

## Priority 1: Critical for Production

### 1. Comprehensive Test Suite
**Status:** Not Started  
**Estimated Effort:** 3-5 days  
**Priority:** HIGH

#### Unit Tests Required:

**File: `api/internal/pkg/pac-go-server/service/k8s/cluster_test.go`**
```go
// Test cases needed:
- TestGenerateMasterScript_Calico
- TestGenerateMasterScript_Flannel
- TestGenerateMasterScript_Weave
- TestGenerateWorkerScript
- TestValidateKubernetesVersion
- TestValidateCNIPlugin
- TestScriptContainsRequiredCommands
```

**File: `api/internal/pkg/pac-go-server/service/k8s/k8s_test.go`**
```go
// Test cases needed:
- TestDeploy_SingleMaster
- TestDeploy_HACluster
- TestDeploy_InvalidConfig
- TestGetStatus_ClusterReady
- TestGetStatus_ClusterPending
- TestGetStatus_ClusterFailed
- TestDelete_Success
- TestDelete_AlreadyDeleted
```

**File: `api/internal/pkg/pac-go-server/services/k8s_service_test.go`**
```go
// Test cases needed:
- TestGetServiceKubeconfig_Success
- TestGetServiceKubeconfig_Unauthorized
- TestGetServiceKubeconfig_NotK8sService
- TestGetServiceKubeconfig_ClusterNotReady
- TestGetServiceKubeconfig_ExpiredService
- TestGenerateKubeconfig_ValidFormat
- TestAuditLogging
```

**File: `api/controllers/app/service/k8s_test.go`**
```go
// Test cases needed:
- TestReconcile_CreateService
- TestReconcile_UpdateStatus
- TestReconcile_DeleteService
- TestReconcile_ErrorHandling
```

#### Integration Tests Required:

**File: `api/test/integration/k8s_service_test.go`**
```go
// Test scenarios:
- Full lifecycle: Create → Deploy → Ready → Delete
- Concurrent service creation
- Service expiration and cleanup
- Kubeconfig download flow
- Error recovery scenarios
```

#### E2E Tests Required:

**File: `api/test/e2e/k8s_deployment_test.go`**
```go
// Test scenarios:
- Create K8s catalog via API
- Deploy K8s service
- Wait for cluster ready (with timeout)
- Download kubeconfig
- Verify kubectl access to cluster
- Deploy sample workload
- Delete service
- Verify complete cleanup
```

**Acceptance Criteria:**
- [ ] 80%+ code coverage for service layer
- [ ] All critical paths tested
- [ ] Mock external dependencies (IBM Cloud API)
- [ ] Tests run in CI/CD pipeline
- [ ] Tests complete in < 5 minutes

---

### 2. Auto-Cleanup Safety Mechanisms
**Status:** Design Complete, Implementation Needed  
**Estimated Effort:** 2-3 days  
**Priority:** HIGH

#### Implementation Plan:

**File: `api/apis/app/v1alpha1/service_types.go`**
```go
// Add to ServiceSpec:
type ServiceSpec struct {
    // ... existing fields ...
    
    // Cleanup configuration
    CleanupConfig *CleanupConfig `json:"cleanupConfig,omitempty"`
}

type CleanupConfig struct {
    // Grace period in hours before cleanup (default: 24)
    GracePeriodHours int `json:"gracePeriodHours,omitempty"`
    
    // Require explicit confirmation before cleanup
    RequireConfirmation bool `json:"requireConfirmation,omitempty"`
    
    // Create backup before cleanup
    CreateBackup bool `json:"createBackup,omitempty"`
    
    // Send notification before cleanup
    NotifyBeforeCleanup bool `json:"notifyBeforeCleanup,omitempty"`
    
    // Allow manual override to prevent auto-cleanup
    PreventAutoCleanup bool `json:"preventAutoCleanup,omitempty"`
}

// Add to ServiceStatus:
type ServiceStatus struct {
    // ... existing fields ...
    
    // Cleanup status
    CleanupScheduledAt *metav1.Time `json:"cleanupScheduledAt,omitempty"`
    CleanupWarningsSent int `json:"cleanupWarningsSent,omitempty"`
    BackupCreated bool `json:"backupCreated,omitempty"`
}
```

**File: `api/controllers/app/service/cleanup.go`** (NEW)
```go
package service

import (
    "context"
    "time"
)

type CleanupManager struct {
    // ... fields ...
}

// ScheduleCleanup schedules a service for cleanup after grace period
func (m *CleanupManager) ScheduleCleanup(ctx context.Context, service *Service) error {
    // Set cleanup scheduled time
    // Send initial warning notification
    // Update service status
}

// CheckCleanupEligibility checks if service is ready for cleanup
func (m *CleanupManager) CheckCleanupEligibility(service *Service) (bool, string) {
    // Check grace period elapsed
    // Check if confirmation received (if required)
    // Check if backup created (if required)
    // Check manual override flag
}

// CreateBackup creates a backup before cleanup
func (m *CleanupManager) CreateBackup(ctx context.Context, service *Service) error {
    // Export cluster configuration
    // Save kubeconfig
    // Export workload manifests
    // Store in backup location
}

// SendCleanupWarning sends warning notification
func (m *CleanupManager) SendCleanupWarning(service *Service, hoursRemaining int) error {
    // Send email notification
    // Send in-app notification
    // Log warning
}

// PerformCleanup performs the actual cleanup with safety checks
func (m *CleanupManager) PerformCleanup(ctx context.Context, service *Service) error {
    // Final eligibility check
    // Create backup if not already done
    // Perform cleanup
    // Verify cleanup success
    // Update service status
}
```

**File: `api/controllers/app/service_controller.go`**
```go
// Update reconciliation logic:
func (r *ServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // ... existing code ...
    
    // Handle expired services with grace period
    if service.Status.Expired {
        cleanupMgr := NewCleanupManager(r.Client)
        
        // Schedule cleanup if not already scheduled
        if service.Status.CleanupScheduledAt == nil {
            if err := cleanupMgr.ScheduleCleanup(ctx, service); err != nil {
                return ctrl.Result{}, err
            }
            return ctrl.Result{RequeueAfter: 1 * time.Hour}, nil
        }
        
        // Send periodic warnings
        hoursRemaining := calculateHoursRemaining(service)
        if shouldSendWarning(service, hoursRemaining) {
            cleanupMgr.SendCleanupWarning(service, hoursRemaining)
        }
        
        // Check if ready for cleanup
        eligible, reason := cleanupMgr.CheckCleanupEligibility(service)
        if !eligible {
            logger.Info("Service not eligible for cleanup", "reason", reason)
            return ctrl.Result{RequeueAfter: 1 * time.Hour}, nil
        }
        
        // Perform cleanup with safety checks
        if err := cleanupMgr.PerformCleanup(ctx, service); err != nil {
            return ctrl.Result{}, err
        }
    }
    
    // ... rest of code ...
}
```

**Acceptance Criteria:**
- [ ] Grace period configurable (default 24 hours)
- [ ] Backup created before cleanup
- [ ] Multiple warning notifications sent
- [ ] Manual override option available
- [ ] Dry-run mode for testing
- [ ] Audit trail for all cleanup operations

---

### 3. Rate Limiting for Kubeconfig Endpoint
**Status:** Not Started  
**Estimated Effort:** 1-2 days  
**Priority:** HIGH

#### Implementation Plan:

**File: `api/internal/pkg/pac-go-server/middleware/ratelimit.go`** (NEW)
```go
package middleware

import (
    "sync"
    "time"
    "github.com/gin-gonic/gin"
)

type RateLimiter struct {
    requests map[string][]time.Time
    mu       sync.RWMutex
    limit    int
    window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
    return &RateLimiter{
        requests: make(map[string][]time.Time),
        limit:    limit,
        window:   window,
    }
}

func (rl *RateLimiter) Middleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        userID := getUserID(c)
        endpoint := c.Request.URL.Path
        key := userID + ":" + endpoint
        
        if !rl.Allow(key) {
            c.JSON(429, gin.H{
                "error": "Rate limit exceeded. Maximum 5 downloads per hour.",
                "retry_after": rl.RetryAfter(key),
            })
            c.Abort()
            return
        }
        
        c.Next()
    }
}

func (rl *RateLimiter) Allow(key string) bool {
    rl.mu.Lock()
    defer rl.mu.Unlock()
    
    now := time.Now()
    cutoff := now.Add(-rl.window)
    
    // Clean old requests
    requests := rl.requests[key]
    valid := []time.Time{}
    for _, t := range requests {
        if t.After(cutoff) {
            valid = append(valid, t)
        }
    }
    
    if len(valid) >= rl.limit {
        rl.requests[key] = valid
        return false
    }
    
    valid = append(valid, now)
    rl.requests[key] = valid
    return true
}
```

**File: `api/internal/pkg/pac-go-server/router/router.go`**
```go
// Add rate limiting to kubeconfig endpoint:
func SetupRouter() *gin.Engine {
    // ... existing code ...
    
    // Rate limiter: 5 downloads per hour per user
    kubeconfigLimiter := middleware.NewRateLimiter(5, 1*time.Hour)
    
    api.GET("/services/:name/kubeconfig", 
        kubeconfigLimiter.Middleware(),
        services.GetServiceKubeconfig,
    )
    
    // ... rest of code ...
}
```

**Acceptance Criteria:**
- [ ] Max 5 kubeconfig downloads per hour per user
- [ ] Clear error message when limit exceeded
- [ ] Retry-After header in response
- [ ] Admin users exempt from rate limiting
- [ ] Rate limit configurable via environment variable

---

## Priority 2: Important for Production

### 4. Database Audit Storage
**Status:** Not Started  
**Estimated Effort:** 2 days  
**Priority:** MEDIUM

#### Implementation Plan:

**File: `api/internal/pkg/pac-go-server/models/audit.go`** (NEW)
```go
package models

import "time"

type AuditEvent struct {
    ID          string    `json:"id"`
    Timestamp   time.Time `json:"timestamp"`
    UserID      string    `json:"userId"`
    UserEmail   string    `json:"userEmail"`
    Action      string    `json:"action"` // KUBECONFIG_DOWNLOAD, SERVICE_CREATE, etc.
    ResourceID  string    `json:"resourceId"`
    IPAddress   string    `json:"ipAddress"`
    UserAgent   string    `json:"userAgent"`
    Success     bool      `json:"success"`
    ErrorMsg    string    `json:"errorMsg,omitempty"`
    Metadata    map[string]string `json:"metadata,omitempty"`
}
```

**File: `api/internal/pkg/pac-go-server/db/audit.go`** (NEW)
```go
package db

func (db *Database) CreateAuditEvent(event *models.AuditEvent) error {
    // Store in database
}

func (db *Database) GetAuditEvents(filters AuditFilters) ([]models.AuditEvent, error) {
    // Query audit events
}
```

**File: `api/internal/pkg/pac-go-server/services/k8s_service.go`**
```go
// Update GetServiceKubeconfig:
func GetServiceKubeconfig(c *gin.Context) {
    // ... existing code ...
    
    // Store audit event in database
    event := &models.AuditEvent{
        ID:         generateID(),
        Timestamp:  time.Now(),
        UserID:     userId,
        UserEmail:  userEmail,
        Action:     "KUBECONFIG_DOWNLOAD",
        ResourceID: serviceName,
        IPAddress:  c.ClientIP(),
        UserAgent:  c.Request.UserAgent(),
        Success:    true,
        Metadata: map[string]string{
            "cluster_id": service.Status.K8sCluster.ClusterID,
            "is_admin":   fmt.Sprintf("%v", isAdmin),
        },
    }
    
    if err := dbCon.CreateAuditEvent(event); err != nil {
        logger.Error("Failed to store audit event", zap.Error(err))
        // Don't fail the request, just log the error
    }
    
    // ... rest of code ...
}
```

**Acceptance Criteria:**
- [ ] All kubeconfig downloads stored in database
- [ ] Audit events queryable via API
- [ ] Retention policy (90 days default)
- [ ] Export capability for compliance
- [ ] Dashboard for audit visualization

---

### 5. Credential Rotation
**Status:** Not Started  
**Estimated Effort:** 3 days  
**Priority:** MEDIUM

#### Implementation Plan:

**File: `api/internal/pkg/pac-go-server/service/k8s/credentials.go`** (NEW)
```go
package k8s

type CredentialManager struct {
    // ... fields ...
}

// RotateCredentials rotates cluster credentials
func (cm *CredentialManager) RotateCredentials(clusterID string) error {
    // Generate new certificates
    // Update cluster with new certs
    // Invalidate old kubeconfigs
    // Notify users of rotation
}

// ScheduleRotation schedules automatic credential rotation
func (cm *CredentialManager) ScheduleRotation(clusterID string, interval time.Duration) {
    // Schedule periodic rotation
}
```

**Acceptance Criteria:**
- [ ] Automatic rotation every 90 days
- [ ] Manual rotation on-demand
- [ ] Notification before rotation
- [ ] Grace period for old credentials
- [ ] Audit trail for rotations

---

### 6. Monitoring and Alerting
**Status:** Not Started  
**Estimated Effort:** 2-3 days  
**Priority:** MEDIUM

#### Implementation Plan:

**Metrics to Track:**
- Cluster deployment success rate
- Average deployment time
- Cluster health status
- Kubeconfig download count
- Failed deployments
- Resource utilization

**Alerts to Configure:**
- Cluster deployment failure
- Cluster unhealthy for > 5 minutes
- High kubeconfig download rate (potential security issue)
- Cleanup failures
- Resource quota exceeded

**File: `api/internal/pkg/pac-go-server/metrics/k8s.go`** (NEW)
```go
package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
    K8sDeploymentDuration = prometheus.NewHistogram(...)
    K8sDeploymentTotal = prometheus.NewCounterVec(...)
    K8sClusterHealth = prometheus.NewGaugeVec(...)
    KubeconfigDownloads = prometheus.NewCounterVec(...)
)
```

**Acceptance Criteria:**
- [ ] Prometheus metrics exposed
- [ ] Grafana dashboard created
- [ ] Alert rules configured
- [ ] PagerDuty integration
- [ ] Slack notifications

---

## Priority 3: Nice to Have

### 7. Enhanced UI Features
**Estimated Effort:** 2 days  
**Priority:** LOW

- Cluster resource usage visualization
- Real-time deployment progress
- Cluster logs viewer
- Node status dashboard
- Cost estimation

### 8. Advanced Features
**Estimated Effort:** 5+ days  
**Priority:** LOW

- Multi-region cluster support
- Cluster upgrade capability
- Backup and restore
- Disaster recovery
- Auto-scaling configuration

---

## Implementation Timeline

### Week 1-2: Critical Items
- [ ] Comprehensive test suite
- [ ] Auto-cleanup safety mechanisms
- [ ] Rate limiting

### Week 3: Important Items
- [ ] Database audit storage
- [ ] Credential rotation
- [ ] Monitoring and alerting

### Week 4: Nice to Have
- [ ] Enhanced UI features
- [ ] Advanced features (if time permits)

---

## Success Criteria

### Before Production Deployment:
- [ ] All Priority 1 items completed
- [ ] 80%+ test coverage
- [ ] Security audit passed
- [ ] Load testing completed
- [ ] Documentation updated
- [ ] Runbook created for operations team

### Production Readiness Checklist:
- [ ] Tests passing in CI/CD
- [ ] Security review approved
- [ ] Performance benchmarks met
- [ ] Monitoring configured
- [ ] Alerts tested
- [ ] Rollback plan documented
- [ ] On-call team trained

---

## Notes
- Each item should be a separate PR for easier review
- Tests should be added incrementally with each feature
- Security items (rate limiting, audit storage) are highest priority
- Monitor production metrics closely after deployment