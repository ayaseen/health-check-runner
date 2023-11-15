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

package cmd

import (
	"github.com/spf13/cobra"
	"gitlab.consulting.redhat.com/meta/health-check-runner/pkg/openshift"
	"os"
)

// rootCmd represents the base command when called without any subcommands
//var onlyOpenShift bool
//var onlyApplication bool
//
//var progress = make(chan int)

var rootCmd = &cobra.Command{
	Use:   "hc-runner",
	Short: "Performs a health check against OpenShift clusters",
	Long: `This application to help do a list of health check lists against OpenShift clusters.
The application runs a list of checks and generates a asicdoc formatted documentation file for each check`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		check, _ := cmd.Flags().GetString("check")

		switch check {
		case "openshift":

			// Run checklist
			openshift.CheckLists()

		case "application":
			openshift.Applicationchecklists()

		}

	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()

	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.health-check-runner.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.PersistentFlags().String("check", "openshift", "Run health check on OpenShift cluster")

}
