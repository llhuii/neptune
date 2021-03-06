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

// ModelInformer provides access to a shared informer and lister for
// Models.
type ModelInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.ModelLister
}

type modelInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewModelInformer constructs a new informer for Model type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewModelInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredModelInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredModelInformer constructs a new informer for Model type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredModelInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.NeptuneV1alpha1().Models(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.NeptuneV1alpha1().Models(namespace).Watch(context.TODO(), options)
			},
		},
		&neptunev1alpha1.Model{},
		resyncPeriod,
		indexers,
	)
}

func (f *modelInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredModelInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *modelInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&neptunev1alpha1.Model{}, f.defaultInformer)
}

func (f *modelInformer) Lister() v1alpha1.ModelLister {
	return v1alpha1.NewModelLister(f.Informer().GetIndexer())
}
