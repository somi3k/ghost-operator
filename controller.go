/*
Copyright 2017 The Kubernetes Authors.

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

package main

import (
	"fmt"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	appslisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"

	v1alpha1 "github.com/somi3k/ghost-operator/pkg/apis/ghostcontroller/v1alpha1"
	clientset "github.com/somi3k/ghost-operator/pkg/generated/clientset/versioned"
	samplescheme "github.com/somi3k/ghost-operator/pkg/generated/clientset/versioned/scheme"
	informers "github.com/somi3k/ghost-operator/pkg/generated/informers/externalversions/ghostcontroller/v1alpha1"
	listers "github.com/somi3k/ghost-operator/pkg/generated/listers/ghostcontroller/v1alpha1"
)

const (
	GHOST_IMAGE = "ghost:2.21"
	GHOST_EXTERNAL_PORT = 32500
	GHOST_VOLUME_SIZE = 1000000000
	GHOST_API_VERSION = "ghostcontroller.somi3k/v1alpha1"
	GHOST_CONTAINER_NAME = "ghost-blog"
	GHOST_CLAIM_NAME = "ghost-blog-content"
	GHOST_VOLUME_NAME = "ghost-blog-persistent-store"
)

const controllerAgentName = "ghost-operator"

const (
	// SuccessSynced is used as part of the Event 'reason' when a Ghost is synced
	SuccessSynced = "Synced"
	// ErrResourceExists is used as part of the Event 'reason' when a Ghost fails
	// to sync due to a Deployment of the same name already existing.
	ErrResourceExists = "ErrResourceExists"

	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to a Deployment already existing
	MessageResourceExists = "Resource %q already exists and is not managed by Ghost"
	// MessageResourceSynced is the message used for an Event fired when a Ghost
	// is synced successfully
	MessageResourceSynced = "Ghost synced successfully"
)


// Controller is the controller implementation for Ghost resources
type GhostController struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// ghostclientset is a clientset for our own API group
	ghostclientset clientset.Interface

	deploymentsLister appslisters.DeploymentLister
	deploymentsSynced cache.InformerSynced
	ghostsLister        listers.GhostLister
	ghostsSynced        cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}


// NewController returns a new ghost controller
func NewController(

	kubeclientset kubernetes.Interface,
	ghostclientset clientset.Interface,
	deploymentInformer appsinformers.DeploymentInformer,
	ghostInformer informers.GhostInformer) *GhostController {

	 utilruntime.HandleError(fmt.Errorf("*************************** NewController() \n"))

	// Create event broadcaster
	// Add ghost-operator types to the default Kubernetes Scheme so Events can be
	// logged for ghost-operator types.
	utilruntime.Must(samplescheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &GhostController{
		kubeclientset:     kubeclientset,
		ghostclientset:   ghostclientset,
		deploymentsLister: deploymentInformer.Lister(),
		deploymentsSynced: deploymentInformer.Informer().HasSynced,
		ghostsLister:        ghostInformer.Lister(),
		ghostsSynced:        ghostInformer.Informer().HasSynced,
		workqueue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Ghosts"),
		recorder:          recorder,
	}

	klog.Info("Setting up event handlers")
	// Set up an event handler for when Ghost resources change
	ghostInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueGhost,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueGhost(new)
		},
	})
	// Set up an event handler for when Deployment resources change. This
	// handler will lookup the owner of the given Deployment, and if it is
	// owned by a Ghost resource will enqueue that Ghost resource for
	// processing. This way, we don't need to implement custom logic for
	// handling Deployment resources. More info on this pattern:
	// https://github.com/kubernetes/community/blob/8cafef897a22026d42f5e5bb3f104febe7e29830/contributors/devel/controllers.md
	deploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newDepl := new.(*appsv1.Deployment)
			oldDepl := old.(*appsv1.Deployment)
			if newDepl.ResourceVersion == oldDepl.ResourceVersion {
				// Periodic resync will send update events for all known Deployments.
				// Two different versions of the same Deployment will always have different RVs.
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})
	return controller
}


// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *GhostController) Run(threadiness int, stopCh <-chan struct{}) error {

	 utilruntime.HandleError(fmt.Errorf("*************************** Run() \n"))

	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting Ghost controller")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.deploymentsSynced, c.ghostsSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting workers")
	// Launch two workers to process Foo resources
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}


// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *GhostController) runWorker() {

	 utilruntime.HandleError(fmt.Errorf("*************************** runWorker() \n"))

	for c.processNextWorkItem() {
	}
}


// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *GhostController) processNextWorkItem() bool {

	 utilruntime.HandleError(fmt.Errorf("*************************** processNextWorkItem() \n"))

	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// Foo resource to be synced.
		if err := c.syncHandler(key); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}
	return true
}


// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Ghost resource
// with the current status of the resource.
func (c *GhostController) syncHandler(key string) error {

	 utilruntime.HandleError(fmt.Errorf("*************************** syncHandler() \n"))

	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the Ghost resource with this namespace/name
	ghost, err := c.ghostsLister.Ghosts(namespace).Get(name)
	if err != nil {
		// The Foo resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("ghost '%s' in work queue no longer exists", key))
			return nil
		}
		return err
	}

	deploymentName := ghost.Name
	if deploymentName == "" {
		// We choose to absorb the error here as the worker would requeue the
		// resource otherwise. Instead, the next time the resource is updated
		// the resource will be queued again.
		utilruntime.HandleError(fmt.Errorf("%s: deployment name must be specified", key))
		return nil
	}

	// Get the deployment with the name specified in Ghost.spec
	deployment, err := c.deploymentsLister.Deployments(ghost.Namespace).Get(deploymentName)
	// If the resource doesn't exist, we'll create it
	if errors.IsNotFound(err) {
		 utilruntime.HandleError(fmt.Errorf("---------syncHandler()----------- Get(deploymentName) isNotFound error\n"))
		//deployment, err = c.kubeclientset.AppsV1().Deployments(ghost.Namespace).Create(newDeployment(ghost))
	} else {
		 utilruntime.HandleError(fmt.Errorf("---------------syncHandler()-------------- deployment.Status.Replicas", deployment.Status.Replicas))
		 utilruntime.HandleError(fmt.Errorf("---------------syncHandler()-------------- deployment.Status.AvailableReplicas", deployment.Status.AvailableReplicas))
		 utilruntime.HandleError(fmt.Errorf("---------------syncHandler()-------------- deployment.Status.Replicas", ghost.Status.AvailableReplicas))
	}

    var hostname string
	if ghost.Status.Url == "" {
		 utilruntime.HandleError(fmt.Errorf("------------- syncHandler() -------- ghost.Status.Url is EMPTY, calling deployGhost() \n"))
		err, deploymentName, hostname, deployment = c.deployGhost(ghost)
	} else {
		hostname = ghost.Status.Url
		 utilruntime.HandleError(fmt.Errorf("--------------- syncHandler() -------- ghost.Status.Url = ", ghost.Status.Url))
	}

	err = c.updateGhostStatus(ghost, deployment, hostname)
	c.recorder.Event(ghost, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)

	if err != nil {
		 utilruntime.HandleError(fmt.Errorf("$$$$$$$$$$$$ updateGhostStatus failed $$$$$$$ syncHandler()"))
		panic(err)
	}

	 utilruntime.HandleError(fmt.Errorf("----------------- syncHandler() --------aft------ ghost.Status.Url = ", ghost.Status.Url))


	// If an error occurs during Get/Create, we'll requeue the item so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		return err
	}

	//If the Deployment is not controlled by this Ghost resource, we should log
	//a warning to the event recorder and ret
	//if !metav1.IsControlledBy(deployment, ghost) {
	//	msg := fmt.Sprintf(MessageResourceExists, deployment.Name)
	//	c.recorder.Event(ghost, corev1.EventTypeWarning, ErrResourceExists, msg)
	//	return fmt.Errorf(msg)
	//}

	// If this number of the replicas on the Ghost resource is specified, and the
	// number does not equal the current desired replicas on the Deployment, we
	// should update the Deployment resource.
//	if ghost.Spec.Replicas != nil && *ghost.Spec.Replicas != *deployment.Spec.Replicas {
//		klog.V(4).Infof("Ghost %s replicas: %d, deployment replicas: %d", name, *ghost.Spec.Replicas, *deployment.Spec.Replicas)
//		deployment, err = c.kubeclientset.AppsV1().Deployments(ghost.Namespace).Update(newDeployment(ghost))
//		deployment, err = c.kubeclientset.AppsV1().Deployments(ghost.Namespace).Update(newDeployment(ghost))
//
//	}

	// If an error occurs during Update, we'll requeue the item so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	//if err != nil {
	//	return err
	//}

	// Finally, we update the status block of the Ghost resource to reflect the
	// current state of the world

	//err = c.updateGhostStatus(ghost, deployment)
	//if err != nil {
	//	return err
	//}

	return nil
}


//func (c *GhostController) updateGhostStatus(ghost *v1alpha1.Ghost, deployment *appsv1.Deployment) error {
func (c *GhostController) updateGhostStatus(ghost *v1alpha1.Ghost, deployment *appsv1.Deployment, hostname string) error {

	 utilruntime.HandleError(fmt.Errorf("*************************** updateGhostStatus() \n"))

	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	ghostCopy := ghost.DeepCopy()
	ghostCopy.Status.AvailableReplicas = deployment.Status.AvailableReplicas
	ghostCopy.Status.Url = hostname

	 utilruntime.HandleError(fmt.Errorf("-------- updateGhostStatus() ----before--- ghostCopy.Status.Url = ", ghostCopy.Status.Url))
	 utilruntime.HandleError(fmt.Errorf("-------- updateGhostStatus() ----before--- ghost.Status.Url = ", ghost.Status.Url))

	//ghostCopy.Status.AvailableReplicas = deployment.Status.AvailableReplicas


	// If the CustomResourceSubresources feature gate is not enabled,
	// we must use Update instead of UpdateStatus to update the Status block of the Ghost resource.
	// UpdateStatus will not allow changes to the Spec of the resource,
	// which is ideal for ensuring nothing other than resource status has been updated.
	_, err := c.ghostclientset.GhostcontrollerV1alpha1().Ghosts(ghost.Namespace).Update(ghostCopy)
	if err != nil {
		 utilruntime.HandleError(fmt.Errorf("$$$$$$$$$$$$$ problem in updateGhostStatus: %e", err))
	}

	 utilruntime.HandleError(fmt.Errorf("-------- updateGhostStatus() ----after----- ghostCopy.Status.Url = ", ghostCopy.Status.Url))
	 utilruntime.HandleError(fmt.Errorf("-------- updateGhostStatus() ----after----- ghost.Status.Url = ", ghost.Status.Url))

	return err
}


// enqueueGhost takes a Ghost resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Ghost.
func (c *GhostController) enqueueGhost(obj interface{}) {

	 utilruntime.HandleError(fmt.Errorf("*************************** enqueueGhost() \n"))

	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}


// handleObject will take any resource implementing metav1.Object and attempt
// to find the Ghost resource that 'owns' it. It does this by looking at the
// objects metadata.ownerReferences field for an appropriate OwnerReference.
// It then enqueues that Ghost resource to be processed. If the object does not
// have an appropriate OwnerReference, it will simply be skipped.
func (c *GhostController) handleObject(obj interface{}) {


	 utilruntime.HandleError(fmt.Errorf("*************************** handleObject() \n"))

	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		klog.V(4).Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}
	klog.V(4).Infof("Processing object: %s", object.GetName())
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		// If this object is not owned by a Ghost, we should not do anything more
		// with it.
		if ownerRef.Kind != "Ghost" {
			return
		}

		ghost, err := c.ghostsLister.Ghosts(object.GetNamespace()).Get(ownerRef.Name)
		if err != nil {
			klog.V(4).Infof("ignoring orphaned object '%s' of ghost '%s'", object.GetSelfLink(), ownerRef.Name)
			return
		}

		c.enqueueGhost(ghost)
		return
	}
}


func (c *GhostController) deployGhost(ghost *v1alpha1.Ghost) (error, string, string, *appsv1.Deployment) {

	 utilruntime.HandleError(fmt.Errorf("*************************** deployGhost() \n"))

	c.createPersistentVolume(ghost)
	c.createPersistentVolumeClaim(ghost)
	c.createService(ghost)

	replica := int32(1)
	err, deployName, hostname, deployment := c.createDeployment(ghost, &replica)

	 utilruntime.HandleError(fmt.Errorf("------------ deployGhost() ------------ deployName = \n", deployName))
	 utilruntime.HandleError(fmt.Errorf("------------ deployGhost() ------------ hostname = \n", hostname))

	if err != nil {
		return err, "", "", nil
	}

	return err, deployName, hostname, deployment
}


// createDeployment creates a new Deployment for a Ghost resource. It also sets
// the appropriate OwnerReferences on the resource so handleObject can discover
// the Ghost resource that 'owns' it.
func (c *GhostController) createDeployment(ghost *v1alpha1.Ghost, replica *int32) (error, string, string, *appsv1.Deployment) {

	 utilruntime.HandleError(fmt.Errorf("*************************** createDeployment() \n"))

	namespace := getNamespace(ghost)

	deploymentName := ghost.Name

	hostname := "http://" + ghost.Spec.MinikubeIP + ":" + fmt.Sprintf("%d", GHOST_EXTERNAL_PORT)

	 utilruntime.HandleError(fmt.Errorf("------------ createDeployment() ----------- hostname = ", hostname))

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: GHOST_API_VERSION,
					Kind:       "Ghost",
					Name:       ghost.Name,
					UID:        ghost.UID,
				},
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: replica,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
				    "app": GHOST_CONTAINER_NAME,
			    },
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": GHOST_CONTAINER_NAME,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  GHOST_CONTAINER_NAME,
							Image: GHOST_IMAGE,
							ImagePullPolicy: "IfNotPresent",
							Env: []corev1.EnvVar{
								{
									Name: "url",
									Value: hostname,
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name: GHOST_CONTAINER_NAME,
									ContainerPort: 2368,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name: GHOST_VOLUME_NAME,
									MountPath: "/var/lib/ghost/content",
								},
							},

						},
					},
					Volumes: []corev1.Volume{
						{
							Name: GHOST_VOLUME_NAME,
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: GHOST_CLAIM_NAME,
								},
							},
						},
					},
				},
			},
		},
	}

	 utilruntime.HandleError(fmt.Errorf("------------ creating deployment ------------ createDeployment()  \n"))

	result, err := c.kubeclientset.AppsV1().Deployments(namespace).Create(deployment)

	if err != nil {
		panic(err)
		return err, "", "", nil
	}

	 utilruntime.HandleError(fmt.Errorf("------------ created deployment %s ------------ createDeployment()  \n", result.GetObjectMeta().GetName()))

	return nil, result.ObjectMeta.GetName(), hostname, deployment
}


func(c *GhostController) createPersistentVolume(ghost *v1alpha1.Ghost) {

	 utilruntime.HandleError(fmt.Errorf("*************************** createPersistentVolume() \n"))

	persistentVolume :=  &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: GHOST_CLAIM_NAME,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: GHOST_API_VERSION,
					Kind:       "Ghost",
					Name:       ghost.Name,
					UID:        ghost.UID,
				},
			},
		},
		Spec: corev1.PersistentVolumeSpec{
			StorageClassName: "standard",
			Capacity: corev1.ResourceList{corev1.ResourceStorage:
				*resource.NewQuantity(GHOST_VOLUME_SIZE, resource.DecimalSI)},
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/mnt/data",
				},
			},
		},
	}
	persistentVolumeClient := c.kubeclientset.CoreV1().PersistentVolumes()

	 utilruntime.HandleError(fmt.Errorf("------------ creating pv ------------ createPersistentVolume()  \n"))

	result, err := persistentVolumeClient.Create(persistentVolume)
	if err != nil {
		panic(err)
	}
	fmt.Printf("------------ created pv %q ------------ createPersistentVolume() \n " , result.GetObjectMeta().GetName())
}


func (c *GhostController) createPersistentVolumeClaim(ghost *v1alpha1.Ghost) {

	 utilruntime.HandleError(fmt.Errorf("*************************** createPersistentVolumeClaim() \n"))

	persistentVolumeClaim := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: GHOST_CLAIM_NAME,
			Labels: map[string]string{"app": GHOST_CONTAINER_NAME},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: GHOST_API_VERSION,
					Kind:       "Ghost",
					Name:       ghost.Name,
					UID:        ghost.UID,
				},
			},
		},
		Spec:  corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceStorage:
				*resource.NewQuantity(1000000000, resource.DecimalSI)}},
		},
	}

	namespace := getNamespace(ghost)
	persistentVolumeClaimClient := c.kubeclientset.CoreV1().PersistentVolumeClaims(namespace)

	 utilruntime.HandleError(fmt.Errorf("------------ creating pvc ------------ createPersistentVolumeClaim()  \n"))
	result, err := persistentVolumeClaimClient.Create(persistentVolumeClaim)
	if err != nil {
		panic(err)
	}
	fmt.Printf("------------ created pv %q ------------ createPersistentVolumeClaim()  \n",
		result.GetObjectMeta().GetName())
}


func (c *GhostController) createService(ghost *v1alpha1.Ghost) {

	 utilruntime.HandleError(fmt.Errorf("*************************** createService() \n"))

	namespace := getNamespace(ghost)
	serviceClient := c.kubeclientset.CoreV1().Services(namespace)
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: GHOST_CONTAINER_NAME,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: GHOST_API_VERSION,
					Kind:       "Ghost",
					Name:       ghost.Name,
					UID:        ghost.UID,
				},
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeNodePort,
			Ports: []corev1.ServicePort{
				{
					Name:       "ghost-port",
					Port:       80,
					TargetPort: intstr.FromInt(2368),
					NodePort:   GHOST_EXTERNAL_PORT,
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector: map[string]string{"app": GHOST_CONTAINER_NAME},
		},
	}

	result1, err1 := serviceClient.Create(service)
	if err1 != nil {
		panic(err1)
	}

	 utilruntime.HandleError(fmt.Errorf("------------ created svc ----------- createService() " + result1.GetObjectMeta().GetName() + "\n"))

	//nodePort1 := result1.Spec.Ports[0].NodePort
	//nodePort := fmt.Sprint(nodePort1)

	// Parse ServiceIP and Port
	serviceIP := result1.Spec.ClusterIP
	fmt.Printf("------------ createService() : serviceIP = ", serviceIP)

	//servicePortInt := result1.Spec.Ports[0].Port
	//servicePort := fmt.Sprint(servicePortInt)

	//serviceURI := GHOST_URL + ":" + servicePort

	//fmt.Printf("------------ createService() : serviceURI = ", GHOST_URL)

}


func getNamespace(ghost *v1alpha1.Ghost) string {

	 utilruntime.HandleError(fmt.Errorf("*************************** getNamespace() \n"))

	namespace := corev1.NamespaceDefault
	if ghost.Namespace != "" {
		namespace = ghost.Namespace
	}
	return namespace
}