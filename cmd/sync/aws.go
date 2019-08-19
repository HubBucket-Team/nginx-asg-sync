package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	yaml "gopkg.in/yaml.v2"
)

// AWSClient allows you to get the list of IP addresses of instanes of an Auto Scaling group. It implements the CloudProvider interface
type AWSClient struct {
	svcEC2         ec2iface.EC2API
	svcAutoscaling autoscalingiface.AutoScalingAPI
	config         *awsConfig
}

// NewAWSClient creates an AWSClient
func NewAWSClient() *AWSClient {
	return &AWSClient{}
}

// Configure configures the AWSClient with necessary parameters
func (client *AWSClient) Configure() error {
	httpClient := &http.Client{Timeout: connTimeoutInSecs * time.Second}
	cfg := &aws.Config{Region: aws.String(client.config.Region), HTTPClient: httpClient}

	session, err := session.NewSession(cfg)
	if err != nil {
		return err
	}

	svcAutoscaling := autoscaling.New(session)
	svcEC2 := ec2.New(session)
	client.svcEC2 = svcEC2
	client.svcAutoscaling = svcAutoscaling
	return nil
}

// ValidateAndSaveConfig parses and validates AWSClient config and saves it
func (client *AWSClient) ValidateAndSaveConfig(configPath string) error {
	data, err := ioutil.ReadFile(*configFile)
	if err != nil {
		return err
	}

	cfg := &awsConfig{}
	err = yaml.Unmarshal(data, cfg)
	if err != nil {
		return err
	}

	err = validateAWSConfig(cfg)
	if err != nil {
		return err
	}

	client.config = cfg
	return nil
}

// CheckIfScalingGroupExists checks if the Auto Scaling group exists
func (client *AWSClient) CheckIfScalingGroupExists(name string) (bool, error) {
	_, exists, err := client.getAutoscalingGroup(name)
	return exists, fmt.Errorf("couldn't check if an AutoScaling group exists: %v", err)
}

// GetUpstreamsConfig returns the configuration for Upstreams. It returns a map with the Upstream name as key and the AutoScaling group as value
func (client *AWSClient) GetUpstreamsConfig() []Upstream {
	var upstreams []Upstream
	for _, u := range client.config.Upstreams {
		u := Upstream{
			Name:         u.Name,
			Port:         u.Port,
			ScalingGroup: u.AutoscalingGroup,
			Kind:         u.Kind,
		}
		upstreams = append(upstreams, u)
	}
	return upstreams
}

// GetPrivateIPsForScalingGroup returns the list of IP addresses of instanes of the Auto Scaling group
func (client *AWSClient) GetPrivateIPsForScalingGroup(name string) ([]string, error) {
	group, exists, err := client.getAutoscalingGroup(name)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, fmt.Errorf("autoscaling group %v doesn't exist", name)
	}

	instances, err := client.getInstancesOfAutoscalingGroup(group)
	if err != nil {
		return nil, err
	}

	var result []string
	for _, ins := range instances {
		if len(ins.NetworkInterfaces) > 0 && ins.NetworkInterfaces[0].PrivateIpAddress != nil {
			result = append(result, *ins.NetworkInterfaces[0].PrivateIpAddress)
		}
	}

	return result, nil
}

// GetSyncIntervalInSeconds returns the SyncIntervalInSeconds config value
func (client *AWSClient) GetSyncIntervalInSeconds() time.Duration {
	return client.config.SyncIntervalInSeconds
}

// GetAPIEndpoint returns the APIEndpoint config value
func (client *AWSClient) GetAPIEndpoint() string {
	return client.config.APIEndpoint
}

func (client *AWSClient) getAutoscalingGroup(name string) (*autoscaling.Group, bool, error) {
	params := &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{
			aws.String(name),
		},
	}

	resp, err := client.svcAutoscaling.DescribeAutoScalingGroups(params)
	if err != nil {
		return nil, false, err
	}

	if len(resp.AutoScalingGroups) != 1 {
		return nil, false, nil
	}

	return resp.AutoScalingGroups[0], true, nil
}

func (client *AWSClient) getInstancesOfAutoscalingGroup(group *autoscaling.Group) ([]*ec2.Instance, error) {
	var result []*ec2.Instance

	if len(group.Instances) == 0 {
		return result, nil
	}

	var ids []*string
	for _, ins := range group.Instances {
		ids = append(ids, ins.InstanceId)
	}
	params := &ec2.DescribeInstancesInput{
		InstanceIds: ids,
	}

	resp, err := client.svcEC2.DescribeInstances(params)
	if err != nil {
		return result, err
	}
	for _, res := range resp.Reservations {
		result = append(result, res.Instances...)
	}

	return result, nil
}
