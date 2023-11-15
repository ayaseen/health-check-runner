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
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var alertState string

const alertName = `Alerts are not configured to be sent to a notification system`

func alerts() {

	out, err := exec.Command("oc", "-n", "openshift-monitoring", "exec", "-c", "prometheus",
		"prometheus-k8s-0", "--", "curl", "-s", "http://localhost:9090/api/v1/alerts").Output()
	if err != nil {
		log.Fatal("Failed to execute command:", err)

	}

	alertState = strings.TrimSpace(string(out))

	// Check if the alert with the specified name exists
	if !strings.Contains(alertState, alertName) {
		color.Green("OpenShift alerts are forwarded to an external system\tPASSED")
	} else {
		color.Red("OpenShift alerts are forwarded to an external system\tFAILED")
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
		output.WriteString(alertsProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}
}

func alertsProcess(line string) string {
	if strings.HasPrefix(line, "<<Ensure OpenShift alerts are forwarded to an external system that is monitored>>") {
		if !strings.Contains(alertState, alertName) {
			return line + "\n\n| OpenShift alerts are forwarded to an external system\n\n" + GetKeyChanges("nochnage")
		} else {
			return line + "\n\n| OpenShift alerts are not forwarded to an external system\n\n" + GetKeyChanges("required")
		}
	}

	if strings.HasPrefix(line, "== Ensure OpenShift alerts are forwarded to an external system that is monitored") {

		version, _ := getOpenShiftVersion()

		if !strings.Contains(alertState, alertName) {
			return line + "\n\n" + GetChanges("nochange") +
				"\n\n**Observation**\n\nOOpenShift alerts are forwarded to an external system\n\n" +
				"**Recommendation**\n\nNo changes are required if the alerts been forwarded to external system. \n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"monitoring/index#sending-notifications-to-external-systems_managing-alerts[Sending Notification to external system]\n"

		} else {
			return line + "\n\n" + GetChanges("recommended") +
				"\n\n**Observation**\n\nOpenShift alerts are not forwarded to an external system\n\n" +
				"**Recommendation**\n\nRouting alerts to receivers enables you to send timely notifications to the appropriate teams when failures occur. \n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"monitoring/index#sending-notifications-to-external-systems_managing-alerts[Sending Notification to external system]\n"

		}
	}

	return line + "\n"
}
