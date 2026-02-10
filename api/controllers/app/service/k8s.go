package service

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	pac "github.com/IBM/power-access-cloud/api/apis/app/v1alpha1"
	"github.com/IBM/power-access-cloud/api/controllers/app/scope"
	k8ssvc "github.com/IBM/power-access-cloud/api/internal/pkg/pac-go-server/service/k8s"
)

// K8sService implements the Interface for K8s cluster services
type K8sService struct {
	scope      *scope.ServiceScope
	k8sService k8ssvc.Service
	logger     *zap.Logger
}

// NewK8sService creates a new K8s service
func NewK8sService(scope *scope.ServiceScope, logger *zap.Logger) Interface {
	return &K8sService{
		scope:      scope,
		k8sService: k8ssvc.NewK8sService(logger),
		logger:     logger,
	}
}

// Reconcile reconciles the K8s cluster service
func (s *K8sService) Reconcile(ctx context.Context) error {
	s.logger.Info("Reconciling K8s cluster service",
		zap.String("service", s.scope.Service.Name),
		zap.String("namespace", s.scope.Service.Namespace),
	)

	// Get the catalog
	catalog := &pac.Catalog{}
	catalogKey := s.scope.Service.Spec.Catalog
	if err := s.scope.Client.Get(ctx, catalogKey, catalog); err != nil {
		return errors.Wrap(err, "failed to get catalog")
	}

	// Verify it's a K8s catalog
	if catalog.Spec.Type != pac.CatalogTypeK8s {
		return fmt.Errorf("catalog type mismatch: expected K8s, got %s", catalog.Spec.Type)
	}

	// Check current service state
	switch s.scope.Service.Status.State {
	case "":
		// New service, start provisioning
		return s.provisionCluster(ctx, catalog)
	case pac.ServiceStateProvisioning:
		// Check provisioning status
		return s.checkProvisioningStatus(ctx)
	case pac.ServiceStateRunning:
		// Service is running, nothing to do
		s.logger.Info("K8s cluster service is running")
		return nil
	case pac.ServiceStateFailed:
		// Service failed, log error
		s.logger.Error("K8s cluster service is in failed state",
			zap.String("message", s.scope.Service.Status.Message))
		return nil
	default:
		s.logger.Warn("Unknown service state", zap.String("state", string(s.scope.Service.Status.State)))
		return nil
	}
}

// provisionCluster provisions a new K8s cluster
func (s *K8sService) provisionCluster(ctx context.Context, catalog *pac.Catalog) error {
	s.logger.Info("Provisioning K8s cluster",
		zap.String("service", s.scope.Service.Name),
		zap.String("catalog", catalog.Name),
	)

	// Update service state to in progress
	s.scope.Service.Status.State = pac.ServiceStateInProgress
	s.scope.Service.Status.Message = "Provisioning K8s cluster nodes..."

	// Deploy the cluster
	clusterInfo, err := s.k8sService.DeployCluster(catalog.Spec.K8s, s.scope.Service.Spec.SSHKeys)
	if err != nil {
		s.scope.Service.Status.State = pac.ServiceStateFailed
		s.scope.Service.Status.Message = fmt.Sprintf("Failed to deploy cluster: %v", err)
		return errors.Wrap(err, "failed to deploy K8s cluster")
	}

	// Store cluster information in service status
	s.scope.Service.Status.K8sCluster.ClusterID = clusterInfo.ClusterID
	s.scope.Service.Status.K8sCluster.State = "provisioning"
	s.scope.Service.Status.K8sCluster.TotalNodes = len(clusterInfo.MasterNodes) + len(clusterInfo.WorkerNodes)
	s.scope.Service.Status.K8sCluster.ReadyNodes = 0
	
	// Store cluster ID in annotations for later reference
	if s.scope.Service.Annotations == nil {
		s.scope.Service.Annotations = make(map[string]string)
	}
	s.scope.Service.Annotations["cluster-id"] = clusterInfo.ClusterID

	s.scope.Service.Status.AccessInfo = "Cluster is being provisioned. Access information will be available once ready."

	s.logger.Info("K8s cluster provisioning initiated",
		zap.String("cluster_id", clusterInfo.ClusterID),
	)

	return nil
}

// checkProvisioningStatus checks the status of cluster provisioning
func (s *K8sService) checkProvisioningStatus(ctx context.Context) error {
	s.logger.Info("Checking K8s cluster provisioning status",
		zap.String("service", s.scope.Service.Name),
	)

	// Get cluster status (this would use the cluster ID from service annotations)
	clusterID := s.scope.Service.Annotations["cluster-id"]
	if clusterID == "" {
		s.logger.Warn("Cluster ID not found in service annotations")
		return nil
	}

	status, err := s.k8sService.GetClusterStatus(clusterID)
	if err != nil {
		return errors.Wrap(err, "failed to get cluster status")
	}

	// Update K8s cluster status
	s.scope.Service.Status.K8sCluster.State = status.State
	s.scope.Service.Status.K8sCluster.ReadyNodes = status.ReadyNodes
	s.scope.Service.Status.K8sCluster.TotalNodes = status.TotalNodes
	s.scope.Service.Status.K8sCluster.KubeVersion = status.KubeVersion
	s.scope.Service.Status.K8sCluster.APIServerURL = status.MasterEndpoint

	// Update service status based on cluster status
	switch status.State {
	case "running":
		s.scope.Service.Status.State = pac.ServiceStateCreated
		s.scope.Service.Status.SetSuccessful()
		s.scope.Service.Status.Message = ""
		
		// Get access information
		accessInfo, err := s.k8sService.GetClusterAccessInfo(clusterID)
		if err != nil {
			s.logger.Error("Failed to get cluster access info", zap.Error(err))
		} else {
			// Update K8s cluster status with access info
			s.scope.Service.Status.K8sCluster.MasterIPs = accessInfo.MasterIPs
			s.scope.Service.Status.K8sCluster.DashboardURL = accessInfo.DashboardURL
			
			// Use the template function for consistent formatting
			s.scope.Service.Status.AccessInfo = pac.K8sAccessInfoTemplate(
				accessInfo.APIServerURL,
				accessInfo.DashboardURL,
				accessInfo.MasterIPs,
			)
		}
	case "failed":
		s.scope.Service.Status.State = pac.ServiceStateFailed
		s.scope.Service.Status.Message = fmt.Sprintf("Cluster provisioning failed: %s", status.Message)
	default:
		s.scope.Service.Status.State = pac.ServiceStateInProgress
		s.scope.Service.Status.Message = fmt.Sprintf("Cluster provisioning in progress: %s (%d/%d nodes ready)",
			status.Message, status.ReadyNodes, status.TotalNodes)
	}

	return nil
}

// Delete deletes the K8s cluster
func (s *K8sService) Delete(ctx context.Context) (bool, error) {
	s.logger.Info("Deleting K8s cluster service",
		zap.String("service", s.scope.Service.Name),
	)

	// Get cluster ID from service annotations
	clusterID := s.scope.Service.Annotations["cluster-id"]
	if clusterID == "" {
		s.logger.Warn("Cluster ID not found, skipping cluster deletion")
		return true, nil
	}

	// Delete the cluster
	if err := s.k8sService.DeleteCluster(clusterID); err != nil {
		s.logger.Error("Failed to delete K8s cluster",
			zap.String("cluster_id", clusterID),
			zap.Error(err),
		)
		return false, errors.Wrap(err, "failed to delete K8s cluster")
	}

	// Clear K8s cluster status
	s.scope.Service.Status.ClearK8sClusterStatus()
	
	s.logger.Info("K8s cluster deleted successfully",
		zap.String("cluster_id", clusterID),
	)

	return true, nil
}

// Made with Bob
