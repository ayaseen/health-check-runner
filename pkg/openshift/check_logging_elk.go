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
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var elk string

// Check Logging

func CheckElasticSearch() {

	out, _ := exec.Command("oc", "get", "Elasticsearch", "elasticsearch", "-n", "openshift-logging", "-o", "yaml").Output()
	//if err != nil {
	//	fmt.Println("Error executing oc command:", err)
	//	return
	//}

	elk = string(out)

	loggingConfigure, _ := checkLoggingConfigure()
	diskUsage := getDiskStorageUsage(elk)

	if elk != "[]" && loggingConfigure != "" {
		if diskUsage != -1 {
			//color.Red("ElasticSearch Disk storage usage more than 90%\t\tCRITICAL")
		}
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
		output.WriteString(LoggingElasticSearchProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}
}

func LoggingElasticSearchProcess(line string) string {

	if strings.HasPrefix(line, "<<OpenShift Logging Elasticsearch storage has sufficient space>>") {
		loggingConfigure, _ := checkLoggingConfigure()

		diskUsage := getDiskStorageUsage(elk)

		if elk != "[]" && loggingConfigure != "" {
			if diskUsage != -1 {
				return line + "\n\n| ElasticSearch Disk storage usage reaches 95%\n\n" + GetKeyChanges("recommended")
			} else {
				return line + "\n\n| ElasticSearch Disk storage usage is normal\n\n" + GetKeyChanges("nochange")
			}
		} else {
			return line + "\n\n|Not Applicable" + GetKeyChanges("na") + "\n\n"
		}

	}

	if strings.HasPrefix(line, "== OpenShift Logging Elasticsearch storage has sufficient space") {

		version, _ := getOpenShiftVersion()
		loggingConfigure, _ := checkLoggingConfigure()
		diskUsage := getDiskStorageUsage(elk)

		if elk != "[]" && loggingConfigure != "" {

			if diskUsage != -1 {
				return line + "\n\n" + GetChanges("recommended") +
					"\n\n**Observation**\n\nThe current level of free space on OpenShift Elasticsearch pods is not sufficient for now or in the near future." +
					"When the disk space reaches 95% used Elasticsearch has a protective function that locks the indices stopping new data from being written to them." +
					" This is to stop Elasticsearch from using any further disk causing the disk to become exhausted.\n\n" +
					"**Recommendation**\n\nExpand the available storage available to Elasticsearch or reduce the log retention period\n\n" +
					"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
					"html-single/logging/index#cluster-logging-external[Forwarding logs to external third-party logging systems]\n"
			} else {
				return line + "\n\n" + GetChanges("nochange") +
					"\n\n**Observation**\n\nThe current level of free space on OpenShift Elasticsearch pods is normal. \n\n" +
					"**Recommendation**\n\nNone \n\n" +
					"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
					"html-single/logging/index#cluster-logging-external[Forwarding logs to external third-party logging systems]\n"

			}
		} else {

			return line + "\n\n" + GetChanges("na") +
				"\n\n**Observation**\n\nOpenShift Logging is not installed or configured. \n\n" +
				"**Recommendation**\n\nYou need to deploy the logging subsystem to aggregate all the logs from your OpenShift Container Platform cluster," +
				" such as node system audit logs, application container logs, and infrastructure logs. \n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/logging/cluster-logging-deploying[Installing OpenShift Logging]\n" +
				"* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/logging/configuring-your-logging-deployment[Cluster Logging custom resource]\n" +
				"* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/logging/cluster-logging[Understanding Red Hat OpenShift Logging]\n"
		}

	}

	return line + "\n"
}

func getDiskStorageUsage(output string) int {
	conditions := strings.SplitAfter(output, "- conditions:")
	for _, condition := range conditions {
		if strings.Contains(condition, "type: NodeStorage") && strings.Contains(condition, "status: \"True\"") {
			message := extractMessage(condition)
			return extractDiskUsage(message)
		}
	}
	return -1
}

func extractMessage(condition string) string {
	re := regexp.MustCompile(`(?m)message:\s+(.*)\n`)
	match := re.FindStringSubmatch(condition)
	if len(match) == 2 {
		return match[1]
	}
	return ""
}

func extractDiskUsage(message string) int {
	re := regexp.MustCompile(`\((.*?)\)`)
	match := re.FindStringSubmatch(message)
	if len(match) == 2 {
		diskUsageStr := match[1]
		diskUsageStr = strings.TrimSuffix(diskUsageStr, "%")
		diskUsageFloat, err := strconv.ParseFloat(diskUsageStr, 64)
		if err != nil {
			log.Printf("Failed to convert disk usage value to float: %v\n", err)
			return -1
		}
		diskUsage := int(diskUsageFloat)
		return diskUsage
	}
	return -1
}
