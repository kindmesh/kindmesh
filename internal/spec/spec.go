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
	Pod2GwIP      map[string]string
	ServiceList   []string
	ClusterDomain string
}

type RouterRequst struct {
}
