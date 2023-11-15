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
	"context"
	"fmt"
	"github.com/fatih/color"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"path/filepath"
	"strings"
)

const probeDescription = `
=== What is a probe?

A probe is an OpenShift action that periodically performs diagnostics on a running container. Probes can be added in template or via console. There are three types of probes: Readiness, Liveness and Startup

Readiness Probe: Identifies when a container is able to handle external traffic received from a service. Even though the container is running it should not receive any requests.
**If the test fails for readiness probe the container will be taken out of service**.

Liveness Probe: Checks whether a pod is healthy by running a command or making a network request inside the container.
**If a test fails then the container is restarted**.

Startup Probe: Provides a way to defer the execution of liveness and readiness probes until a container indicates it is able to handle them. Kubernetes will not direct the other probe types to a container if it has a startup probe that has not yet succeeded.
**If a test fails then the container is restarted**.

OpenShift provided three options that control these probes:

1.  Open a TCP socket on the pod IP
2.  Perform an HTTP GET against a URL on a container that must return 200 OK
3.  Run a command in the container that must return exit code 0`

// Initialize counter for workloads without probes
var workloadsWithoutProbes, workloadsWithProbes, totalProjects int

func applicationProbes() {

	appName := ""

	config, err := getClusterConfig()
	if err != nil {
		fmt.Println("Error getting cluster config:", err)
		os.Exit(1)
	}

	ocClient, err := getClientSet(config)
	if err != nil {
		fmt.Println("Error getting client set:", err)
		os.Exit(1)
	}

	// Get the list of projects in the OpenShift cluster
	projects, err := ocClient.CoreV1().Namespaces().List(context.Background(), v1.ListOptions{})
	if err != nil {
		panic(err)
	}

	// Check each project for the application workloads
	for _, project := range projects.Items {
		// Exclude projects with the prefix 'openshift' or 'kube'
		if strings.HasPrefix(project.Name, "openshift") ||
			strings.HasPrefix(project.Name, "kube") ||
			strings.Contains(project.Name, "operator") ||
			strings.Contains(project.Name, "multicluster") ||
			strings.Contains(project.Name, "open-cluster") {
			continue
		}

		// Get the list of DeploymentConfigs in the project
		dcs, err := ocClient.AppsV1().Deployments(project.Name).List(context.Background(), v1.ListOptions{})
		if err != nil {
			panic(err)

		}

		// Check each DeploymentConfig for the app
		for _, dc := range dcs.Items {
			// Check if the DeploymentConfig is for an operator and skip it
			if strings.Contains(dc.Name, "operator") {
				continue
			}
			// Check if the app name is specified and skip to the next if it does not match
			if appName != "" && dc.Name != appName {
				continue
			}

			// Check if the Readiness Probe is configured
			if dc.Spec.Template.Spec.Containers[0].ReadinessProbe != nil {
				workloadsWithProbes++
				totalProjects++
			} else {
				//fmt.Printf("Readiness Probe is not configured for the application %s in project %s\n", dc.Name, project.Name)
				workloadsWithoutProbes++
			}

			// Check if the Liveness Probe is configured
			if dc.Spec.Template.Spec.Containers[0].LivenessProbe != nil {
				workloadsWithProbes++
			} else {
				//fmt.Printf("Liveness Probe is not configured for the application %s in project %s\n", dc.Name, project.Name)
				workloadsWithoutProbes++
			}
		}

	}
	// Create the output file for writing
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
		output.WriteString(applicationProbesProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}
	// Print the total number of workloads without probes
	if workloadsWithProbes > totalProjects {
		color.Green("Liveness and Readiness are configured\t\t\tPASSED")

	} else {
		color.Red("Liveness and Readiness are configured\t\t\tFAILED")
	}
}

func applicationProbesProcess(line string) string {

	if strings.HasPrefix(line, "<<Readiness and Liveness Probes>>") {

		if workloadsWithProbes == totalProjects {
			return line + "\n\n|Readiness and Liveness Probes not configured\n\n" + GetKeyChanges("recommended") + "\n\n"
		} else {
			return line + "\n\n|Readiness and Liveness Probes are configured\n\n" + GetKeyChanges("nochange") + "\n\n"
		}

	}

	if strings.HasPrefix(line, "== Readiness and Liveness Probes") {
		version, _ := getOpenShiftVersion()

		if workloadsWithProbes == totalProjects {
			return line + "\n\n" + GetChanges("recommended") + "\n\n" + probeDescription + "\n\n" +
				"\n\n**Observation**\n\nNot all Deployments, DeploymentConfigs or StatefulSets specify Liveness," +
				" Readiness and Startup Probes. This could lead to running pods that are in an unhealthy state.\n" +
				"**Recommendation**\n\nDevelopers involvement in creating the probes are important for the application's success.\n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/building_applications/index#application-health[Application health]\n"
		} else {
			return line + "\n\n" + GetChanges("nochange") +
				"\n\n**Observation**\n\nContainer probes configured for workloads." + "\n\n" + probeDescription + "\n\n" +
				"**Recommendation**\n\nNone\n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/building_applications/index#application-health[Application health]\n"
		}
	}
	return line + "\n"
}
