// +build !ignore_autogenerated

/*
Copyright 2018 The Knative Authors

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

// This file was autogenerated by deepcopy-gen. Do not edit it manually!

package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterIngress) DeepCopyInto(out *ClusterIngress) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterIngress.
func (in *ClusterIngress) DeepCopy() *ClusterIngress {
	if in == nil {
		return nil
	}
	out := new(ClusterIngress)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterIngress) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	} else {
		return nil
	}
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterIngressBackend) DeepCopyInto(out *ClusterIngressBackend) {
	*out = *in
	out.ServicePort = in.ServicePort
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterIngressBackend.
func (in *ClusterIngressBackend) DeepCopy() *ClusterIngressBackend {
	if in == nil {
		return nil
	}
	out := new(ClusterIngressBackend)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterIngressBackendSplit) DeepCopyInto(out *ClusterIngressBackendSplit) {
	*out = *in
	if in.Backend != nil {
		in, out := &in.Backend, &out.Backend
		if *in == nil {
			*out = nil
		} else {
			*out = new(ClusterIngressBackend)
			**out = **in
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterIngressBackendSplit.
func (in *ClusterIngressBackendSplit) DeepCopy() *ClusterIngressBackendSplit {
	if in == nil {
		return nil
	}
	out := new(ClusterIngressBackendSplit)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterIngressList) DeepCopyInto(out *ClusterIngressList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ClusterIngress, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterIngressList.
func (in *ClusterIngressList) DeepCopy() *ClusterIngressList {
	if in == nil {
		return nil
	}
	out := new(ClusterIngressList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterIngressList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	} else {
		return nil
	}
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterIngressRule) DeepCopyInto(out *ClusterIngressRule) {
	*out = *in
	if in.Hosts != nil {
		in, out := &in.Hosts, &out.Hosts
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	in.ClusterIngressRuleValue.DeepCopyInto(&out.ClusterIngressRuleValue)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterIngressRule.
func (in *ClusterIngressRule) DeepCopy() *ClusterIngressRule {
	if in == nil {
		return nil
	}
	out := new(ClusterIngressRule)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterIngressRuleValue) DeepCopyInto(out *ClusterIngressRuleValue) {
	*out = *in
	if in.HTTP != nil {
		in, out := &in.HTTP, &out.HTTP
		if *in == nil {
			*out = nil
		} else {
			*out = new(HTTPClusterIngressRuleValue)
			(*in).DeepCopyInto(*out)
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterIngressRuleValue.
func (in *ClusterIngressRuleValue) DeepCopy() *ClusterIngressRuleValue {
	if in == nil {
		return nil
	}
	out := new(ClusterIngressRuleValue)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterIngressSpec) DeepCopyInto(out *ClusterIngressSpec) {
	*out = *in
	if in.Backend != nil {
		in, out := &in.Backend, &out.Backend
		if *in == nil {
			*out = nil
		} else {
			*out = new(ClusterIngressBackend)
			**out = **in
		}
	}
	if in.TLS != nil {
		in, out := &in.TLS, &out.TLS
		*out = make([]ClusterIngressTLS, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Rules != nil {
		in, out := &in.Rules, &out.Rules
		*out = make([]ClusterIngressRule, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterIngressSpec.
func (in *ClusterIngressSpec) DeepCopy() *ClusterIngressSpec {
	if in == nil {
		return nil
	}
	out := new(ClusterIngressSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterIngressStatus) DeepCopyInto(out *ClusterIngressStatus) {
	*out = *in
	in.LoadBalancer.DeepCopyInto(&out.LoadBalancer)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterIngressStatus.
func (in *ClusterIngressStatus) DeepCopy() *ClusterIngressStatus {
	if in == nil {
		return nil
	}
	out := new(ClusterIngressStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterIngressTLS) DeepCopyInto(out *ClusterIngressTLS) {
	*out = *in
	if in.Hosts != nil {
		in, out := &in.Hosts, &out.Hosts
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterIngressTLS.
func (in *ClusterIngressTLS) DeepCopy() *ClusterIngressTLS {
	if in == nil {
		return nil
	}
	out := new(ClusterIngressTLS)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HTTPClusterIngressPath) DeepCopyInto(out *HTTPClusterIngressPath) {
	*out = *in
	if in.Backends != nil {
		in, out := &in.Backends, &out.Backends
		*out = make([]ClusterIngressBackendSplit, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.AppendHeaders != nil {
		in, out := &in.AppendHeaders, &out.AppendHeaders
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Retries != nil {
		in, out := &in.Retries, &out.Retries
		if *in == nil {
			*out = nil
		} else {
			*out = new(HTTPRetry)
			**out = **in
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HTTPClusterIngressPath.
func (in *HTTPClusterIngressPath) DeepCopy() *HTTPClusterIngressPath {
	if in == nil {
		return nil
	}
	out := new(HTTPClusterIngressPath)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HTTPClusterIngressRuleValue) DeepCopyInto(out *HTTPClusterIngressRuleValue) {
	*out = *in
	if in.Paths != nil {
		in, out := &in.Paths, &out.Paths
		*out = make([]HTTPClusterIngressPath, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HTTPClusterIngressRuleValue.
func (in *HTTPClusterIngressRuleValue) DeepCopy() *HTTPClusterIngressRuleValue {
	if in == nil {
		return nil
	}
	out := new(HTTPClusterIngressRuleValue)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HTTPRetry) DeepCopyInto(out *HTTPRetry) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HTTPRetry.
func (in *HTTPRetry) DeepCopy() *HTTPRetry {
	if in == nil {
		return nil
	}
	out := new(HTTPRetry)
	in.DeepCopyInto(out)
	return out
}
