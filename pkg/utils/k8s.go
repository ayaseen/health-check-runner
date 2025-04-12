package utils

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// GetClusterConfig returns the Kubernetes client configuration
func GetClusterConfig() (*rest.Config, error) {
	// Try to use in-cluster config first
	config, err := rest.InClusterConfig()
	if err != nil {
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

		// Build config from kubeconfig file
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build config from kubeconfig: %v", err)
		}
	}

	return config, nil
}

// GetClientSet returns a Kubernetes clientset
func GetClientSet() (*kubernetes.Clientset, error) {
	config, err := GetClusterConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %v", err)
	}

	return clientset, nil
}

// RunCommand executes a command and returns its output
func RunCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("command failed: %v, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// RunCommandWithTimeout executes a command with a timeout and returns its output
func RunCommandWithTimeout(timeout int, name string, args ...string) (string, error) {
	// Create command
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Create a channel to signal command completion
	done := make(chan error, 1)

	// Start the command
	err := cmd.Start()
	if err != nil {
		return "", fmt.Errorf("failed to start command: %v", err)
	}

	// Wait for the command in a goroutine
	go func() {
		done <- cmd.Wait()
	}()

	// Create a timer for timeout
	timer := make(chan bool, 1)
	if timeout > 0 {
		go func() {
			time.Sleep(time.Duration(timeout) * time.Second)
			timer <- true
		}()
	}

	// Wait for either command completion or timeout
	select {
	case <-timer:
		// Timeout occurred, kill the command
		cmd.Process.Kill()
		return "", fmt.Errorf("command timed out after %d seconds", timeout)
	case err := <-done:
		// Command completed
		if err != nil {
			return "", fmt.Errorf("command failed: %v, stderr: %s", err, stderr.String())
		}
		return stdout.String(), nil
	}
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
	_, err := RunCommand("oc", "whoami")
	return err == nil
}

// GetOpenShiftVersion returns the OpenShift version
func GetOpenShiftVersion() (string, error) {
	out, err := RunCommand("oc", "get", "clusterversion", "-o", "jsonpath={.items[].status.history[0].version}")
	if err != nil {
		return "", fmt.Errorf("failed to get OpenShift version: %v", err)
	}

	version := strings.TrimSpace(out)
	if version == "" {
		return "", fmt.Errorf("empty OpenShift version")
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

// CompressDirectory compresses a directory to a zip file
func CompressDirectory(sourcePath, destPath string, password string) error {
	// Implementation would go here
	// This is a placeholder for now
	return nil
}
