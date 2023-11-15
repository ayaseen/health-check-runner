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

var verifySslAPI bool

func apiServerCertificate() {
	cmd := exec.Command("oc", "whoami", "--show-server=true")
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
	var apiURL string
	for scanner.Scan() {
		line := scanner.Text()
		apiURL = line
		apiURL = strings.Replace(apiURL, "https://api-int", "https://api", 1)
	}
	if apiURL == "" {
		fmt.Println("Error: unable to retrieve OpenShift API URL")
		return
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
	}

	client := &http.Client{Transport: tr}

	_, err = client.Get(apiURL)
	if err != nil {
		if _, ok := err.(x509.UnknownAuthorityError); ok {
			fmt.Println("The OpenShift API URL has self-signed certificates.")
		} else {
			color.Red("API certificate has expired or is not yet valid \tFAILED")
			verifySslAPI = false
		}

	} else {
		color.Green("API Certificate configured with CA \t\tPASSED")
		verifySslAPI = true
	}

	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		fmt.Println("Error getting current directory:", err)
		return
	}

	// Create the output file for writing
	outfile, err := os.Create(filepath.Join(dir, "resources/api_server_certificates.adoc"))
	if err != nil {
		fmt.Println("Error creating output file:", err)
		return
	}
	defer outfile.Close()

	// Read each line of the input file
	scannerAPI := bufio.NewScanner(strings.NewReader(tpl))
	for scannerAPI.Scan() {
		line := scannerAPI.Text()
		outfile.WriteString(apiServerCertificateProcess(line))
	}

	if err = scannerAPI.Err(); err != nil {
		fmt.Println("Error scanning file:", err)
		return
	}
}

func apiServerCertificateProcess(line string) string {

	if strings.HasPrefix(line, "=") {

		if verifySslAPI != true {
			return line + "API server certificates\n\n" + GetChanges("recommended") + "\n\n"
		} else {
			return line + "API server certificates\n\n" + GetChanges("nochange") + "\n\n"
		}

	}

	if strings.HasPrefix(line, "**Observation**") {
		if verifySslAPI != true {
			return line + "\n\nAPI has self-singed certificate, expired or is not yet valid.\n"

		} else {
			return line + "\n\nAPI Certificate configured with CA and valid.\n"

		}
	}

	if strings.HasPrefix(line, "**Recommendation**") {
		if verifySslAPI != true {
			return line + "\n\nThe default API server certificate is issued by an internal OpenShift Container Platform cluster CA." +
				" Clients outside of the cluster will not be able to verify the API server’s certificate by default. " +
				"This certificate can be replaced by one that is issued by a CA that clients trust.\n"
		} else {
			return line + "\n\nNone\n"

		}

	}

	if strings.Contains(line, "https://access.redhat.com/documentation/en-us/openshift_container_platform/") {
		version, _ := getOpenShiftVersion()

		return strings.Replace(line, "/openshift_container_platform/4.10", "/openshift_container_platform/"+version, -1) +
			"html-single/security_and_compliance/configuring-certificates#api-server-certificates[API server certificates]\n"
	}
	return line + "\n"
}
