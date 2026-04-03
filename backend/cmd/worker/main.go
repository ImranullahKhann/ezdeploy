package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ezdeploy/backend/internal/config"
	"ezdeploy/backend/internal/db"
	"ezdeploy/backend/internal/deployment"
	"ezdeploy/backend/internal/logging"
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
			job, err := deployService.ClaimJob(ctx, "deploy", 5*time.Minute)
			if err != nil {
				logger.Error("failed to claim job", "error", err)
				continue
			}

			if job == nil {
				continue
			}

			logger.Info("processing job", "job_id", job.ID, "type", job.JobType)
			
			// For Phase 5, we just log that we are processing and complete the job.
			// Actual deployment logic will be in Phase 6 and 7.
			
			var payload map[string]any
			if err := json.Unmarshal(job.Payload, &payload); err != nil {
				logger.Error("failed to unmarshal payload", "job_id", job.ID, "error", err)
				_ = deployService.FailJob(ctx, job.ID, "invalid payload: "+err.Error())
				continue
			}

			deploymentID, ok := payload["deployment_id"].(string)
			if !ok {
				logger.Error("missing deployment_id in payload", "job_id", job.ID)
				_ = deployService.FailJob(ctx, job.ID, "missing deployment_id")
				continue
			}

			// Simulate some work
			logger.Info("starting deployment", "deployment_id", deploymentID)
			_ = deployService.UpdateStatus(ctx, deploymentID, deployment.StatusBuilding)
			_ = deployService.AddEvent(ctx, deploymentID, "build_started", "Build started (mock)", nil)
			
			time.Sleep(2 * time.Second)
			
			_ = deployService.UpdateStatus(ctx, deploymentID, deployment.StatusRunning)
			_ = deployService.AddEvent(ctx, deploymentID, "deployment_ready", "Deployment is ready (mock)", nil)
			
			if err := deployService.CompleteJob(ctx, job.ID); err != nil {
				logger.Error("failed to complete job", "job_id", job.ID, "error", err)
			} else {
				logger.Info("job completed", "job_id", job.ID)
			}
		}
	}
}
