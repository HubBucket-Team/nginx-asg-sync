package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	nginx "github.com/nginxinc/nginx-plus-go-client/client"
)

var configFile = flag.String("config_path", "/etc/nginx/aws.yaml", "Path to the config file")
var logFile = flag.String("log_path", "", "Path to the log file. If the file doesn't exist, it will be created")
var cloudProvider = flag.String("cloud_provider", "AWS", "CloudProvider selected. Valid values are: 'AWS'")
var version = "0.2-1"

const connTimeoutInSecs = 10

func main() {
	flag.Parse()

	if *logFile != "" {
		logF, err := os.OpenFile(*logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
		if err != nil {
			log.Printf("Couldn't open the log file: %v", err)
			os.Exit(10)
		}
		log.SetOutput(io.MultiWriter(logF, os.Stderr))
	}

	if *cloudProvider != "" {
		if !validateCloudProvider(*cloudProvider) {
			log.Printf("Invalid Cloud Provider %v", *cloudProvider)
			os.Exit(10)
		}
	} else {
		log.Printf("cloud_provider is required")
		os.Exit(10)
	}

	log.Printf("nginx-asg-sync version %s", version)

	var cloudProviderClient CloudProvider
	if *cloudProvider == "AWS" {
		cloudProviderClient = NewAWSClient()
	}

	err := cloudProviderClient.ValidateAndSaveConfig(*configFile)
	if err != nil {
		log.Printf("Couldn't parse or validate the config file %v: %v", *configFile, err)
		os.Exit(10)
	}

	err = cloudProviderClient.Configure()
	if err != nil {
		log.Printf("Couldn't configure cloud provider client: %v", err)
		os.Exit(10)
	}

	httpClient := &http.Client{Timeout: connTimeoutInSecs * time.Second}
	nginxClient, err := nginx.NewNginxClient(httpClient, cloudProviderClient.GetAPIEndpoint())

	if err != nil {
		log.Printf("Couldn't create NGINX client: %v", err)
		os.Exit(10)
	}

	for _, ups := range cloudProviderClient.GetUpstreamsConfig() {
		if ups.Kind == "http" {
			err = nginxClient.CheckIfUpstreamExists(ups.Name)
		} else {
			err = nginxClient.CheckIfStreamUpstreamExists(ups.Name)
		}

		if err != nil {
			log.Printf("Problem with the NGINX configuration: %v", err)
			os.Exit(10)
		}

		exists, err := cloudProviderClient.CheckIfScalingGroupExists(ups.ScalingGroup)
		if err != nil {
			log.Printf("Error with cloud provider configuration: %v", err)
			os.Exit(10)
		} else if !exists {
			log.Printf("Warning: Scaling group '%v' doesn't exist in the cloud provider", ups.ScalingGroup)
		}
	}

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGTERM)

	for {
		for _, upstream := range cloudProviderClient.GetUpstreamsConfig() {
			ips, err := cloudProviderClient.GetPrivateIPsForScalingGroup(upstream.ScalingGroup)
			if err != nil {
				log.Printf("Couldn't get the IP addresses for %v: %v", upstream.ScalingGroup, err)
				continue
			}

			if upstream.Kind == "http" {
				var upsServers []nginx.UpstreamServer
				for _, ip := range ips {
					backend := fmt.Sprintf("%v:%v", ip, upstream.Port)
					upsServers = append(upsServers, nginx.UpstreamServer{
						Server:   backend,
						MaxFails: 1,
					})
				}

				added, removed, err := nginxClient.UpdateHTTPServers(upstream.Name, upsServers)
				if err != nil {
					log.Printf("Couldn't update HTTP servers in NGINX: %v", err)
					continue
				}

				if len(added) > 0 || len(removed) > 0 {
					log.Printf("Updated HTTP servers of %v; Added: %v, Removed: %v", upstream, added, removed)
				}
			} else {
				var upsServers []nginx.StreamUpstreamServer
				for _, ip := range ips {
					backend := fmt.Sprintf("%v:%v", ip, upstream.Port)
					upsServers = append(upsServers, nginx.StreamUpstreamServer{
						Server:   backend,
						MaxFails: 1,
					})
				}

				added, removed, err := nginxClient.UpdateStreamServers(upstream.Name, upsServers)
				if err != nil {
					log.Printf("Couldn't update Steam servers in NGINX: %v", err)
					continue
				}

				if len(added) > 0 || len(removed) > 0 {
					log.Printf("Updated Stream servers of %v; Added: %v, Removed: %v", upstream, added, removed)
				}
			}

		}

		select {
		case <-time.After(cloudProviderClient.GetSyncIntervalInSeconds() * time.Second):
		case <-sigterm:
			log.Println("Terminating...")
			return
		}
	}
}
