package processor

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
