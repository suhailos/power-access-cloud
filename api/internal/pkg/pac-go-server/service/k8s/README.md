# Kubernetes Cluster Deployment on CentOS

This package provides functionality to deploy Kubernetes clusters on CentOS-based infrastructure.

## Overview

The K8s service implementation allows users to provision production-ready Kubernetes clusters with the following features:

- Multi-master high availability setup
- Configurable worker nodes
- Multiple CNI plugin options (Calico, Flannel, Weave)
- Automated cluster setup and configuration
- CentOS-optimized deployment scripts

## Architecture

### Components

1. **ClusterDeployer**: Generates deployment scripts for master and worker nodes
2. **K8sService**: Manages cluster lifecycle (create, status, delete)
3. **K8sService Controller**: Integrates with the PAC service controller

### Deployment Flow

```
1. User creates K8s catalog with cluster specifications
2. User provisions service from K8s catalog
3. Service controller initiates cluster deployment
4. VMs are provisioned for master and worker nodes
5. Setup scripts are executed on each node:
   - System prerequisites (SELinux, swap, kernel modules)
   - Container runtime (containerd)
   - Kubernetes components (kubelet, kubeadm, kubectl)
6. Master node is initialized with kubeadm
7. CNI plugin is installed
8. Worker nodes join the cluster
9. Cluster status is monitored and reported
10. Access information (kubeconfig) is provided to user
```

## CentOS Setup Steps

The implementation follows these steps based on Kubernetes on CentOS best practices:

### 1. System Prerequisites

```bash
# Disable SELinux
sudo setenforce 0
sudo sed -i 's/^SELINUX=enforcing$/SELINUX=permissive/' /etc/selinux/config

# Disable swap
sudo swapoff -a
sudo sed -i '/ swap / s/^\(.*\)$/#\1/g' /etc/fstab

# Load kernel modules
cat <<EOF | sudo tee /etc/modules-load.d/k8s.conf
overlay
br_netfilter
EOF

sudo modprobe overlay
sudo modprobe br_netfilter

# Configure sysctl
cat <<EOF | sudo tee /etc/sysctl.d/k8s.conf
net.bridge.bridge-nf-call-iptables  = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward                 = 1
EOF

sudo sysctl --system
```

### 2. Install Container Runtime (containerd)

```bash
# Install containerd
sudo yum install -y yum-utils device-mapper-persistent-data lvm2
sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
sudo yum install -y containerd.io

# Configure containerd
sudo mkdir -p /etc/containerd
containerd config default | sudo tee /etc/containerd/config.toml
sudo sed -i 's/SystemdCgroup = false/SystemdCgroup = true/' /etc/containerd/config.toml

sudo systemctl restart containerd
sudo systemctl enable containerd
```

### 3. Install Kubernetes Components

```bash
# Add Kubernetes repository
cat <<EOF | sudo tee /etc/yum.repos.d/kubernetes.repo
[kubernetes]
name=Kubernetes
baseurl=https://pkgs.k8s.io/core:/stable:/v1.28/rpm/
enabled=1
gpgcheck=1
gpgkey=https://pkgs.k8s.io/core:/stable:/v1.28/rpm/repodata/repomd.xml.key
exclude=kubelet kubeadm kubectl cri-tools kubernetes-cni
EOF

# Install Kubernetes
sudo yum install -y kubelet kubeadm kubectl --disableexcludes=kubernetes
sudo systemctl enable --now kubelet
```

### 4. Initialize Master Node

```bash
# Initialize cluster
sudo kubeadm init --pod-network-cidr=10.244.0.0/16 --service-cidr=10.96.0.0/12

# Configure kubectl
mkdir -p $HOME/.kube
sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config

# Install CNI plugin (example: Calico)
kubectl apply -f https://docs.projectcalico.org/manifests/calico.yaml
```

### 5. Join Worker Nodes

```bash
# On master, generate join command
kubeadm token create --print-join-command

# On worker nodes, run the join command
sudo kubeadm join <master-ip>:6443 --token <token> --discovery-token-ca-cert-hash sha256:<hash>
```

## Configuration Options

### K8s Catalog Specification

```yaml
apiVersion: app.pac.io/v1alpha1
kind: Catalog
metadata:
  name: k8s-cluster-small
spec:
  type: K8s
  description: "Small Kubernetes cluster for development"
  capacity:
    cpu: "8"
    memory: 16384
  expiry: 7
  image_thumbnail_reference: "https://example.com/k8s-icon.png"
  k8s:
    crn: "crn:v1:bluemix:public:power-iaas:..."
    kubernetes_version: "1.28"
    master_count: 1
    worker_count: 2
    os_image: "CentOS-Stream-8"
    network: "pac-network"
    pod_network_cidr: "10.244.0.0/16"
    service_cidr: "10.96.0.0/12"
    cni_plugin: "calico"
```

### Supported CNI Plugins

- **Calico**: Default, recommended for production
- **Flannel**: Lightweight, simple overlay network
- **Weave**: Easy to set up, good for small clusters

## API Usage

### Create K8s Catalog

```bash
curl -X POST https://pac-api/api/v1/catalogs \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "k8s-dev-cluster",
    "type": "K8s",
    "description": "Development K8s cluster",
    "capacity": {"cpu": 8, "memory": 16384},
    "expiry": 7,
    "image_thumbnail_reference": "https://example.com/k8s.png",
    "k8s": {
      "crn": "crn:v1:bluemix:public:power-iaas:...",
      "kubernetes_version": "1.28",
      "master_count": 1,
      "worker_count": 2,
      "os_image": "CentOS-Stream-8",
      "cni_plugin": "calico"
    }
  }'
```

### Deploy K8s Cluster

```bash
curl -X POST https://pac-api/api/v1/services \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "display_name": "My K8s Cluster",
    "catalog_name": "k8s-dev-cluster"
  }'
```

### Get Cluster Status

```bash
curl -X GET https://pac-api/api/v1/services/<service-name> \
  -H "Authorization: Bearer <token>"
```

## Cluster Access

Once the cluster is provisioned, users can access it using:

1. **Kubeconfig**: Download from the service access info
2. **API Server**: Direct access to the Kubernetes API
3. **Dashboard**: Web UI for cluster management (if enabled)

Example kubeconfig retrieval:
```bash
# Get service details
SERVICE_INFO=$(curl -X GET https://pac-api/api/v1/services/<service-name> \
  -H "Authorization: Bearer <token>")

# Extract kubeconfig from access_info
echo "$SERVICE_INFO" | jq -r '.status.access_info' > kubeconfig

# Use kubectl
export KUBECONFIG=./kubeconfig
kubectl get nodes
```

## Monitoring and Management

### Cluster Health Checks

The service automatically monitors:
- Node status (Ready/NotReady)
- Control plane health
- Network connectivity
- Pod scheduling capability

### Cluster Operations

- **Scale**: Add/remove worker nodes
- **Upgrade**: Update Kubernetes version
- **Backup**: Etcd backup and restore
- **Delete**: Clean removal of all resources

## Troubleshooting

### Common Issues

1. **Nodes not joining cluster**
   - Check firewall rules
   - Verify network connectivity
   - Check join token validity

2. **Pods not scheduling**
   - Verify CNI plugin installation
   - Check node resources
   - Review pod requirements

3. **API server unreachable**
   - Check master node status
   - Verify certificates
   - Check network configuration

### Logs

```bash
# Check kubelet logs
sudo journalctl -u kubelet -f

# Check container runtime
sudo journalctl -u containerd -f

# Check pod logs
kubectl logs -n kube-system <pod-name>
```

## Security Considerations

- SELinux is disabled for compatibility (can be configured in permissive mode)
- Firewall rules are configured for required ports
- SSH keys are used for node access
- RBAC is enabled by default
- Network policies can be configured with CNI plugins

## Future Enhancements

- [ ] Multi-master HA setup with load balancer
- [ ] Automated backup and restore
- [ ] Cluster autoscaling
- [ ] Monitoring stack integration (Prometheus/Grafana)
- [ ] Logging stack integration (ELK/Loki)
- [ ] Service mesh integration (Istio/Linkerd)
- [ ] GitOps integration (ArgoCD/Flux)

## References

- [Kubernetes Official Documentation](https://kubernetes.io/docs/)
- [CentOS Container Documentation](https://docs.centos.org/en-US/containers/)
- [kubeadm Documentation](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/)
- [CNI Plugin Documentation](https://kubernetes.io/docs/concepts/extend-kubernetes/compute-storage-net/network-plugins/)