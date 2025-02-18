# gitlab-fleet-webhook-operator

The `gitlab-fleet-webhook-operator` is a Kubernetes controller that integrates GitLab with Rancher Fleet. It automates the creation and management of GitLab environments and deployments based on the state of `GitRepo` custom resources in your Kubernetes cluster.

## Getting Started

### Prerequisites
- go version v1.23.0+
- docker version 17.03+.
- kubectl version v1.27+.
- Access to a Kubernetes v1.27+ cluster.

### Required Environment Variables
The controller requires the following environment variables to be set:

- `GITLAB_TOKEN`: The personal access token for authenticating with the GitLab API.
- `GITLAB_URL`: The base URL of your GitLab instance (e.g., `https://git.mydomain.com/api/v4`).

### Required Labels for GitRepo Objects
The `GitRepoReconciler` requires the following labels to be set on `GitRepo` objects:

- `gitlab.com/repo-name`: The name of the GitLab repository.
- `gitlab.com/environment`: The name of the environment to be managed in GitLab.
- `gitlab.com/environment-url`: The URL of the environment.

Example:

```sh
kubectl label -n fleet-local gitrepo sample-simple gitlab.com/repo-name=approval-rule-testing
kubectl label -n fleet-local gitrepo sample-simple gitlab.com/environment=staging
kubectl label -n fleet-local gitrepo sample-simple gitlab.com/environment-url=http://staging.example.com
```

The controller will then automatically create an environment and deployments within the Gitlab repository.

### To Deploy on the cluster
**Build and your image**

```sh
docker build . -t gitlab-fleet-webhook-operator:local
```

or get the latest iamge published by a Github Action from ghcr.io 

```sh
ghcr.io/localleon/gitlab-fleet-notifications:main 
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh 
kubectl create ns gitlab-fleet-operator
# Create secret with Gitlab-Token for Authentication
export GITLAB_TOKEN="YOUR_GITLAB_TOKEN_WITH_WRITE_API"
kubectl create secret -n gitlab-fleet-operator generic gitlab-token --from-literal GITLAB_TOKEN=$GITLAB_TOKEN
# Deploy the operator 
kubectl apply -n gitlab-fleet-operator -f gitlab-fleet-webhook-operator/config/manager/
```

You can edit the deployments to your liking if you want to deploy to a different namespace or use a different Gitlab-URL. 

## Contributing

Contributions to the project are welcome, however this is currently more of a proof of concept

## License

Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.