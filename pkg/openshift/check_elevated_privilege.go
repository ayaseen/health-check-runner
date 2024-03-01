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
	ocappsv1client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	k8sappsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"os"
	"path/filepath"
	"strings"
)

var (
	foundPodsWithPrivileged bool
	table                   Table
)

// Table represents a markdown table
type Table struct {
	Rows []TableRow
}

// TableRow represents a row in the table
type TableRow struct {
	Namespace        string
	Deployment       string
	DeploymentConfig string
}

// AddRow adds a new row to the table
func (t *Table) AddRow(namespace, deployment, deploymentConfig string) {
	row := TableRow{
		Namespace:        namespace,
		Deployment:       deployment,
		DeploymentConfig: deploymentConfig,
	}
	t.Rows = append(t.Rows, row)
}

// GenerateMarkdown generates the markdown representation of the table
func (t *Table) GenerateMarkdown() string {
	var builder strings.Builder

	// Header row
	builder.WriteString("{set:cellbgcolor:!}\n[cols=\"1,3\", options=header]\n|===\n")
	builder.WriteString("|Namespace\n|Deployment\n\n")

	// Data rows
	for _, row := range t.Rows {
		builder.WriteString("{set:cellbgcolor:!}\n|")
		builder.WriteString(row.Namespace)
		builder.WriteString("\n|")
		builder.WriteString(row.Deployment)
		builder.WriteString("\n")
		builder.WriteString("{set:cellbgcolor:!}\n|")
		builder.WriteString(row.Namespace)
		builder.WriteString("\n|")
		builder.WriteString(row.DeploymentConfig)
		builder.WriteString("\n|===\n")
	}

	return builder.String()
}

// Check cluster operators status
func checkElevatedPrivileges() {

	config, err := getClusterConfig()
	if err != nil {
		fmt.Println("Error getting cluster config:", err)
		os.Exit(1)
	}

	// Create the Kubernetes clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	// Create the OpenShift Apps client
	ocAppsClient, err := ocappsv1client.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	// Create the table
	table = Table{}

	// Retrieve all namespaces
	namespaces, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err)
	}

	// Iterate through namespaces and retrieve running pods
	for _, namespace := range namespaces.Items {
		if shouldExcludeNamespace(namespace.Name) {
			continue
		}

		pods, err := clientset.CoreV1().Pods(namespace.Name).List(context.TODO(), metav1.ListOptions{
			LabelSelector: labels.Set{}.AsSelector().String(),
		})
		if err != nil {
			fmt.Printf("Error retrieving pods in namespace %s: %v\n", namespace.Name, err)
			continue
		}

		for _, pod := range pods.Items {
			if shouldExcludePod(pod.Name) {
				continue
			}

			for _, container := range pod.Spec.Containers {
				if container.SecurityContext != nil && container.SecurityContext.Privileged != nil && *container.SecurityContext.Privileged {
					foundPodsWithPrivileged = true
					// Check if the pod is associated with a Deployment
					if deployment := getAssociatedDeployment(clientset, namespace.Name, pod); deployment != nil {
						// Add row to the table
						table.AddRow(namespace.Name, deployment.Name, "")
					}

					// Check if the pod is associated with a DeploymentConfig
					if deploymentConfig := getAssociatedDeploymentConfig(ocAppsClient, namespace.Name, pod); deploymentConfig != "" {

						// Update the last row with DeploymentConfig value
						lastRowIndex := len(table.Rows) - 1
						lastRow := &table.Rows[lastRowIndex]
						lastRow.DeploymentConfig = deploymentConfig
					}

					break
				}
			}
		}
	}

	//if foundPodsWithPrivileged == false {
	//	color.Green("Elevated Privileged\t\t\t\t\tPASSED")
	//} else {
	//	// Generate and print the markdown table
	//	color.Red("Elevated Privileged\t\t\t\t\tFAILED")
	//
	//	//markdown := table.GenerateMarkdown()
	//	//fmt.Println(markdown)
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
		output.WriteString(elevatedPrivilegesResultsProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}
}

func elevatedPrivilegesResultsProcess(line string) string {

	//version, err := getOpenShiftVersion()
	//if err != nil {
	//	return line + " Error getting OpenShift version: " + err.Error() + "\n"
	//}

	// To populate Key Summary section
	if strings.HasPrefix(line, "<<Elevated Privileges>>") {
		if foundPodsWithPrivileged == false {
			return line + "\n\n| No user workload use Elevated Privileges in unprivileged way \n\n" + GetKeyChanges("nochange")
		} else {
			return line + "\n\n| At least one user workload use an unjustified non restricted SCC \n\n" + GetKeyChanges("recommended")
		}
	}

	// To populate body section
	if strings.HasPrefix(line, "== Elevated Privileges") {

		markdown := table.GenerateMarkdown()
		version, _ := getOpenShiftVersion()

		//Render the change status
		if foundPodsWithPrivileged == true {
			return line + "\n\n" + GetChanges("recommended") + "\n\n" +
				"\n\n**Observation**\n\nAt least one user workload use an unjustified non restricted SCC \n\n" + markdown +
				"**Recommendation**\n\nSCC allows host access to namespaces, file systems, and PIDs. It should only be used by trusted pods. Grant with caution.\n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/authentication_and_authorization/index#managing-pod-security-policies[Default Security Context Constraint]\n"

		} else {
			return line + "\n\n" + GetChanges("nochange") + "\n\n" +
				"\n\n**Observation**\n\nNone\n\n" +
				"**Recommendation**\n\nNone\n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/authentication_and_authorization/index#managing-pod-security-policies[Default Security Context Constraint]\n"

		}
	}
	return line + "\n"
}

func shouldExcludeNamespace(namespace string) bool {
	excludedPrefixes := []string{"openshift", "default", "kube", "open"}

	for _, prefix := range excludedPrefixes {
		if strings.HasPrefix(namespace, prefix) {
			return true
		}
	}

	return false
}

func shouldExcludePod(podName string) bool {
	excludedSuffixes := []string{"-build", "-deploy"}

	for _, suffix := range excludedSuffixes {
		if strings.HasSuffix(podName, suffix) {
			return true
		}
	}

	return false
}

func getAssociatedDeployment(clientset *kubernetes.Clientset, namespace string, pod v1.Pod) *k8sappsv1.Deployment {
	ownerReferences := pod.OwnerReferences
	for _, owner := range ownerReferences {
		if owner.Kind == "ReplicaSet" {
			replicaSet, err := clientset.AppsV1().ReplicaSets(namespace).Get(context.TODO(), owner.Name, metav1.GetOptions{})
			if err == nil {
				deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), replicaSet.OwnerReferences[0].Name, metav1.GetOptions{})
				if err == nil {
					return deployment
				}
			}
		}
	}
	return nil
}

func getAssociatedDeploymentConfig(ocAppsClient *ocappsv1client.AppsV1Client, namespace string, pod v1.Pod) string {
	ownerReferences := pod.OwnerReferences
	for _, owner := range ownerReferences {
		if owner.Kind == "ReplicationController" {
			// Pod is associated with a ReplicationController, check if it belongs to a DeploymentConfig
			deploymentConfigs, err := ocAppsClient.DeploymentConfigs(namespace).List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				fmt.Printf("Error listing DeploymentConfigs in namespace %s: %v\n", namespace, err)
				return ""
			}

			for _, dc := range deploymentConfigs.Items {
				if annotationsMatch(dc.Spec.Template.Annotations, pod.Annotations) {
					return dc.Name
				}
			}
		}
	}

	return "" // Return empty string if no matching DeploymentConfig is found
}

func annotationsMatch(annotations map[string]string, podAnnotations map[string]string) bool {
	for key, value := range annotations {
		podValue, ok := podAnnotations[key]
		if !ok || value != podValue {
			return false
		}
	}
	return true
}
