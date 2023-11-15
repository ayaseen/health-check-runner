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

var podImageStatus, podPendingStatus bool

type PodInfo struct {
	Name      string
	Namespace string
	Status    string
}

func podImagePullStatus() {

	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		fmt.Println("Error getting current directory:", err)
		return
	}

	// Create the output file for writing
	outfile, err := os.Create(filepath.Join(dir, "resources/pod_status.adoc"))
	if err != nil {
		fmt.Println("Error creating output file:", err)
		return
	}
	defer outfile.Close()

	// Read each line of the input file
	scanner := bufio.NewScanner(strings.NewReader(tpl))
	for scanner.Scan() {
		line := scanner.Text()
		outfile.WriteString(podImagePullProcess(line))
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error scanning file:", err)
		return
	}
}
func checkPodImagePullStatus() ([]PodInfo, error) {
	// Get the OpenShift config
	config, err := getClusterConfig()
	if err != nil {
		return nil, err
	}

	// Create the Kubernetes client
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	// Get all pods in the cluster
	podList, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var podInfoList []PodInfo

	// Check each pod for ImagePull issues
	for _, pod := range podList.Items {
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.State.Waiting != nil && (containerStatus.State.Waiting.Reason == "ErrImagePull" || containerStatus.State.Waiting.Reason == "ImagePullBackOff") {
				podInfoList = append(podInfoList, PodInfo{Name: pod.Name, Namespace: pod.Namespace, Status: containerStatus.State.Waiting.Reason})
				podImageStatus = true
			}
		}
	}

	if podImageStatus != true {
		color.Green("All Pods are Healthy\t\t\t\t\tPASSED")
	} else {
		color.Cyan("Some Pods can not pull image\t\t\t\tWarning")
	}

	return podInfoList, nil
}

// Check Pod status if pending

func checkPendingPods() ([]PodInfo, error) {
	// Get the OpenShift config
	config, err := getClusterConfig()
	if err != nil {
		return nil, err
	}

	// Create the Kubernetes client
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	// Get all pods in the cluster
	podList, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var podInfoList []PodInfo

	// Check each pod for Pending issues
	for _, pod := range podList.Items {
		if pod.Status.Phase == corev1.PodPending {
			podInfoList = append(podInfoList, PodInfo{Name: pod.Name, Namespace: pod.Namespace, Status: string(pod.Status.Phase)})
		}
	}

	return podInfoList, nil
}

func podImagePullProcess(line string) string {

	if strings.HasPrefix(line, "=") {
		// Render the change status
		if podImageStatus == true {
			return line + "OpenShift Pods Health\n\n" + GetChanges("advisory") + "\n\n"
		} else {
			return line + "OpenShift Pods Health\n\n" + GetChanges("nochange") + "\n\n"
		}
	}

	if strings.HasPrefix(line, "**Observation**") {

		podInfoList, err := checkPodImagePullStatus()
		var podStatus string
		if err != nil {
			fmt.Println(err)
		}

		if podImageStatus == true {

			for _, podInfo := range podInfoList {
				//fmt.Printf("Pod %s in namespace %s has %s issues\n", podInfo.Name, podInfo.Namespace, podInfo.Status)
				podStatus = "\n\nPod " + podInfo.Name + " in namespace" + podInfo.Namespace + " has" + podInfo.Status + " issues\n"
			}
			return line + podStatus
		} else {
			return line + "\n\nNone\n"

		}

		podPending, err := checkPendingPods()

		if err != nil {
			fmt.Println(err)
		}

		if podPendingStatus == true {

			for _, podInfo := range podPending {
				//fmt.Printf("Pod %s in namespace %s has %s issues\n", podInfo.Name, podInfo.Namespace, podInfo.Status)
				podStatus = "\n\nPod " + podInfo.Name + " in namespace" + podInfo.Namespace + " has" + podInfo.Status + " issues\n"
			}
			return line + podStatus
		} else {
			return line + "\n\nNone\n"

		}
	}
	if strings.HasPrefix(line, "**Recommendation**") {
		if podPendingStatus == true {
			return line + "\n\nContact Red Hat support to investigate the issues of Unhealthy Pods\n"
		} else {
			return line + "\n\nNone\n"

		}

	}

	return line + "\n"
}
