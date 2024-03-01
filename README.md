# OpenShift Health Check Application


This is a command-line interface (CLI) application written in Golang that performs a health check against OpenShift clusters.
The application runs a list of checks and generates a basic documentation file for each check.

In this tool, there are more than 40 checks to make sure Openshift comply with recommendations and best practice.

> **Note**
> This application tested on openshift version: 4.9  and later

> **Note**
> This application is not fully completed, it's still under development and might produce unwanted results!

# Usage

To use this application, follow these steps:

* Clone the repository to your local machine.
* Build the main.go file using the go build command.
* on Linux OS: CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o hc-runner main.go
* Run the executable file using the ./main command.

* hc-runner is ready to use with the latest changes!
* Check the generated healthcheck-body.adoc document under resources to see the output.

Dependencies
* This application requires Golang to be installed on your machine.
* KUBECONFIG set in env, or use oc login to gain OpenShift cluster access

# Running the checks
The application runs against a list of checks defined in the pkg/openshift/openshift_check_list.go file.
You can add or modify the checks in this file to suit your requirements.

To run the checks, simply execute the ./hc-runner command. The application will run each check and generate a documentation file for each one under resources.

The generated file is protected with a password "7e5eed48001f9a407bbb87b29c32871b"

# Documentation
The application generates a basic documentation file for each check in the docs directory.
The documentation file includes the check name, description, and the result of the check, a compressed file including all document that been generated with encrypted password.

# Contributing
We welcome contributions from the community. If you find a bug or have a feature request, please open an issue on the repository.
If you would like to contribute code, please fork the repository and submit a pull request.

# Demo

Run  ./main

![health-check-demo](images/health-check.png)





