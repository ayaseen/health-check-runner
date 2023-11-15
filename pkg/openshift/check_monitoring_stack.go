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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"strings"
)

var configMapStackExist, ok bool
var data string

type PrometheusK8sConfig struct {
	Retention           string                 `yaml:"retention"`
	Resources           map[string]interface{} `yaml:"resources"`
	VolumeClaimTemplate map[string]interface{} `yaml:"volumeClaimTemplate"`
}

type ConfigData struct {
	EnableUserWorkload bool                `yaml:"enableUserWorkload"`
	PrometheusK8s      PrometheusK8sConfig `yaml:"prometheusK8s"`
}

func monitoringStack() {
	config, err := getClusterConfig()
	if err != nil {
		fmt.Println("Error getting cluster config:", err)
		os.Exit(1)
	}

	clientset, err := getClientSet(config)
	if err != nil {
		fmt.Println("Error getting client set:", err)
		os.Exit(1)
	}

	// Check if the configMap exists
	cm, err := clientset.CoreV1().ConfigMaps("openshift-monitoring").Get(context.TODO(), "cluster-monitoring-config", metav1.GetOptions{})
	if err != nil {
		configMapStackExist = false
	}

	// Check if the key exists and has the correct value
	data, ok = cm.Data["config.yaml"]
	//if !ok {
	//	fmt.Println("ConfigMap does not contain config.yaml")
	//	os.Exit(1)
	//}

	if err = yaml.Unmarshal([]byte(data), &configMapData); err != nil {
		fmt.Printf("Error unmarshaling YAML: %v\n", err)

	}

	if !hasPrometheusK8sVolumeClaimTemplate(data) && configMapStackExist == false && !ok {
		color.Red("OpenShift Monitoring storage is enabled\t\tFAILED")
	} else {
		color.Green("OpenShift Monitoring storage is enabled\t\tPASSED")
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
		output.WriteString(monitoringStackProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}
}

func monitoringStackProcess(line string) string {

	if strings.HasPrefix(line, "<<Monitoring components need high performance/local persistent storage to maintain consistent state after a pod restart>>") {
		if !hasPrometheusK8sVolumeClaimTemplate(data) {
			return line + "\n\n| OpenShift monitoring components do not have the appropriate storage \n\n" + GetKeyChanges("required")
		} else {
			return line + "\n\n| OpenShift monitoring components do have the appropriate storage \n\n" + GetKeyChanges("nochange")
		}
	}

	if strings.HasPrefix(line, "== Monitoring components need high performance/local persistent storage to maintain consistent state after a pod restart") {

		version, _ := getOpenShiftVersion()

		if !hasPrometheusK8sVolumeClaimTemplate(data) {
			return line + "\n\n" + GetChanges("required") +
				"\n\n**Observation**\n\nThe OpenShift monitoring components do not have the appropriate storage configured for a production environment and are not protected against data loss.\n\n" +
				"**Recommendation**\n\nConfigure persistent storage for the OpenShift monitoring components per the OpenShift documentation\n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"html-single/monitoring/configuring-the-monitoring-stack[Understanding the monitoring stack]\n" +
				"* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"html-single/monitoring/index#configuring_persistent_storage_configuring-the-monitoring-stack[Monitoring Configuring persistent storage]\n"

		} else {
			return line + "\n\n" + GetChanges("nochange") +
				"\n\n**Observation**\n\nThe OpenShift monitoring components do have the appropriate storage. \n\n" +
				"**Recommendation**\n\nNone. \n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"html-single/monitoring/configuring-the-monitoring-stack[Understanding the monitoring stack]\n" +
				"* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"html-single/monitoring/index#configuring_persistent_storage_configuring-the-monitoring-stack[Monitoring Configuring persistent storage]\n"

		}

	}

	return line + "\n"
}

func hasPrometheusK8sVolumeClaimTemplate(data string) bool {
	var configData ConfigData

	if err := yaml.Unmarshal([]byte(data), &configData); err != nil {
		fmt.Printf("Error unmarshaling YAML: %v\n", err)
		return false
	}

	return configData.PrometheusK8s.VolumeClaimTemplate != nil
}
