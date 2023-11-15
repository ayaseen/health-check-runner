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
	"fmt"
	"github.com/fatih/color"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var loggingHealth string

// Check Logging

func checkLoggingHealth() {

	out, err := exec.Command("oc", "get", "clusterlogging", "instance", "-n", "openshift-logging", "-o", "yaml").Output()

	// Parse the YAML output and check the Elasticsearch status
	loggingHealth = string(out)

	if loggingHealth != "" {
		if strings.Index(loggingHealth, "status: yellow") != -1 || strings.Index(loggingHealth, "status: red") != -1 {
			color.Red("The logging is healthy\t\t\t\t\tFAILED")

		} else {
			color.Green("The logging is healthy\t\t\t\t\tPASSED")
		}
	} else {
		color.HiCyan("The logging is healthy\t\t\t\t\tSKIPPED")
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
		output.WriteString(LoggingHealthProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}
}

func checkLoggingStatus() (string, error) {

	// Get the Installation method type
	out, err := exec.Command("oc", "get", "clusterlogging", "instance", "-n", "openshift-logging", "-o", "yaml").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func LoggingHealthProcess(line string) string {

	if strings.HasPrefix(line, "<<OpenShift logging deployment is functioning and healthy>>") {
		loggingConfigure, _ := checkLoggingConfigure()

		if loggingConfigure != "" {
			if strings.Index(loggingHealth, "status: yellow") != -1 ||
				strings.Index(loggingHealth, "status: red") != -1 {
				return line + "\n\n| OpenShift Logging not healthy \n\n" + GetKeyChanges("recommended")
			} else {
				return line + "\n\n| OpenShift Logging is healthy \n\n" + GetKeyChanges("nochange")
			}
		} else {
			return line + "\n\n|Not Applicable" + GetKeyChanges("na") + "\n\n"
		}
	}

	if strings.HasPrefix(line, "== OpenShift logging deployment is functioning and healthy") {

		version, _ := getOpenShiftVersion()
		loggingConfigure, _ := checkLoggingConfigure()

		if loggingConfigure != "" {
			if strings.Index(loggingHealth, "status: yellow") != -1 ||
				strings.Index(loggingHealth, "status: red") != -1 {
				return line + "\n\n" + GetChanges("recommended") +
					"\n\n**Observation**\n\nOpenShift logging is not healthy.It’s health status is either yellow or red." +
					" An unhealthy logging system could lead to incomplete log collection and/or total loss of log data.\n\n" +
					"**Recommendation**\n\nPerform an investigation into the root cause of logging issue\n\n" +
					"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
					"html-single/html-single/logging/index#cluster-logging-cluster-status[OpenShift logging status]\n" +
					"* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
					"html-single/html-single/logging/index#cluster-logging-log-store-status[OpenShift Elasticsearch log store status]\n"
			} else {
				return line + "\n\n" + GetChanges("nochange") +
					"\n\n**Observation**\n\nOpenShift logging is healthy. \n\n" +
					"**Recommendation**\n\nNone \n\n" +
					"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
					"html-single/html-single/logging/index#cluster-logging-cluster-status[OpenShift logging status]\n" +
					"* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
					"html-single/html-single/logging/index#cluster-logging-log-store-status[OpenShift Elasticsearch log store status]\n"

			}
		} else {

			return line + "\n\n" + GetChanges("na") +
				"\n\n**Observation**\n\nOpenShift Logging is not installed or configured. \n\n" +
				"**Recommendation**\n\nYou need to deploy the logging subsystem to aggregate all the logs from your OpenShift Container Platform cluster," +
				" such as node system audit logs, application container logs, and infrastructure logs. \n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/logging/cluster-logging-deploying[Installing OpenShift Logging]\n" +
				"* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/logging/configuring-your-logging-deployment[Cluster Logging custom resource]\n" +
				"* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/logging/cluster-logging[Understanding Red Hat OpenShift Logging]\n"
		}

	}

	return line + "\n"
}
