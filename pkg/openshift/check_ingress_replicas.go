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

func ingressControllerReplica() {

	ingressReplica, err := exec.Command("oc", "get", "ingresscontroller/default", "-n", "openshift-ingress-operator",
		"-o", `jsonpath='{.spec.replicas}'`).Output()

	if err != nil {
		fmt.Println("Error: Failed to retrieve operator ingresscontroller information")
	}

	// Remove the surrounding quotes from the output.
	ingressReplicaCount := strings.Trim(string(ingressReplica), "'")

	if string(ingressReplicaCount) != "3" {
		//color.Red("Default OpenShift ingress replica match\t\tFAILED")
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
		output.WriteString(IngressReplicaProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}
}

func checkIngressReplica() (string, error) {

	// Get the ingress controller type
	out, err := exec.Command("oc", "get", "ingresscontroller/default", "-n", "openshift-ingress-operator",
		"-o", `jsonpath='{.spec.replicas}'`).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func getIngressReplica() (string, error) {

	// Get the ingress controller type
	out, err := exec.Command("oc", "get", "po", "-n", "openshift-ingress").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func IngressReplicaProcess(line string) string {
	if strings.HasPrefix(line, "<<Ingress Controller Replica Count>>") {
		ingressReplica, err := checkIngressReplica()
		if err != nil {
			return line + " Error checking default ingress: " + err.Error() + "\n"
		}

		// Remove the surrounding quotes from the output.
		ingressReplicaCount := strings.Trim(ingressReplica, "'")

		if string(ingressReplicaCount) != "3" {
			return line + "\n\n| Default OpenShift ingress replica not match\n\n" + GetKeyChanges("recommended")
		} else {
			return line + "\n\n| Default OpenShift ingress replica match\n\n" + GetKeyChanges("nochange")
		}
	}

	if strings.HasPrefix(line, "== Ingress Controller Replica Count") {
		ingressReplica, _ := checkIngressReplica()
		ingressReplicaPods, _ := getIngressReplica()
		version, _ := getOpenShiftVersion()

		// Remove the surrounding quotes from the output.
		ingressReplicaCount := strings.Trim(ingressReplica, "'")

		if string(ingressReplicaCount) != "3" {
			return line + "\n\n" + GetChanges("recommended") + "\n\n[source, bash]\n----\n" + ingressReplicaPods + "\n----\n" +
				"\n\n**Observation**\n\nDefault OpenShift ingress replica not match\n\n" +
				"**Recommendation**\n\nOpenShift ingress pods should place distributed on infra nodes with replica count three (3). \n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/networking/index#configuring-ingress[Configure Ingress Controller]\n"

		} else {
			return line + "\n\n" + GetChanges("nochange") + "\n\n[source, bash]\n----\n" + ingressReplicaPods + "\n----\n" +
				"\n\n**Observation**\n\nDefault OpenShift ingress replica match\n\n" +
				"**Recommendation**\n\nNone. \n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/networking/index#configuring-ingress[Configure Ingress Controller]\n"

		}
	}

	return line + "\n"
}
