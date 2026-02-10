package k8s

import (
	"fmt"
	"strings"

	pac "github.com/IBM/power-access-cloud/api/apis/app/v1alpha1"
)

// ClusterDeployer handles K8s cluster deployment on CentOS
type ClusterDeployer struct {
	Catalog pac.K8sCatalog
	SSHKeys []string
}

// NewClusterDeployer creates a new K8s cluster deployer
func NewClusterDeployer(catalog pac.K8sCatalog, sshKeys []string) *ClusterDeployer {
	return &ClusterDeployer{
		Catalog: catalog,
		SSHKeys: sshKeys,
	}
}

// GenerateMasterSetupScript generates the setup script for master nodes
func (cd *ClusterDeployer) GenerateMasterSetupScript(isMasterInit bool) string {
	script := `#!/bin/bash
set -e

# Update system
echo "Updating system packages..."
sudo yum update -y

# Disable SELinux
echo "Disabling SELinux..."
sudo setenforce 0
sudo sed -i 's/^SELINUX=enforcing$/SELINUX=permissive/' /etc/selinux/config

# Disable swap
echo "Disabling swap..."
sudo swapoff -a
sudo sed -i '/ swap / s/^\(.*\)$/#\1/g' /etc/fstab

# Load kernel modules
echo "Loading kernel modules..."
cat <<EOF | sudo tee /etc/modules-load.d/k8s.conf
overlay
br_netfilter
EOF

sudo modprobe overlay
sudo modprobe br_netfilter

# Configure sysctl parameters
echo "Configuring sysctl parameters..."
cat <<EOF | sudo tee /etc/sysctl.d/k8s.conf
net.bridge.bridge-nf-call-iptables  = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward                 = 1
EOF

sudo sysctl --system

# Install containerd
echo "Installing containerd..."
sudo yum install -y yum-utils device-mapper-persistent-data lvm2
sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
sudo yum install -y containerd.io

# Configure containerd
echo "Configuring containerd..."
sudo mkdir -p /etc/containerd
containerd config default | sudo tee /etc/containerd/config.toml
sudo sed -i 's/SystemdCgroup = false/SystemdCgroup = true/' /etc/containerd/config.toml

sudo systemctl restart containerd
sudo systemctl enable containerd

# Add Kubernetes repository
echo "Adding Kubernetes repository..."
cat <<EOF | sudo tee /etc/yum.repos.d/kubernetes.repo
[kubernetes]
name=Kubernetes
baseurl=https://pkgs.k8s.io/core:/stable:/v` + cd.Catalog.KubernetesVersion + `/rpm/
enabled=1
gpgcheck=1
gpgkey=https://pkgs.k8s.io/core:/stable:/v` + cd.Catalog.KubernetesVersion + `/rpm/repodata/repomd.xml.key
exclude=kubelet kubeadm kubectl cri-tools kubernetes-cni
EOF

# Install Kubernetes components
echo "Installing Kubernetes components..."
sudo yum install -y kubelet kubeadm kubectl --disableexcludes=kubernetes

sudo systemctl enable --now kubelet

# Configure firewall
echo "Configuring firewall..."
sudo firewall-cmd --permanent --add-port=6443/tcp
sudo firewall-cmd --permanent --add-port=2379-2380/tcp
sudo firewall-cmd --permanent --add-port=10250/tcp
sudo firewall-cmd --permanent --add-port=10251/tcp
sudo firewall-cmd --permanent --add-port=10252/tcp
sudo firewall-cmd --permanent --add-port=10255/tcp
sudo firewall-cmd --reload
`

	if isMasterInit {
		podCIDR := cd.Catalog.PodNetworkCIDR
		if podCIDR == "" {
			podCIDR = "10.244.0.0/16"
		}
		serviceCIDR := cd.Catalog.ServiceCIDR
		if serviceCIDR == "" {
			serviceCIDR = "10.96.0.0/12"
		}

		script += fmt.Sprintf(`
# Initialize Kubernetes cluster
echo "Initializing Kubernetes cluster..."
sudo kubeadm init --pod-network-cidr=%s --service-cidr=%s

# Configure kubectl for root user
mkdir -p $HOME/.kube
sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config

# Install CNI plugin
echo "Installing CNI plugin..."
`, podCIDR, serviceCIDR)

		cniPlugin := cd.Catalog.CNIPlugin
		if cniPlugin == "" {
			cniPlugin = "calico"
		}

		switch strings.ToLower(cniPlugin) {
		case "calico":
			script += `kubectl apply -f https://docs.projectcalico.org/manifests/calico.yaml
`
		case "flannel":
			script += `kubectl apply -f https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel.yml
`
		case "weave":
			script += `kubectl apply -f https://github.com/weaveworks/weave/releases/download/v2.8.1/weave-daemonset-k8s.yaml
`
		default:
			script += `kubectl apply -f https://docs.projectcalico.org/manifests/calico.yaml
`
		}

		script += `
# Generate join command for worker nodes
echo "Generating join command..."
kubeadm token create --print-join-command > /tmp/kubeadm-join-command.sh
chmod +x /tmp/kubeadm-join-command.sh

echo "Master node setup complete!"
echo "Join command saved to /tmp/kubeadm-join-command.sh"
`
	}

	script += `
echo "Setup complete!"
`
	return script
}

// GenerateWorkerSetupScript generates the setup script for worker nodes
func (cd *ClusterDeployer) GenerateWorkerSetupScript(joinCommand string) string {
	script := `#!/bin/bash
set -e

# Update system
echo "Updating system packages..."
sudo yum update -y

# Disable SELinux
echo "Disabling SELinux..."
sudo setenforce 0
sudo sed -i 's/^SELINUX=enforcing$/SELINUX=permissive/' /etc/selinux/config

# Disable swap
echo "Disabling swap..."
sudo swapoff -a
sudo sed -i '/ swap / s/^\(.*\)$/#\1/g' /etc/fstab

# Load kernel modules
echo "Loading kernel modules..."
cat <<EOF | sudo tee /etc/modules-load.d/k8s.conf
overlay
br_netfilter
EOF

sudo modprobe overlay
sudo modprobe br_netfilter

# Configure sysctl parameters
echo "Configuring sysctl parameters..."
cat <<EOF | sudo tee /etc/sysctl.d/k8s.conf
net.bridge.bridge-nf-call-iptables  = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward                 = 1
EOF

sudo sysctl --system

# Install containerd
echo "Installing containerd..."
sudo yum install -y yum-utils device-mapper-persistent-data lvm2
sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
sudo yum install -y containerd.io

# Configure containerd
echo "Configuring containerd..."
sudo mkdir -p /etc/containerd
containerd config default | sudo tee /etc/containerd/config.toml
sudo sed -i 's/SystemdCgroup = false/SystemdCgroup = true/' /etc/containerd/config.toml

sudo systemctl restart containerd
sudo systemctl enable containerd

# Add Kubernetes repository
echo "Adding Kubernetes repository..."
cat <<EOF | sudo tee /etc/yum.repos.d/kubernetes.repo
[kubernetes]
name=Kubernetes
baseurl=https://pkgs.k8s.io/core:/stable:/v` + cd.Catalog.KubernetesVersion + `/rpm/
enabled=1
gpgcheck=1
gpgkey=https://pkgs.k8s.io/core:/stable:/v` + cd.Catalog.KubernetesVersion + `/rpm/repodata/repomd.xml.key
exclude=kubelet kubeadm kubectl cri-tools kubernetes-cni
EOF

# Install Kubernetes components
echo "Installing Kubernetes components..."
sudo yum install -y kubelet kubeadm kubectl --disableexcludes=kubernetes

sudo systemctl enable --now kubelet

# Configure firewall
echo "Configuring firewall..."
sudo firewall-cmd --permanent --add-port=10250/tcp
sudo firewall-cmd --permanent --add-port=30000-32767/tcp
sudo firewall-cmd --reload

`

	if joinCommand != "" {
		script += fmt.Sprintf(`
# Join the cluster
echo "Joining the cluster..."
sudo %s

echo "Worker node setup complete!"
`, joinCommand)
	} else {
		script += `
echo "Worker node prepared. Run the join command from master to complete setup."
`
	}

	return script
}

// GetClusterInfo returns information about the cluster configuration
func (cd *ClusterDeployer) GetClusterInfo() map[string]interface{} {
	return map[string]interface{}{
		"kubernetes_version": cd.Catalog.KubernetesVersion,
		"master_count":       cd.Catalog.MasterCount,
		"worker_count":       cd.Catalog.WorkerCount,
		"os_image":           cd.Catalog.OSImage,
		"pod_network_cidr":   cd.Catalog.PodNetworkCIDR,
		"service_cidr":       cd.Catalog.ServiceCIDR,
		"cni_plugin":         cd.Catalog.CNIPlugin,
	}
}

// Made with Bob
