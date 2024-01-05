package routes

import (
	"cronus/internal/cronus"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

func ListCronjobsHandler(manager *cronus.CronJobManager) gin.HandlerFunc {
	return func(context *gin.Context) {
		jobs, err := manager.ListCronJobs()

		if err != nil {
			context.JSON(http.StatusInternalServerError, gin.H{
				"error": "Error listing cron jobs",
				"msg":   err.Error(),
			})
		}

		if jobs == nil {
			context.JSON(http.StatusOK, gin.H{})
		} else {
			context.JSON(http.StatusOK, jobs)
		}
	}
}

func podLogsToSingleString(podLogs map[string]string) string {
	buf := ""

	for pod, logs := range podLogs {
		buf += fmt.Sprintf("Pod: %s\n----------------------------------\n%s\n", pod, logs)
	}

	return buf
}

func GetPodLogs(manager *cronus.CronJobManager) gin.HandlerFunc {
	return func(context *gin.Context) {
		name := context.Param("name")
		namespace := context.Param("namespace")

		fmt.Printf("Fetch logs for %s/%s\n", namespace, name)

		logs, err := manager.GetPodLogs(name, namespace)

		if err != nil {
			context.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to fetch logs",
				"msg":   err.Error(),
			})
			return
		}

		logMsg := podLogsToSingleString(logs)

		context.String(http.StatusOK, logMsg)
	}
}
