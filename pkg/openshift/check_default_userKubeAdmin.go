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
var defaultUserValidate []byte

func defaultOpenShiftUser() {

	var err error
	defaultUserValidate, err = exec.Command("oc", "get", "secrets", "kubeadmin", "-n", "kube-system").Output()
	//if err != nil {
	//	fmt.Println("Error: Unable to connect to OpenShift Cluster")
	//	os.Exit(1)
	//
	//}

	//if string(defaultUserValidate) != "" {
	//	color.Red("Kubeadmin user is absent\t\t\t\tFAILED")
	//
	//} else {
	//	color.Green("Kubeadmin user is absent\t\t\t\tPASSED")
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
		output.WriteString(defaultUserProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}
}

func checkDefaultOpenshiftUser() (string, error) {

	// Get the encryption type of the etcd server
	out, err := exec.Command("oc", "get", "secrets", "kubeadmin", "-n", "kube-system").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func defaultUserProcess(line string) string {

	if strings.HasPrefix(line, "<<Kubeadmin user disabled>>") {

		if string(defaultUserValidate) != "" {
			return line + "\n\n| Default OpenShift `kubeadmin` is present \n\n" + GetKeyChanges("required")
		} else {
			return line + "\n\n| NA \n\n" + GetKeyChanges("nochange")
		}

	}

	if strings.HasPrefix(line, "== Kubeadmin user disabled") {

		defaultUser, _ := checkDefaultOpenshiftUser()
		version, _ := getOpenShiftVersion()

		if string(defaultUserValidate) != "" {
			return line + "\n\n" + GetChanges("required") + "\n\n[source, bash]\n----\n" + defaultUser + "\n----\n" +
				"\n\n**Observation**\n\nKubeadmin user is not disabled \n\n" +
				"**Recommendation**\n\nThis user is for temporary post installation steps and should be removed to avoid any potential security breach\n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/authentication_and_authorization/removing-kubeadmin[Removing Kubeadmin user]\n"
		} else {
			return line + "\n\n" + GetChanges("nochange") + "\n\n[source, bash]\n----\n" + defaultUser + "\n----\n" +
				"\n\n**Observation**\n\nKubeadmin user is disabled, no action is required. \n\n" +
				"**Recommendation**\n\nNone\n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/authentication_and_authorization/removing-kubeadmin[Removing Kubeadmin user]\n"
		}
	}

	return line + "\n"
}
