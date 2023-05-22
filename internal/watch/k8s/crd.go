package k8s

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/kindmesh/kindmesh/internal/watch/processor"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
)

func watchCRDs() {
	client, err := dynamic.NewForConfig(getRestConfig())
	if err != nil {
		log.Fatal(err)
	}

	resource := schema.GroupVersionResource{Group: "kindmesh.io", Version: "v1", Resource: "l7services"}
	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(client, time.Minute, corev1.NamespaceAll, nil)
	informer := factory.ForResource(resource).Informer()

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			processor.GlobalCache.NewCRDEvent(watch.Added, obj)
		},
		UpdateFunc: func(oldObj, obj interface{}) {
			processor.GlobalCache.NewCRDEvent(watch.Modified, obj)
		},
		DeleteFunc: func(obj interface{}) {
			processor.GlobalCache.NewCRDEvent(watch.Deleted, obj)
		},
	})

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	informer.Run(ctx.Done())
}
