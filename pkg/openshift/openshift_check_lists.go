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
	"fmt"
	"github.com/briandowns/spinner"
	"log"
	"os"
	"time"
)

var provider string

// Main function to call sub lists
var (
	functions []func() = []func(){
		checkCO,
		etcdHealth,
		clusterVersion,
		clusterCNIPlubin,
		checkElevatedPrivileges,
		nodeStatus,
		checkLoggingHealth,
		vmwareVersion,
		etcdEncryption,
		vmwareHA,
		installationType,
		emptyDirVolume,
		defaultOpenShiftUser,
		resourceLimit,
		checkInfraNode,
		defaultNodeSchedule,
		resourceQuota,
		selfProvisioners,
		controlNodeSchedule,
		defaultProject,
		serviceMonitor,
		proxySettings,
		defaultIngressCertificate,
		infrastructureProvider,
		networkPolicy,
		nodeUsage,
		vmwareNetworkType,
		vmwareStorageType,
		checkInfraConfigPool,
		providerTopology,
		monitoringUserWorkload,
		checkLogging,
		ingressControllerPlacement,
		applicationProbes,
		vmwareNodePlacement,
		vmwareDatastoreMultitenancy,
		monitoringStack,
		identityProvider,
		ingressControllerReplica,
		CheckElasticSearch,
		checkLoggingForwardersOPS,
		checkLoggingForwarders,
		internalRegistry,
		IngressControllerType,
		ClusterDefaultSCC,
		InfraTaints,
		KubeletConfig,
		alerts,
		checkLoggingPlacement,

		//podImagePullStatus,

		//etcdBackup,

		//apiServerCertificate,

	}
)

func CheckLists() {

	// Get Infrastructure provider
	provider, _ = CheckInfrastructureProvider()

	// Create source folder if it doesn't exist
	err := CreateDirResources(DestPath)
	if err != nil {
		log.Fatal(err)
	}

	_, err = getClusterConfig()
	if err != nil {
		os.Exit(1)

	}

	err = checkOCExist()
	if err != nil {
		fmt.Println("Error: OpenShift cluster is not reachable through API")
		os.Exit(1)
	}

	ocpAccess := verifyOpenShiftAPIAccess()

	if ocpAccess == true {
		fmt.Println("Error: OpenShift cluster is not reachable through API")
		os.Exit(1)
	}

	fmt.Println("Starting CheckLists...")

	s := spinner.New(spinner.CharSets[24], 100*time.Millisecond, spinner.WithWriter(os.Stderr))
	s.Suffix = " Checking... OpenShift  "
	s.FinalMSG = "All checks completed!\n"
	_ = s.Color("red", "bold")

	s.Start()

	for _, f := range functions {
		f()
		time.Sleep(4 * time.Second)
	}
	s.Stop()

	// Compress resource folder and protect a random password
	Compress(DestPath, CompressFile)

}
