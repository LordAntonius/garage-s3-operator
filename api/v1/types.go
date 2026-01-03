package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

/* *************************************
   GarageS3Instance API Schema and types
   *************************************/

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
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

/* **************************************
   GarageS3AccessKey API Schema and types
   **************************************/

// GarageS3AccessKey is the Schema for a Garage S3 access key.
type GarageS3AccessKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GarageS3AccessKeySpec   `json:"spec"`
	Status GarageS3AccessKeyStatus `json:"status,omitempty"`
}

// GarageS3AccessKeyList contains a list of GarageS3AccessKey
type GarageS3AccessKeyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GarageS3AccessKey `json:"items"`
}

type GarageS3AccessKeySpec struct {
	InstanceRef     GarageS3InstanceRef `json:"instanceRef"`
	CanCreateBucket bool                `json:"canCreateBucket,omitempty"`
	Expiration      string              `json:"expiration,omitempty"`
	NeverExpires    bool                `json:"neverExpires,omitempty"`
}

// GarageS3InstanceRef references a GarageS3Instance by name and namespace.
type GarageS3InstanceRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// GarageS3AccessKeyStatus represents the observed state of the GarageS3AccessKey.
type GarageS3AccessKeyStatus struct {
	Secret     string             `json:"secret,omitempty"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

/* **************************************
   GarageS3Bucket API Schema and types
 **************************************/

// GarageS3Bucket is the Schema for an S3 Bucket related to a Garage S3 Instance.
type GarageS3Bucket struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GarageS3BucketSpec   `json:"spec"`
	Status GarageS3BucketStatus `json:"status,omitempty"`
}

// GarageS3BucketList contains a list of GarageS3Bucket
type GarageS3BucketList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GarageS3Bucket `json:"items"`
}

// GarageS3BucketSpec describes the desired state of a Bucket.
type GarageS3BucketSpec struct {
	// Reference to the GarageS3Instance (name + namespace) this Bucket belongs to.
	InstanceRef GarageS3InstanceRef `json:"instanceRef"`

	// Website access configuration for the bucket
	WebsiteAccess *GarageS3WebsiteAccess `json:"websiteAccess,omitempty"`

	// Optional quota in buckets or bytes to apply to this bucket
	Quota *GarageS3BucketQuota `json:"quota,omitempty"`

	// List of additional aliases associated with this bucket
	AdditionalAliases []string `json:"additionalAliases,omitempty"`

	// List of permissions to apply to this bucket
	Permissions []GarageS3BucketPermission `json:"permissions,omitempty"`
}

// GarageS3BucketQuota describes optional quota limits
type GarageS3BucketQuota struct {
	MaxObjects *int64 `json:"maxObjects,omitempty"`
	MaxBytes   *int64 `json:"maxBytes,omitempty"`
}

// GarageS3WebsiteAccess describes website access configuration for a bucket
type GarageS3WebsiteAccess struct {
	// Enabled controls whether the bucket is configured as a website
	Enabled bool `json:"enabled"`
	// IndexDocument is the name of the index document (e.g., index.html)
	IndexDocument string `json:"indexDocument,omitempty"`
	// ErrorDocument is the name of the error document (e.g., error.html)
	ErrorDocument string `json:"errorDocument,omitempty"`
}

// GarageS3BucketPermission represents an ACL/permission to grant on the bucket
type GarageS3BucketPermission struct {
	// Name of the GarageS3AccessKey to which to apply the permission
	AccessKeyName string `json:"accessKeyName"`

	// Grant read permission
	Read bool `json:"read,omitempty"`

	// Grant write permission
	Write bool `json:"write,omitempty"`

	// Grant owner permission
	Owner bool `json:"owner,omitempty"`
}

// GarageS3BucketStatus represents the observed state of the Bucket
type GarageS3BucketStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}
