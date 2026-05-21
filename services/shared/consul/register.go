package consul

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"github.com/hashicorp/consul/api"
)

type RegistrationConfig struct {
	Name           string
	ServiceName    string
	Port           int
	HealthEndpoint string
	ConsulAddr     string
}

func RegisterService(ctx context.Context, cfg RegistrationConfig) (*api.Client, error) {
	consulCfg := api.DefaultConfig()
	consulCfg.Address = cfg.ConsulAddr

	client, err := api.NewClient(consulCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create consul client: %w", err)
	}

	// Get the service's IP address for registration
	host := cfg.Name
	if net.ParseIP(host) == nil && host != "localhost" {
		// Use the service name as DNS name
		host = cfg.Name
	}

	registration := &api.AgentServiceRegistration{
		ID:      cfg.ServiceName + "-1",
		Name:    cfg.ServiceName,
		Port:    cfg.Port,
		Address: host,
		Check: &api.AgentServiceCheck{
			HTTP:     fmt.Sprintf("http://%s:%d%s", host, cfg.Port, cfg.HealthEndpoint),
			Interval: "10s",
			Timeout:  "5s",
		},
	}

	if err := client.Agent().ServiceRegister(registration); err != nil {
		return nil, fmt.Errorf("failed to register service: %w", err)
	}

	slog.Info("registered with consul", "service", cfg.ServiceName, "address", host, "port", cfg.Port)
	return client, nil
}

func DeregisterService(client *api.Client, serviceID string) error {
	if client == nil {
		return nil
	}
	return client.Agent().ServiceDeregister(serviceID)
}
