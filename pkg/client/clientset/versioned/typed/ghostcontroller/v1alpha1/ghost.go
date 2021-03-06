/*
Copyright The Kubernetes Authors.

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

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"time"

	v1alpha1 "github.com/somi3k/ghost-operator/pkg/apis/ghostcontroller/v1alpha1"
	scheme "github.com/somi3k/ghost-operator/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// GhostsGetter has a method to return a GhostInterface.
// A group's client should implement this interface.
type GhostsGetter interface {
	Ghosts(namespace string) GhostInterface
}

// GhostInterface has methods to work with Ghost resources.
type GhostInterface interface {
	Create(*v1alpha1.Ghost) (*v1alpha1.Ghost, error)
	Update(*v1alpha1.Ghost) (*v1alpha1.Ghost, error)
	UpdateStatus(*v1alpha1.Ghost) (*v1alpha1.Ghost, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.Ghost, error)
	List(opts v1.ListOptions) (*v1alpha1.GhostList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Ghost, err error)
	GhostExpansion
}

// ghosts implements GhostInterface
type ghosts struct {
	client rest.Interface
	ns     string
}

// newGhosts returns a Ghosts
func newGhosts(c *GhostcontrollerV1alpha1Client, namespace string) *ghosts {
	return &ghosts{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the ghost, and returns the corresponding ghost object, and an error if there is any.
func (c *ghosts) Get(name string, options v1.GetOptions) (result *v1alpha1.Ghost, err error) {
	result = &v1alpha1.Ghost{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("ghosts").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Ghosts that match those selectors.
func (c *ghosts) List(opts v1.ListOptions) (result *v1alpha1.GhostList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.GhostList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("ghosts").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested ghosts.
func (c *ghosts) Watch(opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("ghosts").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch()
}

// Create takes the representation of a ghost and creates it.  Returns the server's representation of the ghost, and an error, if there is any.
func (c *ghosts) Create(ghost *v1alpha1.Ghost) (result *v1alpha1.Ghost, err error) {
	result = &v1alpha1.Ghost{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("ghosts").
		Body(ghost).
		Do().
		Into(result)
	return
}

// Update takes the representation of a ghost and updates it. Returns the server's representation of the ghost, and an error, if there is any.
func (c *ghosts) Update(ghost *v1alpha1.Ghost) (result *v1alpha1.Ghost, err error) {
	result = &v1alpha1.Ghost{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("ghosts").
		Name(ghost.Name).
		Body(ghost).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *ghosts) UpdateStatus(ghost *v1alpha1.Ghost) (result *v1alpha1.Ghost, err error) {
	result = &v1alpha1.Ghost{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("ghosts").
		Name(ghost.Name).
		SubResource("status").
		Body(ghost).
		Do().
		Into(result)
	return
}

// Delete takes name of the ghost and deletes it. Returns an error if one occurs.
func (c *ghosts) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("ghosts").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *ghosts) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	var timeout time.Duration
	if listOptions.TimeoutSeconds != nil {
		timeout = time.Duration(*listOptions.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("ghosts").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Timeout(timeout).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched ghost.
func (c *ghosts) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Ghost, err error) {
	result = &v1alpha1.Ghost{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("ghosts").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
