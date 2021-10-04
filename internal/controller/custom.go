package controller

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	kubescheme "k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corelisterv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	handlers "github.com/supercaracal/kubernetes-controller-template/internal/handler"
	workers "github.com/supercaracal/kubernetes-controller-template/internal/worker"
	customclient "github.com/supercaracal/kubernetes-controller-template/pkg/generated/clientset/versioned"
	customscheme "github.com/supercaracal/kubernetes-controller-template/pkg/generated/clientset/versioned/scheme"
	custominformers "github.com/supercaracal/kubernetes-controller-template/pkg/generated/informers/externalversions"
	customlisterv1 "github.com/supercaracal/kubernetes-controller-template/pkg/generated/listers/supercaracal/v1"
)

const (
	informerReSyncDuration = 10 * time.Second
	cleanupDuration        = 10 * time.Second
	reconcileDuration      = 5 * time.Second
	resourceName           = "FooBars"
	controllerName         = "kubernetes-controller-template"
)

// CustomController is
type CustomController struct {
	builtin   *builtinTool
	custom    *customTool
	workQueue workqueue.RateLimitingInterface
}

type builtinTool struct {
	client  kubernetes.Interface
	factory kubeinformers.SharedInformerFactory
	pod     *podInfo
}

type customTool struct {
	client   customclient.Interface
	factory  custominformers.SharedInformerFactory
	resource *customResourceInfo
}

type podInfo struct {
	informer cache.SharedIndexInformer
	lister   corelisterv1.PodLister
}

type customResourceInfo struct {
	informer cache.SharedIndexInformer
	lister   customlisterv1.FooBarLister
}

// NewCustomController is
func NewCustomController(cfg *rest.Config) (*CustomController, error) {
	if err := customscheme.AddToScheme(kubescheme.Scheme); err != nil {
		return nil, err
	}

	builtin, err := buildBuiltinTools(cfg)
	if err != nil {
		return nil, err
	}

	custom, err := buildCustomResourceTools(cfg)
	if err != nil {
		return nil, err
	}

	wq := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), resourceName)

	h := handlers.NewInformerHandler(wq)
	custom.resource.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    h.OnAdd,
		UpdateFunc: h.OnUpdate,
		DeleteFunc: h.OnDelete,
	})

	return &CustomController{builtin: builtin, custom: custom, workQueue: wq}, nil
}

// Run is
func (c *CustomController) Run(stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workQueue.ShutDown()

	c.builtin.factory.Start(stopCh)
	c.custom.factory.Start(stopCh)

	if ok := cache.WaitForCacheSync(stopCh, c.builtin.pod.informer.HasSynced, c.custom.resource.informer.HasSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	eventBroadcaster := record.NewBroadcaster()
	defer eventBroadcaster.Shutdown()

	if w := eventBroadcaster.StartStructuredLogging(0); w != nil {
		defer w.Stop()
	}

	if w := eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: c.builtin.client.CoreV1().Events("")}); w != nil {
		defer w.Stop()
	}

	recorder := eventBroadcaster.NewRecorder(kubescheme.Scheme, corev1.EventSource{Component: controllerName})

	worker := workers.NewReconciler(
		&workers.ResourceClient{
			Builtin: c.builtin.client,
			Custom:  c.custom.client,
		},
		&workers.ResourceLister{
			Pod:            c.builtin.pod.lister,
			CustomResource: c.custom.resource.lister,
		},
		c.workQueue,
		recorder,
	)

	go wait.Until(worker.Run, reconcileDuration, stopCh)
	go wait.Until(worker.Clean, cleanupDuration, stopCh)

	klog.V(4).Info("Controller is ready")
	<-stopCh
	klog.V(4).Info("Shutting down controller")

	return nil
}

func buildBuiltinTools(cfg *rest.Config) (*builtinTool, error) {
	cli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	info := kubeinformers.NewSharedInformerFactory(cli, informerReSyncDuration)
	pod := info.Core().V1().Pods()
	p := podInfo{informer: pod.Informer(), lister: pod.Lister()}

	return &builtinTool{client: cli, factory: info, pod: &p}, nil
}

func buildCustomResourceTools(cfg *rest.Config) (*customTool, error) {
	cli, err := customclient.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	info := custominformers.NewSharedInformerFactory(cli, informerReSyncDuration)
	cr := info.Supercaracal().V1().FooBars()
	r := customResourceInfo{informer: cr.Informer(), lister: cr.Lister()}

	return &customTool{client: cli, factory: info, resource: &r}, nil
}
