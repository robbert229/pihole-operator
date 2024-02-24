/*
Copyright 2024.

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PiHoleSpec defines the desired state of PiHole
type PiHoleSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Replicas specifies how many instances should be deployed.
	//
	// Default: 1
	// +optional
	Replicas *int32 `json:"replicas"`

	// Resources configures the resource requirements for the pihole instances.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources"`

	// +optional
	DNS PiHoleSpecDNS `json:"dns"`

	// DNSRecordSelector is used to configure DNSRecord discovery.
	DNSRecordSelector *metav1.LabelSelector `json:"dnsRecordSelector"`

	// DNSRecordSelectorNamespace contains the namespaces to match for
	// DNSRecord discovery. An empty label selector matches all namespaces.
	// A null label selector matches the current namespace only.
	DNSRecordNamespaceSelector *metav1.LabelSelector `json:"dnsRecordNamespaceSelector,omitempty"`

	// Paused when true, configures the controller to take no actions.
	Paused bool `json:"paused,omitempty"`

	// Version of pihole being deployed. The operator uses this information
	// to generate the PiHole ReplicaSet + configuration files.
	//
	// If not specified, the operator assumes the latest upstream version of
	// PiHole available at the time when the version of the operator was
	// released.
	//
	// +optional
	Version *string `json:"version,omitempty"`
}

// PiHoleSpecDNS contains the dns settings for the pi hole instance.
type PiHoleSpecDNS struct {
	UpstreamServers []string `json:"upstreamServers"`
}

// PiHoleStatus defines the observed state of PiHole
type PiHoleStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// PiHole is the Schema for the piholes API
type PiHole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PiHoleSpec   `json:"spec,omitempty"`
	Status PiHoleStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PiHoleList contains a list of PiHole
type PiHoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PiHole `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PiHole{}, &PiHoleList{})
}
