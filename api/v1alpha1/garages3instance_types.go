/*
MIT License

Copyright (c) 2025 LordAntonius

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GarageS3InstanceSpec defines the desired state of GarageS3Instance.
type GarageS3InstanceSpec struct {

	// URL of accessible Garage S3 instance.
	// +kubebuilder:default:="127.0.0.1"
	// +optional
	Url string `json:"url,omitempty"`

	// Port of Garage admin API
	// +kubebuilder:default:=3903
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +optional
	Port int32 `json:"port,omitempty"`

	// Secret where admin token for Garage admin API is stored.
	// Token is found in field token.
	Secret string `json:"adminTokenSecret"`
}

// GarageS3InstanceStatus defines the observed state of GarageS3Instance.
type GarageS3InstanceStatus struct {

	// Status of the Garage S3 instance connection.
	// HTTP-like numeric status code (allows wider range than uint8)
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=65535
	StatusCode int32 `json:"statusCode"`

	// Message providing more details about the connection status.
	Message string `json:"message"`

	// Timestamp of the last status update.
	LastUpdated metav1.Time `json:"lastUpdated"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// GarageS3Instance is the Schema for the garages3instances API.
type GarageS3Instance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GarageS3InstanceSpec   `json:"spec,omitempty"`
	Status GarageS3InstanceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GarageS3InstanceList contains a list of GarageS3Instance.
type GarageS3InstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GarageS3Instance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GarageS3Instance{}, &GarageS3InstanceList{})
}
