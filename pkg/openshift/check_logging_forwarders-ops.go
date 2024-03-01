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

var loggingForwarderOPS string

// Check Logging

func checkLoggingForwardersOPS() {

	out, err := exec.Command("oc", "get", "clusterLogForwarder", "-n", "openshift-logging", "-o", "yaml").Output()

	loggingForwarderOPS = string(out)

	loggingConfigure, _ := checkLoggingConfigure()
	loggingForwaders, _ := checkLoggingForwarder()

	if loggingConfigure != "" {
		if loggingForwaders != "" {
			if !strings.Contains(loggingForwarderOPS, "outputs: []") ||
				!strings.Contains(loggingForwarderOPS, "pipelines: []") ||
				!strings.Contains(loggingForwarderOPS, "inputs: []") {
				//color.Green("Log forwarding audit, infra, applications\t\tPASSED")
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
		output.WriteString(LoggingForwardersOPSProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}
}

func LoggingForwardersOPSProcess(line string) string {

	if strings.HasPrefix(line, "<<Forward OpenShift audit logs to external logging system>>") ||
		strings.HasPrefix(line, "<<Forward OpenShift application logs to external logging system>>") ||
		strings.HasPrefix(line, "<<Forward OpenShift infrastructure logs to external logging system>>") {

		loggingConfigure, _ := checkLoggingConfigure()
		loggingForwarderExits, _ := checkLoggingForwarder()

		if loggingConfigure != "" && loggingForwarderExits != "" {
			if strings.Contains(loggingForwarderOPS, "outputs: []") ||
				strings.Contains(loggingForwarderOPS, "pipelines: []") ||
				strings.Contains(loggingForwarderOPS, "inputs: []") {
				return line + "\n\n| OpenShift Logging Forwarder not configured \n\n" + GetKeyChanges("recommended")
			} else {
				return line + "\n\n| OpenShift Logging Forwarder is configured \n\n" + GetKeyChanges("nochange")
			}
		} else {
			return line + "\n\n|Not Applicable" + GetKeyChanges("na") + "\n\n"
		}
	}

	if strings.HasPrefix(line, "== Forward OpenShift audit logs to external logging system") ||
		strings.HasPrefix(line, "== Forward OpenShift application logs to external logging system") ||
		strings.HasPrefix(line, "== Forward OpenShift infrastructure logs to external logging system") {

		version, _ := getOpenShiftVersion()
		loggingConfigure, _ := checkLoggingConfigure()
		loggingForwarderExits, _ := checkLoggingForwarder()

		if loggingConfigure != "" && loggingForwarderExits != "" {

			if strings.Contains(loggingForwarderOPS, "outputs: []") ||
				strings.Contains(loggingForwarderOPS, "pipelines: []") ||
				strings.Contains(loggingForwarderOPS, "inputs: []") {
				return line + "\n\n" + GetChanges("recommended") +
					"\n\n**Observation**\n\nLog forwarding is not configured for the OpenShift audit pipeline to an external logging system." +
					"An external logging system protects the confidentiality, integrity, and availability of audit logs\n\n" +
					"**Recommendation**\n\nConfigure external log forwarding for audit, infrastructure and applications logs\n\n" +
					"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
					"html-single/logging/index#cluster-logging-external[Forwarding logs to external third-party logging systems]\n"
			} else {
				return line + "\n\n" + GetChanges("nochange") +
					"\n\n**Observation**\n\nLog forwarding is configured for the OpenShift audit pipeline to an external logging system.\n\n" +
					"**Recommendation**\n\nNone\n\n" +
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
