package v1

import "k8s.io/apimachinery/pkg/runtime"

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
