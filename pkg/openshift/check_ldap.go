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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var identityProviderType string

func identityProvider() {

	// Run the 'oc get oauth -o jsonpath={.spec.identityProviders[*].type}' command
	out, err := exec.Command("oc", "get", "oauth", "-o", "jsonpath={.items[*].spec.identityProviders[*].type}").Output()
	if err != nil {
		fmt.Printf("Error getting identity provider type: %v", err)
	}

	// Check if LDAP is one of the identity provider types
	identityProviderType = strings.TrimSpace(string(out))
	//if strings.Contains(identityProviderType, "LDAP") {
	//	// Check if the LDAP configuration is secure
	//	if strings.Contains(identityProviderType, "ldaps") {
	//		color.Green("Integrate with identity provider (LDAP) securely\tPASSED")
	//
	//	} else {
	//		color.Yellow("Integrate with identity provider not securely\t\tREVISED")
	//
	//	}
	//} else {
	//	color.Red("Integrate with identity provider (LDAP)\t\tFAILED")
	//
	//}
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
		output.WriteString(identityProviderProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}
}

func checkIdentityProvider() (string, error) {

	// Get the encryption type of the etcd server
	out, err := exec.Command("oc", "get", "oauth", "-o", "jsonpath={.items[*].spec}").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func identityProviderProcess(line string) string {

	if strings.HasPrefix(line, "<<Identity Provider Type>>") {
		if identityProviderType != "" {
			switch identityProviderType {
			case "LDAP":
				// Check if the LDAP configuration is secure
				return line + "\n\n|" + identityProviderType + GetKeyChanges("nochange") + "\n\n"
			case "HTPasswd":
				return line + "\n\n|" + identityProviderType + GetKeyChanges("advisory") + "\n\n"
			}
		} else {
			return line + "\n\n|Identity Provider Type is not set\n\n" + GetKeyChanges("required") + "\n\n"
		}
	}

	if strings.HasPrefix(line, "<<Identity Provider Search URL>>") {
		idpProvider, _ := checkIdentityProvider()

		if identityProviderType != "" {
			switch identityProviderType {
			case "LDAP":
				if strings.Contains(idpProvider, "(objectclass=*)(|(memberOf=") {
					return line + "\n\n|Search filter is set" + GetKeyChanges("nochange") + "\n\n"
				} else {
					return line + "\n\n|Search filter is not set" + GetKeyChanges("advisory") + "\n\n"
				}
			default:
				return line + "\n\n|Not Applicable" + GetKeyChanges("na") + "\n\n"

			}
		} else {
			return line + "\n\n|Identity Provider Type is not set\n\n" + GetKeyChanges("na") + "\n\n"
		}
	}

	if strings.HasPrefix(line, "<<LDAP Encrypted Connection>>") {
		idpProvider, _ := checkIdentityProvider()

		if identityProviderType != "" {
			switch identityProviderType {
			case "LDAP":
				if strings.Contains(idpProvider, "ldaps") {
					return line + "\n\n|LDAP is secure" + GetKeyChanges("nochange") + "\n\n"
				} else {
					return line + "\n\n|LDAP is not secure" + GetKeyChanges("recommended") + "\n\n"
				}
			default:
				return line + "\n\n|Not Applicable" + GetKeyChanges("na") + "\n\n"

			}
		} else {
			return line + "\n\n|Identity Provider Type is not set\n\n" + GetKeyChanges("na") + "\n\n"
		}
	}

	if strings.HasPrefix(line, "== Identity Provider Type") {
		idpProvider, _ := checkIdentityProvider()
		version, _ := getOpenShiftVersion()

		if idpProvider != "{}" {
			if strings.Contains(identityProviderType, "LDAP") {
				if strings.Contains(idpProvider, "ldaps") && strings.Contains(idpProvider, "(memberOf") {
					return line + "\n\n" + GetChanges("nochange") +
						"\n\n**Observation**\n\nLDAP is configured as an identity provider" +
						"**Recommendation**\n\nNone\n\n" +
						"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
						"html-single/authentication_and_authorization/index#configuring-ldap-identity-provider[Configuring an LDAP identity provider]\n" +
						"\n\n\n\n== Identity Provider Search URL\n\n" + GetChanges("nochange") +
						"\n\n**Observation**\n\nIdentity Provider Search URL is configured" +
						"\n\n**Recommendation**\n\nNone\n\n" +
						"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
						"html-single/authentication_and_authorization/index#configuring-ldap-identity-provider[Configuring an LDAP identity provider]\n" +
						"\n\n\n\n== LDAP Encrypted Connection\n\n" + GetChanges("nochange") +
						"\n\n**Observation**\n\nLDAP is configured as an identity provider securely." +
						"\n\n**Recommendation**\n\nNone\n\n" +
						"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
						"html-single/authentication_and_authorization/index#configuring-ldap-identity-provider[Configuring an LDAP identity provider]\n"

				} else if !strings.Contains(idpProvider, "ldaps") && strings.Contains(idpProvider, "memberOf") {
					return line + "\n\n" + GetChanges("nochange") +
						"\n\n**Observation**\n\nLDAP is configured as an identity provider" +
						"**Recommendation**\n\nNone\n\n" +
						"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
						"/html-single/authentication_and_authorization/index#configuring-ldap-identity-provider[Configuring an LDAP identity provider]\n" +
						"\n\n\n\n== Identity Provider Search URL\n\n" + GetChanges("nochange") +
						"\n\n**Observation**\n\nIdentity Provider Search URL is configured" +
						"\n\n**Recommendation**\n\nNone\n\n" +
						"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
						"/html-single/authentication_and_authorization/index#configuring-ldap-identity-provider[Configuring an LDAP identity provider]\n" +
						"\n\n\n\n== LDAP Encrypted Connection\n\n" + GetChanges("recommended") +
						"\n\n**Observation**\n\nWith an unencrypted LDAP connection attackers could easily gain access to sensitive information." +
						"\n\n**Recommendation**\n\nEnsure LDAP server support encrypted communications, then update the connection string in the OAuth Identity Provider to use LDAPS.\n\n" +
						"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
						"/html-single/authentication_and_authorization/index#configuring-ldap-identity-provider[Configuring an LDAP identity provider]\n"

				} else {
					return line + "\n\n" + GetChanges("nochange") +
						"\n\n**Observation**\n\nLDAP is configured as an identity provider" +
						"**Recommendation**\n\nNone\n\n" +
						"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
						"/html-single/authentication_and_authorization/index#configuring-ldap-identity-provider[Configuring an LDAP identity provider]\n" +
						"\n\n\n\n== Identity Provider Search URL\n\n" + GetChanges("recommended") +
						"\n\n**Observation**\n\nIdentity Provider Search URL is not configured" +
						"\n\n**Recommendation**\n\nIt is recommended to limit LDAP search by group.\n\n" +
						"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
						"/html-single/authentication_and_authorization/index#configuring-ldap-identity-provider[Configuring an LDAP identity provider]\n" +
						"\n\n\n\n== LDAP Encrypted Connection\n\n" + GetChanges("recommended") +
						"\n\n**Observation**\n\nLDAP is configured as an identity provider and not secure." +
						"\n\n**Recommendation**\n\nIt is recommended to use LDAPS to secure the communication between OpenShift and the LDAP.\n\n" +
						"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
						"/html-single/authentication_and_authorization/index#configuring-ldap-identity-provider[Configuring an LDAP identity provider]\n"
				}
			} else if strings.Contains(identityProviderType, "HTPasswd") {

				return line + "\n\nOnly " + "`" + identityProviderType + "`" + " is used as an identity provider, there is no integration with a central identity provider.\n\n"
			}
		} else {
			return line + "\n\n" + GetChanges("required") +
				"\n\n**Observation**\n\nThere is no integration with a central identity provider (LDAP)." +
				"\n\n**Recommendation**\n\nIntegrate with an existing central identity provider such as Active Directory LDAP\n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/authentication_and_authorization/index#configuring-ldap-identity-provider[Configuring an LDAP identity provider]\n" +
				"\n\n\n\n== Identity Provider Search URL\n\n" + GetChanges("na") +
				"\n\n**Observation**\n\nThere is no integration with a central identity provider (LDAP)." +
				"\n\n**Recommendation**\n\nIntegrate with an existing central identity provider such as Active Directory LDAP\n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/authentication_and_authorization/index#configuring-ldap-identity-provider[Configuring an LDAP identity provider]\n" +
				"\n\n\n\n== LDAP Encrypted Connection\n\n" + GetChanges("na") +
				"\n\n**Observation**\n\nThere is no integration with a central identity provider (LDAP)." +
				"\n\n**Recommendation**\n\nIntegrate with an existing central identity provider such as Active Directory LDAP\n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/authentication_and_authorization/index#configuring-ldap-identity-provider[Configuring an LDAP identity provider]\n"
		}
	}
	return line + "\n"
}
