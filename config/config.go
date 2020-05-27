package config

import (
	"fmt"
	"github.com/crabkun/crab/compress"
	"github.com/crabkun/crab/crypto"
)

type BaseConfig struct {
	Mode     string `json:"mode"`
	LogLevel string `json:"log_level"`
}

func (c *BaseConfig) Validate() error {
	if c.Mode == "" {
		return fmt.Errorf("mode empty")
	}
	if c.LogLevel == "" {
		return fmt.Errorf("log level (log_level) empty")
	}
	return nil
}

type MasterConfig struct {
	ListenAt  string `json:"listen_at"`
	MasterKey string `json:"master_key"`
}

func (c *MasterConfig) Validate() error {
	if c.ListenAt == "" {
		return fmt.Errorf("listen address (listen_at) empty")
	}
	if c.MasterKey == "" {
		return fmt.Errorf("master key (master_key) empty")
	}
	return nil
}

type ServerConfig struct {
	Master    string        `json:"master"`
	MasterKey string        `json:"master_key"`
	Ports     []*PortConfig `json:"ports"`
}

func (c *ServerConfig) Validate() error {
	if c.Master == "" {
		return fmt.Errorf("master address (master) empty")
	}
	if c.MasterKey == "" {
		return fmt.Errorf("master key (master_key) empty")
	}
	if len(c.Ports) == 0 {
		return fmt.Errorf("ports empty")
	}

	for i, v := range c.Ports {
		pe := v.Validate()
		if pe != nil {
			return fmt.Errorf("port (at pos %d) validate failed :%s", i, pe.Error())
		}
	}
	return nil
}

func (c *ServerConfig) GetPort(portKey string) (*PortConfig, bool) {
	// todo optimize
	for i, v := range c.Ports {
		if v.PortKey == portKey {
			return c.Ports[i], true
		}
	}
	return nil, false
}

type PortConfig struct {
	Mark           string `json:"mark"`
	LocalAddress   string `json:"local_address"`
	PortKey        string `json:"port_key"`
	EncryptMethod  string `json:"encrypt_method"`
	CompressMethod string `json:"compress_method"`
}

func (c *PortConfig) Validate() error {
	if c.LocalAddress == "" {
		return fmt.Errorf("local address (local_address) empty")
	}
	if c.PortKey == "" {
		return fmt.Errorf("port key (port_key) empty")
	}
	if c.EncryptMethod == "" {
		return fmt.Errorf("encrypt method (encrypt_method) empty")
	}
	if c.CompressMethod == "" {
		return fmt.Errorf("compress method (compress_method) empty")
	}
	if _, _, err := crypto.GetCrypto(c.EncryptMethod); err != nil {
		return err
	}
	if _, _, err := compress.GetCompress(c.CompressMethod); err != nil {
		return err
	}
	return nil
}

type ClientConfig struct {
	Master string        `json:"master"`
	Ports  []*PortConfig `json:"ports"`
}

func (c *ClientConfig) Validate() error {
	if c.Master == "" {
		return fmt.Errorf("master address (master) empty")
	}
	if len(c.Ports) == 0 {
		return fmt.Errorf("ports empty")
	}

	for i, v := range c.Ports {
		pe := v.Validate()
		if pe != nil {
			return fmt.Errorf("port (at pos %d) validate failed :%s", i, pe.Error())
		}
	}
	return nil
}
