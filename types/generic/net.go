package generic

import "fmt"

type AddressPort struct {
	Address string
	Port int
	Type string
}

func (a AddressPort) String() string {
	return fmt.Sprintf("%s:%d", a.Address, a.Port)
}

func (a AddressPort) Network() string {
	return a.Type
}