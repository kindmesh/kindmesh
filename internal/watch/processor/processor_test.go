package processor

import (
	"testing"
	"time"

	"github.com/kindmesh/kindmesh/internal/spec"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/watch"
)

type mocker struct {
	dns    *spec.DNSRequest
	router *spec.RouterRequst
}

func (m *mocker) AllocateForNames(names map[string]bool) (map[string]string, error) {
	ret := map[string]string{}
	for k := range names {
		ret[k] = "127.0.0.1"
	}
	return ret, nil
}
func (m *mocker) Emit(dns *spec.DNSRequest, router *spec.RouterRequst) {
	m.dns = dns
	m.router = router
	// fmt.Printf("dns %v %+v\n", *dns, *router)
}

func TestProcessor(t *testing.T) {
	hostIP := "192.168.0.1"
	clusterDomain := "svc.cluster.local"
	mock := &mocker{}
	p := newProcessor(hostIP, clusterDomain, mock, mock.Emit)
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
			ObjectMeta: metav1.ObjectMeta{Namespace: "default2"},
		},
	})
	time.Sleep(time.Millisecond * 20)
	p.stop()

	assert.Equal(t, "default", mock.dns.Pod2NS["127.0.0.1"])
	assert.Equal(t, "default2", mock.dns.Pod2NS["127.0.0.2"])
	assert.Equal(t, "127.0.0.1", mock.dns.NS2GwIP["default2"])

	assert.Equal(t, "ratings.default.", mock.dns.ServiceList[0])
	assert.Equal(t, "svc.cluster.local", mock.dns.ClusterDomain)
}
