# User Guide: Deploying Applications on Your K8s Cluster

## What You Get After K8s Cluster Deployment

When your Kubernetes cluster is successfully deployed, you receive:

### 1. **Cluster Access Information**

In the service details, you'll see:

```
✅ Cluster is Ready!

Cluster ID: k8s-cluster-1707584123
Kubernetes Version: v1.28.0
Nodes: 3/3 ready
API Server: https://10.0.1.10:6443
Dashboard: https://10.0.1.10:30443/dashboard
Master Node IPs: 10.0.1.10

[Download Kubeconfig Button]
```

### 2. **Kubeconfig File**

Click the "Download Kubeconfig" button to get your cluster configuration file. This file contains:

- Cluster connection details
- API server endpoint
- Authentication certificates (placeholders with instructions)
- Context configuration

### 3. **SSH Access to Nodes**

You can SSH into any cluster node using the SSH keys you registered:

```bash
ssh root@<master-ip>
```

## How to Start Deploying Your Applications

### Step 1: Set Up kubectl

1. **Download and install kubectl** (if not already installed):

```bash
# Linux
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
chmod +x kubectl
sudo mv kubectl /usr/local/bin/

# macOS
brew install kubectl

# Windows
choco install kubernetes-cli
```

2. **Configure kubectl with your kubeconfig**:

```bash
# Download kubeconfig from PAC UI
# Save it as kubeconfig.yaml

# Set KUBECONFIG environment variable
export KUBECONFIG=./kubeconfig.yaml

# Or copy to default location
mkdir -p ~/.kube
cp kubeconfig.yaml ~/.kube/config
```

3. **Get certificates from master node** (one-time setup):

```bash
# SSH to master node
ssh root@<master-ip>

# Get CA certificate
sudo cat /etc/kubernetes/pki/ca.crt | base64 -w 0

# Get admin client certificate
sudo cat /etc/kubernetes/pki/admin.crt | base64 -w 0

# Get admin client key
sudo cat /etc/kubernetes/pki/admin.key | base64 -w 0
```

4. **Update kubeconfig with certificates**:

Edit your kubeconfig.yaml and replace:
- `<CERTIFICATE_AUTHORITY_DATA>` with CA cert
- `<CLIENT_CERTIFICATE_DATA>` with admin cert
- `<CLIENT_KEY_DATA>` with admin key

5. **Verify connection**:

```bash
kubectl get nodes
kubectl cluster-info
kubectl get pods --all-namespaces
```

### Step 2: Deploy Your First Application

#### Example 1: Deploy NGINX Web Server

```bash
# Create deployment
kubectl create deployment nginx --image=nginx:latest

# Expose as service
kubectl expose deployment nginx --port=80 --type=NodePort

# Get service details
kubectl get svc nginx

# Access the application
curl http://<node-ip>:<node-port>
```

#### Example 2: Deploy a Custom Application

Create a deployment file `my-app.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  replicas: 3
  selector:
    matchLabels:
      app: my-app
  template:
    metadata:
      labels:
        app: my-app
    spec:
      containers:
      - name: my-app
        image: your-registry/your-app:latest
        ports:
        - containerPort: 8080
---
apiVersion: v1
kind: Service
metadata:
  name: my-app-service
spec:
  selector:
    app: my-app
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: LoadBalancer
```

Deploy it:

```bash
kubectl apply -f my-app.yaml
kubectl get deployments
kubectl get pods
kubectl get services
```

#### Example 3: Deploy with Helm

```bash
# Install Helm
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# Add Helm repository
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update

# Deploy WordPress
helm install my-wordpress bitnami/wordpress

# Get WordPress URL
kubectl get svc my-wordpress
```

### Step 3: Access Your Applications

#### NodePort Services

```bash
# Get node IP and port
kubectl get nodes -o wide
kubectl get svc <service-name>

# Access application
curl http://<node-ip>:<node-port>
```

#### LoadBalancer Services

```bash
# Get external IP
kubectl get svc <service-name>

# Access application
curl http://<external-ip>
```

#### Port Forwarding (for testing)

```bash
# Forward local port to pod
kubectl port-forward pod/<pod-name> 8080:80

# Access at localhost
curl http://localhost:8080
```

### Step 4: Monitor Your Applications

```bash
# View pod logs
kubectl logs <pod-name>

# Follow logs in real-time
kubectl logs -f <pod-name>

# Describe pod for details
kubectl describe pod <pod-name>

# Get pod metrics
kubectl top pods

# Get node metrics
kubectl top nodes
```

### Step 5: Scale Your Applications

```bash
# Scale deployment
kubectl scale deployment <deployment-name> --replicas=5

# Autoscale based on CPU
kubectl autoscale deployment <deployment-name> --min=2 --max=10 --cpu-percent=80

# Check autoscaler status
kubectl get hpa
```

## Common Use Cases

### 1. Microservices Application

Deploy a multi-tier application:

```yaml
# Database
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: postgres
spec:
  serviceName: postgres
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: postgres:14
        env:
        - name: POSTGRES_PASSWORD
          value: mysecretpassword
        ports:
        - containerPort: 5432
---
# Backend API
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: api
  template:
    metadata:
      labels:
        app: api
    spec:
      containers:
      - name: api
        image: your-api:latest
        ports:
        - containerPort: 3000
---
# Frontend
apiVersion: apps/v1
kind: Deployment
metadata:
  name: frontend
spec:
  replicas: 2
  selector:
    matchLabels:
      app: frontend
  template:
    metadata:
      labels:
        app: frontend
    spec:
      containers:
      - name: frontend
        image: your-frontend:latest
        ports:
        - containerPort: 80
```

### 2. CI/CD Pipeline

Deploy Jenkins:

```bash
helm repo add jenkins https://charts.jenkins.io
helm install jenkins jenkins/jenkins

# Get admin password
kubectl exec --namespace default -it svc/jenkins -c jenkins -- /bin/cat /run/secrets/additional/chart-admin-password

# Access Jenkins
kubectl port-forward svc/jenkins 8080:8080
```

### 3. Monitoring Stack

Deploy Prometheus and Grafana:

```bash
# Add Prometheus Helm repo
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

# Install Prometheus
helm install prometheus prometheus-community/kube-prometheus-stack

# Access Grafana
kubectl port-forward svc/prometheus-grafana 3000:80

# Default credentials: admin/prom-operator
```

### 4. Logging Stack

Deploy ELK Stack:

```bash
# Add Elastic Helm repo
helm repo add elastic https://helm.elastic.co
helm repo update

# Install Elasticsearch
helm install elasticsearch elastic/elasticsearch

# Install Kibana
helm install kibana elastic/kibana

# Install Filebeat
helm install filebeat elastic/filebeat
```

## Best Practices

### 1. Use Namespaces

```bash
# Create namespace
kubectl create namespace my-app

# Deploy to namespace
kubectl apply -f my-app.yaml -n my-app

# Set default namespace
kubectl config set-context --current --namespace=my-app
```

### 2. Use ConfigMaps and Secrets

```bash
# Create ConfigMap
kubectl create configmap app-config --from-file=config.json

# Create Secret
kubectl create secret generic app-secret --from-literal=password=mysecret

# Use in deployment
spec:
  containers:
  - name: app
    envFrom:
    - configMapRef:
        name: app-config
    - secretRef:
        name: app-secret
```

### 3. Set Resource Limits

```yaml
spec:
  containers:
  - name: app
    resources:
      requests:
        memory: "64Mi"
        cpu: "250m"
      limits:
        memory: "128Mi"
        cpu: "500m"
```

### 4. Use Health Checks

```yaml
spec:
  containers:
  - name: app
    livenessProbe:
      httpGet:
        path: /health
        port: 8080
      initialDelaySeconds: 30
      periodSeconds: 10
    readinessProbe:
      httpGet:
        path: /ready
        port: 8080
      initialDelaySeconds: 5
      periodSeconds: 5
```

### 5. Use Persistent Storage

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: app-storage
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
---
spec:
  containers:
  - name: app
    volumeMounts:
    - name: storage
      mountPath: /data
  volumes:
  - name: storage
    persistentVolumeClaim:
      claimName: app-storage
```

## Troubleshooting

### Pod Not Starting

```bash
# Check pod status
kubectl get pods
kubectl describe pod <pod-name>

# Check logs
kubectl logs <pod-name>

# Check events
kubectl get events --sort-by='.lastTimestamp'
```

### Service Not Accessible

```bash
# Check service
kubectl get svc
kubectl describe svc <service-name>

# Check endpoints
kubectl get endpoints <service-name>

# Test from within cluster
kubectl run test --image=busybox -it --rm -- wget -O- http://<service-name>
```

### Resource Issues

```bash
# Check node resources
kubectl top nodes
kubectl describe nodes

# Check pod resources
kubectl top pods
kubectl describe pod <pod-name>
```

## Additional Resources

- **Kubernetes Documentation**: https://kubernetes.io/docs/
- **kubectl Cheat Sheet**: https://kubernetes.io/docs/reference/kubectl/cheatsheet/
- **Helm Charts**: https://artifacthub.io/
- **Kubernetes Patterns**: https://k8spatterns.io/

## Support

If you encounter issues:
1. Check the service details in PAC UI for cluster status
2. Review cluster logs on master node
3. Contact your administrator
4. Submit feedback through PAC platform

---

**Congratulations!** You now have a fully functional Kubernetes cluster ready for deploying your applications. Start small, experiment, and scale as needed!