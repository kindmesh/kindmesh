package processor

import (
	"context"
	"fmt"
	"sync"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
)

type processor struct {
	eventChan chan *metaEvent
	hostIP    string
	dnsInfo   *dnsInfo

	ns2GwIP map[string]string

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	buildInterval time.Duration
}

type metaEvent struct {
	eventType watch.EventType
	object    interface{}
}

type dnsInfo struct {
	pod2ns        map[string]string
	serviceList   map[string]bool
	clusterDomain string
}

func newProcessor(hostIP, clusterDomain string) *processor {
	ctx, cancel := context.WithCancel(context.Background())
	return &processor{
		eventChan: make(chan *metaEvent, 10000),
		hostIP:    hostIP,
		dnsInfo: &dnsInfo{
			pod2ns:        map[string]string{},
			serviceList:   map[string]bool{},
			clusterDomain: clusterDomain,
		},
		ns2GwIP:       map[string]string{},
		ctx:           ctx,
		cancel:        cancel,
		wg:            sync.WaitGroup{},
		buildInterval: time.Millisecond * 100,
	}
}

func (p *processor) addEvent(e *metaEvent) {
	p.eventChan <- e
}

func (p *processor) processEvent() {
	hasEvent := false
	for {
		var e *metaEvent
		select {
		case e = <-p.eventChan:
			hasEvent = true
		default:
			if hasEvent {
				return
			}
			select {
			case e = <-p.eventChan:
				hasEvent = true
			case <-p.ctx.Done():
				return
			}
		}
		if svc, ok := e.object.(*L7Service); ok {
			domain := svc.MetaData.Name + "." + svc.MetaData.Namespace + "."
			switch e.eventType {
			case watch.Added:
				p.dnsInfo.serviceList[domain] = true
			case watch.Modified:
				p.dnsInfo.serviceList[domain] = true
			case watch.Deleted:
				delete(p.dnsInfo.serviceList, domain)
			default:
			}
		}
		if pod, ok := e.object.(*v1.Pod); ok {
			if pod.Status.HostIP != p.hostIP {
				continue
			}
			switch e.eventType {
			case watch.Added:
				p.dnsInfo.pod2ns[pod.Status.PodIP] = pod.Namespace
			case watch.Modified:
				p.dnsInfo.pod2ns[pod.Status.PodIP] = pod.Namespace
			case watch.Deleted:
				delete(p.dnsInfo.pod2ns, pod.Status.PodIP)
			default:
			}
		}
	}
}

type DnsRequst struct {
	Pod2NS        map[string]string
	Pod2GwIP      map[string]string
	ServiceList   []string
	ClusterDomain string
}

func (p *processor) processMetaEvent() {
	p.wg.Add(1)
	defer p.wg.Done()
	for {
		p.processEvent()
		select {
		case <-p.ctx.Done():
			return
		default:
		}
		p.buildMeta()
	}
}
func (p *processor) buildMeta() {
	p.buildDNS()
	p.buildEnvoy()
}

func (p *processor) buildDNS() {
	var serviceList []string
	for k := range p.dnsInfo.serviceList {
		serviceList = append(serviceList, k)
	}

	pod2gw := p.rebuildPod2GwIP(p.dnsInfo.pod2ns)

	dnsReq := DnsRequst{
		Pod2NS:        p.dnsInfo.pod2ns,
		Pod2GwIP:      pod2gw,
		ServiceList:   serviceList,
		ClusterDomain: p.dnsInfo.clusterDomain,
	}

	fmt.Println("dns Req is", dnsReq)
}

func (p *processor) buildEnvoy() {
}

func (p *processor) rebuildPod2GwIP(pod2ns map[string]string) map[string]string {
	ret := map[string]string{}
	needDelete := map[string]bool{}
	for ns := range p.ns2GwIP {
		needDelete[ns] = true
	}
	for ip, ns := range pod2ns {
		delete(needDelete, ns)
		if v, ok := p.ns2GwIP[ns]; ok {
			ret[ip] = v
		} else {
			p.allocGwIp(ns)
			ret[ip] = p.ns2GwIP[ns]
		}
	}
	for ns := range needDelete {
		delete(p.ns2GwIP, ns)
	}
	return ret
}

func (p *processor) allocGwIp(ns string) {
	// ip := net.ParseIP("169.254.10.1")
	// mgr := netif.NewNetifManager([]net.IP{ip})
	// if err := mgr.AddDummyDevice("kindmesh"); err != nil {
	// panic(err)
	// }
	p.ns2GwIP[ns] = "gwip"
}

func (p *processor) stop() {
	p.cancel()
	p.wg.Wait()
}
