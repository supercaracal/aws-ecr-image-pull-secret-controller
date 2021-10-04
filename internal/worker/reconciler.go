package worker

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	corelisterv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	customapiv1 "github.com/supercaracal/kubernetes-controller-template/pkg/apis/supercaracal/v1"
	customclient "github.com/supercaracal/kubernetes-controller-template/pkg/generated/clientset/versioned"
	customlisterv1 "github.com/supercaracal/kubernetes-controller-template/pkg/generated/listers/supercaracal/v1"
)

// Reconciler is
type Reconciler struct {
	client    *ResourceClient
	lister    *ResourceLister
	workQueue workqueue.RateLimitingInterface
	recorder  record.EventRecorder
}

// ResourceClient is
type ResourceClient struct {
	Builtin kubernetes.Interface
	Custom  customclient.Interface
}

// ResourceLister is
type ResourceLister struct {
	Pod            corelisterv1.PodLister
	CustomResource customlisterv1.FooBarLister
}

const (
	childLifetime = 3 * time.Minute
)

var (
	customGroup = customapiv1.SchemeGroupVersion.WithKind("FooBar")
	delOpts     = metav1.DeleteOptions{PropagationPolicy: func(s metav1.DeletionPropagation) *metav1.DeletionPropagation { return &s }(metav1.DeletePropagationBackground)}
	updOpts     = metav1.UpdateOptions{}
	creOpts     = metav1.CreateOptions{}
	trueP       = func(b bool) *bool { return &b }(true)
)

// NewReconciler is
func NewReconciler(
	cli *ResourceClient,
	list *ResourceLister,
	wq workqueue.RateLimitingInterface,
	rec record.EventRecorder,
) *Reconciler {

	return &Reconciler{client: cli, lister: list, workQueue: wq, recorder: rec}
}

// Run is
func (r *Reconciler) Run() {
	for r.processNextWorkItem() {
	}
}

func (r *Reconciler) processNextWorkItem() bool {
	obj, shutdown := r.workQueue.Get()
	if shutdown {
		return false
	}

	if err := r.sync(obj); err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

func (r *Reconciler) sync(obj interface{}) error {
	defer r.workQueue.Done(obj)

	var key string
	var ok bool

	if key, ok = obj.(string); !ok {
		r.workQueue.Forget(obj)
		return fmt.Errorf("expected string in workqueue but got %#v", obj)
	}

	if err := r.create(key); err != nil {
		r.workQueue.AddRateLimited(key)
		return fmt.Errorf("error syncing '%s': %w, requeuing", key, err)
	}

	r.workQueue.Forget(obj)
	return nil
}

func (r *Reconciler) create(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return fmt.Errorf("invalid resource key: %s: %w", key, err)
	}

	parent, err := r.lister.CustomResource.FooBars(namespace).Get(name)
	if err != nil {
		if kubeerrors.IsNotFound(err) {
			return fmt.Errorf("custom resource '%s' in work queue no longer exists: %w", key, err)
		}

		return err
	}

	klog.V(4).Infof("Dequeued object %s successfully from work queue", key)
	child, err := r.createChildPod(parent)
	if err != nil {
		return err
	}
	r.recorder.Eventf(parent, corev1.EventTypeNormal, "SuccessfulCreate", "Created pod %s/%s", child.Namespace, child.Name)
	klog.V(4).Infof("Created pod %s/%s successfully", child.Namespace, child.Name)

	return r.update(parent)
}

func (r *Reconciler) update(obj *customapiv1.FooBar) (err error) {
	cpy := obj.DeepCopy()
	cpy.Status.Succeeded = true
	_, err = r.client.Custom.SupercaracalV1().FooBars(obj.Namespace).Update(context.TODO(), cpy, updOpts)
	return
}

func (r *Reconciler) createChildPod(parent *customapiv1.FooBar) (*corev1.Pod, error) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:            fmt.Sprintf("%s-%d", parent.Name, time.Now().UnixMicro()),
			Namespace:       parent.Namespace,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(parent, customGroup)},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				corev1.Container{
					Name:            "main",
					Image:           "gcr.io/distroless/static-debian11:debug-amd64",
					Command:         []string{"echo"},
					Args:            []string{parent.Spec.Message},
					SecurityContext: &corev1.SecurityContext{ReadOnlyRootFilesystem: trueP},
				},
			},
		},
	}

	return r.client.Builtin.CoreV1().Pods(parent.Namespace).Create(context.TODO(), pod, creOpts)
}

// Clean is
func (r *Reconciler) Clean() {
	parents, err := r.lister.CustomResource.List(labels.Everything())
	if err != nil {
		if !kubeerrors.IsNotFound(err) {
			utilruntime.HandleError(err)
		}
		return
	}

	allPods, err := r.lister.Pod.List(labels.Everything())
	if err != nil {
		if !kubeerrors.IsNotFound(err) {
			utilruntime.HandleError(err)
		}
		return
	}

	baseTime := metav1.NewTime(time.Now().Add(-childLifetime))

	var parent *customapiv1.FooBar
	for _, pod := range allPods {
		parent = findParent(parents, pod)
		if parent == nil {
			continue
		}

		if pod.Status.Phase != corev1.PodSucceeded {
			continue
		}

		if baseTime.Before(pod.Status.StartTime) {
			continue
		}

		if err := r.client.Builtin.CoreV1().Pods(parent.Namespace).Delete(context.TODO(), pod.Name, delOpts); err != nil {
			utilruntime.HandleError(err)
			continue
		}

		r.recorder.Eventf(parent, corev1.EventTypeNormal, "SuccessfulDelete", "Deleted pod %s/%s", pod.Namespace, pod.Name)
		klog.V(4).Infof("Deleted pod %s/%s successfully", pod.Namespace, pod.Name)
	}
}

func findParent(parents []*customapiv1.FooBar, pod *corev1.Pod) *customapiv1.FooBar {
	for _, parent := range parents {
		if metav1.IsControlledBy(pod, parent) {
			return parent
		}
	}

	return nil
}
