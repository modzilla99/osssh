package neutron

type Port struct {
	// UUID for the port.
	ID string `json:"id"`

	// Network that this port is associated with.
	NetworkID string `json:"network_id"`

	// Specifies IP addresses for the port thus associating the port itself with
	// the subnets where the IP addresses are picked from
	FixedIPs []IP `json:"fixed_ips"`

	// Identifies the device (e.g., virtual server) using this port.
	DeviceID string `json:"device_id"`

	// Show the Binding Hypervisor
	HostID string `json:"binding:host_id"`
}

// IP is a sub-struct that represents an individual IP.
type IP struct {
	SubnetID  string `json:"subnet_id"`
	IPAddress string `json:"ip_address,omitempty"`
}
