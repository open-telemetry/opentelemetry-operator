//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Exporter) DeepCopyInto(out *Exporter) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Exporter.
func (in *Exporter) DeepCopy() *Exporter {
	if in == nil {
		return nil
	}
	out := new(Exporter)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Instrumentation) DeepCopyInto(out *Instrumentation) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Instrumentation.
func (in *Instrumentation) DeepCopy() *Instrumentation {
	if in == nil {
		return nil
	}
	out := new(Instrumentation)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Instrumentation) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InstrumentationList) DeepCopyInto(out *InstrumentationList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Instrumentation, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InstrumentationList.
func (in *InstrumentationList) DeepCopy() *InstrumentationList {
	if in == nil {
		return nil
	}
	out := new(InstrumentationList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *InstrumentationList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InstrumentationSpec) DeepCopyInto(out *InstrumentationSpec) {
	*out = *in
	out.Exporter = in.Exporter
	out.Java = in.Java
	out.NodeJS = in.NodeJS
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InstrumentationSpec.
func (in *InstrumentationSpec) DeepCopy() *InstrumentationSpec {
	if in == nil {
		return nil
	}
	out := new(InstrumentationSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InstrumentationStatus) DeepCopyInto(out *InstrumentationStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InstrumentationStatus.
func (in *InstrumentationStatus) DeepCopy() *InstrumentationStatus {
	if in == nil {
		return nil
	}
	out := new(InstrumentationStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JavaSpec) DeepCopyInto(out *JavaSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JavaSpec.
func (in *JavaSpec) DeepCopy() *JavaSpec {
	if in == nil {
		return nil
	}
	out := new(JavaSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NodeJSSpec) DeepCopyInto(out *NodeJSSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NodeJSSpec.
func (in *NodeJSSpec) DeepCopy() *NodeJSSpec {
	if in == nil {
		return nil
	}
	out := new(NodeJSSpec)
	in.DeepCopyInto(out)
	return out
}
