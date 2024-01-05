package main

import (
	"cronus/internal/api"
	"cronus/internal/cronus"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Create a bidirectional channel
	stopCh := make(chan struct{})
	defer close(stopCh)

	// Handle OS signals for graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Start your CronJobManager in a separate goroutine
	go func() {
		manager, err := cronus.NewCronJobManager(stopCh)
		if err != nil {
			panic(err)
		}

		apiRouter := api.NewCronusAPI(manager)
		if err := apiRouter.Run(":8080"); err != nil {
			panic(err)
		}
	}()

	// Block until a signal is received
	<-sigs
	// Signal the informer to stop
	stopCh <- struct{}{}
}
