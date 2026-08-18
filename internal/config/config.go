package config

import (
	"fmt"
	"strings"

	"github.com/crabkun/crab/internal/compress"
	"github.com/crabkun/crab/internal/crypto"
)

const (
	ModeMaster = "master"
	ModeServer = "server"
	ModeClient = "client"
)

// File is the on-disk JSON document. Field names are unchanged so existing
// config files keep working.
type File struct {
	Mode      string `mapstructure:"mode" json:"mode"`
	LogLevel  string `mapstructure:"log_level" json:"log_level"`
	ListenAt  string `mapstructure:"listen_at" json:"listen_at"`
	Master    string `mapstructure:"master" json:"master"`
	MasterKey string `mapstructure:"master_key" json:"master_key"`
	Ports     []Port `mapstructure:"ports" json:"ports"`
}

type Port struct {
	Mark           string `mapstructure:"mark" json:"mark"`
	LocalAddress   string `mapstructure:"local_address" json:"local_address"`
	PortKey        string `mapstructure:"port_key" json:"port_key"`
	EncryptMethod  string `mapstructure:"encrypt_method" json:"encrypt_method"`
	CompressMethod string `mapstructure:"compress_method" json:"compress_method"`
}

type Config struct {
	Mode     string
	LogLevel string
	Master   Master
	Server   Server
	Client   Client
}

type Master struct {
	ListenAt  string
	MasterKey string
}

type Server struct {
	Master    string
	MasterKey string
	Ports     []Port
	portIndex map[string]Port
}

type Client struct {
	Master string
	Ports  []Port
}

func (s Server) Port(portKey string) (Port, bool) {
	p, ok := s.portIndex[portKey]
	return p, ok
}

func (f File) Validate() (*Config, error) {
	if strings.TrimSpace(f.Mode) == "" {
		return nil, fmt.Errorf("mode empty")
	}
	if strings.TrimSpace(f.LogLevel) == "" {
		return nil, fmt.Errorf("log level (log_level) empty")
	}

	cfg := &Config{
		Mode:     strings.ToLower(strings.TrimSpace(f.Mode)),
		LogLevel: strings.ToLower(strings.TrimSpace(f.LogLevel)),
	}

	switch cfg.Mode {
	case ModeMaster:
		if f.ListenAt == "" {
			return nil, fmt.Errorf("listen address (listen_at) empty")
		}
		if f.MasterKey == "" {
			return nil, fmt.Errorf("master key (master_key) empty")
		}
		cfg.Master = Master{ListenAt: f.ListenAt, MasterKey: f.MasterKey}
	case ModeServer:
		if f.Master == "" {
			return nil, fmt.Errorf("master address (master) empty")
		}
		if f.MasterKey == "" {
			return nil, fmt.Errorf("master key (master_key) empty")
		}
		if len(f.Ports) == 0 {
			return nil, fmt.Errorf("ports empty")
		}
		index, err := indexPorts(f.Ports)
		if err != nil {
			return nil, err
		}
		cfg.Server = Server{
			Master:    f.Master,
			MasterKey: f.MasterKey,
			Ports:     f.Ports,
			portIndex: index,
		}
	case ModeClient:
		if f.Master == "" {
			return nil, fmt.Errorf("master address (master) empty")
		}
		if len(f.Ports) == 0 {
			return nil, fmt.Errorf("ports empty")
		}
		if _, err := indexPorts(f.Ports); err != nil {
			return nil, err
		}
		cfg.Client = Client{Master: f.Master, Ports: f.Ports}
	default:
		return nil, fmt.Errorf("unsupported mode %q", f.Mode)
	}
	return cfg, nil
}

func indexPorts(ports []Port) (map[string]Port, error) {
	index := make(map[string]Port, len(ports))
	for i, p := range ports {
		if err := p.validate(); err != nil {
			return nil, fmt.Errorf("port (at pos %d) validate failed :%s", i, err.Error())
		}
		if _, exists := index[p.PortKey]; exists {
			return nil, fmt.Errorf("port (at pos %d) duplicate port_key", i)
		}
		index[p.PortKey] = p
	}
	return index, nil
}

func (p Port) validate() error {
	if p.LocalAddress == "" {
		return fmt.Errorf("local address (local_address) empty")
	}
	if p.PortKey == "" {
		return fmt.Errorf("port key (port_key) empty")
	}
	if p.EncryptMethod == "" {
		return fmt.Errorf("encrypt method (encrypt_method) empty")
	}
	if p.CompressMethod == "" {
		return fmt.Errorf("compress method (compress_method) empty")
	}
	if _, _, err := crypto.Lookup(p.EncryptMethod); err != nil {
		return err
	}
	if _, _, err := compress.Lookup(p.CompressMethod); err != nil {
		return err
	}
	return nil
}
