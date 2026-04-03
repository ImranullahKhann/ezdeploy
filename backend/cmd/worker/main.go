package main
import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"


	"ezdeploy/backend/internal/build"
	"ezdeploy/backend/internal/config"
	"ezdeploy/backend/internal/db"
	"ezdeploy/backend/internal/deployment"
	"ezdeploy/backend/internal/logging"
	"ezdeploy/backend/internal/project"
	"ezdeploy/backend/internal/runtime"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func run(parent context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := logging.New(cfg.LogLevel)
	ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	deployService, err := deployment.New(pool)
	if err != nil {
		return err
	}

	projectService, err := project.New(pool)
	if err != nil {
		return err
	}

	buildService, err := build.New(cfg.StorageRoot)
	if err != nil {
		return err
	}

	runtimeService, err := runtime.New()
	if err != nil {
		return err
	}

	logger.Info("worker starting")

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("worker shutting down")
			return nil
		case <-ticker.C:
			// Poll for "deploy" jobs
			job, err := deployService.ClaimJob(ctx, "deploy", 10*time.Minute)
			if err != nil {
				logger.Error("failed to claim job", "error", err)
				continue
			}

			if job == nil {
				continue
			}

			logger.Info("processing job", "job_id", job.ID, "type", job.JobType)
			
			go func(j *deployment.Job) {
				if err := processDeployJob(ctx, j, cfg, deployService, projectService, buildService, runtimeService, logger); err != nil {
					logger.Error("job failed", "job_id", j.ID, "error", err)
					_ = deployService.FailJob(ctx, j.ID, err.Error())
				} else {
					_ = deployService.CompleteJob(ctx, j.ID)
					logger.Info("job completed", "job_id", j.ID)
				}
			}(job)
		}
	}
}

func processDeployJob(
	ctx context.Context, 
	job *deployment.Job, 
	cfg config.Config,
	deployService *deployment.Service,
	projectService *project.Service,
	buildService *build.Service,
	runtimeService *runtime.Service,
	logger *slog.Logger,
) error {
	var payload map[string]any
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	deploymentID, _ := payload["deployment_id"].(string)
	projectID, _ := payload["project_id"].(string)

	dep, err := deployService.GetByID(ctx, deploymentID)
	if err != nil {
		return fmt.Errorf("get deployment: %w", err)
	}

	// For the MVP, we assume the user_id isn't needed here if we have projectID
	// But GetByID in projectService needs userID. 
	// Let's add a internal GetByID to projectService or just bypass for now since worker is internal.
	// Actually, I'll just use a raw query here to get project info for simplicity in the worker.
	
	proj, err := projectService.GetByIDInternal(ctx, projectID)
	if err != nil {
		return fmt.Errorf("get project: %w", err)
	}

	conf, err := projectService.GetConfigInternal(ctx, projectID)
	if err != nil {
		// It's okay if config is missing, we use defaults
		conf = project.ProjectConfig{}
	}

	_ = deployService.UpdateStatus(ctx, deploymentID, deployment.StatusBuilding)
	_ = deployService.AddEvent(ctx, deploymentID, "build_started", "Starting build...", nil)

	// 1. Build
	buildOpts := build.BuildOptions{
		ProjectID:      projectID,
		DeploymentID:   deploymentID,
		RepoURL:        proj.GitRepoURL,
		Branch:         proj.Branch,
		DockerfilePath: getString(conf.DockerfilePath),
		LogWriter:      os.Stdout, // In Phase 8 we will redirect this to a dedicated log service
	}
	if dep.GitBranch != nil {
		buildOpts.Branch = *dep.GitBranch
	}
	if dep.GitCommitSHA != nil {
		buildOpts.CommitSHA = *dep.GitCommitSHA
	}

	imageTag, err := buildService.Build(ctx, buildOpts)
	if err != nil {
		_ = deployService.UpdateStatus(ctx, deploymentID, deployment.StatusBuildFailed)
		_ = deployService.AddEvent(ctx, deploymentID, "build_failed", err.Error(), nil)
		return fmt.Errorf("build: %w", err)
	}

	_ = deployService.AddEvent(ctx, deploymentID, "build_finished", "Build successful", nil)
	_ = deployService.UpdateStatus(ctx, deploymentID, deployment.StatusDeploying)

	// 2. Deploy
	// Allocate a host port if not already assigned
	hostPort := 0
	if dep.Port != nil {
		hostPort = *dep.Port
	} else {
		var err error
		hostPort, err = deployService.AllocatePort(ctx, cfg.RuntimePortMin, cfg.RuntimePortMax)
		if err != nil {
			_ = deployService.UpdateStatus(ctx, deploymentID, deployment.StatusFailed)
			_ = deployService.AddEvent(ctx, deploymentID, "deployment_failed", "failed to allocate port: "+err.Error(), nil)
			return fmt.Errorf("allocate port: %w", err)
		}
	}

	containerPort := 8080
	if conf.Port != nil {
		containerPort = *conf.Port
	}

	startOpts := runtime.StartOptions{
		ProjectID:     projectID,
		DeploymentID:  deploymentID,
		ImageTag:      imageTag,
		HostPort:      hostPort,
		ContainerPort: containerPort,
		Network:       cfg.RuntimeNetwork,
		EnvVars:       conf.EnvVars,
		LogWriter:     os.Stdout,
	}

	containerID, err := runtimeService.StartContainer(ctx, startOpts)
	if err != nil {
		_ = deployService.UpdateStatus(ctx, deploymentID, deployment.StatusFailed)
		_ = deployService.AddEvent(ctx, deploymentID, "deployment_failed", err.Error(), nil)
		return fmt.Errorf("deploy: %w", err)
	}

	shortPrjID := projectID
	if len(shortPrjID) > 12 {
		shortPrjID = shortPrjID[len(shortPrjID)-8:]
	}
	shortDepID := deploymentID
	if len(shortDepID) > 12 {
		shortDepID = shortDepID[len(shortDepID)-8:]
	}
	containerName := fmt.Sprintf("ezd-%s-%s", shortPrjID, shortDepID)
	containerName = strings.ReplaceAll(containerName, "_", "-")
	
	publicURL := fmt.Sprintf("http://localhost:%d", hostPort)
	_ = deployService.UpdateMetadata(ctx, deploymentID, &containerID, &hostPort, &publicURL)

	_ = deployService.AddEvent(ctx, deploymentID, "container_started", "Container started, waiting for health check...", map[string]any{
		"container_id":   containerID,
		"container_name": containerName,
		"host_port":      hostPort,
		"container_port": containerPort,
	})

	// 3. Health Check
	healthPath := getString(conf.HealthcheckPath)
	if healthPath == "" {
		healthPath = "/"
	}

	err = runtimeService.PollHealth(ctx, containerName, containerPort, healthPath, 1*time.Minute)
	if err != nil {
		_ = deployService.UpdateStatus(ctx, deploymentID, deployment.StatusFailed)
		_ = deployService.AddEvent(ctx, deploymentID, "deployment_failed", "Health check failed: "+err.Error(), nil)
		// We should probably stop the container here too
		_ = runtimeService.StopContainer(ctx, containerName)
		return fmt.Errorf("health check: %w", err)
	}

	_ = deployService.UpdateStatus(ctx, deploymentID, deployment.StatusRunning)
	_ = deployService.AddEvent(ctx, deploymentID, "deployment_ready", "Deployment is live and healthy", map[string]any{
		"container_id": containerID,
		"host_port":    hostPort,
		"public_url":   publicURL,
	})

	return nil
}

func getString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
