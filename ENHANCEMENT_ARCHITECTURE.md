# Power Access Cloud Enhancement Architecture

## Overview
This document outlines the architectural enhancements to support multiple resource types beyond CentOS VMs:
1. Kubernetes Clusters (with future OpenShift support)
2. AIX Virtual Machines
3. IBMi Virtual Machines

## Current Architecture Analysis

### Existing Components
- **CRDs**: Catalog and Service custom resources
- **Controllers**: Catalog and Service reconcilers
- **Service Providers**: VM service implementation for CentOS
- **Scope**: PowerVS client integration and resource management
- **Web UI**: React-based self-service portal

### Current Flow
1. Admin creates Catalog CR with VM specifications
2. User requests Service from Catalog
3. Service controller provisions PowerVS VM
4. Time-limited access with auto-expiry

## Enhanced Architecture

### 1. Extended CRD Types

#### Catalog Types
```go
// Current: CatalogTypeVM = "VM"
// New additions:
CatalogTypeK8s = "K8S"
CatalogTypeAIX = "AIX"
CatalogTypeIBMi = "IBMi"
```

#### Catalog Spec Extensions
```yaml
# K8s Catalog
spec:
  type: K8S
  k8s:
    crn: "crn:..."
    cluster_type: "standard|openshift"  # Future: openshift
    version: "1.28"
    worker_count: 3
    worker_flavor: "bx2.4x16"
    zone: "us-south-1"

# AIX Catalog
spec:
  type: AIX
  aix:
    crn: "crn:..."
    image: "AIX-7300-00-00"
    processor_type: "dedicated"
    system_type: "s922"
    capacity:
      cpu: "1"
      memory: 16

# IBMi Catalog
spec:
  type: IBMi
  ibmi:
    crn: "crn:..."
    image: "IBMi-75-01"
    processor_type: "dedicated"
    system_type: "e980"
    capacity:
      cpu: "2"
      memory: 32
    license_repository: "license-repo-name"
```

### 2. Service Status Extensions

```go
type ServiceStatus struct {
    VM VM `json:"vm,omitempty"`
    K8s K8s `json:"k8s,omitempty"`        // New
    AIX AIX `json:"aix,omitempty"`        // New
    IBMi IBMi `json:"ibmi,omitempty"`     // New
    AccessInfo string `json:"accessInfo"`
    // ... existing fields
}

type K8s struct {
    ClusterID string `json:"cluster_id,omitempty"`
    ClusterName string `json:"cluster_name,omitempty"`
    IngressHostname string `json:"ingress_hostname,omitempty"`
    State string `json:"state,omitempty"`
    MasterURL string `json:"master_url,omitempty"`
}

type AIX struct {
    InstanceID string `json:"instance_id,omitempty"`
    IPAddress string `json:"ip_address,omitempty"`
    ExternalIPAddress string `json:"external_ip_address,omitempty"`
    State string `json:"state,omitempty"`
    OSVersion string `json:"os_version,omitempty"`
}

type IBMi struct {
    InstanceID string `json:"instance_id,omitempty"`
    IPAddress string `json:"ip_address,omitempty"`
    ExternalIPAddress string `json:"external_ip_address,omitempty"`
    State string `json:"state,omitempty"`
    OSVersion string `json:"os_version,omitempty"`
    ConsoleURL string `json:"console_url,omitempty"`
}
```

### 3. Service Provider Interface

All service types implement the same interface:
```go
type Interface interface {
    Reconcile(ctx context.Context) error
    Delete(ctx context.Context) (bool, error)
}
```

#### Implementation Structure
```
api/controllers/app/service/
├── interface.go          # Common interface
├── vm.go                 # Existing CentOS VM (keep as-is)
├── k8s.go               # New: K8s cluster provisioning
├── aix.go               # New: AIX VM provisioning
├── ibmi.go              # New: IBMi VM provisioning
└── factory.go           # New: Service factory pattern
```

### 4. Client Extensions

#### PowerVS Client Enhancements
The existing PowerVS client already supports AIX and IBMi through the IBM Cloud Power VS API. We'll add helper methods:

```go
// In api/internal/pkg/client/powervs/client.go
func (c *Client) CreateAIXVM(opts *models.PVMInstanceCreate) error
func (c *Client) CreateIBMiVM(opts *models.PVMInstanceCreate) error
func (c *Client) GetAIXImages() ([]*models.Image, error)
func (c *Client) GetIBMiImages() ([]*models.Image, error)
```

#### New K8s Client
```go
// api/internal/pkg/client/kubernetes/client.go
type Client struct {
    containerService *containerv2.ContainerV2
    accountID string
    region string
}

func (c *Client) CreateCluster(opts *ClusterCreateOptions) (*Cluster, error)
func (c *Client) GetCluster(clusterID string) (*Cluster, error)
func (c *Client) DeleteCluster(clusterID string) error
func (c *Client) GetClusterConfig(clusterID string) ([]byte, error)
```

### 5. Controller Updates

#### Catalog Controller
- Add validation for K8s, AIX, and IBMi catalog types
- Validate images, flavors, and configurations
- Check quota and availability

#### Service Controller
- Factory pattern to instantiate correct service provider
- Handle different provisioning times (K8s takes longer)
- Update requeue logic based on service type

### 6. Scope Enhancements

```go
type ControllerScope struct {
    Logger logr.Logger
    Client client.Client
    Catalog *v1alpha1.Catalog
    PowerVSClient *powervs.Client      // Existing
    PlatformClient *platform.Client    // Existing
    K8sClient *kubernetes.Client       // New
}
```

### 7. Access Information Templates

```go
var K8sAccessInfoTemplate = func(clusterID, masterURL, ingressHostname string) string {
    return fmt.Sprintf(`
Kubernetes Cluster Access:
- Cluster ID: %s
- Master URL: %s
- Ingress: %s
- Download kubeconfig: ibmcloud ks cluster config --cluster %s
`, clusterID, masterURL, ingressHostname, clusterID)
}

var AIXAccessInfoTemplate = func(externalIP, internalIP, osVersion string) string {
    return fmt.Sprintf(`
AIX VM Access:
- External IP: %s
- Internal IP: %s
- OS Version: %s
- SSH: ssh root@%s (use registered SSH key)
`, externalIP, internalIP, osVersion, externalIP)
}

var IBMiAccessInfoTemplate = func(externalIP, internalIP, consoleURL string) string {
    return fmt.Sprintf(`
IBMi VM Access:
- External IP: %s
- Internal IP: %s
- Console: %s
- SSH: ssh qsecofr@%s (use registered SSH key)
`, externalIP, internalIP, consoleURL, externalIP)
}
```

## Implementation Strategy

### Phase 1: Foundation (CRD & Types)
1. Update catalog_types.go with new catalog types
2. Update service_types.go with new status types
3. Add validation rules and kubebuilder markers
4. Regenerate CRDs

### Phase 2: Service Providers
1. Implement k8s.go service provider
2. Implement aix.go service provider
3. Implement ibmi.go service provider
4. Create factory.go for service instantiation

### Phase 3: Client Integration
1. Add K8s client package
2. Enhance PowerVS client with AIX/IBMi helpers
3. Update scope to include new clients

### Phase 4: Controller Updates
1. Update catalog controller validation
2. Update service controller with factory pattern
3. Add type-specific reconciliation logic

### Phase 5: Samples & Documentation
1. Create sample catalogs for each type
2. Update README and documentation
3. Add deployment guides

## Key Design Decisions

### 1. Backward Compatibility
- Existing VM catalogs continue to work unchanged
- No breaking changes to existing APIs
- Additive changes only

### 2. Extensibility
- Factory pattern allows easy addition of new types
- Interface-based design for service providers
- Pluggable client architecture

### 3. Resource Isolation
- Each catalog type has its own spec section
- Status fields are type-specific
- No cross-contamination of configurations

### 4. Time Management
- K8s clusters may take 15-30 minutes to provision
- Adjust requeue intervals based on service type
- Maintain existing expiry mechanism

### 5. Access Control
- Same RBAC model for all resource types
- SSH key management for VMs
- Kubeconfig distribution for K8s

## Future Enhancements

### OpenShift Support
- Add `cluster_type: "openshift"` to K8s catalog
- Implement OpenShift-specific provisioning
- Add OpenShift console access info

### Multi-Node K8s
- Support for node pools
- Auto-scaling configurations
- Different worker flavors per pool

### Advanced Networking
- VPC integration for K8s
- Custom network configurations for AIX/IBMi
- Load balancer provisioning

### Monitoring & Observability
- Prometheus metrics for each service type
- Health checks and status monitoring
- Usage analytics per resource type

## Security Considerations

1. **Credential Management**: Use Kubernetes secrets for cloud credentials
2. **Network Isolation**: Ensure proper network segmentation
3. **Access Logging**: Audit all resource access
4. **Quota Enforcement**: Prevent resource exhaustion
5. **Data Encryption**: Encrypt sensitive data at rest and in transit

## Testing Strategy

1. **Unit Tests**: Test each service provider independently
2. **Integration Tests**: Test full provisioning workflows
3. **E2E Tests**: Test user journeys for each resource type
4. **Performance Tests**: Validate concurrent provisioning
5. **Chaos Tests**: Test failure scenarios and recovery

## Migration Path

For existing deployments:
1. Deploy updated CRDs (backward compatible)
2. Update controller deployment
3. Existing VM catalogs continue working
4. Add new catalog types incrementally
5. No downtime required