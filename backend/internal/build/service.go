package build

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

type Service struct {
	storageRoot string
}

func New(storageRoot string) (*Service, error) {
	if err := os.MkdirAll(storageRoot, 0755); err != nil {
		return nil, fmt.Errorf("create storage root: %w", err)
	}
	return &Service{storageRoot: storageRoot}, nil
}

type BuildOptions struct {
	ProjectID      string
	DeploymentID   string
	RepoURL        string
	Branch         string
	CommitSHA      string
	DockerfilePath string // relative to repo root
	BuildMethod    string // "dockerfile" or "buildpack"
	BuildCmd       string
	StartCmd       string
	InstallCmd     string
	OutputDir      string
	Port           int
	EnvVars        map[string]interface{}
	LogWriter      io.Writer
}

func (s *Service) Build(ctx context.Context, opts BuildOptions) (string, error) {
	buildDir := filepath.Join(s.storageRoot, "builds", opts.DeploymentID)
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return "", fmt.Errorf("create build dir: %w", err)
	}
	// defer os.RemoveAll(buildDir) // Keep it for now to debug Phase 6

	// 1. Clone repo
	if err := s.cloneRepo(ctx, opts, buildDir); err != nil {
		return "", err
	}

	// 2. Build Docker image
	imageTag := fmt.Sprintf("ezdeploy-%s:%s", opts.ProjectID, opts.DeploymentID)
	
	// Choose build method based on configuration
	if opts.BuildMethod == "buildpack" {
		// Build using commands - generate Dockerfile
		if err := s.buildFromCommands(ctx, opts, buildDir, imageTag); err != nil {
			return "", err
		}
	} else {
		// Build using Dockerfile (default)
		if err := s.dockerBuild(ctx, opts, buildDir, imageTag); err != nil {
			return "", err
		}
	}

	return imageTag, nil
}

func (s *Service) BuildStatic(ctx context.Context, opts BuildOptions) (string, error) {
	buildDir := filepath.Join(s.storageRoot, "builds", opts.DeploymentID)
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return "", fmt.Errorf("create build dir: %w", err)
	}
	defer os.RemoveAll(buildDir)

	// 1. Clone repo
	if err := s.cloneRepo(ctx, opts, buildDir); err != nil {
		return "", err
	}

	// 2. Build Docker image that performs the build
	imageTag := fmt.Sprintf("ezdeploy-static-build-%s:%s", opts.ProjectID, opts.DeploymentID)
	containerName := fmt.Sprintf("ezdeploy-static-container-%s", opts.DeploymentID)

	// Detect base image from project structure
	baseImage := s.detectBaseImage(buildDir)
	fmt.Fprintf(opts.LogWriter, "Using base image for build: %s\n", baseImage)

	// Generate a Dockerfile from the build commands
	dockerfileContent := s.generateDockerfile(baseImage, opts)
	dockerfilePath := filepath.Join(buildDir, "Dockerfile.static")
	
	if err := os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0644); err != nil {
		return "", fmt.Errorf("write generated Dockerfile: %w", err)
	}

	fmt.Fprintf(opts.LogWriter, "Building static artifacts in container %s...\n", imageTag)

	// Build the image (this runs the RUN commands inside the container)
	cmd := exec.CommandContext(ctx, "docker", "build", "-t", imageTag, "-f", "Dockerfile.static", ".")
	cmd.Dir = buildDir
	cmd.Stdout = opts.LogWriter
	cmd.Stderr = opts.LogWriter
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("docker build static: %w", err)
	}

	// 3. Extract output directory from the container
	outputDir := opts.OutputDir
	if outputDir == "" {
		outputDir = "dist" // default
	}

	artifactDir := filepath.Join(s.storageRoot, "static-sites", opts.ProjectID, opts.DeploymentID)
	if err := os.MkdirAll(artifactDir, 0755); err != nil {
		return "", fmt.Errorf("create artifact dir: %w", err)
	}

	fmt.Fprintf(opts.LogWriter, "Extracting artifacts from %s to %s...\n", outputDir, artifactDir)

	// Create a temporary container to copy files from it
	createCmd := exec.CommandContext(ctx, "docker", "create", "--name", containerName, imageTag)
	if err := createCmd.Run(); err != nil {
		return "", fmt.Errorf("docker create for extraction: %w", err)
	}
	defer func() {
		// Cleanup: remove container and image
		_ = exec.Command("docker", "rm", "-f", containerName).Run()
		_ = exec.Command("docker", "rmi", imageTag).Run()
	}()

	// Copy the output directory from the container to the host artifact directory
	// Note: docker cp <container>:<src>/. <dest> copies contents correctly
	cpCmd := exec.CommandContext(ctx, "docker", "cp", fmt.Sprintf("%s:/app/%s/.", containerName, outputDir), artifactDir)
	if err := cpCmd.Run(); err != nil {
		return "", fmt.Errorf("docker cp static artifacts: %w", err)
	}

	return artifactDir, nil
}

func (s *Service) cloneRepo(ctx context.Context, opts BuildOptions, buildDir string) error {
	fmt.Fprintf(opts.LogWriter, "Cloning %s (branch: %s)...\n", opts.RepoURL, opts.Branch)

	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", "--branch", opts.Branch, opts.RepoURL, ".")
	cmd.Dir = buildDir
	cmd.Stdout = opts.LogWriter
	cmd.Stderr = opts.LogWriter

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone: %w", err)
	}

	if opts.CommitSHA != "" {
		fmt.Fprintf(opts.LogWriter, "Checking out commit %s...\n", opts.CommitSHA)
		cmd := exec.CommandContext(ctx, "git", "fetch", "--depth", "1", "origin", opts.CommitSHA)
		cmd.Dir = buildDir
		cmd.Stdout = opts.LogWriter
		cmd.Stderr = opts.LogWriter
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git fetch commit: %w", err)
		}

		cmd = exec.CommandContext(ctx, "git", "checkout", opts.CommitSHA)
		cmd.Dir = buildDir
		cmd.Stdout = opts.LogWriter
		cmd.Stderr = opts.LogWriter
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git checkout commit: %w", err)
		}
	}

	return nil
}

func (s *Service) dockerBuild(ctx context.Context, opts BuildOptions, buildDir, imageTag string) error {
	dockerfilePath := opts.DockerfilePath
	if dockerfilePath == "" {
		dockerfilePath = "Dockerfile"
	}

	fmt.Fprintf(opts.LogWriter, "Building docker image %s using %s...\n", imageTag, dockerfilePath)

	cmd := exec.CommandContext(ctx, "docker", "build", "-t", imageTag, "-f", dockerfilePath, ".")
	cmd.Dir = buildDir
	cmd.Stdout = opts.LogWriter
	cmd.Stderr = opts.LogWriter

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker build: %w", err)
	}

	return nil
}

func (s *Service) buildFromCommands(ctx context.Context, opts BuildOptions, buildDir, imageTag string) error {
	fmt.Fprintf(opts.LogWriter, "Building docker image %s from commands...\n", imageTag)

	// Detect base image from project structure
	baseImage := s.detectBaseImage(buildDir)
	fmt.Fprintf(opts.LogWriter, "Detected base image: %s\n", baseImage)

	// Generate a Dockerfile from the build and start commands
	dockerfileContent := s.generateDockerfile(baseImage, opts)
	dockerfilePath := filepath.Join(buildDir, "Dockerfile.generated")
	
	if err := os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0644); err != nil {
		return fmt.Errorf("write generated Dockerfile: %w", err)
	}

	fmt.Fprintf(opts.LogWriter, "Generated Dockerfile:\n%s\n", dockerfileContent)

	// Build using the generated Dockerfile
	cmd := exec.CommandContext(ctx, "docker", "build", "-t", imageTag, "-f", "Dockerfile.generated", ".")
	cmd.Dir = buildDir
	cmd.Stdout = opts.LogWriter
	cmd.Stderr = opts.LogWriter

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker build from commands: %w", err)
	}

	return nil
}

func (s *Service) detectBaseImage(buildDir string) string {
	// Check for common project files to determine base image
	if _, err := os.Stat(filepath.Join(buildDir, "package.json")); err == nil {
		return "node:18-alpine"
	}
	if _, err := os.Stat(filepath.Join(buildDir, "requirements.txt")); err == nil {
		return "python:3.11-slim"
	}
	if _, err := os.Stat(filepath.Join(buildDir, "go.mod")); err == nil {
		return "golang:1.21-alpine"
	}
	if _, err := os.Stat(filepath.Join(buildDir, "Gemfile")); err == nil {
		return "ruby:3.2-slim"
	}
	// Default to Node.js as it's most common for web services
	return "node:18-alpine"
}

func (s *Service) generateDockerfile(baseImage string, opts BuildOptions) string {
	dockerfile := fmt.Sprintf("FROM %s\n\n", baseImage)
	dockerfile += "WORKDIR /app\n\n"
	
	// Copy all files
	dockerfile += "COPY . .\n\n"
	
	// Add environment variables if provided
	if opts.EnvVars != nil && len(opts.EnvVars) > 0 {
		dockerfile += "# Environment Variables\n"
		for key, value := range opts.EnvVars {
			dockerfile += fmt.Sprintf("ENV %s=%v\n", key, value)
		}
		dockerfile += "\n"
	}
	
	// Install dependencies if install command is provided
	if opts.InstallCmd != "" {
		dockerfile += fmt.Sprintf("# Install dependencies\nRUN %s\n\n", opts.InstallCmd)
	}
	
	// Run build command if provided
	if opts.BuildCmd != "" {
		dockerfile += fmt.Sprintf("# Build application\nRUN %s\n\n", opts.BuildCmd)
	}
	
	// Expose port
	port := opts.Port
	if port == 0 {
		port = 8080
	}
	dockerfile += fmt.Sprintf("EXPOSE %d\n\n", port)
	
	// Set start command
	startCmd := opts.StartCmd
	if startCmd == "" {
		startCmd = "npm start" // default
	}
	dockerfile += fmt.Sprintf("CMD %s\n", startCmd)
	
	return dockerfile
}
