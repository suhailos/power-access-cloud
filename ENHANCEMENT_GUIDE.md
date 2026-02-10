# Power Access Cloud Enhancement Guide

## Overview

This guide describes the enhancements made to IBM Power Access Cloud to support multiple resource types beyond CentOS VMs:

1. **Kubernetes Clusters** - Standard and OpenShift clusters (OpenShift support is future-ready)
2. **AIX Virtual Machines** - AIX operating system on Power Systems
3. **IBMi Virtual Machines** - IBMi operating system on Power Systems

All enhancements maintain **backward compatibility** with existing CentOS VM catalogs and the time-limited, free access model.

## Architecture Changes

### New Catalog Types

The system now supports four catalog types:

- `VM` - CentOS/Linux Virtual Machines (existing)
- `K8S` - Kubernetes Clusters (new)
- `AIX` - AIX Virtual Machines (new)
- `IBMi` - IBMi Virtual Machines (new)

### Component Updates

1. **CRD Types** (`api/apis/app/v1alpha1/`)
   - Extended `CatalogType` enum with K8S, AIX, IBMi
   - Added type-specific catalog specs (K8sCatalog, AIXCatalog, IBMiCatalog)
   - Added type-specific service status (K8s, AIX, IBMi)

2. **Service Providers** (`api/controllers/app/service/`)
   - `vm.go` - Existing CentOS VM provider
   - `k8s.go` - New Kubernetes cluster provider
   - `aix.go` - New AIX VM provider
   - `ibmi.go` - New IBMi VM provider

3. **Client Integration** (`api/internal/pkg/client/`)
   - `kubernetes/` - New K8s client for cluster management
   - Enhanced PowerVS client usage for AIX and IBMi

4. **Controllers** (`api/controllers/app/`)
   - Updated catalog controller with validation for all types
   - Updated service controller with factory pattern for service instantiation
   - Type-specific requeue intervals (K8s takes longer)

## Usage Guide

### Creating Catalogs

#### Kubernetes Cluster Catalog

```yaml
apiVersion: app.pac.io/v1alpha1
kind: Catalog
metadata:
  name: standard-k8s-cluster
spec:
  type: K8S
  capacity:
    cpu: '4'
    memory: 16
  description: 'Standard Kubernetes cluster with 3 worker nodes'
  expiry: 7
  image_thumbnail_reference: 'https://example.com/k8s-icon.png'
  k8s:
    crn: 'crn:v1:bluemix:public:power-iaas:us-south:a/<accountID>:<instanceID>::'
    cluster_type: standard  # or 'openshift' for future
    version: '1.28'
    worker_count: 3
    worker_flavor: 'bx2.4x16'
    zone: 'us-south-1'
```

#### AIX VM Catalog

```yaml
apiVersion: app.pac.io/v1alpha1
kind: Catalog
metadata:
  name: medium-aix-vm
spec:
  type: AIX
  capacity:
    cpu: '1'
    memory: 16
  description: 'Medium size AIX VM with AIX 7.3'
  expiry: 14
  image_thumbnail_reference: 'https://example.com/aix-icon.png'
  aix:
    crn: 'crn:v1:bluemix:public:power-iaas:lon06:a/<accountID>:<instanceID>::'
    image: 'AIX-7300-00-00'
    network: 'my-network'
    processor_type: dedicated
    system_type: s922
    capacity:
      cpu: '1'
      memory: 16
```

#### IBMi VM Catalog

```yaml
apiVersion: app.pac.io/v1alpha1
kind: Catalog
metadata:
  name: large-ibmi-vm
spec:
  type: IBMi
  capacity:
    cpu: '2'
    memory: 32
  description: 'Large size IBMi VM with IBMi 7.5'
  expiry: 14
  image_thumbnail_reference: 'https://example.com/ibmi-icon.png'
  ibmi:
    crn: 'crn:v1:bluemix:public:power-iaas:lon06:a/<accountID>:<instanceID>::'
    image: 'IBMi-75-01'
    network: 'my-network'
    processor_type: dedicated
    system_type: e980
    capacity:
      cpu: '2'
      memory: 32
    license_repository: 'ibmi-license-repo'
```

### Requesting Services

Users request services the same way regardless of catalog type:

```yaml
apiVersion: app.pac.io/v1alpha1
kind: Service
metadata:
  name: my-k8s-cluster
spec:
  user_id: "user@example.com"
  display_name: "My Development Cluster"
  catalog:
    name: standard-k8s-cluster
  ssh_keys:
    - "ssh-rsa AAAAB3NzaC1yc2E..."
  expiry: "2024-12-31T23:59:59Z"
```

### Access Information

Each service type provides specific access information:

#### Kubernetes Cluster
```
Kubernetes Cluster Access:
- Cluster ID: abc123
- Master URL: https://c100.us-south.containers.cloud.ibm.com:12345
- Ingress: mycluster-abc123.us-south.containers.appdomain.cloud
- Download kubeconfig: ibmcloud ks cluster config --cluster abc123
```

#### AIX VM
```
AIX VM Access:
- External IP: 169.48.123.45
- Internal IP: 192.168.1.10
- OS Version: AIX-7300-00-00
- SSH: ssh root@169.48.123.45 (use registered SSH key)
```

#### IBMi VM
```
IBMi VM Access:
- External IP: 169.48.123.46
- Internal IP: 192.168.1.11
- Console: https://cloud.ibm.com/power/instances/.../virtual-servers/...
- SSH: ssh qsecofr@169.48.123.46 (use registered SSH key)
```

## Configuration

### Environment Variables

For Kubernetes cluster provisioning:
```bash
export IBMCLOUD_API_KEY="your-api-key"
```

For PowerVS (VM, AIX, IBMi):
```bash
export IBMCLOUD_API_KEY="your-api-key"
```

### Catalog Validation

The system validates:

**For K8S:**
- Cluster type is 'standard' or 'openshift'
- Worker count is between 1 and 10
- Version, flavor, and zone are specified

**For AIX/IBMi:**
- PowerVS instance is active
- Image exists and is active
- Network exists (if specified)
- System type is valid (s922, e980, etc.)
- Processor type is valid (shared, dedicated, capped)

## Provisioning Times

Different resource types have different provisioning times:

- **CentOS VM**: 5-10 minutes
- **AIX VM**: 10-15 minutes
- **IBMi VM**: 15-20 minutes
- **K8s Cluster**: 20-30 minutes

The system automatically adjusts requeue intervals based on resource type.

## Expiry and Cleanup

All resources follow the same expiry model:

1. Resources are provisioned with a time limit (default: 5 days, configurable per catalog)
2. Users can request extensions (subject to approval)
3. Expired resources are automatically cleaned up after 24 hours
4. Users receive notifications before expiry

## Migration from Existing Deployment

1. **Deploy Updated CRDs**
   ```bash
   kubectl apply -f api/config/crd/bases/
   ```

2. **Update Controller**
   ```bash
   kubectl apply -f api/config/manager/manager.yaml
   ```

3. **Existing VM Catalogs Continue Working**
   - No changes needed to existing catalogs
   - Existing services continue to function
   - Zero downtime migration

4. **Add New Catalogs**
   ```bash
   kubectl apply -f api/config/samples/app_v1alpha1_catalog_k8s.yaml
   kubectl apply -f api/config/samples/app_v1alpha1_catalog_aix.yaml
   kubectl apply -f api/config/samples/app_v1alpha1_catalog_ibmi.yaml
   ```

## Development and Testing

### Running Locally

```bash
cd api
make install  # Install CRDs
make run      # Run controller locally
```

### Testing Catalog Creation

```bash
# Test K8s catalog
kubectl apply -f config/samples/app_v1alpha1_catalog_k8s.yaml
kubectl get catalogs standard-k8s-cluster -o yaml

# Test AIX catalog
kubectl apply -f config/samples/app_v1alpha1_catalog_aix.yaml
kubectl get catalogs medium-aix-vm -o yaml

# Test IBMi catalog
kubectl apply -f config/samples/app_v1alpha1_catalog_ibmi.yaml
kubectl get catalogs large-ibmi-vm -o yaml
```

### Testing Service Provisioning

```bash
# Create a service
kubectl apply -f config/samples/app_v1alpha1_service.yaml

# Watch service status
kubectl get services my-service -w

# Check service details
kubectl describe service my-service
```

## Known Limitations

1. **K8s Client Implementation**: The Kubernetes client (`api/internal/pkg/client/kubernetes/client.go`) is a placeholder. Full implementation requires:
   - IBM Cloud Kubernetes Service SDK integration
   - Proper authentication and authorization
   - Cluster lifecycle management APIs

2. **OpenShift Support**: While the CRD supports `cluster_type: openshift`, full OpenShift provisioning requires additional implementation.

3. **Network Configuration**: Advanced networking features (VPC, custom subnets) are supported in the CRD but may need additional validation logic.

4. **License Management**: IBMi license repository integration is basic and may need enhancement for production use.

## Future Enhancements

1. **OpenShift Clusters**: Complete implementation of OpenShift cluster provisioning
2. **Multi-Zone K8s**: Support for multi-zone Kubernetes clusters
3. **Auto-Scaling**: Dynamic worker node scaling for K8s clusters
4. **Advanced Monitoring**: Resource utilization tracking and alerts
5. **Cost Tracking**: Usage-based cost reporting per user/group
6. **Backup/Restore**: Automated backup for VMs and cluster configurations

## Troubleshooting

### Catalog Not Ready

Check catalog status:
```bash
kubectl describe catalog <catalog-name>
```

Common issues:
- PowerVS instance not active
- Image not found or not active
- Invalid CRN format
- Network not found

### Service Stuck in IN_PROGRESS

Check service status:
```bash
kubectl describe service <service-name>
```

Common issues:
- Insufficient quota in PowerVS
- Network IP exhaustion
- Image provisioning failure
- K8s cluster creation timeout

### Access Information Not Showing

Ensure service is in CREATED state:
```bash
kubectl get service <service-name> -o jsonpath='{.status.state}'
```

## Support

For issues or questions:
1. Check the [FAQ](support/docs/FAQ.md)
2. Review logs: `kubectl logs -n pac-system deployment/pac-controller-manager`
3. Open an issue on GitHub

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on contributing to this project.