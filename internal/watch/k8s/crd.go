package k8s

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
)

func watchCRDs() {
	client, err := dynamic.NewForConfig(getRestConfig())
	if err != nil {
		log.Fatal(err)
	}

	resource := schema.GroupVersionResource{Group: "webapp.tutorial.kubebuilder.io", Version: "v1", Resource: "guestbooks"}
	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(client, time.Minute, corev1.NamespaceAll, nil)
	informer := factory.ForResource(resource).Informer()

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			u := obj.(*unstructured.Unstructured)
			bs, _ := u.MarshalJSON()

			logrus.WithFields(logrus.Fields{
				"name":      u.GetName(),
				"namespace": u.GetNamespace(),
				"labels":    u.GetLabels(),
			}).Infof("received add event! %s", string(bs))
		},
		UpdateFunc: func(oldObj, obj interface{}) {
			u := obj.(*unstructured.Unstructured)
			bs, _ := u.MarshalJSON()
			logrus.Infof("received update event %s!", string(bs))
		},
		DeleteFunc: func(obj interface{}) {
			u := obj.(*unstructured.Unstructured)
			bs, _ := u.MarshalJSON()
			logrus.Infof("received delete event %s!", string(bs))
		},
	})

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	informer.Run(ctx.Done())
}
