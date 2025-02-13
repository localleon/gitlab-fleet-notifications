package main

import (
	"context"

	fleetapi "github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type GitRepoReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *GitRepoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&fleetapi.GitRepo{}).
		Owns(&fleetapi.GitRepo{}). // Ensures it watches updates
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}

func (r *GitRepoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// reconcile implementation
	log := log.FromContext(ctx)

	// GitRepo
	gitrepo := &fleetapi.GitRepo{}
	// Fetch the gitrepo instance
	if err := r.Get(ctx, req.NamespacedName, gitrepo); err != nil {
		// gitrepo object not found, it might have been deleted
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// Check if environment exists, if not create
	repoName, ok1 := gitrepo.Labels["gitlab.com/repo-name"]
	environmentName, ok2 := gitrepo.Labels["gitlab.com/environment"]
	environmentURL, ok3 := gitrepo.Labels["gitlab.com/environment-url"]

	if ok1 && ok2 && ok3 {
		// Check Project-ID and Environments from configured repo
		id, _ := getProjectID(gitlabClient, repoName)
		// Prepare the environment
		environmentID, err := prepareProjectEnvironment(id, environmentName, environmentURL, ctx)
		if err != nil {
			log.Error(err, "Failed to create environment for project "+repoName)
			return ctrl.Result{}, err
		} else {
			log.Info("Environment prepared successfully ", "environment", environmentName, "gitRepo", gitrepo.Name)
		}

		// Reconcile the deployment job#
		reconcileGitRepoDeployment(id, environmentName, gitrepo, ctx)
		log.Info("Reconciled GitRepo Deployment for", "environment", environmentName, "gitRepo", gitrepo.Name)

		// Update Environment status
		reconcileEnvironmentStatus(id, environmentID, gitrepo, ctx)
		log.Info("Reconciled Environment Status for", "environment", environmentName, "gitRepo", gitrepo.Name)

	} else {
		log.Info("Unable to setup/check environment for " + gitrepo.Name + " required labels for operator are not set")
	}

	return ctrl.Result{}, nil
}

func reconcileGitRepoDeployment(projectID int, environmentName string, gitRepo *fleetapi.GitRepo, ctx context.Context) {
	deploymentID, deployStatus := checkIfGitlabDeploymentExists(projectID, environmentName, gitRepo)

	if !deployStatus { // Deployment does not exist for current commit

		deploymentID, _ = createGitlabDeployment(projectID, environmentName, gitRepo)
	}

	// We now check if the desired resources are ready and update the deployment status
	if gitRepo.Status.ResourceCounts.DesiredReady == gitRepo.Status.ResourceCounts.Ready {
		// Set the deployment status based on the condition from GitRepo object
		gitlabClient.Deployments.UpdateProjectDeployment(projectID, deploymentID, &gitlab.UpdateProjectDeploymentOptions{
			Status: gitlab.Ptr(gitlab.DeploymentStatusSuccess),
		})
	}

	// If the Fleet GitJob has an error, we do not reconcile and fail the deployment
	if gitRepo.Status.Display.Error {
		gitlabClient.Deployments.UpdateProjectDeployment(projectID, deploymentID, &gitlab.UpdateProjectDeploymentOptions{
			Status: gitlab.Ptr(gitlab.DeploymentStatusFailed),
		})
	}
}

func reconcileEnvironmentStatus(projectID int, environmentID int, gitRepo *fleetapi.GitRepo, ctx context.Context) {
	staticDescriptionPrefix := "This environment is currently beeing controlled by Rancher Fleet and Gitlab messages are propagated via a custom Kubernetes controller. Please visit the Rancher Fleet Page of your Cluster deployment for more informatin. Here's the information from Fleet: \n\n"
	// We always handle the latest status update and check if there's a message available
	objCondition := getLatestWranglerObjectCondition(gitRepo.Status.Conditions)
	if objCondition != nil {
		descriptionMessage := ""
		// Construct description string from current messages
		if objCondition.Message != "" {
			descriptionMessage = staticDescriptionPrefix + "Status: " + string(objCondition.Type) + ", with message: " + objCondition.Message

		} else { // If we do not have a message at the latest condition object, we can assume the environment is reconciled
			descriptionMessage = staticDescriptionPrefix + "Environment is currently reconciled. Cluster-state is in sync with Repository"
		}
		// Propagate information to Gitlab
		gitlabClient.Environments.EditEnvironment(projectID, environmentID, &gitlab.EditEnvironmentOptions{
			Description: &descriptionMessage,
		})
	}
}
