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

var managementState string

func internalRegistry() {

	out, err := exec.Command("oc", "get", "configs.imageregistry.operator.openshift.io",
		"cluster", "-o", "jsonpath={.spec.storage.managementState}").Output()
	if err != nil {
		log.Fatal("Failed to execute command:", err)

	}

	managementState = strings.TrimSpace(string(out))

	//if string(managementState) == "Managed" {
	//	color.Green("Openshift internal registry is functioning\t\tPASSED")
	//} else {
	//	color.Red("Openshift internal registry is functioning\t\tFAILED")
	//}

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
		output.WriteString(internalRegistryProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}
}

func internalRegistryProcess(line string) string {
	if strings.HasPrefix(line, "<<Openshift internal registry is functioning and running>>") {
		if string(managementState) == "Managed" {
			return line + "\n\n| Openshift internal registry is functioning and running \n\n" + GetKeyChanges("nochange")
		} else {
			return line + "\n\n| Openshift internal registry is not functioning and running \n\n" + GetKeyChanges("required")
		}
	}

	if strings.HasPrefix(line, "== Openshift internal registry is functioning and running") {

		version, _ := getOpenShiftVersion()

		if string(managementState) == "Managed" {
			return line + "\n\n" + GetChanges("nochange") +
				"\n\n**Observation**\n\nOpenshift internal registry is functioning and running\n\n" +
				"**Recommendation**\n\nNo changes are required if the 'managementState' is set to 'Managed'. \n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"html-single/registry/index[OpenShift registry]\n"

		} else {
			return line + "\n\n" + GetChanges("required") +
				"\n\n**Observation**\n\nOpenshift internal registry is not functioning and running\n\n" +
				"**Recommendation**\n\nChanges are required if the 'managementState' is not set to 'Managed'. \n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"html-single/registry/index[OpenShift registry]\n"

		}
	}

	return line + "\n"
}
