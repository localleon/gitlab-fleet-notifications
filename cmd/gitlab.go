package main

import (
	"context"
	"fmt"
	"os"

	fleetapi "github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/log"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// setupGitlabClient creates a Gitlab Client to our Gitlab instance specified by the environment vars
func setupGitlabClient() (*gitlab.Client, error) {
	// Get Tokens for auth from env
	gitlab_token := os.Getenv("GITLAB_TOKEN")
	gitlab_url := os.Getenv("GITLAB_URL") // "https://git.mydomain.com/api/v4"

	git, err := gitlab.NewClient(gitlab_token, gitlab.WithBaseURL(gitlab_url))
	return git, err
}

func getProjectID(git *gitlab.Client, projectName string) (int, error) {
	projects, _, err := git.Projects.ListProjects(&gitlab.ListProjectsOptions{
		Search: gitlab.Ptr(projectName),
	})
	if err != nil {
		return 0, err
	}
	if len(projects) == 0 {
		return 0, fmt.Errorf("gitlab project not found")
	}
	return projects[0].ID, nil
}

func prepareProjectEnvironment(projectID int, environmentName, environmentURL string, ctx context.Context) (int, error) {
	// Get Logging endpoint from context
	log := log.FromContext(ctx)

	environments, _, err := gitlabClient.Environments.ListEnvironments(projectID, nil)
	if err != nil {
		log.Error(err, "Failed to get environments for project")
	}
	// Check if the environment already exists
	environmentID, environmentExists := checkIfEnvironmentExists(environmentName, environments)
	if environmentExists {
		log.Info("Environment " + environmentName + " already exists")
		return environmentID, nil
	} else {
		// Create the environment if it does not exist
		environment, _, err := gitlabClient.Environments.CreateEnvironment(projectID, &gitlab.CreateEnvironmentOptions{
			Name:        gitlab.Ptr(environmentName),
			ExternalURL: gitlab.Ptr(environmentURL), // Adds the URL without https/http from the labels
		})
		if err != nil {
			return 0, err
		}
		return environment.ID, nil
	}
}

func checkIfEnvironmentExists(environmentName string, environments []*gitlab.Environment) (int, bool) {
	for _, env := range environments {
		if env.Name == environmentName {
			return env.ID, true
		}
	}
	return 0, false
}

func checkIfGitlabDeploymentExists(projectID int, environmentName string, gitRepo *fleetapi.GitRepo) (int, bool) {
	deplyoments, _, err := gitlabClient.Deployments.ListProjectDeployments(projectID, &gitlab.ListProjectDeploymentsOptions{
		Environment: &environmentName,
	})
	if err != nil { // Deployment probaly doesnt exist
		return 0, false // Deployment does not exist
	}
	for _, deployment := range deplyoments {
		if deployment.SHA == gitRepo.Status.Commit { // Deployment exists and matches current SHA value in GitRepo

			return deployment.ID, true
		}
	}
	return 0, false // Deployment does not exist
}

func createGitlabDeployment(projectID int, environmentName string, gitRepo *fleetapi.GitRepo) (int, error) {
	// Create the Deployment in the project referencing the environment from the current commitID
	deployments, _, err := gitlabClient.Deployments.CreateProjectDeployment(projectID, &gitlab.CreateProjectDeploymentOptions{
		Environment: &environmentName,
		Ref:         &gitRepo.Spec.Branch,
		Tag:         gitlab.Ptr(false), // we don't reference tags in fleet, only commits
		SHA:         &gitRepo.Status.Commit,
		Status:      gitlab.Ptr(gitlab.DeploymentStatusRunning),
	})

	return deployments.ID, err
}
