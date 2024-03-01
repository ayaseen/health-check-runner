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

var isSchedule bool

func controlNodeSchedule() {

	// Execute the command: oc get scheduler cluster -o yaml
	out, err := exec.Command("oc", "get", "scheduler", "cluster", "-o", "yaml").Output()
	if err != nil {
		log.Fatal(err)
	}

	// Parse the output as YAML
	controlNode := string(out)

	// Check if the mastersSchedulable field is set to false
	if strings.Contains(controlNode, "mastersSchedulable: false") {
		isSchedule = false
		//color.Green("Masters nodes are not schedule\t\t\t\tPASSED")
	} else {
		isSchedule = true
		//color.Red("Masters nodes are schedule\t\t\t\tFAILED")
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
		output.WriteString(controlNodesProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}
}

func controlNodesProcess(line string) string {
	if strings.HasPrefix(line, "<<Control Nodes Schedulable>>") {

		if isSchedule != false {
			return line + "\n\n|Masters nodes are Schedulable\n\n" + GetKeyChanges("required") + "\n\n"
		} else {
			return line + "\n\n|Masters nodes are not Schedulable\n\n" + GetKeyChanges("nochange") + "\n\n"
		}
	}

	if strings.HasPrefix(line, "== Control Nodes Schedulable") {

		version, _ := getOpenShiftVersion()

		if isSchedule != false {
			return line + "\n\n" + GetChanges("recommended") +
				"\n\n**Observation**\n\nMasters nodes are Schedulable. \n\n" +
				"**Recommendation**\n\nAvoid running user workload application on master nodes.\n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"html-single/nodes/index#working-with-nodes[Working with the nodes]\n"

		} else {
			return line + "\n\n" + GetChanges("nochange") +
				"\n\n**Observation**\n\nMasters nodes are not Schedulable. \n\n" +
				"**Recommendation**\n\nNone.\n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"html-single/nodes/index#working-with-nodes[Working with the nodes]\n"

		}
	}

	return line + "\n"
}
