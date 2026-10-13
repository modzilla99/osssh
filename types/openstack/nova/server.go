package nova

type Server struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Status             string `json:"status"`
	HypervisorHostname string `json:"OS-EXT-SRV-ATTR:hypervisor_hostname"`
}
