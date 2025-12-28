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
