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
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	routev1client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/config"
	"gitlab.consulting.redhat.com/meta/health-check-runner/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var (
	etcdDiskBackendCommit string
	etcdDiskWalFsync      string
	etcdNetworkPeerRound  string
	resultMetrics         string
	token                 []byte
	passed                bool
	etcdDiskBackend       bool
	etcdDiskWal           bool
	etcdNetworkPeer       bool
)

type query struct {
	name        string
	queryString string
}

const (
	namespace, serviceAccount = "openshift-monitoring", "prometheus-hc-api"
)

// Create rolebinding for prometheus-hc-api
func createRoleBinding() {
	cmd := exec.Command("oc", "adm", "policy", "add-cluster-role-to-user", "cluster-monitoring-view", "-z", serviceAccount, "-n", namespace)
	_, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Errorf("failed to create role binding: %v", err)
	}

}

// Create rolebinding for prometheus-hc-api
func deleteRoleBinding() {
	// Set the required command and arguments
	command := "oc"
	args := []string{
		"adm",
		"policy",
		"remove-cluster-role-from-user",
		"cluster-monitoring-view",
		"-z",
		serviceAccount,
		"-n",
		namespace,
	}

	// Create the command
	cmd := exec.Command(command, args...)

	// Run the command
	err := cmd.Run()

	// Check for errors
	if err != nil {
		fmt.Printf("Error running command: %s\n", err)
		os.Exit(1)
	}
}

// Create service account
func createServiceAccount() {

	cmd := exec.Command("oc", "create", "sa", serviceAccount, "-n", namespace)

	_, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Errorf("failed to create service account: %v", err)
	}

	createRoleBinding()

}

// Create service account
func deleteServiceAccount() {

	cmd := exec.Command("oc", "delete", "sa", serviceAccount, "-n", namespace)

	_, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Errorf("failed to delete service account: %v", err)
	}

}

func getSecretToken() []byte {

	checkOCPVersion := utils.CompareOpenShiftVersion()

	var getSAToken []byte

	if checkOCPVersion {
		getSAToken, _ = exec.Command("oc", "create", "token", "-n", namespace, serviceAccount).Output()

	} else {
		getSAToken, _ = exec.Command("oc", "sa", "get-token", "-n", namespace, serviceAccount).Output()
	}
	//saToken := string(getSAToken)
	//saToken = removeSpaces(saToken)

	return getSAToken
}

func etcdHealth() {

	// Create a custom TLS config that skips verification
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	// Define the queries to run
	queries := []query{
		{name: "etcd_disk_backend_commit_duration_seconds_bucket", queryString: "histogram_quantile(0.99, irate(etcd_disk_backend_commit_duration_seconds_bucket[5m])) > 0.03"},
		{name: "etcd_disk_wal_fsync_duration_seconds_bucket", queryString: "histogram_quantile(0.99, irate(etcd_disk_wal_fsync_duration_seconds_bucket[10m])) > 0.015"},
		{name: "etcd_network_peer_round_trip_time_seconds_bucket", queryString: "histogram_quantile(0.99, irate(etcd_network_peer_round_trip_time_seconds_bucket[10m])) > 0.06"},
	}
	configKube, err := getClusterConfig()
	if err != nil {
		fmt.Println("Error getting cluster config:", err)
		os.Exit(1)
	}

	clientset, err := getClientSet(configKube)
	if err != nil {
		fmt.Println("Error getting client set:", err)
		os.Exit(1)
	}

	// List all service accounts in the openshift-monitoring namespace.
	saList, err := clientset.CoreV1().ServiceAccounts("openshift-monitoring").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error listing service accounts: %v\n", err)

	}

	// Check if the service account already exists.
	found := false
	for _, sa := range saList.Items {
		if sa.Name == serviceAccount {
			found = true
			break
		}
	}

	// If the service account does not exist, create it.
	if found {
		deleteServiceAccount()
	}
	createServiceAccount()

	routeClient, err := routev1client.NewForConfig(configKube)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating Route client: %v\n", err)

	}

	route, err := routeClient.Routes("openshift-monitoring").Get(context.TODO(), "prometheus-k8s", metav1.GetOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error retrieving Prometheus route: %v\n", err)

	}

	token = bytes.TrimSpace(getSecretToken())

	promAPI, err := api.NewClient(api.Config{
		Address: "https://" + route.Spec.Host,
		RoundTripper: config.NewAuthorizationCredentialsRoundTripper(
			"Bearer",
			config.Secret(token), // Assuming token is a string variable containing the bearer token
			&http.Transport{
				TLSClientConfig: tlsConfig,
			},
		),
	})

	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		os.Exit(1)
	}

	v1api := v1.NewAPI(promAPI)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Loop over the queries and run them
	for _, q := range queries {
		etcdMetrics, _, err1 := v1api.Query(ctx, q.queryString, time.Now(), v1.WithTimeout(10*time.Second))
		if err1 != nil {
			fmt.Println(err1)
			continue
		}
		resultMetrics = etcdMetrics.String()

		// Check if the result is empty

		if resultMetrics == "" {

			switch q.name {
			case "etcd_disk_backend_commit_duration_seconds_bucket":
				etcdDiskBackend = true
			case "etcd_disk_wal_fsync_duration_seconds_bucket":
				etcdDiskWal = true
			case "etcd_network_peer_round_trip_time_seconds_bucket":
				etcdNetworkPeer = true

			}
			passed = true
			break

		} else {
			//	color.Cyan("ETCD Health\t\t\t\t\t\tWarning")
			break

		}
	}

	//if passed {
	//	color.Green("ETCD is Healthy\t\t\t\t\tPASSED")
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
		output.WriteString(etcdETCDHealthProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}

	// Delete serviceAccount and role binding at the end of execution
	defer deleteServiceAccount()
	defer deleteRoleBinding()
}

func etcdETCDHealthProcess(line string) string {
	if strings.HasPrefix(line, "<<Etcd Performance>>") {

		// Render the change status
		if etcdDiskBackend != true {
			etcdDiskBackendCommit = "\n\nReported etcd_disk_backend_commit_duration_seconds_bucket metric 10 minutes rate 99th percentile is more than 25ms.\n"
		}

		if etcdDiskWal != true {
			etcdDiskWalFsync = "\nReported etcd_disk_wal_fsync_duration_seconds_bucket metric 10 minutes rate 99th percentile is more than 20ms.\n"
		}

		if etcdNetworkPeer != true {
			etcdNetworkPeerRound = "\nReported etcd_network_peer_round_trip_time_seconds_bucket metric 10 minutes rate 99th percentile is more than 50ms.\n"
		}

		// Check if the result is empty
		if resultMetrics == "" || (etcdDiskBackend && etcdDiskWal && etcdNetworkPeer) {
			return line + "\n\n| ETCD is healthy \n\n" + GetKeyChanges("nochange")
		} else {
			return line + "\n\n| ETCD is not healthy \n\n" + GetKeyChanges("required")
		}

	}

	if strings.HasPrefix(line, "== Etcd Performance") {
		version, _ := getOpenShiftVersion()

		if etcdDiskBackend != true {
			etcdDiskBackendCommit = "\n\nReported etcd_disk_backend_commit_duration_seconds_bucket metric 10 minutes rate 99th percentile is more than 25ms.\n"
		}

		if etcdDiskWal != true {
			etcdDiskWalFsync = "\nReported etcd_disk_wal_fsync_duration_seconds_bucket metric 10 minutes rate 99th percentile is more than 20ms.\n"
		}

		if etcdNetworkPeer != true {
			etcdNetworkPeerRound = "\nReported etcd_network_peer_round_trip_time_seconds_bucket metric 10 minutes rate 99th percentile is more than 50ms.\n"
		}

		// Check if the result is empty
		if resultMetrics == "" || (etcdDiskBackend && etcdDiskWal && etcdNetworkPeer) {
			return line + "\n\n" + GetChanges("nochange") +
				"\n\nReported etcd_disk_backend_commit_duration_seconds_bucket metric 5 minutes rate 99th percentile is less than 25ms.\n" +
				"\n\nReported etcd_disk_wal_fsync_duration_seconds_bucket metric 5 minutes rate 99th percentile is less than 20ms.\n" +
				"\n\nReported etcd_network_peer_round_trip_time_seconds_bucket metric 5 minutes rate 99th percentile is less than 50ms.\n" +
				"\n\n**Observation**\n\nETCD is healthy \n\n" +
				"**Recommendation**\n\nNone\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/scalability_and_performance/index#recommended-etcd-practices_recommended-host-practices[ETCD Recommended Practice]\n"
		} else {
			return line + "\n\n" + GetChanges("required") + "\n\n" + etcdDiskBackendCommit + etcdDiskWalFsync + etcdNetworkPeerRound + "\n" + resultMetrics + "\n" +
				"\n\n**Observation**\n\nETCD is not healthy \n\n" +
				"**Recommendation**\n\nCheck the reference for more details.\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/scalability_and_performance/index#recommended-etcd-practices_recommended-host-practices[ETCD Recommended Practice]\n"
		}

	}

	return line + "\n"
}
