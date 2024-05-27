package nova

import "time"

type Server struct {
	ID string `json:"id"`
	TenantID string `json:"tenant_id"`
	UserID string `json:"user_id"`
	Name string `json:"name"`
	Updated time.Time `json:"updated"`
	Created time.Time `json:"created"`
	HostID string `json:"hostid"`
	HypervisorHostname string `json:"OS-EXT-SRV-ATTR:hypervisor_hostname"`
	Status string `json:"status"`
	Progress int `json:"progress"`
	AccessIPv4 string `json:"accessIPv4"`
	AccessIPv6 string `json:"accessIPv6"`
	Image map[string]interface{} `json:"-"`
	Flavor map[string]interface{} `json:"flavor"`
	Addresses map[string]interface{} `json:"addresses"`
	Metadata map[string]string `json:"metadata"`
	Links []interface{} `json:"links"`
	KeyName string `json:"key_name"`
	AdminPass string `json:"adminPass"`
	SecurityGroups []map[string]interface{} `json:"security_groups"`
	Tags *[]string `json:"tags"`
	ServerGroups *[]string `json:"server_groups"`
}