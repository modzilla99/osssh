package neutron

import "time"

type Port struct {
	// UUID for the port.
	ID string `json:"id"`

	// Network that this port is associated with.
	NetworkID string `json:"network_id"`

	// Human-readable name for the port. Might not be unique.
	Name string `json:"name"`

	// Describes the port.
	Description string `json:"description"`

	// Administrative state of port. If false (down), port does not forward
	// packets.
	AdminStateUp bool `json:"admin_state_up"`

	// Indicates whether network is currently operational. Possible values include
	// `ACTIVE', `DOWN', `BUILD', or `ERROR'. Plug-ins might define additional
	// values.
	Status string `json:"status"`

	// Mac address to use on this port.
	MACAddress string `json:"mac_address"`

	// Specifies IP addresses for the port thus associating the port itself with
	// the subnets where the IP addresses are picked from
	FixedIPs []IP `json:"fixed_ips"`

	// TenantID is the project owner of the port.
	TenantID string `json:"tenant_id"`

	// ProjectID is the project owner of the port.
	ProjectID string `json:"project_id"`

	// Identifies the entity (e.g.: dhcp agent) using this port.
	DeviceOwner string `json:"device_owner"`

	// Specifies the IDs of any security groups associated with a port.
	SecurityGroups []string `json:"security_groups"`

	// Identifies the device (e.g., virtual server) using this port.
	DeviceID string `json:"device_id"`

	// Identifies the list of IP addresses the port will recognize/accept
	AllowedAddressPairs []AddressPair `json:"allowed_address_pairs"`

	// Tags optionally set via extensions/attributestags
	Tags []string `json:"tags"`

	// PropagateUplinkStatus enables/disables propagate uplink status on the port.
	PropagateUplinkStatus bool `json:"propagate_uplink_status"`

	// Extra parameters to include in the request.
	ValueSpecs map[string]string `json:"value_specs"`

	// Show the Binding Hypervisor
	HostID string `json:"binding:host_id"`

	// RevisionNumber optionally set via extensions/standard-attr-revisions
	RevisionNumber int `json:"revision_number"`

	// Timestamp when the port was created
	CreatedAt time.Time `json:"created_at"`

	// Timestamp when the port was last updated
	UpdatedAt time.Time `json:"updated_at"`
}

// IP is a sub-struct that represents an individual IP.
type IP struct {
	SubnetID  string `json:"subnet_id"`
	IPAddress string `json:"ip_address,omitempty"`
}

// AddressPair contains the IP Address and the MAC address.
type AddressPair struct {
	IPAddress  string `json:"ip_address,omitempty"`
	MACAddress string `json:"mac_address,omitempty"`
}