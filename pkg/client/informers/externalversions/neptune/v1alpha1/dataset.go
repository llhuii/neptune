// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	time "time"

	neptunev1alpha1 "github.com/edgeai-neptune/neptune/pkg/apis/neptune/v1alpha1"
	versioned "github.com/edgeai-neptune/neptune/pkg/client/clientset/versioned"
	internalinterfaces "github.com/edgeai-neptune/neptune/pkg/client/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/edgeai-neptune/neptune/pkg/client/listers/neptune/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// DatasetInformer provides access to a shared informer and lister for
// Datasets.
type DatasetInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.DatasetLister
}

type datasetInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewDatasetInformer constructs a new informer for Dataset type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewDatasetInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredDatasetInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredDatasetInformer constructs a new informer for Dataset type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredDatasetInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.NeptuneV1alpha1().Datasets(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.NeptuneV1alpha1().Datasets(namespace).Watch(context.TODO(), options)
			},
		},
		&neptunev1alpha1.Dataset{},
		resyncPeriod,
		indexers,
	)
}

func (f *datasetInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredDatasetInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *datasetInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&neptunev1alpha1.Dataset{}, f.defaultInformer)
}

func (f *datasetInformer) Lister() v1alpha1.DatasetLister {
	return v1alpha1.NewDatasetLister(f.Informer().GetIndexer())
}
