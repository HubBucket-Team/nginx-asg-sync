package main

import "time"

// CloudProvider is the interface to connect with any cloud provider.
type CloudProvider interface {
	GetPrivateIPsForScalingGroup(name string) ([]string, error)
	ValidateAndSaveConfig(configPath string) error
	CheckIfScalingGroupExists(name string) (bool, error)
	Configure() error

	GetUpstreamsConfig() []Upstream
	GetAPIEndpoint() string
	GetSyncIntervalInSeconds() time.Duration
}

// Upstream is the cloud agnostic representation of an Upstream (eg, common fields for every cloud provider)
type Upstream struct {
	Name         string
	Port         int
	ScalingGroup string
	Kind         string
}

func validateCloudProvider(provider string) bool {
	providers := map[string]bool{
		"AWS": true,
	}

	return providers[provider]
}
