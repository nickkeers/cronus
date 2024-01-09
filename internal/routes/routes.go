package routes

import (
	"cronus/internal/cronus"
	"fmt"
	"github.com/gin-gonic/gin"
	"html/template"
	"net/http"
	"time"
)

type CronusAPI struct {
	router      *gin.Engine
	cronManager *cronus.CronJobManager
}

func NewCronusAPI(manager *cronus.CronJobManager) *CronusAPI {
	api := &CronusAPI{
		router:      gin.Default(),
		cronManager: manager,
	}

	api.router.SetFuncMap(template.FuncMap{
		"readableDateTime": readableDateTime,
	})

	api.setupRoutes()

	return api
}

func readableDateTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

func (c *CronusAPI) setupRoutes() {
	c.router.Static("/assets", "./assets")
	c.router.LoadHTMLGlob("./assets/html/*")

	c.router.GET("/", func(context *gin.Context) {
		jobs, err := c.cronManager.GetCronJobAndPods()

		if err != nil || jobs == nil {
			fmt.Println("no jobs found")
			context.HTML(http.StatusInternalServerError, "index.html", gin.H{
				"error": err.Error(),
			})
			return
		}

		fmt.Printf("jobs: %+v\n", jobs)

		context.HTML(http.StatusOK, "index.gohtml", gin.H{
			"title":       "Index",
			"CronJobInfo": jobs,
		})
	})

	c.router.GET("/api/cronjobs", ListCronjobsHandler(c.cronManager))
	c.router.GET("/api/logs/:namespace/:name/:type", GetLogsForAllPods(c.cronManager))
}

func (c *CronusAPI) Run(addr string) error {
	err := c.router.Run(addr)
	if err != nil {
		return err
	}
	return nil
}
