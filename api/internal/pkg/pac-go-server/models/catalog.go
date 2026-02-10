package models

type Catalog struct {
	ID                      string        `json:"id"`
	Type                    string        `json:"type"`
	Name                    string        `json:"name"`
	Description             string        `json:"description"`
	Capacity                Capacity      `json:"capacity"`
	Retired                 bool          `json:"retired"`
	Expiry                  int           `json:"expiry"`
	ImageThumbnailReference string        `json:"image_thumbnail_reference"`
	VM                      VM            `json:"vm"`
	K8s                     K8s           `json:"k8s"`
	Status                  CatalogStatus `json:"status"`
}

type CatalogStatus struct {
	Ready   bool   `json:"ready"`
	Message string `json:"message,omitempty"`
}

type VM struct {
	CRN           string   `json:"crn"`
	ProcessorType string   `json:"processor_type"`
	SystemType    string   `json:"system_type"`
	Image         string   `json:"image"`
	Network       string   `json:"network"`
	Capacity      Capacity `json:"capacity"`
}

type K8s struct {
	CRN               string   `json:"crn"`
	KubernetesVersion string   `json:"kubernetes_version"`
	MasterCount       int      `json:"master_count"`
	WorkerCount       int      `json:"worker_count"`
	OSImage           string   `json:"os_image"`
	Network           string   `json:"network"`
	PodNetworkCIDR    string   `json:"pod_network_cidr"`
	ServiceCIDR       string   `json:"service_cidr"`
	CNIPlugin         string   `json:"cni_plugin"`
	Capacity          Capacity `json:"capacity"`
}
