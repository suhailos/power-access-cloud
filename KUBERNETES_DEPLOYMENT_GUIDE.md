# Kubernetes Cluster Deployment Guide

This guide provides instructions for deploying Kubernetes clusters on CentOS using the Power Access Cloud (PAC) platform.

## Overview

The PAC platform now supports automated Kubernetes cluster deployment on CentOS-based infrastructure. This implementation provides:

- **Automated Setup**: Complete cluster provisioning with minimal manual intervention
- **CentOS Optimized**: Scripts tailored for CentOS Stream 8 and later
- **Production Ready**: Support for multi-master HA configurations
- **Flexible CNI**: Choice of Calico, Flannel, or Weave network plugins
- **Full Lifecycle Management**: Create, monitor, scale, and delete clusters

## Architecture

### Components

```
┌─────────────────────────────────────────────────────────────┐
│                     PAC Platform                             │
├─────────────────────────────────────────────────────────────┤
│  API Layer                                                   │
│  ├── Catalog Management (K8s catalog type)                  │
│  ├── Service Management (K8s service provisioning)          │
│  └── User Management (Authentication & Authorization)       │
├─────────────────────────────────────────────────────────────┤
│  Controller Layer                                            │
│  ├── Service Controller (Reconciliation loop)               │
│  ├── K8s Service Handler (Cluster lifecycle)                │
│  └── Catalog Controller (Catalog validation)                │
├─────────────────────────────────────────────────────────────┤
│  K8s Service Layer                                           │
│  ├── Cluster Deployer (Script generation)                   │
│  ├── K8s Service (Cluster operations)                       │
│  └── Status Monitor (Health checks)                         │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│              Infrastructure Provider (IBM Cloud)             │
│  ├── VM Provisioning (Master & Worker nodes)                │
│  ├── Network Configuration                                  │
│  └── Storage Management                                     │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                  Kubernetes Cluster                          │
│  ├── Master Nodes (Control Plane)                           │
│  │   ├── API Server                                         │
│  │   ├── Controller Manager                                 │
│  │   ├── Scheduler                                          │
│  │   └── etcd                                               │
│  └── Worker Nodes (Data Plane)                              │
│      ├── kubelet                                             │
│      ├── kube-proxy                                          │
│      └── Container Runtime (containerd)                     │
└─────────────────────────────────────────────────────────────┘
```

## Prerequisites

### System Requirements

- **Operating System**: CentOS Stream 8 or later
- **Minimum Resources per Node**:
  - Master: 2 vCPU, 4GB RAM, 20GB disk
  - Worker: 2 vCPU, 4GB RAM, 20GB disk
- **Network**: Private network with internet access
- **Firewall**: Required ports open (see Network Requirements)

### Network Requirements

**Master Nodes:**
- 6443/tcp - Kubernetes API server
- 2379-2380/tcp - etcd server client API
- 10250/tcp - Kubelet API
- 10251/tcp - kube-scheduler
- 10252/tcp - kube-controller-manager
- 10255/tcp - Read-only Kubelet API

**Worker Nodes:**
- 10250/tcp - Kubelet API
- 30000-32767/tcp - NodePort Services

**All Nodes:**
- SSH access (port 22)
- Internet access for package downloads

## Deployment Steps

### Step 1: Create K8s Catalog

Create a catalog that defines the Kubernetes cluster configuration:

```bash
curl -X POST https://pac-api.example.com/api/v1/catalogs \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "k8s-dev-cluster",
    "type": "K8s",
    "description": "Development Kubernetes cluster",
    "capacity": {
      "cpu": 8,
      "memory": 16384
    },
    "expiry": 7,
    "image_thumbnail_reference": "https://kubernetes.io/images/kubernetes-horizontal-color.png",
    "k8s": {
      "crn": "crn:v1:bluemix:public:power-iaas:us-south:a/account-id:instance-id::",
      "kubernetes_version": "1.28",
      "master_count": 1,
      "worker_count": 2,
      "os_image": "CentOS-Stream-8",
      "network": "pac-network",
      "pod_network_cidr": "10.244.0.0/16",
      "service_cidr": "10.96.0.0/12",
      "cni_plugin": "calico"
    }
  }'
```

**Catalog Parameters:**

- `name`: Unique identifier for the catalog
- `type`: Must be "K8s"
- `description`: Human-readable description
- `capacity`: Total cluster capacity (sum of all nodes)
- `expiry`: Days until cluster expires
- `k8s.crn`: Cloud Resource Name for infrastructure
- `k8s.kubernetes_version`: K8s version (e.g., "1.28", "1.27")
- `k8s.master_count`: Number of master nodes (1 or 3 for HA)
- `k8s.worker_count`: Number of worker nodes
- `k8s.os_image`: CentOS image name
- `k8s.network`: Network name or ID
- `k8s.pod_network_cidr`: Pod network CIDR (default: 10.244.0.0/16)
- `k8s.service_cidr`: Service CIDR (default: 10.96.0.0/12)
- `k8s.cni_plugin`: CNI plugin (calico, flannel, weave)

### Step 2: Provision K8s Cluster

Deploy a cluster from the catalog:

```bash
curl -X POST https://pac-api.example.com/api/v1/services \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "display_name": "My Development Cluster",
    "catalog_name": "k8s-dev-cluster"
  }'
```

### Step 3: Monitor Deployment

Check the deployment status:

```bash
# Get service details
curl -X GET https://pac-api.example.com/api/v1/services/${SERVICE_NAME} \
  -H "Authorization: Bearer ${ACCESS_TOKEN}"
```

**Status States:**
- `provisioning`: Cluster is being created
- `running`: Cluster is ready and healthy
- `failed`: Deployment failed (check message for details)
- `deleting`: Cluster is being deleted

### Step 4: Access the Cluster

Once the cluster is running, retrieve access information:

```bash
# Get service details including access info
SERVICE_INFO=$(curl -X GET https://pac-api.example.com/api/v1/services/${SERVICE_NAME} \
  -H "Authorization: Bearer ${ACCESS_TOKEN}")

# Extract kubeconfig
echo "${SERVICE_INFO}" | jq -r '.status.access_info' > kubeconfig

# Configure kubectl
export KUBECONFIG=./kubeconfig

# Verify cluster access
kubectl get nodes
kubectl get pods --all-namespaces
```

## Configuration Examples

### Development Cluster (Single Master)

```yaml
apiVersion: app.pac.io/v1alpha1
kind: Catalog
metadata:
  name: k8s-dev
spec:
  type: K8s
  description: "Development cluster"
  capacity:
    cpu: "8"
    memory: 16384
  expiry: 7
  k8s:
    kubernetes_version: "1.28"
    master_count: 1
    worker_count: 2
    os_image: "CentOS-Stream-8"
    cni_plugin: "calico"
```

### Production Cluster (HA Masters)

```yaml
apiVersion: app.pac.io/v1alpha1
kind: Catalog
metadata:
  name: k8s-prod
spec:
  type: K8s
  description: "Production HA cluster"
  capacity:
    cpu: "24"
    memory: 49152
  expiry: 30
  k8s:
    kubernetes_version: "1.28"
    master_count: 3
    worker_count: 5
    os_image: "CentOS-Stream-8"
    cni_plugin: "calico"
    pod_network_cidr: "10.244.0.0/16"
    service_cidr: "10.96.0.0/12"
```

## CNI Plugin Selection

### Calico (Recommended)

**Pros:**
- Production-grade network policy support
- High performance
- Excellent documentation
- Active community

**Use Cases:**
- Production workloads
- Multi-tenant environments
- Security-focused deployments

### Flannel

**Pros:**
- Simple and lightweight
- Easy to troubleshoot
- Low resource overhead

**Use Cases:**
- Development environments
- Simple networking requirements
- Learning Kubernetes

### Weave

**Pros:**
- Easy setup
- Built-in encryption
- Good for small clusters

**Use Cases:**
- Small to medium clusters
- Security-conscious deployments
- Quick prototyping

## Cluster Operations

### Scaling Worker Nodes

To add worker nodes, update the catalog and create a new service, or manually provision additional VMs and join them to the cluster.

### Upgrading Kubernetes

1. Create a new catalog with the desired version
2. Deploy a new cluster
3. Migrate workloads
4. Delete the old cluster

### Backup and Restore

**Backup etcd:**
```bash
ETCDCTL_API=3 etcdctl snapshot save snapshot.db \
  --endpoints=https://127.0.0.1:2379 \
  --cacert=/etc/kubernetes/pki/etcd/ca.crt \
  --cert=/etc/kubernetes/pki/etcd/server.crt \
  --key=/etc/kubernetes/pki/etcd/server.key
```

**Restore etcd:**
```bash
ETCDCTL_API=3 etcdctl snapshot restore snapshot.db \
  --data-dir=/var/lib/etcd-restore
```

### Deleting a Cluster

```bash
curl -X DELETE https://pac-api.example.com/api/v1/services/${SERVICE_NAME} \
  -H "Authorization: Bearer ${ACCESS_TOKEN}"
```

## Troubleshooting

### Common Issues

#### 1. Nodes Not Joining Cluster

**Symptoms:**
- Worker nodes stuck in "NotReady" state
- Join command fails

**Solutions:**
```bash
# Check kubelet status
sudo systemctl status kubelet

# Check kubelet logs
sudo journalctl -u kubelet -f

# Verify network connectivity
ping <master-ip>

# Check firewall rules
sudo firewall-cmd --list-all

# Regenerate join token
kubeadm token create --print-join-command
```

#### 2. Pods Not Scheduling

**Symptoms:**
- Pods stuck in "Pending" state
- No nodes available

**Solutions:**
```bash
# Check node status
kubectl get nodes

# Check node resources
kubectl describe node <node-name>

# Check pod events
kubectl describe pod <pod-name>

# Verify CNI plugin
kubectl get pods -n kube-system | grep -E 'calico|flannel|weave'
```

#### 3. API Server Unreachable

**Symptoms:**
- kubectl commands timeout
- Unable to connect to cluster

**Solutions:**
```bash
# Check API server status
sudo systemctl status kube-apiserver

# Check API server logs
sudo journalctl -u kube-apiserver -f

# Verify certificates
sudo kubeadm certs check-expiration

# Check network connectivity
curl -k https://<master-ip>:6443/healthz
```

### Log Locations

- **Kubelet**: `journalctl -u kubelet`
- **Containerd**: `journalctl -u containerd`
- **API Server**: `/var/log/kube-apiserver.log`
- **Controller Manager**: `/var/log/kube-controller-manager.log`
- **Scheduler**: `/var/log/kube-scheduler.log`

## Best Practices

### Security

1. **Enable RBAC**: Always use Role-Based Access Control
2. **Network Policies**: Implement network segmentation
3. **Pod Security**: Use Pod Security Standards
4. **Secrets Management**: Use external secret managers
5. **Regular Updates**: Keep Kubernetes and OS updated

### High Availability

1. **Multiple Masters**: Use 3 masters for production
2. **Load Balancer**: Use external load balancer for API server
3. **etcd Backup**: Regular automated backups
4. **Node Distribution**: Spread nodes across availability zones

### Monitoring

1. **Metrics Server**: Install for resource metrics
2. **Prometheus**: Deploy for comprehensive monitoring
3. **Grafana**: Visualize cluster metrics
4. **Alerting**: Set up alerts for critical issues

### Resource Management

1. **Resource Requests**: Set appropriate requests/limits
2. **Quotas**: Implement namespace quotas
3. **LimitRanges**: Define default limits
4. **Autoscaling**: Use HPA and VPA

## Support and Documentation

- **PAC Documentation**: [Link to PAC docs]
- **Kubernetes Documentation**: https://kubernetes.io/docs/
- **CentOS Documentation**: https://docs.centos.org/
- **Issue Tracker**: [Link to issue tracker]

## Appendix

### A. Firewall Configuration Script

```bash
#!/bin/bash
# Configure firewall for Kubernetes

# Master node
if [ "$NODE_TYPE" == "master" ]; then
  sudo firewall-cmd --permanent --add-port=6443/tcp
  sudo firewall-cmd --permanent --add-port=2379-2380/tcp
  sudo firewall-cmd --permanent --add-port=10250/tcp
  sudo firewall-cmd --permanent --add-port=10251/tcp
  sudo firewall-cmd --permanent --add-port=10252/tcp
  sudo firewall-cmd --permanent --add-port=10255/tcp
fi

# Worker node
if [ "$NODE_TYPE" == "worker" ]; then
  sudo firewall-cmd --permanent --add-port=10250/tcp
  sudo firewall-cmd --permanent --add-port=30000-32767/tcp
fi

sudo firewall-cmd --reload
```

### B. Health Check Script

```bash
#!/bin/bash
# Check cluster health

echo "Checking node status..."
kubectl get nodes

echo "Checking system pods..."
kubectl get pods -n kube-system

echo "Checking component status..."
kubectl get componentstatuses

echo "Checking cluster info..."
kubectl cluster-info
```

### C. Useful kubectl Commands

```bash
# Get cluster information
kubectl cluster-info
kubectl version

# Node management
kubectl get nodes
kubectl describe node <node-name>
kubectl cordon <node-name>
kubectl drain <node-name>
kubectl uncordon <node-name>

# Pod management
kubectl get pods --all-namespaces
kubectl describe pod <pod-name>
kubectl logs <pod-name>
kubectl exec -it <pod-name> -- /bin/bash

# Resource usage
kubectl top nodes
kubectl top pods

# Troubleshooting
kubectl get events --sort-by='.lastTimestamp'
kubectl describe <resource-type> <resource-name>
```

## Changelog

- **v1.0.0** (2026-02-10): Initial K8s deployment implementation
  - Added K8s catalog type
  - Implemented CentOS-based deployment scripts
  - Added support for Calico, Flannel, and Weave CNI plugins
  - Integrated with PAC service controller