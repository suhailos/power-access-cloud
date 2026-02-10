package k8s

import (
	pac "github.com/IBM/power-access-cloud/api/apis/app/v1alpha1"
)

// Service defines the interface for K8s cluster operations
type Service interface {
	// DeployCluster deploys a new K8s cluster
	DeployCluster(catalog pac.K8sCatalog, sshKeys []string) (*ClusterInfo, error)
	
	// GetClusterStatus returns the status of a K8s cluster
	GetClusterStatus(clusterID string) (*ClusterStatus, error)
	
	// DeleteCluster deletes a K8s cluster
	DeleteCluster(clusterID string) error
	
	// GetClusterAccessInfo returns access information for the cluster
	GetClusterAccessInfo(clusterID string) (*ClusterAccessInfo, error)
}

// ClusterInfo contains information about a deployed cluster
type ClusterInfo struct {
	ClusterID     string                 `json:"cluster_id"`
	MasterNodes   []NodeInfo             `json:"master_nodes"`
	WorkerNodes   []NodeInfo             `json:"worker_nodes"`
	Status        string                 `json:"status"`
	Configuration map[string]interface{} `json:"configuration"`
}

// NodeInfo contains information about a cluster node
type NodeInfo struct {
	NodeID     string `json:"node_id"`
	IPAddress  string `json:"ip_address"`
	Role       string `json:"role"` // master or worker
	Status     string `json:"status"`
	InstanceID string `json:"instance_id"`
}

// ClusterStatus represents the current status of a cluster
type ClusterStatus struct {
	State          string   `json:"state"` // provisioning, running, failed, deleting
	Message        string   `json:"message"`
	ReadyNodes     int      `json:"ready_nodes"`
	TotalNodes     int      `json:"total_nodes"`
	KubeVersion    string   `json:"kube_version"`
	MasterEndpoint string   `json:"master_endpoint"`
	Conditions     []string `json:"conditions"`
}

// ClusterAccessInfo contains access information for the cluster
type ClusterAccessInfo struct {
	KubeconfigContent string   `json:"kubeconfig_content"`
	MasterEndpoint    string   `json:"master_endpoint"`
	MasterIPs         []string `json:"master_ips"`
	DashboardURL      string   `json:"dashboard_url,omitempty"`
	APIServerURL      string   `json:"api_server_url"`
}

// Made with Bob
