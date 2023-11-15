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

// Check project template

func ingressControllerPlacement() {

	ingressPlacement, err := exec.Command("oc", "get", "ingresscontroller/default", "-n", "openshift-ingress-operator",
		"-o", `jsonpath={.spec.nodePlacement.nodeSelector.matchLabels}`).Output()

	if err != nil {
		fmt.Println("Error: Failed to retrieve operator ingresscontroller information")
	}

	if strings.Contains(string(ingressPlacement), "node-role.kubernetes.io/infra") {
		color.Green("Ingress Controller Placement is set\t\t\tPASSED")
	} else {
		color.Red("Ingress Controller Placement not set\t\t\tFAILED")
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
		output.WriteString(IngressControllerPlacementProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}
}

func checkIngressControllerPlacement() (string, error) {

	// Get the ingress controller type
	out, err := exec.Command("oc", "get", "ingresscontroller/default", "-n", "openshift-ingress-operator",
		"-o", `jsonpath={.spec.nodePlacement.nodeSelector.matchLabels}`).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func getIngressNodePlacement() (string, error) {

	// Get the ingress controller type
	out, err := exec.Command("oc", "get", "po", "-n", "openshift-ingress", "-o", "wide").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func IngressControllerPlacementProcess(line string) string {
	if strings.HasPrefix(line, "<<Ingress Controller Placement>>") {
		ingressPlacement, err := checkIngressControllerPlacement()
		if err != nil {
			return line + " Error checking default ingress placement: " + err.Error() + "\n"
		}

		if strings.Contains(ingressPlacement, "node-role.kubernetes.io/infra") {
			return line + "\n\n| Ingress Controller Placement is set\n\n" + GetKeyChanges("nochange")
		} else {
			return line + "\n\n| Ingress Controller Placement not set\n\n" + GetKeyChanges("recommended")
		}
	}

	if strings.HasPrefix(line, "== Ingress Controller Placement") {
		ingressPlacement, _ := checkIngressControllerPlacement()
		ingressNodePlacement, _ := getIngressNodePlacement()
		version, _ := getOpenShiftVersion()

		if strings.Contains(ingressPlacement, "node-role.kubernetes.io/infra") {
			return line + "\n\n" + GetChanges("nochange") + "\n\n[source, bash]\n----\n" + ingressNodePlacement + "\n----\n" +
				"\n\n**Observation**\n\nDefault OpenShift ingress Placement is set.\n\n" +
				"**Recommendation**\n\nNone. \n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/networking/index#configuring-ingress[Configure Ingress Controller]\n"
		} else {
			return line + "\n\n" + GetChanges("recommended") + "\n\n[source, bash]\n----\n" + ingressNodePlacement + "\n----\n" +
				"\n\n**Observation**\n\nDefault OpenShift ingress Placement not set\n\n" +
				"**Recommendation**\n\nOpenShift ingress pods should place on infra nodes. \n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/networking/index#configuring-ingress[Configure Ingress Controller]\n"
		}
	}

	return line + "\n"
}
