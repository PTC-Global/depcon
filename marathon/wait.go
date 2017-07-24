package marathon

import (
	"github.com/ContainX/depcon/pkg/logger"
	"github.com/ContainX/depcon/utils"
	"time"
)

var logWait = logger.GetLogger("depcon.deploy.wait")

func (c *MarathonClient) WaitForApplication(id string, timeout time.Duration) error {
	t_now := time.Now()
	t_stop := t_now.Add(timeout)

	logWaitApplication(id)
	for {
		if time.Now().After(t_stop) {
			return ErrorTimeout
		}

		app, err := c.GetApplication(id)
		if err == nil {
			if app.DeploymentID == nil || len(app.DeploymentID) <= 0 {
				logWait.Info("Application deployment has completed for %s, elapsed time %s", id, utils.ElapsedStr(time.Since(t_now)))
				if app.HealthChecks != nil && len(app.HealthChecks) > 0 {
					err := c.WaitForApplicationHealthy(id, timeout)
					if err != nil {
						logWait.Errorf("Error waiting for application '%s' to become healthy: %s", id, err.Error())
					}
				} else {
					logWait.Warning("No health checks defined for '%s', skipping waiting for healthy state", id)
				}
				return nil
			}
		}
		logWaitApplication(id)
		time.Sleep(time.Duration(2) * time.Second)
	}
}

func (c *MarathonClient) WaitForApplicationHealthy(id string, timeout time.Duration) error {
	t_now := time.Now()
	t_stop := t_now.Add(timeout)
	duration := time.Duration(2) * time.Second
	for {
		if time.Now().After(t_stop) {
			return ErrorTimeout
		}
		app, err := c.GetApplication(id)
		if err != nil {
			return err
		}
		total := app.TasksStaged + app.TasksRunning
		diff := total - app.TasksHealthy
		if diff == 0 {
			logWait.Info("%v of %v expected instances are healthy.  Elapsed health check time of %s", app.TasksHealthy, total, utils.ElapsedStr(time.Since(t_now)))
			return nil
		}
		logWait.Info("%v healthy instances.  Waiting for %v total instances. Retrying check in %v seconds", app.TasksHealthy, total, duration)
		time.Sleep(duration)
	}
}

func (c *MarathonClient) WaitForDeployment(id string, timeout time.Duration) error {

	t_now := time.Now()
	t_stop := t_now.Add(timeout)

	logWaitDeployment(id)

	for {
		if time.Now().After(t_stop) {
			return ErrorTimeout
		}
		if found, _ := c.HasDeployment(id); !found {
			logWait.Info("Deployment has completed for %s, elapsed time %s", id, utils.ElapsedStr(time.Since(t_now)))
			return nil
		}
		logWaitDeployment(id)
		time.Sleep(time.Duration(2) * time.Second)
	}
}

func logWaitDeployment(id string) {
	logWait.Info("Waiting for deployment %s", id)
}

func logWaitApplication(id string) {
	logWait.Info("Waiting for application deployment to complete for %s", id)
}
