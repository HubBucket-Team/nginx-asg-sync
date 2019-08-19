package main

import (
	"fmt"
	"time"
)

type awsConfig struct {
	Region                string
	APIEndpoint           string        `yaml:"api_endpoint"`
	SyncIntervalInSeconds time.Duration `yaml:"sync_interval_in_seconds"`
	Upstreams             []awsUpstream
}

type awsUpstream struct {
	Name             string
	AutoscalingGroup string `yaml:"autoscaling_group"`
	Port             int
	Kind             string
}

const errorMsgFormat = "The mandatory field %v is either empty or missing in the config file"
const intervalErrorMsg = "The mandatory field sync_interval_in_seconds is either 0 or missing in the config file"
const upstreamNameErrorMsg = "The mandatory field name is either empty or missing for an upstream in the config file"
const upstreamErrorMsgFormat = "The mandatory field %v is either empty or missing for the upstream %v in the config file"
const upstreamPortErrorMsgFormat = "The mandatory field port is either zero or missing for the upstream %v in the config file"
const upstreamKindErrorMsgFormat = "The mandatory field kind is either not equal to http or tcp or missing for the upstream %v in the config file"

func validateAWSConfig(cfg *awsConfig) error {
	if cfg.Region == "" {
		return fmt.Errorf(errorMsgFormat, "region")
	}
	if cfg.APIEndpoint == "" {
		return fmt.Errorf(errorMsgFormat, "api_endpoint")
	}
	if cfg.SyncIntervalInSeconds == 0 {
		return fmt.Errorf(intervalErrorMsg)
	}

	if len(cfg.Upstreams) == 0 {
		return fmt.Errorf("There is no upstreams found in the config file")
	}

	for _, ups := range cfg.Upstreams {
		if ups.Name == "" {
			return fmt.Errorf(upstreamNameErrorMsg)
		}
		if ups.AutoscalingGroup == "" {
			return fmt.Errorf(upstreamErrorMsgFormat, "autoscaling_group", ups.Name)
		}
		if ups.Port == 0 {
			return fmt.Errorf(upstreamPortErrorMsgFormat, ups.Name)
		}
		if ups.Kind == "" || !(ups.Kind == "http" || ups.Kind == "stream") {
			return fmt.Errorf(upstreamKindErrorMsgFormat, ups.Name)
		}
	}

	return nil
}
