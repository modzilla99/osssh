package ports

import "github.com/gophercloud/gophercloud/openstack/networking/v2/ports"

type ListOpts struct {
	Status         string   `q:"status"`
	Name           string   `q:"name"`
	Description    string   `q:"description"`
	AdminStateUp   *bool    `q:"admin_state_up"`
	NetworkID      string   `q:"network_id"`
	TenantID       string   `q:"tenant_id"`
	ProjectID      string   `q:"project_id"`
	DeviceOwner    string   `q:"device_owner"`
	MACAddress     string   `q:"mac_address"`
	ID             string   `q:"id"`
	DeviceID       string   `q:"device_id"`
	Limit          int      `q:"limit"`
	Marker         string   `q:"marker"`
	SortKey        string   `q:"sort_key"`
	SortDir        string   `q:"sort_dir"`
	Tags           string   `q:"tags"`
	TagsAny        string   `q:"tags-any"`
	NotTags        string   `q:"not-tags"`
	NotTagsAny     string   `q:"not-tags-any"`
	SecurityGroups []string `q:"security_groups"`
	FixedIPs       []ports.FixedIPOpts
	HostID         string   `q:"host_id"`
}