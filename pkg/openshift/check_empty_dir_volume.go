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
	appsv1client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var emptyDirCount int

func emptyDirVolume() {
	config, err := getClusterConfig()
	if err != nil {
		fmt.Println("Error getting cluster config:", err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Println("Error creating client:", err)
		os.Exit(1)
	}

	appsClientset, err := appsv1client.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating apps client: %v", err)
	}

	excludedPrefixes := []string{
		"openshift",
		"kube",
		"operator",
		"multicluster",
		"open-cluster",
		"default",
	}

	projects, err := clientset.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Error listing projects: %v", err)
	}

	for _, project := range projects.Items {
		if isExcludedProject(project.Name, excludedPrefixes) {
			continue
		}

		pods, err := clientset.CoreV1().Pods(project.Name).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			log.Fatalf("Error listing pods in project %s: %v", project.Name, err)
		}

		for _, pod := range pods.Items {
			isEmptyDirVolume := false

			volumes := pod.Spec.Volumes
			for _, volume := range volumes {
				if volume.EmptyDir != nil {
					isEmptyDirVolume = true
					break
				}
			}

			if isEmptyDirVolume {
				emptyDirCount++
			}
		}

		deployments, err := appsClientset.DeploymentConfigs(project.Name).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			log.Fatalf("Error listing deployment configs in project %s: %v", project.Name, err)
		}

		for _, deployment := range deployments.Items {
			podTemplate := deployment.Spec.Template

			pods, err := clientset.CoreV1().Pods(project.Name).List(context.Background(), metav1.ListOptions{})
			if err != nil {
				log.Fatalf("Error listing pods in project %s: %v", project.Name, err)
			}

			for _, pod := range pods.Items {
				if pod.Name == podTemplate.Name {
					isEmptyDirVolume := false

					volumes := podTemplate.Spec.Volumes
					for _, volume := range volumes {
						if volume.EmptyDir != nil {
							isEmptyDirVolume = true
							break
						}
					}

					if isEmptyDirVolume {
						emptyDirCount++
					}
				}
			}
		}
	}

	//fmt.Printf("Number of pods with EmptyDir volume: %d\n", emptyDirCount)

	if emptyDirCount == 0 {
		color.Green("EmptyDir Volumes not in use\t\t\t\tPASSED")
	} else {
		color.Red("EmptyDir Volumes in use\t\t\t\tFAILED")
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
	scanner := bufio.NewScanner(inputFile)

	// Create a buffer to hold the output
	var output strings.Builder

	// Process each line of the input file
	for scanner.Scan() {
		line := scanner.Text()
		output.WriteString(emptyDirVolumeProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}

}

func emptyDirVolumeProcess(line string) string {
	// To populate Key Summary section
	if strings.HasPrefix(line, "<<EmptyDir Volumes in use>>") {
		// Render the change status
		if emptyDirCount == 0 {
			return line + "\n\n| NA \n\n" + GetKeyChanges("nochange")
		} else {
			return line + "\n\n| Some Application Pods use EmptyDir \n\n" + GetKeyChanges("advisory")
		}
	}

	if strings.HasPrefix(line, "== EmptyDir Volumes in use") {

		version, _ := getOpenShiftVersion()

		// Render the change status
		if emptyDirCount == 0 {
			return line + "\n\n" + GetChanges("nochange") +
				"\n\n**Observation**\n\nApplication pods use persistent storage. \n\n" +
				"**Recommendation**\n\nNA. \n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"html-single/nodes/index#nodes-containers-volumes[Container Storage]\n"

		} else {
			return line + "\n\n" + GetChanges("advisory") +
				"\n\n**Observation**\n\nSome Application Pods use EmptyDir. \n\n" +
				"**Recommendation**\n\nFiles in a container are ephemeral." +
				"As such, when a container crashes or stops, the data is lost. You can use volumes to persist the data used by the containers in a pod. \n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"html-single/nodes/index#nodes-containers-volumes[Container Storage]\n"

		}

	}
	return line + "\n"
}

func isExcludedProject(projectName string, excludedPrefixes []string) bool {
	for _, prefix := range excludedPrefixes {
		if strings.HasPrefix(projectName, prefix) {
			return true
		}
	}
	return false
}
