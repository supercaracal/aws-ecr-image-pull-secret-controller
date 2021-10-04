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
	"k8s.io/klog/v2"

	workers "github.com/supercaracal/aws-ecr-image-pull-secret-controller/internal/worker"
)

const (
	informerReSyncDuration = 10 * time.Second
	reconciliationDuration = 10 * time.Second
	controllerName         = "aws-ecr-image-pull-secret-controller"
)

// CustomController is
type CustomController struct {
	builtin *builtinTool
}

type builtinTool struct {
	client  kubernetes.Interface
	factory kubeinformers.SharedInformerFactory
	secret  *secretInfo
}

type secretInfo struct {
	informer cache.SharedIndexInformer
	lister   corelisterv1.SecretLister
}

// NewCustomController is
func NewCustomController(cfg *rest.Config) (*CustomController, error) {
	builtin, err := buildBuiltinTools(cfg)
	if err != nil {
		return nil, err
	}

	return &CustomController{builtin: builtin}, nil
}

// Run is
func (c *CustomController) Run(stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()

	c.builtin.factory.Start(stopCh)

	if ok := cache.WaitForCacheSync(stopCh, c.builtin.secret.informer.HasSynced); !ok {
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

	worker := workers.NewReconciler(c.builtin.client, c.builtin.secret.lister, recorder)

	go wait.Until(worker.Run, reconciliationDuration, stopCh)

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
	secret := info.Core().V1().Secrets()
	s := secretInfo{informer: secret.Informer(), lister: secret.Lister()}

	return &builtinTool{client: cli, factory: info, secret: &s}, nil
}
