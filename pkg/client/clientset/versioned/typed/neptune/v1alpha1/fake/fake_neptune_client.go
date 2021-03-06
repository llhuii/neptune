// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1alpha1 "github.com/edgeai-neptune/neptune/pkg/client/clientset/versioned/typed/neptune/v1alpha1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeNeptuneV1alpha1 struct {
	*testing.Fake
}

func (c *FakeNeptuneV1alpha1) Datasets(namespace string) v1alpha1.DatasetInterface {
	return &FakeDatasets{c, namespace}
}

func (c *FakeNeptuneV1alpha1) FederatedLearningJobs(namespace string) v1alpha1.FederatedLearningJobInterface {
	return &FakeFederatedLearningJobs{c, namespace}
}

func (c *FakeNeptuneV1alpha1) JointInferenceServices(namespace string) v1alpha1.JointInferenceServiceInterface {
	return &FakeJointInferenceServices{c, namespace}
}

func (c *FakeNeptuneV1alpha1) Models(namespace string) v1alpha1.ModelInterface {
	return &FakeModels{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeNeptuneV1alpha1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
