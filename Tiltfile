load('ext://namespace', 'namespace_create', 'namespace_inject')

development_namespace='cronus'

namespace_create(development_namespace)

# There is almost certainly a way of just listing these files
# or using the kustomization file, but I'm being crude
k8s_yaml([
  'manifests/cronjob.yaml',
  'manifests/deployment.yaml',
  'manifests/kubectl-deployment.yaml',
  'manifests/role.yaml',
  'manifests/rolebinding.yaml',
  'manifests/service.yaml',
  'manifests/serviceaccount.yaml',
])


# Build: tell Tilt what images to build from which directories

docker_build('nickkeers/cronus', '.')

# Watch: tell Tilt how to connect locally/categorise things

k8s_resource('cronus', labels=["app"])
k8s_resource('hello', labels=["cronjobs"])