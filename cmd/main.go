package main

import "cronus/internal/api"

func main() {
	apiRouter := api.NewCronusAPI()

	if err := apiRouter.Run(":8080"); err != nil {
		// just temporary, I promise
		panic(err)
	}
}
