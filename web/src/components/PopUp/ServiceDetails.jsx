import React from "react";
import { Modal, Button } from "@carbon/react";
import { useNavigate } from "react-router-dom";
import { Download } from "@carbon/icons-react";
import UserService from "../../services/UserService";

const ServiceDetails = ({pagename, selectRows, setActionProps }) => {
    let navigate = useNavigate();
    
    const service = selectRows[0];
    const isK8sCluster = service.status.k8s_cluster && service.status.k8s_cluster.cluster_id;
    const isClusterReady = service.status.state === "CREATED";
    
    const downloadKubeconfig = () => {
      const token = UserService.getToken();
      const apiUrl = process.env.REACT_APP_API_URL || 'http://localhost:8080';
      const url = `${apiUrl}/api/v1/services/${service.name}/kubeconfig`;
      
      fetch(url, {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${token}`
        }
      })
      .then(response => {
        if (!response.ok) {
          throw new Error('Failed to download kubeconfig');
        }
        return response.blob();
      })
      .then(blob => {
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `${service.name}-kubeconfig.yaml`;
        document.body.appendChild(a);
        a.click();
        window.URL.revokeObjectURL(url);
        document.body.removeChild(a);
      })
      .catch(error => {
        console.error('Error downloading kubeconfig:', error);
        alert('Failed to download kubeconfig. Please try again.');
      });
    };
    
    const renderAccessInfo = () => {
      if (service.status.access_info === "...") {
        return "Under process";
      }
      
      if (isK8sCluster) {
        // K8s cluster access info
        return (
          <div style={{whiteSpace: 'pre-line'}}>
            {service.status.access_info}
          </div>
        );
      } else {
        // VM access info
        return `Access your VM at the external IP address: ${service.status.access_info}. Use the SSH public key you added before you deployed this service.`;
      }
    };
    
    const renderK8sClusterDetails = () => {
      if (!isK8sCluster) return null;
      
      const cluster = service.status.k8s_cluster;
      return (
        <>
          <p><strong>Cluster ID</strong>: {cluster.cluster_id}</p>
          <p><strong>Kubernetes Version</strong>: {cluster.kube_version || 'N/A'}</p>
          <p><strong>Nodes</strong>: {cluster.ready_nodes}/{cluster.total_nodes} ready</p>
          {cluster.api_server_url && <p><strong>API Server</strong>: {cluster.api_server_url}</p>}
          {cluster.dashboard_url && <p><strong>Dashboard</strong>: {cluster.dashboard_url}</p>}
          {cluster.master_ips && cluster.master_ips.length > 0 && (
            <p><strong>Master Node IPs</strong>: {cluster.master_ips.join(', ')}</p>
          )}
          {isClusterReady && (
            <div style={{marginTop: '1rem', padding: '1rem', backgroundColor: '#f4f4f4', borderRadius: '4px'}}>
              <p style={{marginBottom: '0.5rem'}}><strong>Ready to deploy your applications!</strong></p>
              <Button
                kind="primary"
                size="sm"
                renderIcon={Download}
                onClick={downloadKubeconfig}
                style={{marginTop: '0.5rem'}}
              >
                Download Kubeconfig
              </Button>
              <p style={{marginTop: '0.5rem', fontSize: '0.875rem', color: '#525252'}}>
                Download the kubeconfig file to access your cluster with kubectl and start deploying your applications.
              </p>
            </div>
          )}
        </>
      );
    };

   
  return (
    <Modal
      modalHeading="Service details"
      onRequestClose={() => {
        setActionProps("");
      }}
      onRequestSubmit={() => {
        setActionProps("");
        navigate(pagename);
      }}
      open={true}
      primaryButtonText={"OK"}
      secondaryButtonText={"Cancel"}

    >
      
      {/* <p><strong>ID</strong>: {service.id}</p>
        <p><strong>Name</strong>: {service.name}</p> */}
        <p><strong>Display name</strong>: {service.display_name}</p>
      
      <p><strong>Catalog name</strong>: {service.catalog_name}</p>
        <p><strong>Expiry</strong>: {service.expiry.split("T")[0]}</p>
      
        {renderK8sClusterDetails()}
        
        <p><strong>Access information</strong>: {renderAccessInfo()}</p>
        {service.status.state==="PENDING EXTENSION"&&<p><strong>Requested extension date</strong>: {service.status.extentiondate.split('T')[0]}<br/><br/><strong>Justification for extending</strong>: {service.status.justification}</p>}
        
        <p><strong>Status</strong>: {service.status.state}</p>
        {service.status.message && <p><strong>Message</strong>: {service.status.message}</p>}
    </Modal>
  );
};

export default ServiceDetails;
