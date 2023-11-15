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

func installationType() {

	var err error
	ocpInstallType, err := exec.Command("oc", "get", "cm", "-n", "openshift-config", "openshift-install").Output()

	if strings.Contains(string(ocpInstallType), "openshift-install") {
		color.HiCyan("Installation Type is IPI\t\t\t\tCHECKED")

	} else {
		color.HiCyan("Installation Type is UPI\t\t\t\tCHECKED")
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
		output.WriteString(installationTypeProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}
}

func checkInstallType() (string, error) {

	// Get the Installation method type
	out, err := exec.Command("oc", "get", "cm", "-n", "openshift-config", "openshift-install").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func checkMachineSet() (string, error) {

	// Get the Installation method type
	out, err := exec.Command("oc", "get", "machinesets", "-n", "openshift-machine-api", "-o", "jsonpath={.items[*].spec.replicas}").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func installationTypeProcess(line string) string {

	if strings.HasPrefix(line, "<<Installation Type>>") {
		installType, _ := checkInstallType()

		if strings.Contains(installType, "openshift-install") {
			return line + "\n\n| OpenShift IPI is configured \n\n" + GetKeyChanges("nochange")
		} else {
			return line + "\n\n| OpenShift UPI is configured \n\n" + GetKeyChanges("nochange")
		}

	}

	// Check if UPI and has additional machineSets
	if strings.HasPrefix(line, "<<Check if user-provisioned infrastructure (UPI) using additional MachineSets>>") ||
		strings.HasPrefix(line, "<<Check if user-provisioned infrastructure (UPI) OpenShift Provisioning Automation exists>>") {
		installType, err := checkInstallType()

		machineSet, err := checkMachineSet()
		if err != nil {
			return line + " Error checking machine set: " + err.Error() + "\n"
		}

		if !strings.Contains(installType, "openshift-install") {
			if machineSet != "" {
				return line + "\n\n| Additional MachineSet in place \n\n" + GetKeyChanges("nochange")
			} else {
				return line + "\n\n| No additional MachineSet \n\n" + GetKeyChanges("nochange")

			}
		} else {
			return line + "\n\n|Not Applicable" + GetKeyChanges("na") + "\n\n"
		}

	}

	// Check When IPI is configured (Set Load Balancer NA)
	if strings.HasPrefix(line, "<<Load Balancer Type>>") ||
		strings.HasPrefix(line, "<<Load Balancer Health Checks Enabled>>") ||
		strings.HasPrefix(line, "<<Load Balancer Balancing Algorithm>>") ||
		strings.HasPrefix(line, "<<Load Balancer SSL Settings>>") ||
		strings.HasPrefix(line, "<<Load Balancer VIPs Consistently Configured>>") {

		installType, _ := checkInstallType()

		if strings.Contains(installType, "openshift-install") {
			return line + "\n\n| IPI provided internal HAProxy \n\n" + GetKeyChanges("na")

		} else {
			return line + "\n\n|" + GetKeyChanges("eval") + "\n\n"
		}

	}

	if strings.HasPrefix(line, "== Installation Type") {
		installType, _ := checkInstallType()

		version, _ := getOpenShiftVersion()

		if strings.Contains(installType, "openshift-install") {
			return line + "\n\n" + GetChanges("nochange") +
				"\n\n**Observation**\n\nOpenShift installation Method is IPI\n\n" +
				"**Recommendation**\n\nNone\n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/installing/index\n"
		} else {
			return line + "\n\n" + GetChanges("nochange") +
				"\n\n**Observation**\n\nOpenShift installation Method is UPI\n\n" +
				"**Recommendation**\n\nNone\n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/installing/index\n"
		}
	}

	// Check When IPI is configured (Set Load Balancer NA)
	if strings.HasPrefix(line, "== Load Balancer Type") ||
		strings.HasPrefix(line, "== Load Balancer Health Checks Enabled") ||
		strings.HasPrefix(line, "== Load Balancer Balancing Algorithm") ||
		strings.HasPrefix(line, "== Load Balancer SSL Settings") ||
		strings.HasPrefix(line, "== Load Balancer VIPs Consistently Configured") {

		installType, _ := checkInstallType()
		version, _ := getOpenShiftVersion()

		if strings.Contains(installType, "openshift-install") {

			return line + "\n\n" + GetChanges("na") +
				"\n\n**Observation**\n\nIPI provided internal VIP HAProxy\n\n" +
				"**Recommendation**\n\nNone\n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/installing/index\n"
		} else {

			return line + "\n\n" + GetChanges("eval") +
				"\n\n**Observation**\n\nTo Be Evaluated\n\n" +
				"**Recommendation**\n\nTo Be Evaluated\n\n"
		}
	}
	// UPI machineSet

	if strings.HasPrefix(line, "== Check if user-provisioned infrastructure (UPI) using additional MachineSets") ||
		strings.HasPrefix(line, "== Check if user-provisioned infrastructure (UPI) OpenShift Provisioning Automation exists") {
		installType, _ := checkInstallType()

		machineSet, err := checkMachineSet()
		if err != nil {
			return line + " Error checking machine set: " + err.Error() + "\n"
		}

		version, err := getOpenShiftVersion()
		if err != nil {
			return line + " Error getting OpenShift version: " + err.Error() + "\n"
		}

		if !strings.Contains(installType, "openshift-install") {
			if machineSet != "" {
				return line + "\n\n" + GetChanges("nochange") +
					"\n\n**Observation**\n\nOpenShift installation Method is UPI and has additional machineSet\n\n" +
					"**Recommendation**\n\nNone\n\n" +
					"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
					"/html-single/installing/index\n"
			} else {
				return line + "\n\n" + GetChanges("nochange") +
					"\n\n**Observation**\n\nOpenShift installation Method is UPI and has no additional machineSet\n\n" +
					"**Recommendation**\n\nNone\n\n" +
					"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
					"/html-single/installing/index\n"
			}
		} else {
			return line + "\n\n" + GetChanges("na") +
				"\n\n**Observation**\n\nOpenShift installation Method is IPI\n\n" +
				"**Recommendation**\n\nNone\n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/installing/index\n"
		}
	}

	return line + "\n"
}
