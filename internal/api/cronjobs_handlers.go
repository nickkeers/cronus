package api

import (
	"cronus/internal/cronus"
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

		// encode jobs
		context.JSON(http.StatusOK, jobs)
	}
}
