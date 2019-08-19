package main

import "testing"

var validYaml = []byte(`region: us-west-2
api_endpoint: http://127.0.0.1:8080/api
sync_interval_in_seconds: 5
upstreams:
  - name: backend1
    autoscaling_group: backend-group
    port: 80
    kind: http
  - name: backend2
    autoscaling_group: backend-group
    port: 80
    kind: http
`)

type testInputAWS struct {
	cfg *awsConfig
	msg string
}

func getValidConfig() *awsConfig {
	upstreams := []awsUpstream{
		{
			Name:             "backend1",
			AutoscalingGroup: "backend-group",
			Port:             80,
			Kind:             "http",
		},
	}
	cfg := awsConfig{
		Region:                "us-west-2",
		APIEndpoint:           "http://127.0.0.1:8080/api",
		SyncIntervalInSeconds: 1,
		Upstreams:             upstreams,
	}

	return &cfg
}

func getInvalidConfigInput() []*testInputAWS {
	var input []*testInputAWS

	invalidRegionCfg := getValidConfig()
	invalidRegionCfg.Region = ""
	input = append(input, &testInputAWS{invalidRegionCfg, "invalid region"})

	invalidAPIEndpointCfg := getValidConfig()
	invalidAPIEndpointCfg.APIEndpoint = ""
	input = append(input, &testInputAWS{invalidAPIEndpointCfg, "invalid api_endpoint"})

	invalidSyncIntervalInSecondsCfg := getValidConfig()
	invalidSyncIntervalInSecondsCfg.SyncIntervalInSeconds = 0
	input = append(input, &testInputAWS{invalidSyncIntervalInSecondsCfg, "invalid sync_interval_in_seconds"})

	invalidMissingUpstreamsCfg := getValidConfig()
	invalidMissingUpstreamsCfg.Upstreams = nil
	input = append(input, &testInputAWS{invalidMissingUpstreamsCfg, "no upstreams"})

	invalidUpstreamNameCfg := getValidConfig()
	invalidUpstreamNameCfg.Upstreams[0].Name = ""
	input = append(input, &testInputAWS{invalidUpstreamNameCfg, "invalid name of the upstream"})

	invalidUpstreamAutoscalingGroupCfg := getValidConfig()
	invalidUpstreamAutoscalingGroupCfg.Upstreams[0].AutoscalingGroup = ""
	input = append(input, &testInputAWS{invalidUpstreamAutoscalingGroupCfg, "invalid autoscaling_group of the upstream"})

	invalidUpstreamPortCfg := getValidConfig()
	invalidUpstreamPortCfg.Upstreams[0].Port = 0
	input = append(input, &testInputAWS{invalidUpstreamPortCfg, "invalid port of the upstream"})

	invalidUpstreamKindCfg := getValidConfig()
	invalidUpstreamKindCfg.Upstreams[0].Kind = ""
	input = append(input, &testInputAWS{invalidUpstreamKindCfg, "invalid kind of the upstream"})

	return input
}

func TestValidateConfigNotValid(t *testing.T) {
	input := getInvalidConfigInput()

	for _, item := range input {
		err := validateAWSConfig(item.cfg)
		if err == nil {
			t.Errorf("validateConfig() didn't fail for the invalid config file with %v", item.msg)
		}
	}
}

func TestValidateConfigValid(t *testing.T) {
	cfg := getValidConfig()

	err := validateAWSConfig(cfg)
	if err != nil {
		t.Errorf("validateConfig() failed for the valid config: %v", err)
	}
}
