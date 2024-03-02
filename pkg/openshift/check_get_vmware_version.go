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
	"gitlab.consulting.redhat.com/meta/health-check-runner/pkg/utils"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var vmwareVer string

func vmwareVersion() {

	if provider == "VSphere" {
		username, password, err := utils.GetCredentials()
		if err != nil {
			log.Fatalf("Failed to retrieve credentials: %v", err)
			return
		}

		vCenterURL, err := utils.GetServer()
		if err != nil {
			log.Fatalf("Failed to retrieve server: %v", err)
			return
		}

		vmwareVer, err = utils.GetVmwareVersion(vCenterURL, username, password)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		//color.HiCyan("VMware vSphere Version\t\t\t\t\tCHECKED")

	} else {
		//color.HiCyan("VMware vSphere Version\t\t\t\t\tSKIPPED")
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
		output.WriteString(vmwareVersionProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}
}

func vmwareVersionProcess(line string) string {

	if strings.HasPrefix(line, "VMware vSphere Version") {
		if vmwareVer != "" && provider == "VSphere" {
			return line + "\n\n| " + vmwareVer + " \n\n" + GetKeyChanges("nochange")
		} else {
			return line + "\n\n| Not Applicable \n\n" + GetKeyChanges("na")
		}

	}

	return line + "\n"
}
