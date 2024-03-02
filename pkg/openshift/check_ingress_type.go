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

func IngressControllerType() {

	ingressType, err := exec.Command("oc", "get", "deployment/router-default", "-n", "openshift-ingress",
		"-o", `jsonpath={.metadata.labels.ingresscontroller\.operator\.openshift\.io\/owning-ingresscontroller}`).Output()

	if err != nil {
		fmt.Println("Error: Failed to retrieve operator ingresscontroller information")
	}

	if string(ingressType) != "default" {
		//color.Red("Default OpenShift ingress controller not in use\t\tFAILED")

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
		output.WriteString(IngressControllerTypeProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}
}

func checkIngressControllerType() (string, error) {

	// Get the ingress controller type
	out, err := exec.Command("oc", "get", "deployment/router-default", "-n", "openshift-ingress",
		"-o", `jsonpath={.metadata.labels.ingresscontroller\.operator\.openshift\.io\/owning-ingresscontroller}`).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func IngressControllerTypeProcess(line string) string {
	if strings.HasPrefix(line, "<<Ingress Controller Type>>") {
		ingressType, err := checkIngressControllerType()
		if err != nil {
			return line + " Error checking default ingress: " + err.Error() + "\n"
		}

		if string(ingressType) != "default" {
			return line + "\n\n| Default OpenShift ingress controller not in use\n\n" + GetKeyChanges("na")
		} else {
			return line + "\n\n| Default OpenShift ingress controller in use\n\n" + GetKeyChanges("nochange")
		}
	}

	if strings.HasPrefix(line, "== Ingress Controller Type") {
		ingressType, _ := checkIngressControllerType()
		version, _ := getOpenShiftVersion()

		if string(ingressType) != "default" {
			return line + "\n\n" + GetChanges("na") + "\n\n[source, yaml]\n---\ningresscontroller.operator.openshift.io/owning-ingresscontroller:" + ingressType + "\n----\n" +
				"\n\n**Observation**\n\nDefault OpenShift ingress controller not in use\n\n" +
				"**Recommendation**\n\nNot Applicable. \n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/networking/index#configuring-ingress[Configure Ingress Controller]\n"

		} else {
			return line + "\n\n" + GetChanges("nochange") + "\n\n[source, yaml]\n----\ningresscontroller.operator.openshift.io/owning-ingresscontroller:" +
				ingressType + "\n----\n" +
				"\n\n**Observation**\n\nDefault OpenShift ingress controller in use\n\n" +
				"**Recommendation**\n\nNone. \n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/networking/index#configuring-ingress[Configure Ingress Controller]\n"

		}
	}

	return line + "\n"
}
