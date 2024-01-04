package api

import (
	"cronus/internal/cronus"
	"github.com/gin-gonic/gin"
	"net/http"
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

	api.setupRoutes()

	return api
}

func (c *CronusAPI) setupRoutes() {
	c.router.Static("/assets", "./assets")
	c.router.LoadHTMLGlob("./assets/html/*")

	c.router.GET("/", func(context *gin.Context) {
		context.HTML(http.StatusOK, "index.gohtml", gin.H{
			"title": "Index",
		})
	})

	c.router.GET("/api/cronjobs", ListCronjobsHandler(c.cronManager))
}

func (c *CronusAPI) Run(addr string) error {
	err := c.router.Run(addr)
	if err != nil {
		return err
	}
	return nil
}
