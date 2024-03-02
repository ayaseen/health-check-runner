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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var loggingForwarder string

// Check Logging

func checkLoggingForwarders() {

	out, err := exec.Command("oc", "get", "clusterLogforwarder", "-n", "openshift-logging", "-o", "yaml").Output()
	//if err != nil {
	//	fmt.Println("Error executing oc command:", err)
	//	return
	//}

	loggingForwarder = string(out)
	loggingConfigure, _ := checkLoggingConfigure()
	loggingForwaders, _ := checkLoggingForwarder()

	if loggingConfigure != "" {
		if loggingForwaders != "" {
			if !strings.Contains(loggingForwarderOPS, "outputs: []") ||
				!strings.Contains(loggingForwarderOPS, "pipelines: []") ||
				!strings.Contains(loggingForwarderOPS, "inputs: []") {
				//color.Green("Log forwarding is configured for openshift\t\tPASSED")
			}
		}
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
		output.WriteString(LoggingForwardersProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}
}

func LoggingForwardersProcess(line string) string {

	if strings.HasPrefix(line, "<<Alternative Log Aggregation>>") {
		loggingConfigure, _ := checkLoggingConfigure()

		if loggingConfigure != "" {
			if strings.Contains(loggingForwarder, "outputs: []") ||
				strings.Contains(loggingForwarder, "pipelines: []") ||
				strings.Contains(loggingForwarder, "inputs: []") {
				return line + "\n\n| OpenShift Logging Forwarder not configured \n\n" + GetKeyChanges("recommended")
			} else {
				return line + "\n\n| OpenShift Logging Forwarder is configured \n\n" + GetKeyChanges("nochange")
			}
		} else {
			return line + "\n\n|Not Applicable" + GetKeyChanges("na") + "\n\n"
		}
	}

	if strings.HasPrefix(line, "== Alternative Log Aggregation") {

		version, _ := getOpenShiftVersion()
		loggingConfigure, _ := checkLoggingConfigure()

		if loggingConfigure != "" {

			if strings.Contains(loggingForwarder, "outputs: []") ||
				strings.Contains(loggingForwarder, "pipelines: []") ||
				strings.Contains(loggingForwarder, "inputs: []") {
				return line + "\n\n" + GetChanges("nochange") +
					"\n\n**Observation**\n\nAlternative log aggregation configured for long term audit.\n\n" +
					"**Recommendation**\n\nNone\n\n" +
					"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
					"html-single/logging/index#cluster-logging-external[Forwarding logs to external third-party logging systems]\n"
			} else {
				return line + "\n\n" + GetChanges("recommended") +
					"\n\n**Observation**\n\nThere is no alternative log aggregation configured for long term audit. \n\n" +
					"**Recommendation**\n\nYou need to logging forwarder to aggregate all the logs from your OpenShift Container Platform cluster," +
					" such as node system audit logs, application container logs, and infrastructure logs. \n\n" +
					"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
					"html-single/logging/index#cluster-logging-external[Forwarding logs to external third-party logging systems]\n"

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
