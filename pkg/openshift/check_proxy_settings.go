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
	"github.com/fatih/color"
	"k8s.io/apimachinery/pkg/util/json"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type ProxyConfig struct {
	Spec struct {
		HTTPProxy  string `json:"httpProxy"`
		HTTPSProxy string `json:"httpsProxy"`
		NoProxy    string `json:"noProxy"`
	} `json:"spec"`
}

var proxyConfig ProxyConfig

func proxySettings() {

	// Execute the "oc get proxy/cluster -o json" command
	cmd := exec.Command("oc", "get", "proxy/cluster", "-o", "json")
	proxy, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}

	// Parse the JSON output

	err = json.Unmarshal(proxy, &proxyConfig)
	if err != nil {
		log.Fatal(err)
	}

	// Check if proxy is configured
	if proxyConfig.Spec.HTTPProxy != "" && proxyConfig.Spec.HTTPSProxy != "" && proxyConfig.Spec.NoProxy != "" {
		color.HiCyan("OpenShift Proxy setting is not set\t\t\tCHECKED")
	} else {
		color.HiCyan("OpenShift Proxy setting is set\t\t\t\tCHECKED")
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
		output.WriteString(proxySettingsProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}
}

func checkProxySetting() (string, error) {

	out, err := exec.Command("oc", "get", "proxy/cluster", "-o", "yaml").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func proxySettingsProcess(line string) string {
	if strings.HasPrefix(line, "<<Openshift Proxy Settings>>") {

		if proxyConfig.Spec.HTTPProxy == "" && proxyConfig.Spec.HTTPSProxy == "" && proxyConfig.Spec.NoProxy == "" {
			return line + "\n\n|OpenShift Proxy setting is not set\n\n" + GetKeyChanges("na") + "\n\n"
		} else {
			return line + "\n\n|OpenShift Proxy setting is set\n\n" + GetKeyChanges("na") + "\n\n"
		}
	}

	if strings.HasPrefix(line, "== Openshift Proxy Settings") {
		proxyConf, _ := checkProxySetting()

		if proxyConfig.Spec.HTTPProxy == "" && proxyConfig.Spec.HTTPSProxy == "" && proxyConfig.Spec.NoProxy == "" {
			return line + "\n\n" + GetChanges("na") + "\n[source,bash]\n----\n" + proxyConf + "\n----\n" +
				"\n\n**Observation**\n\nOpenShift Proxy setting is not set. \n\n" +
				"**Recommendation**\n\nNone.\n\n"
		} else {
			return line + "\n\n" + GetChanges("na") + "\n[source,bash]\n----\n" + proxyConf + "\n----\n" +
				"\n\n**Observation**\n\nOpenShift Proxy setting is set. \n\n" +
				"**Recommendation**\n\nNone.\n\n"
		}
	}

	return line + "\n"
}
