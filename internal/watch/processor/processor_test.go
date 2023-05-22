package processor

import (
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/watch"
)

func TestProcessor(t *testing.T) {
	hostIP := "192.168.0.1"
	clusterDomain := "svc.cluster.local"
	p := newProcessor(hostIP, clusterDomain)
	go p.processMetaEvent()

	mc := MetaCache{p: p}
	crd := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      "ratings",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"selector": map[string]interface{}{
					"app": "ratings",
				},
				"protocol":   "http",
				"targetPort": 8000,
			},
		},
	}
	mc.NewPodEvent(watch.Event{
		Type: watch.Added,
		Object: &v1.Pod{
			Status:     v1.PodStatus{HostIP: hostIP, PodIP: "127.0.0.1"},
			ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
		},
	})
	mc.NewCRDEvent(watch.Added, crd)
	mc.NewPodEvent(watch.Event{
		Type: watch.Added,
		Object: &v1.Pod{
			Status:     v1.PodStatus{HostIP: hostIP, PodIP: "127.0.0.2"},
			ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
		},
	})
	time.Sleep(time.Millisecond * 200)
	p.stop()
}
