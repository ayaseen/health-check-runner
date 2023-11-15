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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"os"
	"path/filepath"
	"strings"
)

const (
	excludePrefix    = "openshift"
	excludeContain   = "kube"
	defaultNamespace = "default"
	others           = "open"
)

type NamespaceInfo struct {
	Name          string
	ResourceQuota []corev1.ResourceQuota
	LimitRange    []corev1.LimitRange
}

//var info []NamespaceInfo
//
//var (
//	ns    string
//	quota []corev1.resourceQuota
//	limit []corev1.LimitRange
//)

var resourceQ, resourceL int

func resourceQuota() {
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

	namespaces, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Println(err)

	}
	checkQuotaUserProject, foundResourceQuota := false, false

	for _, namespace := range namespaces.Items {
		if namespace.Name == defaultNamespace ||
			(strings.HasPrefix(namespace.Name, excludePrefix) || strings.Contains(namespace.Name, excludeContain) || strings.Contains(namespace.Name, others)) {
			continue
		}

		var namespaceInfo NamespaceInfo
		namespaceInfo.Name = namespace.Name

		resourceQuotas, err := clientset.CoreV1().ResourceQuotas(namespace.Name).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			fmt.Println(err)

		}

		namespaceInfo.ResourceQuota = resourceQuotas.Items

		//if len(resourceQuotas.Items) > 0 && checkQuotaUserProject {
		//	color.Green("resourceQuota is configured\t\t\t\tPASSED")
		//	break
		//
		//} else {
		//	color.Red("resourceQuota is configured\t\t\t\tFAILED")
		//	//break
		//}

		resourceQ = len(resourceQuotas.Items)
		if resourceQ > 0 {
			foundResourceQuota = true
			break
		}
		checkQuotaUserProject = true

	}

	// To check if there are no user projects
	if !checkQuotaUserProject && foundResourceQuota {
		color.Green("resourceQuota is configured\t\t\t\tPASSED")
	} else {
		color.Red("resourceQuota is configured\t\t\t\tFAILED")
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
		output.WriteString(resourceProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}

}

func resourceLimit() {
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

	namespaces, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Println(err)

	}
	checkQuotaUserProject, foundResourceLimit := false, false

	for _, namespace := range namespaces.Items {
		if namespace.Name == defaultNamespace ||
			(strings.HasPrefix(namespace.Name, excludePrefix) || strings.Contains(namespace.Name, excludeContain) || strings.Contains(namespace.Name, others)) {
			continue
		}

		var namespaceInfo NamespaceInfo
		namespaceInfo.Name = namespace.Name

		limitRanges, err := clientset.CoreV1().LimitRanges(namespace.Name).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			fmt.Println(err)

		}
		namespaceInfo.LimitRange = limitRanges.Items
		resourceL = len(limitRanges.Items)
		//if len(limitRanges.Items) > 0 {
		//	color.Green("LimitRange is configured\t\t\t\tPASSED")
		//	break
		//
		//} else {
		//	color.Red("LimitRange is configured\t\t\t\tFAILED")
		//	//break
		//
		//}

		resourceL = len(limitRanges.Items)
		if resourceL > 0 {
			foundResourceLimit = true
			break
		}

		checkQuotaUserProject = true

	}

	// To check if there are no user projects
	if !checkQuotaUserProject && foundResourceLimit {
		color.Green("LimitRange is configured\t\t\t\tPASSED")
	} else {
		color.Red("LimitRange is configured\t\t\t\tFAILED")
	}

}

func resourceProcess(line string) string {

	if strings.HasPrefix(line, "<<Resource Quotas Defined>>") {
		// Render the change status
		if resourceQ < 1 || resourceL < 1 {
			return line + "\n\n|Resource Requests and Limits not configured for user projects\n\n" + GetKeyChanges("recommended") + "\n\n"
		} else {
			return line + "\n\n| NA \n\n" + GetKeyChanges("nochange")
		}
	}

	if strings.HasPrefix(line, "== Resource Quotas Defined") {
		version, _ := getOpenShiftVersion()

		if resourceQ < 1 || resourceL < 1 {
			return line + "\n\n" + GetChanges("recommended") +
				"\n\n**Observation**\n\nResource Requests and Limits not configured for user projects \n\n" +
				"**Recommendation**\n\nIt is extremely important to create resource quotas and limits, particularly if horizontal pod autoscaling is to be used.\n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/building_applications/index#quotas[Quotas]\n"
		} else {
			return line + "\n\n" + GetChanges("nochange") +
				"\n\n**Observation**\n\nResource Requests and Limits are configured for user projects \n\n" +
				"**Recommendation**\n\nNone.\n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/building_applications/index#quotas[Quotas]\n"
		}
	}

	return line + "\n"
}
