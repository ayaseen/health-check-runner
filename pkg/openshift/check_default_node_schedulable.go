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
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var defaultNodeSelector bool

func defaultNodeSchedule() {

	// Execute the command: oc get scheduler cluster -o yaml
	out, err := exec.Command("oc", "get", "scheduler", "cluster", "-o", "yaml").Output()
	if err != nil {
		log.Fatal(err)
	}

	// Parse the output as YAML
	nodeSelector := string(out)

	// Check if the default Schedulable
	if strings.Contains(nodeSelector, "defaultNodeSelector") {
		defaultNodeSelector = false
		//color.Green("Default Node Selector Set\t\t\t\tPASSED")
	} else {
		defaultNodeSelector = true
		//color.Red("Default Node Selector Set\t\t\t\tFAILED")
	}

	// Create the output file for writing
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
		output.WriteString(defaultNodesProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}
}

func defaultNodesProcess(line string) string {
	if strings.HasPrefix(line, "<<Default Node Selector Set>>") {

		if defaultNodeSelector != false {
			return line + "\n\n|Default Node Selector not Set\n\n" + GetKeyChanges("recommended") + "\n\n"
		} else {
			return line + "\n\n|Default Node Selector Set\n\n" + GetKeyChanges("nochange") + "\n\n"
		}
	}

	if strings.HasPrefix(line, "== Default Node Selector Set") {

		version, _ := getOpenShiftVersion()

		if defaultNodeSelector != false {
			return line + "\n\n" + GetChanges("recommended") +
				"\n\n**Observation**\n\nDefault Node Selector not Set. \n\n" +
				"**Recommendation**\n\nNone\n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"html-single/nodes/index#nodes-scheduler-node-selectors-cluster_nodes-scheduler-node-selectors[Default Node Selector]\n"

		} else {
			return line + "\n\n" + GetChanges("nochange") +
				"\n\n**Observation**\n\nDefault Node Selector Set. \n\n" +
				"**Recommendation**\n\nNone.\n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"html-single/nodes/index#nodes-scheduler-node-selectors-cluster_nodes-scheduler-node-selectors[Default Node Selector]\n"

		}
	}

	return line + "\n"
}
