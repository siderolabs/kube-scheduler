package bmc

import (
	"fmt"

	goipmi "github.com/pensando/goipmi"
)

// Client is a holder for the IPMIClient.
// SEE https://www.intel.com/content/dam/www/public/us/en/documents/product-briefs/ipmi-second-gen-interface-spec-v2-rev1-1.pdf
type Client struct {
	IPMIClient *goipmi.Client
}

type BMCInfo struct {
	// BMC endpoint.
	Endpoint string `json:"endpoint"`
	// BMC port. Defaults to 623.
	Port *uint32 `json:"port,omitempty"`
	// BMC user value.
	User string `json:"user,omitempty"`
	// BMC password value.
	Pass string `json:"pass,omitempty"`
	// BMC Interface Type. Defaults to lanplus.
	Interface *string `json:"interface,omitempty"`
}

// NewClient creates an ipmi client to use.
func NewClient(bmcInfo *BMCInfo) (*Client, error) {
	if bmcInfo.Port == nil {
		n := uint32(623)
		bmcInfo.Port = &n
	}

	if bmcInfo.Interface == nil {
		s := "lanplus"
		bmcInfo.Interface = &s
	}

	conn := &goipmi.Connection{
		Hostname:  bmcInfo.Endpoint,
		Port:      int(*bmcInfo.Port),
		Username:  bmcInfo.User,
		Password:  bmcInfo.Pass,
		Interface: *bmcInfo.Interface,
	}

	ipmiClient, err := goipmi.NewClient(conn)
	if err != nil {
		return nil, err
	}

	if err = ipmiClient.Open(); err != nil {
		return nil, fmt.Errorf("error opening client: %w", err)
	}

	return &Client{IPMIClient: ipmiClient}, nil
}

// Close the client.
func (c *Client) Close() error {
	return c.IPMIClient.Close()
}

// PowerOn will power on a given machine.
func (c *Client) PowerOn() error {
	return c.IPMIClient.Control(goipmi.ControlPowerUp)
}

// PowerOff will power off a given machine.
func (c *Client) PowerOff() error {
	return c.IPMIClient.Control(goipmi.ControlPowerDown)
}

// IsPoweredOn checks current power state.
func (c *Client) IsPoweredOn() (bool, error) {
	status, err := c.Status()
	if err != nil {
		return false, err
	}

	return status.IsSystemPowerOn(), nil
}

// PowerCycle will power cycle a given machine.
func (c *Client) PowerCycle() error {
	return c.IPMIClient.Control(goipmi.ControlPowerCycle)
}

// Status fetches the chassis status.
func (c *Client) Status() (*goipmi.ChassisStatusResponse, error) {
	req := &goipmi.Request{
		NetworkFunction: goipmi.NetworkFunctionChassis,
		Command:         goipmi.CommandChassisStatus,
		Data:            goipmi.ChassisStatusRequest{},
	}

	res := &goipmi.ChassisStatusResponse{}

	err := c.IPMIClient.Send(req, res)
	if err != nil {
		return nil, err
	}

	return res, nil
}
