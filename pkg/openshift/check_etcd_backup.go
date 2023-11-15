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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var cronJobs []string

func etcdBackup() {

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

	cronJobs, err = listCronJobs(clientset)
	if err != nil {
		fmt.Println("Error listing cron jobs:", err)

	}

	if len(cronJobs) == 0 {
		color.Red("ETCD Backup is set\t\t\t\t\tFAILED")

	} else {
		color.Green("ETCD Backup is set\t\t\t\t\tPASSED")

		for _, cronJob := range cronJobs {
			fmt.Println("-", cronJob)
		}
	}

	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		fmt.Println("Error getting current directory:", err)
		return
	}

	// Create the output file for writing
	outfile, err := os.Create(filepath.Join(dir, "resources/etcd_backup.adoc"))
	if err != nil {
		fmt.Println("Error creating output file:", err)
		return
	}
	defer outfile.Close()

	// Read each line of the input file
	scanner := bufio.NewScanner(strings.NewReader(tpl))
	for scanner.Scan() {
		line := scanner.Text()
		outfile.WriteString(etcdResultsProcess(line))
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error scanning file:", err)
		return
	}

}

func listCronJobs(clientset *kubernetes.Clientset) ([]string, error) {
	cronJobList, err := clientset.BatchV1().CronJobs("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	for _, cronJob := range cronJobList.Items {
		if strings.Contains(strings.ToUpper(cronJob.Name), "ETCD") {
			cronJobs = append(cronJobs, cronJob.Name)
		}

	}
	return cronJobs, nil
}

func checkEtcdCO() (string, error) {
	out, err := exec.Command("oc", "get", "co", "etcd").CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func etcdResultsProcess(line string) string {
	if strings.HasPrefix(line, "=") {
		etcdCO, err := checkEtcdCO()
		if err != nil {
			return line + " Error checking etcd CO: " + err.Error() + "\n"
		}
		if len(cronJobs) == 0 {
			return line + "OpenShift ETCD Backup\n\n" + GetChanges("required") + "\n```\n" + etcdCO + "```\n"
		} else {
			return line + "OpenShift ETCD Backup\n\n" + GetChanges("nochange") + "\n```\n" + etcdCO + "```\n"
		}

	}

	if strings.HasPrefix(line, "**Observation**") {
		if len(cronJobs) != 0 {
			return line + "\n\nETCD backup is configured\n"
		} else {
			return line + "\n\nETCD backup is not configured\n"

		}
	}
	if strings.HasPrefix(line, "**Recommendation**") {
		if len(cronJobs) != 0 {
			return line + "\n\nETCD backup is configured\n"
		} else {
			return line + "\n\nETCD data back up should be set on a regular interval.\n"

		}

	}

	if strings.Contains(line, "https://access.redhat.com/documentation/en-us/openshift_container_platform/") {
		version, err := getOpenShiftVersion()
		if err != nil {
			return line + " Error getting OpenShift version: " + err.Error() + "\n"
		}
		return strings.Replace(line, "/openshift_container_platform/4.10", "/openshift_container_platform/"+version, -1) + "html-single/backup_and_restore/index#backup-etcd[ETCD Backup and Restore]\n"
	}
	return line + "\n"
}
