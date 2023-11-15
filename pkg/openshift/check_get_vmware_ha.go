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
	"gitlab.consulting.redhat.com/meta/health-check-runner/pkg/utils"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var nodeNum int
var vmwareHAEnabled bool

func vmwareHA() {

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

		dataCenter, err := utils.GetDatacenter()
		if err != nil {
			log.Fatalf("Failed to retrieve server: %v", err)
			return
		}

		nodeNum, vmwareHAEnabled, err = utils.IsHAClusterEnabled(vCenterURL, username, password, dataCenter)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		color.HiCyan("vCenter Highly Available\t\t\t\tCHECKED")

	} else {
		color.HiCyan("vCenter Highly Available\t\t\t\tSKIPPED")
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
		output.WriteString(vmwareHAProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}
}

func vmwareHAProcess(line string) string {

	if strings.HasPrefix(line, "<<vCenter Highly Available>>") ||
		strings.HasPrefix(line, "<<vSphere HA Enabled>>") {
		if provider == "VSphere" {
			if nodeNum >= 4 && vmwareHAEnabled == true {
				return line + "\n\n| vCenter Highly Available \n\n" + GetKeyChanges("nochange")
			} else {
				return line + "\n\n| Only 2 esxi hosts are provisioned, recommend to have minimum 4 \n\n" + GetKeyChanges("recommended")
			}
		} else {
			return line + "\n\n| Not Applicable \n\n" + GetKeyChanges("na")
		}

	}

	if strings.HasPrefix(line, "== vCenter Highly Available") ||
		strings.HasPrefix(line, "== vSphere HA Enabled") {

		if provider == "VSphere" {
			if nodeNum >= 4 && vmwareHAEnabled == true {
				return line + "\n\n" + GetChanges("nochange") +
					"\n\n**Observation**\n\nvCenter Highly Available\n\n" +
					"**Recommendation**\n\nNone\n\n"
			} else {
				return line + "\n\n" + GetChanges("recommended") +
					"\n\n**Observation**\n\nOnly 2 esxi hosts are provisioned, recommend to have minimum 4\n\n" +
					"**Recommendation**\n\n* https://docs.vmware.com/en/VMware-vSphere/7.0/com.vmware.vsphere.avail.doc/GUID-8BDD82D8-B6BF-4C1B-9E3F-6395CC2FAFD5.html[vCenter HA Cluster Configuration]\n\n" +
					"* https://docs.vmware.com/en/VMware-vSphere/7.0/com.vmware.vsphere.avail.doc/GUID-4A626993-A829-495C-9659-F64BA8B560BD.html[vCenter High Availability]\n"
			}

		} else {
			return line + "\n\n" + GetChanges("na") +
				"\n\n**Observation**\n\n Not Applicable\n\n" +
				"**Recommendation**\n\nNone\n\n"
		}
	}

	return line + "\n"
}
