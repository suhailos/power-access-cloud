package service

import (
	"context"
	"fmt"

	appv1alpha1 "github.com/IBM/power-access-cloud/api/apis/app/v1alpha1"
	"github.com/IBM/power-access-cloud/api/controllers/app/scope"
	"github.com/pkg/errors"
)

var _ Interface = &K8s{}

type K8s struct {
	scope *scope.ServiceScope
}

func NewK8s(scope *scope.ServiceScope) Interface {
	return &K8s{
		scope: scope,
	}
}

func (s *K8s) Reconcile(ctx context.Context) error {
	if s.scope.Service.Status.K8s.ClusterID == "" {
		if err := createK8sCluster(s.scope); err != nil {
			return errors.Wrap(err, "error creating k8s cluster")
		}
	}

	cluster, err := s.scope.K8sClient.GetCluster(s.scope.Service.Status.K8s.ClusterID)
	if err != nil {
		return errors.Wrap(err, "error getting k8s cluster")
	}

	updateK8sStatus(s.scope, cluster)

	return nil
}

func (s *K8s) Delete(ctx context.Context) (bool, error) {
	if s.scope.Service.Status.K8s.ClusterID == "" {
		s.scope.Logger.Info("k8s cluster ID is empty, nothing to clean up")
		return true, nil
	}

	if err := cleanupK8sCluster(s.scope); err != nil {
		return false, errors.Wrap(err, "error cleaning up k8s cluster")
	}
	s.scope.Service.Status.ClearK8sStatus()

	return true, nil
}

func cleanupK8sCluster(scope *scope.ServiceScope) error {
	return scope.K8sClient.DeleteCluster(scope.Service.Status.K8s.ClusterID)
}

func updateK8sStatus(scope *scope.ServiceScope, cluster *K8sCluster) {
	scope.Service.Status.K8s.ClusterID = cluster.ID
	scope.Service.Status.K8s.ClusterName = cluster.Name
	scope.Service.Status.K8s.State = cluster.State
	scope.Service.Status.K8s.MasterURL = cluster.MasterURL
	scope.Service.Status.K8s.IngressHostname = cluster.IngressHostname

	switch cluster.State {
	case "normal", "deployed":
		scope.Service.Status.SetSuccessful()
		scope.Service.Status.State = appv1alpha1.ServiceStateCreated
		scope.Service.Status.AccessInfo = appv1alpha1.K8sAccessInfoTemplate(
			scope.Service.Status.K8s.ClusterID,
			scope.Service.Status.K8s.MasterURL,
			scope.Service.Status.K8s.IngressHostname,
		)
		scope.Service.Status.Message = ""
	case "critical", "failed":
		scope.Service.Status.State = appv1alpha1.ServiceStateFailed
		scope.Service.Status.Message = fmt.Sprintf("k8s cluster creation failed with state: %s", cluster.State)
		scope.Service.Status.AccessInfo = ""
	default:
		scope.Service.Status.State = appv1alpha1.ServiceStateInProgress
		scope.Service.Status.Message = "k8s cluster creation in progress, will update the access info once cluster is ready"
	}
}

func createK8sCluster(scope *scope.ServiceScope) error {
	// Check if cluster already exists
	clusters, err := scope.K8sClient.ListClusters()
	if err != nil {
		return errors.Wrap(err, "error listing clusters")
	}

	for _, cluster := range clusters {
		if cluster.Name == scope.Service.ObjectMeta.Name {
			scope.Logger.Info("k8s cluster already exists, hence skipping the cluster creation", "name", scope.Service.ObjectMeta.Name)
			scope.Service.Status.K8s.ClusterID = cluster.ID
			return nil
		}
	}

	k8sSpec := scope.Catalog.Spec.K8s

	createOpts := &K8sClusterCreateOptions{
		Name:         scope.Service.Name,
		ClusterType:  k8sSpec.ClusterType,
		Version:      k8sSpec.Version,
		WorkerCount:  k8sSpec.WorkerCount,
		WorkerFlavor: k8sSpec.WorkerFlavor,
		Zone:         k8sSpec.Zone,
		VpcID:        k8sSpec.VpcID,
		SubnetID:     k8sSpec.SubnetID,
	}

	cluster, err := scope.K8sClient.CreateCluster(createOpts)
	if err != nil {
		return errors.Wrap(err, "error creating k8s cluster")
	}

	scope.Service.Status.Message = "k8s cluster creation started, will update the access info once cluster is ready"
	scope.Service.Status.K8s.ClusterID = cluster.ID
	scope.Service.Status.K8s.ClusterName = cluster.Name

	return nil
}

// K8sCluster represents a Kubernetes cluster
type K8sCluster struct {
	ID              string
	Name            string
	State           string
	MasterURL       string
	IngressHostname string
}

// K8sClusterCreateOptions represents options for creating a K8s cluster
type K8sClusterCreateOptions struct {
	Name         string
	ClusterType  string
	Version      string
	WorkerCount  int
	WorkerFlavor string
	Zone         string
	VpcID        string
	SubnetID     string
}

// Made with Bob
