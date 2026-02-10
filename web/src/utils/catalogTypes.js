// Catalog type constants and utilities

export const CATALOG_TYPES = {
  VM: 'VM',
  K8S: 'K8S',
  AIX: 'AIX',
  IBMi: 'IBMi'
};

export const getCatalogTypeLabel = (type) => {
  const labels = {
    VM: 'Virtual Machine',
    K8S: 'Kubernetes Cluster',
    AIX: 'AIX Virtual Machine',
    IBMi: 'IBMi Virtual Machine'
  };
  return labels[type] || type;
};

export const getCatalogTypeIcon = (type) => {
  // Using text badges since we don't have logo images
  const icons = {
    VM: '🖥️ VM',
    K8S: '☸️ K8s',
    AIX: '🔷 AIX',
    IBMi: '🔶 IBMi'
  };
  return icons[type] || '📦';
};

export const getCatalogTypeColor = (type) => {
  const colors = {
    VM: '#0f62fe',      // IBM Blue
    K8S: '#326ce5',     // Kubernetes Blue
    AIX: '#00539a',     // AIX Blue
    IBMi: '#d12765'     // IBMi Pink
  };
  return colors[type] || '#525252';
};

export const getCatalogCapacityLabel = (catalog) => {
  switch (catalog.type) {
    case CATALOG_TYPES.K8S:
      return {
        primary: `${catalog.k8s?.worker_count || 0} Workers`,
        secondary: `${catalog.k8s?.worker_flavor || 'N/A'}`,
        version: catalog.k8s?.version || 'N/A'
      };
    case CATALOG_TYPES.AIX:
    case CATALOG_TYPES.IBMi:
    case CATALOG_TYPES.VM:
    default:
      return {
        primary: `vCPU: ${catalog.capacity?.cpu || 0}`,
        secondary: `Memory: ${catalog.capacity?.memory || 0} GB`,
        version: null
      };
  }
};

export const getServiceAccessInfo = (service, catalog) => {
  const type = catalog?.type || 'VM';
  
  switch (type) {
    case CATALOG_TYPES.K8S:
      if (service.status?.k8s) {
        return {
          type: 'K8s Cluster',
          details: [
            `Cluster ID: ${service.status.k8s.cluster_id || 'N/A'}`,
            `Master URL: ${service.status.k8s.master_url || 'N/A'}`,
            `Ingress: ${service.status.k8s.ingress_hostname || 'N/A'}`,
            `State: ${service.status.k8s.state || 'N/A'}`
          ]
        };
      }
      break;
    
    case CATALOG_TYPES.AIX:
      if (service.status?.aix) {
        return {
          type: 'AIX VM',
          details: [
            `External IP: ${service.status.aix.external_ip_address || 'N/A'}`,
            `Internal IP: ${service.status.aix.ip_address || 'N/A'}`,
            `OS Version: ${service.status.aix.os_version || 'N/A'}`,
            `State: ${service.status.aix.state || 'N/A'}`
          ]
        };
      }
      break;
    
    case CATALOG_TYPES.IBMi:
      if (service.status?.ibmi) {
        return {
          type: 'IBMi VM',
          details: [
            `External IP: ${service.status.ibmi.external_ip_address || 'N/A'}`,
            `Internal IP: ${service.status.ibmi.ip_address || 'N/A'}`,
            `Console: ${service.status.ibmi.console_url || 'N/A'}`,
            `OS Version: ${service.status.ibmi.os_version || 'N/A'}`,
            `State: ${service.status.ibmi.state || 'N/A'}`
          ]
        };
      }
      break;
    
    case CATALOG_TYPES.VM:
    default:
      if (service.status?.vm) {
        return {
          type: 'VM',
          details: [
            `External IP: ${service.status.vm.external_ip_address || 'N/A'}`,
            `Internal IP: ${service.status.vm.ip_address || 'N/A'}`,
            `State: ${service.status.vm.state || 'N/A'}`
          ]
        };
      }
  }
  
  return {
    type: 'Service',
    details: ['No details available']
  };
};

export const formatAccessInfo = (accessInfo) => {
  if (!accessInfo) return 'No access information available';
  
  // If it's already formatted with line breaks, return as is
  if (accessInfo.includes('\n')) {
    return accessInfo;
  }
  
  return accessInfo;
};

/**
 * Format service access information based on catalog type
 * @param {string} accessInfo - Raw access information from service status
 * @param {string} catalogType - Type of catalog (VM, K8S, AIX, IBMI)
 * @returns {string} Formatted access information for display
 */
export const formatServiceAccessInfo = (accessInfo, catalogType) => {
  if (!accessInfo || accessInfo === "") {
    return "...";
  }

  // Handle different catalog types
  switch (catalogType) {
    case CATALOG_TYPES.K8S:
      // For K8s, extract kubeconfig download link or master URL
      if (accessInfo.includes("Kubeconfig:")) {
        const kubeconfigStart = accessInfo.indexOf("Kubeconfig:");
        const kubeconfigEnd = accessInfo.indexOf("\n", kubeconfigStart);
        return accessInfo.slice(kubeconfigStart + 11, kubeconfigEnd > 0 ? kubeconfigEnd : accessInfo.length).trim();
      }
      if (accessInfo.includes("Master URL:")) {
        const urlStart = accessInfo.indexOf("Master URL:");
        const urlEnd = accessInfo.indexOf("\n", urlStart);
        return accessInfo.slice(urlStart + 11, urlEnd > 0 ? urlEnd : accessInfo.length).trim();
      }
      return accessInfo;

    case CATALOG_TYPES.AIX:
      // For AIX, extract IP address
      if (accessInfo.includes("ExternalIP:")) {
        const ipStart = accessInfo.indexOf("ExternalIP:");
        const ipEnd = accessInfo.indexOf("\n", ipStart);
        return accessInfo.slice(ipStart + 11, ipEnd > 0 ? ipEnd : accessInfo.length).trim();
      }
      if (accessInfo.includes("IP:")) {
        const ipStart = accessInfo.indexOf("IP:");
        const ipEnd = accessInfo.indexOf("\n", ipStart);
        return accessInfo.slice(ipStart + 3, ipEnd > 0 ? ipEnd : accessInfo.length).trim();
      }
      return accessInfo;

    case CATALOG_TYPES.IBMi:
      // For IBMi, extract console URL or IP
      if (accessInfo.includes("Console URL:")) {
        const urlStart = accessInfo.indexOf("Console URL:");
        const urlEnd = accessInfo.indexOf("\n", urlStart);
        return accessInfo.slice(urlStart + 12, urlEnd > 0 ? urlEnd : accessInfo.length).trim();
      }
      if (accessInfo.includes("IP:")) {
        const ipStart = accessInfo.indexOf("IP:");
        const ipEnd = accessInfo.indexOf("\n", ipStart);
        return accessInfo.slice(ipStart + 3, ipEnd > 0 ? ipEnd : accessInfo.length).trim();
      }
      return accessInfo;

    case CATALOG_TYPES.VM:
    default:
      // For VM (CentOS), extract IP address (original logic)
      if (accessInfo.includes("ExternalIP:")) {
        const beginning = accessInfo.indexOf("ExternalIP:");
        const end = accessInfo.indexOf("use any ");
        if (end > beginning) {
          return accessInfo.slice(beginning + 12, end).trim();
        }
        const lineEnd = accessInfo.indexOf("\n", beginning);
        return accessInfo.slice(beginning + 12, lineEnd > 0 ? lineEnd : accessInfo.length).trim();
      }
      return accessInfo;
  }
};

export const isCatalogType = (type) => {
  return Object.values(CATALOG_TYPES).includes(type);
};

// Made with Bob
