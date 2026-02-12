package nova

type Server struct {
	ID                 string                   `json:"id"`
	Name               string                   `json:"name"`
	HypervisorHostname string                   `json:"OS-EXT-SRV-ATTR:hypervisor_hostname"`
}
