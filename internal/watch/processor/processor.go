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

	services       map[string]*spec.L7Service
	ns2GwIP        map[string]string
	label2Pod      map[string]map[string]bool
	clusterDomain  string
	pod2ns         map[string]string
	currHostPod2ns map[string]string

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

type serviceInfo struct {
	name      string
	namespace string
}

func newProcessor(hostIP, clusterDomain string, gwMalloctor malloctor, emitor emitor) *processor {
	ctx, cancel := context.WithCancel(context.Background())
	return &processor{
		eventChan:      make(chan *metaEvent, 10000),
		hostIP:         hostIP,
		pod2ns:         map[string]string{},
		currHostPod2ns: map[string]string{},
		clusterDomain:  clusterDomain,
		services:       make(map[string]*spec.L7Service),
		label2Pod:      map[string]map[string]bool{},
		gwMalloctor:    gwMalloctor,
		emitor:         emitor,
		ctx:            ctx,
		cancel:         cancel,
		wg:             sync.WaitGroup{},
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
			key := svc.MetaData.Name + "." + svc.MetaData.Namespace
			switch e.eventType {
			case watch.Added, watch.Modified:
				p.services[key] = svc
			case watch.Deleted:
				delete(p.services, key)
			default:
			}
		}
		if pod, ok := e.object.(*v1.Pod); ok {
			// pod label index
			for k, v := range pod.Labels {
				label := fmt.Sprintf("%s:%s", k, v)
				vv, ok := p.label2Pod[label]
				if e.eventType == watch.Deleted {
					if ok {
						delete(vv, pod.Status.PodIP)
					}
				} else {
					if !ok {
						p.label2Pod[label] = map[string]bool{}
					}
					p.label2Pod[label][pod.Status.PodIP] = true
				}
			}
			switch e.eventType {
			case watch.Added, watch.Modified:
				p.pod2ns[pod.Status.PodIP] = pod.Namespace
				if pod.Status.HostIP == p.hostIP {
					p.currHostPod2ns[pod.Status.PodIP] = pod.Namespace
				}
			case watch.Deleted:
				delete(p.pod2ns, pod.Status.PodIP)
				if pod.Status.HostIP == p.hostIP {
					delete(p.currHostPod2ns, pod.Status.PodIP)
				}
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

	if err := p.ensureNS2GwIP(p.currHostPod2ns); err != nil {
		log.Printf("ensureNS2GwIP error %v\n", err)
		return
	}

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
	for _, svc := range p.services {
		serviceList = append(serviceList, svc.MetaData.Name+"."+svc.MetaData.Namespace+".")
	}

	p.dnsRequest = &spec.DNSRequest{
		Pod2NS:        p.currHostPod2ns,
		NS2GwIP:       p.ns2GwIP,
		ServiceList:   serviceList,
		ClusterDomain: p.clusterDomain,
	}
	return nil
}

func (p *processor) buildEnvoy() error {
	req := &spec.RouterRequst{}
	lds := []spec.LDSInfo{}
	for ns, gwIP := range p.ns2GwIP {
		lds = append(lds, spec.LDSInfo{Name: ns, IP: gwIP, Port: 80})
	}
	req.LDS = lds

	rds := []spec.RDSInfo{}
	for ns := range p.ns2GwIP {
		vhosts := []spec.VirtualHostInfo{}
		for _, svc := range p.services {
			name := svc.MetaData.Name + "." + svc.MetaData.Namespace
			// current service + all service.ns + all service.ns.cluster.domain
			domains := []string{name, name + "." + p.clusterDomain}
			if ns == svc.MetaData.Namespace {
				domains = append(domains, svc.MetaData.Name)
			}
			vhosts = append(vhosts, spec.VirtualHostInfo{
				Name:    name,
				Domains: domains,
				Routers: nil, // TODO: from json
				Cluster: svc.MetaData.Namespace + "_" + svc.MetaData.Name,
			})
		}
		rds = append(rds, spec.RDSInfo{Name: ns, VirtualHosts: vhosts})
	}
	req.RDS = rds

	cds := []spec.CDSInfo{}
	for _, svc := range p.services {
		name := svc.MetaData.Namespace + "_" + svc.MetaData.Name
		// TODO: check has subset
		// selector
		ipSet := map[string]bool{}
		isFrist := true
		for k, v := range svc.Spec.Selector {
			isFrist = true
			ipList, ok := p.label2Pod[fmt.Sprintf("%s:%s", k, v)]
			if !ok {
				ipSet = map[string]bool{}
				break
			}
			// first label
			if isFrist {
				for ip := range ipList {
					// double check ip exists
					if _, ok := p.pod2ns[ip]; !ok {
						continue
					}
					ipSet[ip] = true
				}
				continue
			}
			// not first label: try intersection
			for ip := range ipList {
				if !ipSet[ip] {
					delete(ipSet, ip)
				}
			}
		}
		endpoints := []spec.EndpointInfo{}
		for ip := range ipSet {
			if ip == "" {
				fmt.Println("ip is empty?", ipSet)
				continue
			}
			endpoints = append(endpoints, spec.EndpointInfo{IP: ip, Port: svc.Spec.TargetPort})
		}
		cds = append(cds, spec.CDSInfo{
			Name:      name,
			Endpoints: endpoints,
		})
	}
	req.CDS = cds

	p.routerRequst = req
	return nil
}

func (p *processor) ensureNS2GwIP(pod2ns map[string]string) error {
	names := make(map[string]bool)
	for _, ns := range pod2ns {
		names[ns] = true
	}
	ns2GwIP, err := p.gwMalloctor.AllocateForNames(names)
	if err != nil {
		return fmt.Errorf("allocate gw ip error %v", err)
	}
	p.ns2GwIP = ns2GwIP
	return nil
}

func (p *processor) stop() {
	p.cancel()
	p.wg.Wait()
}
