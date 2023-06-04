package spec

const (
	DNS_BIND_IP      = "169.254.99.1"
	ENVOY_CONTROL_IP = "169.254.99.2"
)

type L7Service struct {
	MetaData struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	} `json:"metadata"`
	Spec struct {
		Protocol   string            `json:"protocol"`
		Selector   map[string]string `json:"selector"`
		TargetPort uint32            `json:"targetPort"`
	} `json:"spec"`
}

type DNSRequest struct {
	Pod2NS        map[string]string
	NS2GwIP       map[string]string
	ServiceList   []string
	ClusterDomain string
}

type LDSInfo struct {
	Name string
	IP   string
	Port uint32
}

type RDSInfo struct {
	Name         string
	VirtualHosts []VirtualHostInfo
}

type VirtualHostInfo struct {
	Name    string
	Domains []string
	Routers [][]byte // protojson
	Cluster string
}

type CDSInfo struct {
	Name      string
	Endpoints []EndpointInfo
}

type EndpointInfo struct {
	IP   string
	Port uint32
}

type RouterRequest struct {
	Pod2NS        map[string]string
	NS2GwIP       map[string]string
	ServiceList   map[string]*L7Service
	ClusterDomain string
	Label2Pod     map[string]map[string]bool
}
