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

// var infraNodesExist bool
var infraPool string

// Check cluster operators status
func checkInfraConfigPool() {

	infraPoolConfig, err := exec.Command("oc", "get", "machineconfigpool", "-o", "jsonpath={.items[*].metadata.name}").Output()

	infraPool = string(infraPoolConfig)

	if !strings.Contains(infraPool, "infra") {
		color.Red("Infrastructure Pool is configured\t\t\tFAILED")
	} else {
		color.Green("Infrastructure Pool is configured\t\t\tPASSED")
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
		output.WriteString(InfraPoolProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}

}

func getInfraPool() (string, error) {

	// Get node status
	out, err := exec.Command("oc", "get", "machineconfigpool").CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func InfraPoolProcess(line string) string {

	// To populate Key Summary section
	if strings.HasPrefix(line, "<<Infra machine config pool defined>>") {
		if !strings.Contains(infraPool, "infra") {
			return line + "\n\n| Infra machine config pool not defined\n\n" + GetKeyChanges("recommended")
		} else {
			return line + "\n\n| Infra machine config pool defined\n\n" + GetKeyChanges("nochange")
		}
	}

	// To populate body section
	if strings.HasPrefix(line, "== Infra machine config pool defined") {
		getInfraConfigPool, _ := getInfraPool()
		version, _ := getOpenShiftVersion()

		//Render the change status
		if !strings.Contains(infraPool, "infra") {
			return line + "\n\n" + GetChanges("recommended") + "\n\n[source, bash]\n----\n" + getInfraConfigPool + "\n----\n" +
				"\n\n**Observation**\n\nInfra machine config pool not defined\n\n" +
				"**Recommendation**\n\nIn a production deployment, it is recommended that you deploy at least three machine sets to hold infrastructure components." +
				" Both OpenShift Logging and Red Hat OpenShift Service Mesh deploy Elasticsearch, which requires three instances to be installed on different nodes." +
				" Each of these nodes can be deployed to different availability zones for high availability.\n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/machine_management/index#creating-infrastructure-machinesets\n"

		} else {
			return line + "\n\n" + GetChanges("nochange") + "\n\n[source, bash]\n----\n" + getInfraConfigPool + "\n----\n" +
				"\n\n**Observation**\n\nInfra machine config pool defined. \n\n" +
				"**Recommendation**\n\nIn a production deployment, it is recommended that you deploy at least three machine sets to hold infrastructure components." +
				" Both OpenShift Logging and Red Hat OpenShift Service Mesh deploy Elasticsearch, which requires three instances to be installed on different nodes." +
				" Each of these nodes can be deployed to different availability zones for high availability.\n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/machine_management/index#creating-infrastructure-machinesets\n"
		}
	}
	return line + "\n"
}
