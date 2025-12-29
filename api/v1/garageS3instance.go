package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// GarageS3Instance is the Schema for a Garage S3 instance.
type GarageS3Instance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GarageS3InstanceSpec   `json:"spec"`
	Status GarageS3InstanceStatus `json:"status,omitempty"`
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

// GarageS3InstanceStatus represents the observed state of the GarageS3Instance.
type GarageS3InstanceStatus struct {
	Conditions []GarageS3Condition `json:"conditions,omitempty"`
}

// GarageS3Condition describes a condition for the instance.
type GarageS3Condition struct {
	Status             string      `json:"status"`
	Message            string      `json:"message"`
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`
}
