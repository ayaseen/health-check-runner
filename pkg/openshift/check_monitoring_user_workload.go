/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06

This application performs health checks on OpenShift to provide visibility into various functionalities. It verifies the following aspects:

- OpenShift configurations: Verify OpenShift configuration meets the standard and best practices.
- Security: It examines the security measures in place, such as authentication and authorization configurations.
- Application Probes: It tests the health and readiness probes of deployed applications to ensure they are functioning correctly.
- Resource Usage: It monitors resource consumption of OpenShift clusters, including CPU, memory, and storage.

The purpose of this application is to provide administrators and developers with an overview of OpenShift's health and functionality, helping them identify potential issues and ensure the smooth operation of their OpenShift environment.
*/

package openshift

import (
	"bufio"
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"strings"
)

var configMapData struct {
	EnableUserWorkload bool `json:"enableUserWorkload"`
}

var configMapExist bool

func monitoringUserWorkload() {
	config, err := getClusterConfig()
	if err != nil {
		fmt.Println("Error getting cluster config:", err)
		os.Exit(1)
	}

	clientset, err := getClientSet(config)
	if err != nil {
		fmt.Println("Error getting client set:", err)
		os.Exit(1)
	}

	// Check if the configMap exists
	cm, err := clientset.CoreV1().ConfigMaps("openshift-monitoring").Get(context.TODO(), "cluster-monitoring-config", metav1.GetOptions{})
	if err != nil {
		configMapExist = false
	}

	// Check if the key exists and has the correct value
	dataUser, check := cm.Data["config.yaml"]
	//if !ok {
	//	fmt.Println("ConfigMap does not contain config.yaml")
	//	os.Exit(1)
	//}

	if err = yaml.Unmarshal([]byte(dataUser), &configMapData); err != nil {
		fmt.Printf("Error unmarshaling YAML: %v\n", err)

	}

	if !configMapData.EnableUserWorkload && configMapExist == false && !check {
		//color.Red("User Workload Monitoring is enabled\t\t\tFAILED")

	}

	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		fmt.Println("Error getting current directory:", err)
		return
	}

	inputFile, err := os.Open(filepath.Join(dir, "resources/healthcheck-body.adoc"))
	if err != nil {
		panic(err)
	}
	defer inputFile.Close()

	// Open the output file
	outputFile, err := os.OpenFile("resources/healthcheck-body.adoc", os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer outputFile.Close()

	// Create a scanner to read the input file
	fileScanner := bufio.NewScanner(inputFile)

	// Create a buffer to hold the output
	var output strings.Builder

	// Process each line of the input file
	for fileScanner.Scan() {
		line := fileScanner.Text()
		output.WriteString(monitoringUserWorkloadProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}
}

func monitoringUserWorkloadProcess(line string) string {

	if strings.HasPrefix(line, "<<User Workload Monitoring>>") {
		if !configMapData.EnableUserWorkload {
			return line + "\n\n| User Workload Monitoring not Set \n\n" + GetKeyChanges("recommended")
		} else {
			return line + "\n\n| User Workload Monitoring is Set \n\n" + GetKeyChanges("nochange")
		}
	}

	if strings.HasPrefix(line, "== User Workload Monitoring") {

		version, _ := getOpenShiftVersion()

		if !configMapData.EnableUserWorkload {
			return line + "\n\n" + GetChanges("recommended") +
				"\n\n**Observation**\n\nUser workload monitoring is not configured.\n\n" +
				"**Recommendation**\n\nUsing this feature centralizes monitoring for core platform components and user-defined projects.\n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"html-single/monitoring/configuring-the-monitoring-stack[Understanding the monitoring stack]\n" +
				"* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/monitoring/index#enabling-monitoring-for-user-defined-projects[Enabling monitoring for user-defined projects]"

		} else {
			return line + "\n\n" + GetChanges("nochange") +
				"\n\n**Observation**\n\nUser workload monitoring is configured. \n\n" +
				"**Recommendation**\n\nNone. \n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"html-single/monitoring/configuring-the-monitoring-stack[Understanding the monitoring stack]\n" +
				"* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/monitoring/enabling-monitoring-for-user-defined-projects[Enabling monitoring for user-defined projects]"

		}

	}

	return line + "\n"
}
