package handler

import (
	"fmt"

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

// InformerHandler is
type InformerHandler struct {
	wq workqueue.RateLimitingInterface
}

// NewInformerHandler is
func NewInformerHandler(wq workqueue.RateLimitingInterface) *InformerHandler {
	return &InformerHandler{wq: wq}
}

// OnAdd is
func (h *InformerHandler) OnAdd(obj interface{}) {
	h.tryToHandleObject(obj, "Added")
}

// OnUpdate is
func (h *InformerHandler) OnUpdate(before, after interface{}) {
	if diff := cmp.Diff(before, after); diff != "" {
		klog.V(4).Infof("\n%s", diff)
		h.tryToHandleObject(after, "Updated")
	}
}

// OnDelete is
func (h *InformerHandler) OnDelete(obj interface{}) {
	h.tryToHandleObject(obj, "Deleted")
}

func (h *InformerHandler) tryToHandleObject(obj interface{}, event string) {
	if err := h.handleObject(obj, event); err != nil {
		utilruntime.HandleError(err)
	}
}

func (h *InformerHandler) handleObject(obj interface{}, event string) error {
	var object metav1.Object
	var ok bool

	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			return fmt.Errorf("error decoding object, invalid type")
		}

		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			return fmt.Errorf("error decoding object tombstone, invalid type")
		}

		klog.V(4).Infof("Recovered deleted object %s/%s from tombstone", object.GetNamespace(), object.GetName())
	}

	klog.V(4).Infof("%s object %s/%s", event, object.GetNamespace(), object.GetName())
	if event == "Deleted" {
		return nil
	}

	klog.V(4).Infof("Enqueue object %s/%s to work queue", object.GetNamespace(), object.GetName())
	return h.enqueueCustomResource(object)
}

func (h *InformerHandler) enqueueCustomResource(obj interface{}) error {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		return err
	}

	h.wq.Add(key)
	return nil
}
