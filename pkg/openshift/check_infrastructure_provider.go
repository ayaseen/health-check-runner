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

// Check project template
var infraCloudProvider string

func infrastructureProvider() {

	var err error
	infraProvider, err := exec.Command("oc", "get", "Infrastructure", "cluster", "-o", "jsonpath={.spec.platformSpec.type}").Output()

	infraCloudProvider = strings.TrimSpace(string(infraProvider))

	//if string(infraCloudProvider) == "" {
	//	color.Red("Infrastructure Provider not set\t\t\t\tFAILED")
	//
	//} else {
	//	color.Green("Infrastructure Provider is set\t\t\t\tPASSED")
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
		output.WriteString(infrastructureProviderProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}
}

func infrastructureProviderProcess(line string) string {

	if strings.HasPrefix(line, "<<Infrastructure Provider(s)>>") {
		return line + "\n\n| " + infraCloudProvider + " \n\n" + GetKeyChanges("nochange")
	}

	if strings.HasPrefix(line, "== Infrastructure Provider(s)") {

		version, _ := getOpenShiftVersion()

		if string(infraCloudProvider) != "" {
			return line + "\n\n" + GetChanges("nochange") +
				"\n\n**Observation**\n\nInfrastructure Cloud Provider is " + infraCloudProvider + "\n\n" +
				"**Recommendation**\n\nNone\n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/installing/index\n"
		}
	}

	return line + "\n"
}
