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

var kubeletConfigValidate []byte

// Check project template

func KubeletConfig() {

	var err error
	kubeletConfigValidate, err = exec.Command("oc", "get", "kubeletconfigs").Output()
	if err != nil {
		fmt.Println("Error: Failed to retrieve kubeletconfigs information")

	}

	//
	//if string(kubeletConfigValidate) == "" {
	//	color.Red("Kubelet Configuration (Garbage Collection) is set\tFAILED")
	//
	//} else {
	//
	//	color.Green("Kubelet Configuration (Garbage Collection) is set\tPASSED")
	//
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
		output.WriteString(kubeletConfigProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}
}

func checkKubeletConfig() (string, error) {

	// Get the encryption type of the etcd server
	out, err := exec.Command("oc", "get", "kubeletconfigs").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func kubeletConfigProcess(line string) string {
	if strings.HasPrefix(line, "<<Kubelet Configuration Overridden>>") {
		if string(kubeletConfigValidate) == "" {
			return line + "\n\n| Kubelet Configuration Overridden not configured\n\n" + GetKeyChanges("recommended")
		} else {
			return line + "\n\n| Kubelet Configuration Overridden is configured\n\n" + GetKeyChanges("nochange")
		}
	}

	if strings.HasPrefix(line, "<<Ensure that garbage collection is configured as appropriate>>") {
		if string(kubeletConfigValidate) == "" {
			return line + "\n\n| Garbage Collection configuration should be revised\n\n" + GetKeyChanges("recommended")
		} else {
			return line + "\n\n| Garbage Collection configuration is set\n\n" + GetKeyChanges("nochange")
		}
	}

	if strings.HasPrefix(line, "== Kubelet Configuration Overridden") {
		kubeletConf, _ := checkKubeletConfig()
		version, _ := getOpenShiftVersion()

		if string(kubeletConfigValidate) == "" {
			return line + "\n\n" + GetChanges("recommended") +
				"\n\n**Observation**\n\nKubelet configuration is not enabled to provide garbage collection.\n\n" +
				"**Recommendation**\n\nConfiguring garbage collection for containers and images.\n\n" +
				"*Reference Link(s)*\n\n" +
				"* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/nodes/index#nodes-nodes-garbage-collection[Freeing Garbage Collection]\n"

		} else {
			return line + "\n\n" + GetChanges("nochange") + "\n\n[source, yaml]\n----\n" + kubeletConf + "\n----\n" +
				"\n\n**Observation**\n\nKubelet Configuration Overridden is configured\n\n" +
				"**Recommendation**\n\nNone. \n\n" +
				"*Reference Link(s)*\n\n" +
				"* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/nodes/index#nodes-nodes-garbage-collection[Freeing Garbage Collection]\n"

		}
	}

	if strings.HasPrefix(line, "== Ensure that garbage collection is configured as appropriate") {
		kubeletConf, _ := checkKubeletConfig()
		version, _ := getOpenShiftVersion()

		if string(kubeletConfigValidate) == "" {
			return line + "\n\n" + GetChanges("recommended") +
				"\n\n**Observation**\n\nGarbage Collection configuration is not enabled to provide garbage collection.\n\n" +
				"**Recommendation**\n\nConfiguring garbage collection for containers and images.\n\n" +
				"*Reference Link(s)*\n\n" +
				"* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/nodes/index#nodes-nodes-garbage-collection[Freeing Garbage Collection]\n"

		} else {
			return line + "\n\n" + GetChanges("nochange") + "\n\n[source, yaml]\n----\n" + kubeletConf + "\n----\n" +
				"\n\n**Observation**\n\nGarbage Collection configuration is set\n\n" +
				"**Recommendation**\n\nNone. \n\n" +
				"*Reference Link(s)*\n\n" +
				"* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/nodes/index#nodes-nodes-garbage-collection[Freeing Garbage Collection]\n"

		}
	}

	return line + "\n"
}
