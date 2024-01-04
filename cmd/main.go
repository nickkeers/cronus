package main

import (
	"cronus/internal/api"
	"cronus/internal/cronus"
)

func main() {
	manager, err := cronus.NewCronJobManager()

	if err != nil {
		panic(err)
	}

	apiRouter := api.NewCronusAPI(manager)

	if err := apiRouter.Run(":8080"); err != nil {
		// just temporary, I promise
		panic(err)
	}
}
