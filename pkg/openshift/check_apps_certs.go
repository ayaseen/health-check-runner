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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/fatih/color"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var verifySslAPPS bool

func defaultIngressCertificate() {
	cmd := exec.Command("oc", "whoami", "--show-console")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println("Error getting stdout pipe:", err)
		return
	}
	if err := cmd.Start(); err != nil {
		fmt.Println("Error starting command:", err)
		return
	}
	scanner := bufio.NewScanner(stdout)
	var appURL string
	for scanner.Scan() {
		line := scanner.Text()
		appURL = line

	}
	if appURL == "" {
		fmt.Println("Error: unable to retrieve OpenShift APPS URL")
		return
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
	}

	client := &http.Client{Transport: tr}

	_, err = client.Get(appURL)
	if err != nil {
		if _, ok := err.(x509.UnknownAuthorityError); ok {
			fmt.Println("The OpenShift APPS URL has self-signed certificates.")
		} else {
			color.Red("Ingress Controller Certificate \t\t\tFAILED")
			verifySslAPPS = false
		}

	} else {
		color.Green("Ingress Controller Certificate  \t\t\tPASSED")
		verifySslAPPS = true
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
		output.WriteString(defaultIngressCertificateProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}
}

func defaultIngressCertificateProcess(line string) string {

	if strings.HasPrefix(line, "<<Ingress Controller Certificate>>") {

		if verifySslAPPS != true {
			return line + "\n\n|Default wildcard certificate not configured\n\n" + GetKeyChanges("recommended") + "\n\n"
		} else {
			return line + "\n\n|Default wildcard certificate is configured\n\n" + GetKeyChanges("nochange") + "\n\n"
		}
	}

	if strings.HasPrefix(line, "== Ingress Controller Certificate") {
		version, _ := getOpenShiftVersion()

		if verifySslAPPS != true {
			return line + "\n\n" + GetChanges("recommended") + "\n\n" +
				"\n\n**Observation**\n\nDefault wildcard certificate has self-singed certificate, expired or is not yet valid.\n\n" +
				"**Recommendation**\n\n" + "Replacing the default wildcard certificate with one that is issued by a public CA" +
				" already included in the CA bundle as provided by the container userspace" +
				" allows external clients to connect securely to applications running under the .apps sub-domain.\n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/security_and_compliance/configuring-certificates#replacing-default-ingress_replacing-default-ingresss[Default Ingress Certificate]\n"
		} else {
			return line + "\n\n" + GetChanges("recommended") + "\n\n" +
				"\n\n**Observation**\n\nDefault wildcard certificate configured with CA and valid.\n\n" +
				"**Recommendation**\n\nNone\n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/security_and_compliance/configuring-certificates#replacing-default-ingress_replacing-default-ingresss[Default Ingress Certificate]\n"
		}
	}
	return line + "\n"
}
