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
	"bytes"
	"fmt"
	"github.com/fatih/color"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// NodeUsage represents the CPU and memory usage for a single node
type NodeUsage struct {
	Name   string
	CPU    string
	Memory string
	Status string
}

var cpu, memory float64

func nodeUsage() {

	// Get the output from the `oc adm top node` command
	out, err := exec.Command("oc", "adm", "top", "node").Output()
	if err != nil {
		color.HiYellow("Node usage not available (get nodes.metrics.k8s.io)\tREVISED")
		return
	}

	// boolean variable to keep track of whether a record with high CPU or memory usage has been found
	found := false

	// Parse the command output to retrieve the CPU and memory usage for each node
	var nodeUsages []NodeUsage
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		if match, _ := regexp.MatchString("^NAME", line); match {
			continue
		}
		fields := regexp.MustCompile(`\s+`).Split(line, -1)
		cpu, _ = strconv.ParseFloat(fields[2][:len(fields[2])-1], 64)
		memory, _ = strconv.ParseFloat(fields[4][:len(fields[4])-1], 64)

		nodeUsages = append(nodeUsages, NodeUsage{
			Name:   fields[0],
			CPU:    fields[2],
			Memory: fields[4],
		})
		if cpu > 50 || memory > 50 {
			found = true
			break
		} else {
			found = false

		}

	}
	if found == false {
		//color.Green("All nodes are within the range\t\t\t\tPASSED")
	} else {
		//color.HiYellow("Some node utilize more than 50%\t\t\tREVISED")
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
		output.WriteString(nodeUsageProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}

}

func checkNodeUsageResults() (string, error) {

	// Get nodes usage
	out, err := exec.Command("oc", "adm", "top", "nodes").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func nodeUsageProcess(line string) string {
	if strings.HasPrefix(line, "<<OpenShift Node Usage>>") {
		// Render the change status

		if cpu > 50 || memory > 50 {
			return line + "\n\n| Some node utilize more than 50% \n\n" + GetKeyChanges("advisory")
		} else {
			return line + "\n\n| NA \n\n" + GetKeyChanges("nochange")
		}
	}

	if strings.HasPrefix(line, "== OpenShift Node Usage") {
		usage, _ := checkNodeUsageResults()
		version, _ := getOpenShiftVersion()

		if cpu > 50 || memory > 50 {
			return line + "\n\n" + GetChanges("advisory") + "\n\n[source, bash]\n----\n" + usage + "\n----\n" +
				"\n\n**Observation**\n\nSome node utilize more than 50% \n\n" +
				"**Recommendation**\n\nYou require to add more resources to cluster nodes. \n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version

		} else {
			return line + "\n\n" + GetChanges("nochange") + "\n\n[source, bash]\n----\n" + usage + "\n----\n" +
				"\n\n**Observation**\n\nAll nodes within the usage range \n\n" +
				"**Recommendation**\n\nNone. \n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version

		}

	}

	return line + "\n"
}
