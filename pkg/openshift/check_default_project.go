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

var defaultProjectValidate []byte

// Check project template

func defaultProject() {

	var err error
	defaultProjectValidate, err = exec.Command("oc", "get", "project.config.openshift.io/cluster", "-o", "jsonpath={.spec.projectRequestTemplate}").Output()
	if err != nil {
		fmt.Println("Error: Failed to retrieve cluster project template information")

	}

	if string(defaultProjectValidate) == "" {
		//color.Red("Project template is configured\t\t\t\tFAILED")

	} else {

		//color.Green("Project template is configured\t\t\t\tPASSED")

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
		output.WriteString(defaultProjectProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}
}

func checkDefaultProject() (string, error) {

	// Get the encryption type of the etcd server
	out, err := exec.Command("oc", "get", "project.config.openshift.io/cluster", "-o", "yaml").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func defaultProjectProcess(line string) string {
	if strings.HasPrefix(line, "<<Default Project Template Set>>") {
		if string(defaultProjectValidate) == "" {
			return line + "\n\n| No Default project template\n\n" + GetKeyChanges("recommended")
		} else {
			return line + "\n\n| None\n\n" + GetKeyChanges("nochange")
		}
	}

	if strings.HasPrefix(line, "== Default Project Template Set") {
		defaultproject, _ := checkDefaultProject()
		version, _ := getOpenShiftVersion()

		if string(defaultProjectValidate) == "" {
			return line + "\n\n" + GetChanges("recommended") + "\n\n[source, yaml]\n----\n" + defaultproject + "\n----\n" +
				"\n\n**Observation**\n\nNo Default project template\n\n" +
				"**Recommendation**\n\nCheck in the reference how to create default project template. \n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/building_applications/index#configuring-project-creation[Configuring project creation]\n" +
				"* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/building_applications/index#quotas-setting-per-project[Quotas Setting Per Project]\n"

		} else {
			return line + "\n\n" + GetChanges("nochange") + "\n\n[source, yaml]\n----\n" + defaultproject + "\n----\n" +
				"\n\n**Observation**\n\nDefault project template is set\n\n" +
				"**Recommendation**\n\nNone. \n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/building_applications/index#configuring-project-creation[Configuring project creation]\n" +
				"* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/building_applications/index#quotas-setting-per-project[Quotas Setting Per Project]\n"

		}
	}

	return line + "\n"
}
