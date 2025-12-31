package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto copies all properties of this object into another object of the
// same type that is provided as a pointer.
func (in *GarageS3Instance) DeepCopyInto(out *GarageS3Instance) {
	out.TypeMeta = in.TypeMeta
	out.ObjectMeta = in.ObjectMeta
	out.Spec = GarageS3InstanceSpec{
		Url:              in.Spec.Url,
		Port:             in.Spec.Port,
		AdminTokenSecret: in.Spec.AdminTokenSecret,
	}
}

// DeepCopyObject returns a generically typed copy of an object
func (in *GarageS3Instance) DeepCopyObject() runtime.Object {
	out := GarageS3Instance{}
	in.DeepCopyInto(&out)

	return &out
}

// DeepCopyInto copies the list and its items
func (in *GarageS3InstanceList) DeepCopyInto(out *GarageS3InstanceList) {
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		out.Items = make([]GarageS3Instance, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	} else {
		out.Items = nil
	}
}

// DeepCopyObject returns a generically typed copy of a list object
func (in *GarageS3InstanceList) DeepCopyObject() runtime.Object {
	out := GarageS3InstanceList{}
	in.DeepCopyInto(&out)
	return &out
}

// DeepCopyInto copies all properties of this object into another object of the
// same type that is provided as a pointer.
func (in *GarageS3AccessKey) DeepCopyInto(out *GarageS3AccessKey) {
	out.TypeMeta = in.TypeMeta
	out.ObjectMeta = in.ObjectMeta
	out.Spec = GarageS3AccessKeySpec{
		InstanceRef: GarageS3InstanceRef{
			Name:      in.Spec.InstanceRef.Name,
			Namespace: in.Spec.InstanceRef.Namespace,
		},
		CanCreateBucket: in.Spec.CanCreateBucket,
		Expiration:      in.Spec.Expiration,
		NeverExpires:    in.Spec.NeverExpires,
	}
	// Copy status
	out.Status.Secret = in.Status.Secret
	if in.Status.Conditions != nil {
		out.Status.Conditions = make([]metav1.Condition, len(in.Status.Conditions))
		for i := range in.Status.Conditions {
			out.Status.Conditions[i] = metav1.Condition{
				Type:               in.Status.Conditions[i].Type,
				Status:             in.Status.Conditions[i].Status,
				Reason:             in.Status.Conditions[i].Reason,
				Message:            in.Status.Conditions[i].Message,
				LastTransitionTime: in.Status.Conditions[i].LastTransitionTime,
			}
		}
	} else {
		out.Status.Conditions = nil
	}
}

// DeepCopyObject returns a generically typed copy of an object
func (in *GarageS3AccessKey) DeepCopyObject() runtime.Object {
	out := GarageS3AccessKey{}
	in.DeepCopyInto(&out)

	return &out
}

// DeepCopyInto copies the list and its items
func (in *GarageS3AccessKeyList) DeepCopyInto(out *GarageS3AccessKeyList) {
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		out.Items = make([]GarageS3AccessKey, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	} else {
		out.Items = nil
	}
}

// DeepCopyObject returns a generically typed copy of a list object
func (in *GarageS3AccessKeyList) DeepCopyObject() runtime.Object {
	out := GarageS3AccessKeyList{}
	in.DeepCopyInto(&out)
	return &out
}
