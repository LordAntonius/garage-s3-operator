package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type GarageS3Instance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec GarageS3InstanceSpec `json:"spec"`
}

// GarageS3InstanceList contains a list of GarageS3Instance
type GarageS3InstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GarageS3Instance `json:"items"`
}

type GarageS3InstanceSpec struct {
	Url              string `json:"url"`
	Port             int    `json:"port"`
	AdminTokenSecret string `json:"adminTokenSecret"`
}
