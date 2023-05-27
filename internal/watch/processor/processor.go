package processor

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/kindmesh/kindmesh/internal/spec"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
)

type processor struct {
	eventChan chan *metaEvent
	hostIP    string
	dnsInfo   *dnsInfo

	gwMalloctor malloctor
	emitor      emitor

	dnsRequest   *spec.DNSRequest
	routerRequst *spec.RouterRequst

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
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

func newProcessor(hostIP, clusterDomain string, gwMalloctor malloctor, emitor emitor) *processor {
	ctx, cancel := context.WithCancel(context.Background())
	return &processor{
		eventChan: make(chan *metaEvent, 10000),
		hostIP:    hostIP,
		dnsInfo: &dnsInfo{
			pod2ns:        map[string]string{},
			serviceList:   map[string]bool{},
			clusterDomain: clusterDomain,
		},
		gwMalloctor: gwMalloctor,
		emitor:      emitor,
		ctx:         ctx,
		cancel:      cancel,
		wg:          sync.WaitGroup{},
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
		if svc, ok := e.object.(*spec.L7Service); ok {
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
	if err := p.buildDNS(); err != nil {
		log.Printf("build dns error %v\n", err)
		return
	}
	if err := p.buildEnvoy(); err != nil {
		log.Printf("build envoy api error %v\n", err)
		return
	}
	p.emitor(p.dnsRequest, p.routerRequst)
}

func (p *processor) buildDNS() error {
	var serviceList []string
	for k := range p.dnsInfo.serviceList {
		serviceList = append(serviceList, k)
	}

	pod2gw, err := p.rebuildPod2GwIP(p.dnsInfo.pod2ns)
	if err != nil {
		return err
	}

	p.dnsRequest = &spec.DNSRequest{
		Pod2NS:        p.dnsInfo.pod2ns,
		Pod2GwIP:      pod2gw,
		ServiceList:   serviceList,
		ClusterDomain: p.dnsInfo.clusterDomain,
	}
	return nil
}

func (p *processor) buildEnvoy() error {
	p.routerRequst = &spec.RouterRequst{}
	return nil
}

func (p *processor) rebuildPod2GwIP(pod2ns map[string]string) (map[string]string, error) {
	names := make(map[string]bool)
	for _, ns := range pod2ns {
		names[ns] = true
	}
	ns2GwIP, err := p.gwMalloctor.AllocateForNames(names)
	if err != nil {
		return nil, fmt.Errorf("allocate gw ip error %v", err)
	}
	ret := map[string]string{}
	for ip, ns := range pod2ns {
		ret[ip] = ns2GwIP[ns]
	}
	return ret, nil
}

func (p *processor) stop() {
	p.cancel()
	p.wg.Wait()
}
