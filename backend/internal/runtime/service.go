package runtime

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

type Service struct {
	// No specific fields for now
}

func New() (*Service, error) {
	return &Service{}, nil
}

type StartOptions struct {
	ProjectID     string
	DeploymentID  string
	ImageTag      string
	HostPort      int
	ContainerPort int
	Network       string
	EnvVars       map[string]any
	LogWriter     io.Writer
}

func (s *Service) StartContainer(ctx context.Context, opts StartOptions) (string, error) {
	shortPrjID := opts.ProjectID
	if len(shortPrjID) > 12 {
		shortPrjID = shortPrjID[len(shortPrjID)-8:]
	}
	shortDepID := opts.DeploymentID
	if len(shortDepID) > 12 {
		shortDepID = shortDepID[len(shortDepID)-8:]
	}
	containerName := fmt.Sprintf("ezd-%s-%s", shortPrjID, shortDepID)
	containerName = strings.ReplaceAll(containerName, "_", "-")
	
	fmt.Fprintf(opts.LogWriter, "Starting container %s from image %s (host: %d, container: %d, network: %s)...\n", containerName, opts.ImageTag, opts.HostPort, opts.ContainerPort, opts.Network)

	// Build the docker run command
	args := []string{"run", "-d", "--name", containerName}
	
	if opts.Network != "" {
		args = append(args, "--network", opts.Network)
	}

	// Map host port to container port
	args = append(args, "-p", fmt.Sprintf("%d:%d", opts.HostPort, opts.ContainerPort))

	for k, v := range opts.EnvVars {
		args = append(args, "-e", fmt.Sprintf("%s=%v", k, v))
	}

	args = append(args, opts.ImageTag)

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = opts.LogWriter
	cmd.Stderr = opts.LogWriter

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("docker run: %w", err)
	}

	// Get the real container ID
	idCmd := exec.CommandContext(ctx, "docker", "ps", "-aqf", "name="+containerName)
	output, err := idCmd.Output()
	if err != nil {
		return "", fmt.Errorf("get container id: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

func (s *Service) PollHealth(ctx context.Context, containerName string, port int, path string, timeout time.Duration) error {
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	url := fmt.Sprintf("http://%s:%d%s", containerName, port, path)
	
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	for {
		select {
		case <-timeoutCtx.Done():
			return fmt.Errorf("health check timed out for %s", url)
		case <-ticker.C:
			req, err := http.NewRequestWithContext(timeoutCtx, "GET", url, nil)
			if err != nil {
				continue
			}

			resp, err := client.Do(req)
			if err != nil {
				continue
			}
			resp.Body.Close()

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return nil
			}
		}
	}
}

func (s *Service) StopContainer(ctx context.Context, containerName string) error {
	cmd := exec.CommandContext(ctx, "docker", "stop", containerName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker stop: %w", err)
	}

	cmd = exec.CommandContext(ctx, "docker", "rm", containerName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker rm: %w", err)
	}

	return nil
}
