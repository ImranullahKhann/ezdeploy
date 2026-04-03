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
	if err := s.dockerBuild(ctx, opts, buildDir, imageTag); err != nil {
		return "", err
	}

	return imageTag, nil
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
