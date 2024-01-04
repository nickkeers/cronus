# Cronus

This app shows details of CronJobs in your Kubernetes cluster.

## Project Conventions

This repository uses the [standard Go project layout](https://github.com/golang-standards/project-layout).

For the API part [Gin](https://github.com/gin-gonic/gin) is used.

For the dashboard it uses [Bootstrap 5](https://getbootstrap.com/) + [htmx](https://htmx.org)

## Local Deployment

Pre-requisites:

- Install Docker
- Docker desktop Kubernetes integration enabled

1. `docker build -t nickkeers/cronus:local`
1. `kubectl config use-context docker-desktop`
1. `kubectl apply -k manifests`
1. [`http://localhost:8080`](http://localhost:8080) is available.

Reload the image with:

`docker build -t nickkeers/cronus:local && kubectl rollout -n cronus deployment/cronus`
