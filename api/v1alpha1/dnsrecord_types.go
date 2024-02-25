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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DnsRecordSpec defines the desired state of DnsRecord
type DnsRecordSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// A configures the this DnsRecordSpec to provide an A record dns record.
	A *ASpec `json:"a,omitempty"`

	// CNAME configures the this DnsRecordSpec to provide an CNAME record dns record.
	CNAME *CNAMESpec `json:"cname,omitempty"`
}

// ASpec is configuration for a a record.
type ASpec struct {
	// Domain is the domain of the new cname record.
	Domain string `json:"domain"`

	// IP is where the ip address that the a record should resolve to.
	IP string `json:"ip"`
}

// CNAMESpace is configuration for a cname record.
type CNAMESpec struct {
	// Domain is the domain of the new cname record.
	Domain string `json:"domain"`

	// Target is the domain that the cname should resolve to.
	Target string `json:"target"`
}

// DnsRecordStatus defines the observed state of DnsRecord
type DnsRecordStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// DnsRecord is the Schema for the dnsrecords API
type DnsRecord struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DnsRecordSpec   `json:"spec,omitempty"`
	Status DnsRecordStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DnsRecordList contains a list of DnsRecord
type DnsRecordList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DnsRecord `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DnsRecord{}, &DnsRecordList{})
}
