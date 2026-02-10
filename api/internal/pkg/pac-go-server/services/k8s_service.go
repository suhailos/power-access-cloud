package services

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	pac "github.com/IBM/power-access-cloud/api/apis/app/v1alpha1"
	"github.com/IBM/power-access-cloud/api/internal/pkg/pac-go-server/client"
	log "github.com/IBM/power-access-cloud/api/internal/pkg/pac-go-server/logger"
	"github.com/IBM/power-access-cloud/api/internal/pkg/pac-go-server/utils"
)

// GetServiceKubeconfig godoc
// @Summary			Get kubeconfig for K8s service
// @Description		Download kubeconfig file for accessing the K8s cluster. Requires service ownership or admin role. All downloads are audited.
// @Tags			services
// @Accept			json
// @Produce			text/plain
// @Param			name path string true "service name"
// @Param			Authorization header string true "Insert your access token" default(Bearer <Add access token here>)
// @Success			200 {string} string "Kubeconfig YAML file"
// @Failure			400 {object} map[string]string "Bad request"
// @Failure			401 {object} map[string]string "Unauthorized"
// @Failure			403 {object} map[string]string "Forbidden"
// @Failure			404 {object} map[string]string "Service not found"
// @Failure			429 {object} map[string]string "Too many requests"
// @Router			/api/v1/services/{name}/kubeconfig [get]
// @Security		BearerAuth
func GetServiceKubeconfig(c *gin.Context) {
	logger := log.GetLogger()
	serviceName := c.Param("name")
	
	// Input validation
	if serviceName == "" {
		logger.Error("service name is not set")
		c.JSON(http.StatusBadRequest, gin.H{"error": "service name is required"})
		return
	}

	// Get the service
	service, err := kubeClient.GetService(serviceName)
	if err != nil {
		logger.Error("failed to get service", zap.String("service name", serviceName), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%v", err)})
		return
	}

	// Authentication & Authorization
	config := client.GetConfigFromContext(c.Request.Context())
	kc := client.NewKeyCloakClient(config, c.Request.Context())
	userId := kc.GetUserID()
	userEmail := kc.GetUserEmail()
	isAdmin := kc.IsRole(utils.ManagerRole)

	// Check ownership or admin role
	if !isAdmin && service.Spec.UserID != userId {
		logger.Warn("unauthorized kubeconfig download attempt",
			zap.String("user_id", userId),
			zap.String("user_email", userEmail),
			zap.String("service_name", serviceName),
			zap.String("service_owner", service.Spec.UserID),
		)
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied: you are not the owner of this service"})
		return
	}

	// Validate service type
	if service.Status.K8sCluster.ClusterID == "" {
		logger.Error("service is not a K8s cluster",
			zap.String("service name", serviceName),
			zap.String("user_id", userId),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "service is not a Kubernetes cluster"})
		return
	}

	// Check cluster readiness
	if service.Status.State != pac.ServiceStateCreated {
		logger.Info("cluster not ready for kubeconfig download",
			zap.String("service name", serviceName),
			zap.String("state", string(service.Status.State)),
			zap.String("user_id", userId),
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("cluster is not ready yet, current state: %s", service.Status.State),
			"state": service.Status.State,
		})
		return
	}

	// Check if service is expired
	if service.Status.Expired {
		logger.Warn("kubeconfig download attempted for expired service",
			zap.String("service name", serviceName),
			zap.String("user_id", userId),
		)
		c.JSON(http.StatusForbidden, gin.H{"error": "service has expired"})
		return
	}

	// AUDIT LOG: Record kubeconfig download
	logger.Info("kubeconfig downloaded",
		zap.String("service_name", serviceName),
		zap.String("cluster_id", service.Status.K8sCluster.ClusterID),
		zap.String("user_id", userId),
		zap.String("user_email", userEmail),
		zap.String("ip_address", c.ClientIP()),
		zap.String("user_agent", c.Request.UserAgent()),
		zap.Bool("is_admin", isAdmin),
	)

	// TODO: Store audit event in database for compliance
	// event := models.NewAuditEvent(userId, "KUBECONFIG_DOWNLOAD", serviceName, c.ClientIP())
	// dbCon.CreateAuditEvent(event)

	// Generate kubeconfig
	kubeconfig := generateKubeconfig(service)

	// Security headers
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s-kubeconfig.yaml", serviceName))
	c.Header("Content-Type", "application/x-yaml")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Cache-Control", "no-store, no-cache, must-revalidate")
	c.Header("Pragma", "no-cache")
	
	c.String(http.StatusOK, kubeconfig)
}

// generateKubeconfig generates a kubeconfig file for the K8s cluster
func generateKubeconfig(service pac.Service) string {
	cluster := service.Status.K8sCluster
	clusterName := service.Name
	
	// In a real implementation, this would retrieve actual certificates and tokens
	// For now, we generate a template that users can fill in
	kubeconfig := fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- cluster:
    certificate-authority-data: <CERTIFICATE_AUTHORITY_DATA>
    server: %s
  name: %s
contexts:
- context:
    cluster: %s
    user: %s-admin
  name: %s
current-context: %s
users:
- name: %s-admin
  user:
    client-certificate-data: <CLIENT_CERTIFICATE_DATA>
    client-key-data: <CLIENT_KEY_DATA>

# Cluster Information:
# Cluster ID: %s
# Kubernetes Version: %s
# Master Nodes: %s
# API Server: %s
# Dashboard: %s
# 
# To use this kubeconfig:
# 1. Save this file as kubeconfig.yaml
# 2. Replace <CERTIFICATE_AUTHORITY_DATA>, <CLIENT_CERTIFICATE_DATA>, and <CLIENT_KEY_DATA>
#    with actual base64-encoded certificates from the cluster
# 3. Set KUBECONFIG environment variable: export KUBECONFIG=./kubeconfig.yaml
# 4. Verify connection: kubectl get nodes
# 
# To get certificates from master node:
# SSH to master node: ssh root@%s
# Get CA cert: sudo cat /etc/kubernetes/pki/ca.crt | base64 -w 0
# Get client cert: sudo cat /etc/kubernetes/pki/admin.crt | base64 -w 0
# Get client key: sudo cat /etc/kubernetes/pki/admin.key | base64 -w 0
`,
		cluster.APIServerURL,
		clusterName,
		clusterName,
		clusterName,
		clusterName,
		clusterName,
		clusterName,
		cluster.ClusterID,
		cluster.KubeVersion,
		formatMasterIPs(cluster.MasterIPs),
		cluster.APIServerURL,
		cluster.DashboardURL,
		getMasterIP(cluster.MasterIPs),
	)

	return kubeconfig
}

func formatMasterIPs(ips []string) string {
	if len(ips) == 0 {
		return "N/A"
	}
	result := ""
	for i, ip := range ips {
		if i > 0 {
			result += ", "
		}
		result += ip
	}
	return result
}

func getMasterIP(ips []string) string {
	if len(ips) == 0 {
		return "<MASTER_IP>"
	}
	return ips[0]
}

// Made with Bob
