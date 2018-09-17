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
package v1alpha1

import (
	v1alpha1 "github.com/knative/serving/pkg/apis/networking/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// ClusterIngressLister helps list ClusterIngresses.
type ClusterIngressLister interface {
	// List lists all ClusterIngresses in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.ClusterIngress, err error)
	// ClusterIngresses returns an object that can list and get ClusterIngresses.
	ClusterIngresses(namespace string) ClusterIngressNamespaceLister
	ClusterIngressListerExpansion
}

// clusterIngressLister implements the ClusterIngressLister interface.
type clusterIngressLister struct {
	indexer cache.Indexer
}

// NewClusterIngressLister returns a new ClusterIngressLister.
func NewClusterIngressLister(indexer cache.Indexer) ClusterIngressLister {
	return &clusterIngressLister{indexer: indexer}
}

// List lists all ClusterIngresses in the indexer.
func (s *clusterIngressLister) List(selector labels.Selector) (ret []*v1alpha1.ClusterIngress, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.ClusterIngress))
	})
	return ret, err
}

// ClusterIngresses returns an object that can list and get ClusterIngresses.
func (s *clusterIngressLister) ClusterIngresses(namespace string) ClusterIngressNamespaceLister {
	return clusterIngressNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// ClusterIngressNamespaceLister helps list and get ClusterIngresses.
type ClusterIngressNamespaceLister interface {
	// List lists all ClusterIngresses in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1alpha1.ClusterIngress, err error)
	// Get retrieves the ClusterIngress from the indexer for a given namespace and name.
	Get(name string) (*v1alpha1.ClusterIngress, error)
	ClusterIngressNamespaceListerExpansion
}

// clusterIngressNamespaceLister implements the ClusterIngressNamespaceLister
// interface.
type clusterIngressNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all ClusterIngresses in the indexer for a given namespace.
func (s clusterIngressNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.ClusterIngress, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.ClusterIngress))
	})
	return ret, err
}

// Get retrieves the ClusterIngress from the indexer for a given namespace and name.
func (s clusterIngressNamespaceLister) Get(name string) (*v1alpha1.ClusterIngress, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("clusteringress"), name)
	}
	return obj.(*v1alpha1.ClusterIngress), nil
}
