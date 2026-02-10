/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CatalogTypeVM is VM catalog type
const CatalogTypeVM CatalogType = "VM"

// CatalogTypeK8s is K8s catalog type
const CatalogTypeK8s CatalogType = "K8S"

// CatalogTypeAIX is AIX catalog type
const CatalogTypeAIX CatalogType = "AIX"

// CatalogTypeIBMi is IBMi catalog type
const CatalogTypeIBMi CatalogType = "IBMi"

// CatalogFinalizer is Catalog's finalizer
const CatalogFinalizer = "catalogs.pac.io/finalizer"

// CatalogType is type of catalog
// +kubebuilder:validation:Enum="VM";"K8S";"AIX";"IBMi"
type CatalogType string

type Capacity struct {
	CPU    string `json:"cpu"`
	Memory int    `json:"memory"`
}

// CatalogSpec defines the desired state of Catalog
type CatalogSpec struct {
	// +kubebuilder:validation:Required
	Type CatalogType `json:"type"`
	// +kubebuilder:validation:Required
	Description string `json:"description"`
	// +kubebuilder:validation:Required
	Capacity Capacity `json:"capacity"`
	// Retired says whether the Catalog is retired or not, if retired then it will not be available for provisioning
	Retired bool `json:"retired"`
	// +kubebuilder:default=5
	Expiry int `json:"expiry"`
	// Thumbnail reference for image in Catalog which consists of URL for the catalog used by the UI component to display the thumbnail.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^https?:\/\/.+$`
	ImageThumbnailReference string `json:"image_thumbnail_reference"`
	// +optional
	VM VMCatalog `json:"vm,omitempty"`
	// +optional
	K8s K8sCatalog `json:"k8s,omitempty"`
	// +optional
	AIX AIXCatalog `json:"aix,omitempty"`
	// +optional
	IBMi IBMiCatalog `json:"ibmi,omitempty"`
}

// CatalogStatus defines the observed state of Catalog
type CatalogStatus struct {
	Ready   bool   `json:"ready,omitempty"`
	Message string `json:"message,omitempty"`
}

type VMCatalog struct {
	// +kubebuilder:validation:Required
	CRN string `json:"crn"`
	// +kubebuilder:validation:Required
	ProcessorType string `json:"processor_type"`
	// +kubebuilder:validation:Required
	SystemType string `json:"system_type"`
	// +kubebuilder:validation:Required
	Image string `json:"image"`
	// +optional
	Network string `json:"network"`
	// +optional
	Capacity Capacity `json:"capacity"`
}

type K8sCatalog struct {
	// +kubebuilder:validation:Required
	CRN string `json:"crn"`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=standard;openshift
	ClusterType string `json:"cluster_type"`
	// +kubebuilder:validation:Required
	Version string `json:"version"`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10
	WorkerCount int `json:"worker_count"`
	// +kubebuilder:validation:Required
	WorkerFlavor string `json:"worker_flavor"`
	// +kubebuilder:validation:Required
	Zone string `json:"zone"`
	// +optional
	VpcID string `json:"vpc_id,omitempty"`
	// +optional
	SubnetID string `json:"subnet_id,omitempty"`
}

type AIXCatalog struct {
	// +kubebuilder:validation:Required
	CRN string `json:"crn"`
	// +kubebuilder:validation:Required
	ProcessorType string `json:"processor_type"`
	// +kubebuilder:validation:Required
	SystemType string `json:"system_type"`
	// +kubebuilder:validation:Required
	Image string `json:"image"`
	// +optional
	Network string `json:"network"`
	// +optional
	Capacity Capacity `json:"capacity"`
}

type IBMiCatalog struct {
	// +kubebuilder:validation:Required
	CRN string `json:"crn"`
	// +kubebuilder:validation:Required
	ProcessorType string `json:"processor_type"`
	// +kubebuilder:validation:Required
	SystemType string `json:"system_type"`
	// +kubebuilder:validation:Required
	Image string `json:"image"`
	// +optional
	Network string `json:"network"`
	// +optional
	Capacity Capacity `json:"capacity"`
	// +optional
	LicenseRepository string `json:"license_repository,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Catalog is the Schema for the catalogs API
type Catalog struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec CatalogSpec `json:"spec,omitempty"`
	// +optional
	Status CatalogStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CatalogList contains a list of Catalog
type CatalogList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Catalog `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Catalog{}, &CatalogList{})
}
