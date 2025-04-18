/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file provides Kubernetes and OpenShift API interaction utilities. It includes:

- Functions to establish connections to Kubernetes clusters
- Command execution wrappers for 'oc' and 'kubectl' commands
- Error handling and retry mechanisms for cluster operations
- Utilities for file operations and directory management
- OpenShift version detection and information gathering
- YAML parsing and manipulation functions
- Directory compression and file management operations

These utilities abstract the complexities of interacting with Kubernetes and OpenShift APIs, providing a simplified interface for health checks to gather cluster information.
*/

package utils

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"
)

// GetClusterConfig returns the Kubernetes client configuration
func GetClusterConfig() (*rest.Config, error) {
	// Try to use in-cluster config first
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	// Fall back to kubeconfig file
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		// If KUBECONFIG is not set, use the default location
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %v", err)
		}
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	// Check if the kubeconfig file exists
	if !FileExists(kubeconfig) {
		return nil, fmt.Errorf("kubeconfig file not found at %s", kubeconfig)
	}

	// Build config from kubeconfig file
	config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build config from kubeconfig: %v", err)
	}

	return config, nil
}

// GetClientSet returns a Kubernetes clientset
func GetClientSet() (*kubernetes.Clientset, error) {
	config, err := GetClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes clientset: %w", err)
	}

	return clientset, nil
}

// CommandError represents a command execution error with detailed information
type CommandError struct {
	Command string
	Args    []string
	Err     error
	Stderr  string
}

// Error returns the formatted error message
func (ce *CommandError) Error() string {
	return fmt.Sprintf("command '%s %s' failed: %v\nstderr: %s",
		ce.Command, strings.Join(ce.Args, " "), ce.Err, ce.Stderr)
}

// RunCommand executes a command and returns its output
func RunCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", &CommandError{
			Command: name,
			Args:    args,
			Err:     err,
			Stderr:  stderr.String(),
		}
	}

	return stdout.String(), nil
}

// RunCommandWithTimeout executes a command with a timeout and returns its output
func RunCommandWithTimeout(timeout int, name string, args ...string) (string, error) {
	// Validate timeout
	if timeout <= 0 {
		return "", fmt.Errorf("timeout must be greater than 0")
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	// Create command with context
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	err := cmd.Run()

	// Check if the context deadline exceeded
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("command timed out after %d seconds", timeout)
	}

	if err != nil {
		return "", &CommandError{
			Command: name,
			Args:    args,
			Err:     err,
			Stderr:  stderr.String(),
		}
	}

	return stdout.String(), nil
}

// RunCommandWithRetry executes a command with retries on failure
func RunCommandWithRetry(retries int, delay time.Duration, name string, args ...string) (string, error) {
	var output string
	var err error

	for i := 0; i <= retries; i++ {
		output, err = RunCommand(name, args...)
		if err == nil {
			return output, nil
		}

		if i < retries {
			time.Sleep(delay)
		}
	}

	return output, fmt.Errorf("command failed after %d retries: %w", retries, err)
}

// FileExists checks if a file exists
func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// DirExists checks if a directory exists
func DirExists(dirPath string) bool {
	info, err := os.Stat(dirPath)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

// CreateDirIfNotExists creates a directory if it doesn't exist
func CreateDirIfNotExists(dirPath string) error {
	if !DirExists(dirPath) {
		return os.MkdirAll(dirPath, 0755)
	}
	return nil
}

// IsOCCommandAvailable checks if the 'oc' command is available
func IsOCCommandAvailable() bool {
	_, err := exec.LookPath("oc")
	return err == nil
}

// IsAuthenticatedToCluster checks if authentication to the cluster is working
func IsAuthenticatedToCluster() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "oc", "whoami")
	return cmd.Run() == nil
}

// GetOpenShiftVersion returns the OpenShift version
func GetOpenShiftVersion() (string, error) {
	out, err := RunCommand("oc", "get", "clusterversion", "-o", "jsonpath={.items[].status.history[0].version}")
	if err != nil {
		return "", fmt.Errorf("failed to get OpenShift version: %w", err)
	}

	version := strings.TrimSpace(out)
	if version == "" {
		return "", fmt.Errorf("empty OpenShift version returned")
	}

	return version, nil
}

// GetOpenShiftMajorMinorVersion returns the major and minor version of OpenShift as a string (e.g., "4.10")
func GetOpenShiftMajorMinorVersion() (string, error) {
	version, err := GetOpenShiftVersion()
	if err != nil {
		return "", err
	}

	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid OpenShift version format: %s", version)
	}

	return fmt.Sprintf("%s.%s", parts[0], parts[1]), nil
}

// ParseYAML parses YAML data into the given object
func ParseYAML(data []byte, obj interface{}) error {
	return yaml.Unmarshal(data, obj)
}

// RunCommandWithInput runs a command with the given input on stdin
func RunCommandWithInput(input string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)

	// Set up pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Start the command
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start command: %w", err)
	}

	// Write input to stdin
	_, err = io.WriteString(stdin, input)
	if err != nil {
		return "", fmt.Errorf("failed to write to stdin: %w", err)
	}

	// Close stdin to signal that no more input is coming
	if err := stdin.Close(); err != nil {
		return "", fmt.Errorf("failed to close stdin: %w", err)
	}

	// Wait for the command to complete
	if err := cmd.Wait(); err != nil {
		return "", &CommandError{
			Command: name,
			Args:    args,
			Err:     err,
			Stderr:  stderr.String(),
		}
	}

	return stdout.String(), nil
}

// CompressDirectory compresses a directory to a zip file
func CompressDirectory(sourcePath, destPath string, password string) error {
	// This would require a more complex implementation with a zip library
	// For now, we'll implement a simplified version using external commands
	if password != "" {
		// With password requires additional libraries
		return fmt.Errorf("password-protected zip not implemented")
	}

	// Check if source directory exists
	if !DirExists(sourcePath) {
		return fmt.Errorf("source directory %s does not exist", sourcePath)
	}

	// Ensure parent directory of destination exists
	destDir := filepath.Dir(destPath)
	if err := CreateDirIfNotExists(destDir); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Use the zip command if available
	_, err := exec.LookPath("zip")
	if err == nil {
		cmd := exec.Command("zip", "-r", destPath, ".")
		cmd.Dir = sourcePath

		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create zip archive: %v, stderr: %s", err, stderr.String())
		}
		return nil
	}

	return fmt.Errorf("zip command not available, compression not implemented")
}

// SafeReadFile reads a file with proper error handling
func SafeReadFile(path string) ([]byte, error) {
	if !FileExists(path) {
		return nil, fmt.Errorf("file not found: %s", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	return data, nil
}

// SafeWriteFile writes data to a file with proper error handling
func SafeWriteFile(path string, data []byte, perm os.FileMode) error {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := CreateDirIfNotExists(dir); err != nil {
		return fmt.Errorf("failed to create directory for file %s: %w", path, err)
	}

	if err := os.WriteFile(path, data, perm); err != nil {
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}

	return nil
}

// VerifyOpenShiftAccess checks if the OpenShift API is accessible and if the user is authenticated
func VerifyOpenShiftAccess() (bool, string) {
	// Check if oc command is available
	if !IsOCCommandAvailable() {
		return false, "OpenShift CLI (oc) not found. Please install the OpenShift CLI to run health checks."
	}

	// Try to reach the OpenShift API server
	cmd := exec.Command("oc", "whoami", "--request-timeout=10s")
	out, err := cmd.CombinedOutput()

	// Convert output to string
	output := strings.TrimSpace(string(out))

	// If we have an error, determine the type of error
	if err != nil {
		if strings.Contains(output, "was refused") || strings.Contains(output, "Unable to connect to the server") {
			return false, "Connection to the OpenShift API server was refused. Please check if the server is running and accessible."
		} else if strings.Contains(output, "Missing or incomplete configuration info") ||
			strings.Contains(output, "You must be logged in") {
			return false, "Not authenticated to OpenShift cluster. Please login using 'oc login'."
		}
		return false, fmt.Sprintf("Error accessing OpenShift API: %s", output)
	}

	// No error means we are connected and authenticated
	return true, fmt.Sprintf("Successfully authenticated as user: %s", output)
}
