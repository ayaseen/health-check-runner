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

var selfProvisioner []byte

func selfProvisioners() {
	selfProvisioner, err := exec.Command("oc", "describe", "clusterrolebindings", "self-provisioners").Output()
	if err != nil {
		fmt.Println("Error: Failed to retrieve cluster role binding information")

	}

	if string(selfProvisioner) == "" {
		color.Green("Self-provisioner is disabled\t\t\t\tPASSED")

	} else if strings.Contains(string(selfProvisioner), "system:authenticated:oauth") {
		color.Red("Self-provisioner is disabled\t\t\t\tFAILED")

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
		output.WriteString(selfProvisionerProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}

}

func checkSelfProvisioners() (string, error) {
	out, err := exec.Command("oc", "describe", "clusterrolebindings", "self-provisioners").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil

}

func selfProvisionerProcess(line string) string {
	if strings.HasPrefix(line, "<<Self Provisioner Enabled>>") {
		if string(selfProvisioner) == "" {
			return line + "\n\n| Self Provisioner is not enabled \n\n" + GetKeyChanges("recommended")
		} else {
			return line + "\n\n| NA \n\n" + GetKeyChanges("nochange")
		}

	}

	if strings.HasPrefix(line, "== Self Provisioner Enabled") {

		selfProvisionerRole, err := checkSelfProvisioners()
		if err != nil {
			return line + " Error checking self-provisioner role: " + err.Error() + "\n"
		}
		version, _ := getOpenShiftVersion()

		if string(selfProvisioner) == "" {
			return line + "\n\n" + GetChanges("recommended") + "\n[source,bash]\n----\n" + selfProvisionerRole + "\n----\n" +
				"\n\n**Observation**\n\nSelf Provisioner is enabled for system:authenticated:oauth. Self Provisioner should be disabled to avoid uncontrolled namespace creation. \n\n" +
				"**Recommendation**\n\nRemove self-provisioner from the system:authenticated:oauth group\n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/building_applications/index#disabling-project-self-provisioning_configuring-project-creation[Disabling project self-provisioning]\n"

		} else {
			return line + "\n\n" + GetChanges("nochange") + "\n[source,bash]\n----\n" + selfProvisionerRole + "\n----\n" +
				"\n\n**Observation**\n\nSelf Provisioner is disabled for system:authenticated:oauth.\n\n" +
				"**Recommendation**\n\nNone\n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/building_applications/index#disabling-project-self-provisioning_configuring-project-creation[Disabling project self-provisioning]\n"

		}
	}

	return line + "\n"
}
