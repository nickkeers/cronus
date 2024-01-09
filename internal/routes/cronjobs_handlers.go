package routes

import (
	"cronus/internal/cronus"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

func ListCronjobsHandler(manager *cronus.CronJobManager) gin.HandlerFunc {
	return func(context *gin.Context) {
		jobs, err := manager.GetCronJobAndPods()

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

func GetLogsForSinglePod(manager *cronus.CronJobManager) gin.HandlerFunc {
	return func(context *gin.Context) {
		namespace := context.Param("namespace")
		pod := context.Param("pod")

		log, err := manager.FetchPodLog(pod, namespace)

		if err != nil {
			context.String(http.StatusInternalServerError, "Error: Failed to fetch logs for pod")
			return
		}

		context.String(http.StatusOK, *log)
	}
}

func GetLogsForAllPods(manager *cronus.CronJobManager) gin.HandlerFunc {
	return func(context *gin.Context) {
		name := context.Param("name")
		namespace := context.Param("namespace")
		contentType := context.Param("type")

		if contentType == "" {
			contentType = "json"
		}

		fmt.Printf("Fetch logs for %s/%s\n", namespace, name)

		logs, err := manager.GetPodLogs(name, namespace)

		switch contentType {
		case "text":
			if err != nil {
				context.JSON(http.StatusInternalServerError, gin.H{
					"error": "Failed to fetch logs",
					"msg":   err.Error(),
				})
				return
			}
			logMsg := podLogsToSingleString(logs)

			context.String(http.StatusOK, logMsg)
			return
		case "html":
			context.HTML(http.StatusOK, "logsmodal", gin.H{
				"Title": fmt.Sprintf("Logs for job %s/%s", namespace, name),
				"Body":  podLogsToSingleString(logs),
			})
			return
		default:
			context.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid content type",
				"msg":   "Last parameter must be text or html",
			})
			return
		}
	}
}
