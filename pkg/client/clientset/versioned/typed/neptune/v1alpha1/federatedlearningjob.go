// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	"time"

	v1alpha1 "github.com/edgeai-neptune/neptune/pkg/apis/neptune/v1alpha1"
	scheme "github.com/edgeai-neptune/neptune/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// FederatedLearningJobsGetter has a method to return a FederatedLearningJobInterface.
// A group's client should implement this interface.
type FederatedLearningJobsGetter interface {
	FederatedLearningJobs(namespace string) FederatedLearningJobInterface
}

// FederatedLearningJobInterface has methods to work with FederatedLearningJob resources.
type FederatedLearningJobInterface interface {
	Create(ctx context.Context, federatedLearningJob *v1alpha1.FederatedLearningJob, opts v1.CreateOptions) (*v1alpha1.FederatedLearningJob, error)
	Update(ctx context.Context, federatedLearningJob *v1alpha1.FederatedLearningJob, opts v1.UpdateOptions) (*v1alpha1.FederatedLearningJob, error)
	UpdateStatus(ctx context.Context, federatedLearningJob *v1alpha1.FederatedLearningJob, opts v1.UpdateOptions) (*v1alpha1.FederatedLearningJob, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.FederatedLearningJob, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.FederatedLearningJobList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.FederatedLearningJob, err error)
	FederatedLearningJobExpansion
}

// federatedLearningJobs implements FederatedLearningJobInterface
type federatedLearningJobs struct {
	client rest.Interface
	ns     string
}

// newFederatedLearningJobs returns a FederatedLearningJobs
func newFederatedLearningJobs(c *NeptuneV1alpha1Client, namespace string) *federatedLearningJobs {
	return &federatedLearningJobs{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the federatedLearningJob, and returns the corresponding federatedLearningJob object, and an error if there is any.
func (c *federatedLearningJobs) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.FederatedLearningJob, err error) {
	result = &v1alpha1.FederatedLearningJob{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("federatedlearningjobs").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of FederatedLearningJobs that match those selectors.
func (c *federatedLearningJobs) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.FederatedLearningJobList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.FederatedLearningJobList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("federatedlearningjobs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested federatedLearningJobs.
func (c *federatedLearningJobs) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("federatedlearningjobs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a federatedLearningJob and creates it.  Returns the server's representation of the federatedLearningJob, and an error, if there is any.
func (c *federatedLearningJobs) Create(ctx context.Context, federatedLearningJob *v1alpha1.FederatedLearningJob, opts v1.CreateOptions) (result *v1alpha1.FederatedLearningJob, err error) {
	result = &v1alpha1.FederatedLearningJob{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("federatedlearningjobs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(federatedLearningJob).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a federatedLearningJob and updates it. Returns the server's representation of the federatedLearningJob, and an error, if there is any.
func (c *federatedLearningJobs) Update(ctx context.Context, federatedLearningJob *v1alpha1.FederatedLearningJob, opts v1.UpdateOptions) (result *v1alpha1.FederatedLearningJob, err error) {
	result = &v1alpha1.FederatedLearningJob{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("federatedlearningjobs").
		Name(federatedLearningJob.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(federatedLearningJob).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *federatedLearningJobs) UpdateStatus(ctx context.Context, federatedLearningJob *v1alpha1.FederatedLearningJob, opts v1.UpdateOptions) (result *v1alpha1.FederatedLearningJob, err error) {
	result = &v1alpha1.FederatedLearningJob{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("federatedlearningjobs").
		Name(federatedLearningJob.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(federatedLearningJob).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the federatedLearningJob and deletes it. Returns an error if one occurs.
func (c *federatedLearningJobs) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("federatedlearningjobs").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *federatedLearningJobs) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("federatedlearningjobs").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched federatedLearningJob.
func (c *federatedLearningJobs) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.FederatedLearningJob, err error) {
	result = &v1alpha1.FederatedLearningJob{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("federatedlearningjobs").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
