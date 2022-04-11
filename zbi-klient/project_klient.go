package zklient

import (
	"context"
	"github.com/zbitech/common/pkg/utils"
	"log"

	"github.com/zbitech/common/pkg/errs"
	"github.com/zbitech/common/pkg/logger"
	"github.com/zbitech/common/pkg/model/entity"
	"github.com/zbitech/common/pkg/model/object"
	"github.com/zbitech/common/pkg/vars"
)

func (z ZBIClient) CreateProject(ctx context.Context, request *object.ProjectRequest) (*entity.Project, error) {

	// TODO - check permissions from the context

	// TODO - Validate
	// ValidateName()
	// ValidateNamespace()
	// ValidateVersion()
	// ValidateSubscriptionPlan()

	data_mgr := vars.ManagerFactory.GetProjectDataManager(ctx)

	project, err := data_mgr.CreateProject(ctx, request)
	if err != nil {
		return nil, err
	}

	// TODO - validate request

	spec_arr, err := data_mgr.CreateProjectSpec(ctx, project)
	if err != nil {
		logger.Errorf(ctx, "Project creation failed - %s", err)
		return nil, errs.ErrKubernetesResourceFailed
	}

	objects, err := z.client.GenerateKubernetesObjects(ctx, spec_arr)
	if err != nil {
		logger.Errorf(ctx, "Project kubernetes resource generation failed - %s", err)
		return nil, errs.ErrKubernetesResourceFailed
	}

	resources, err := z.client.ApplyResources(ctx, objects)
	if err != nil {
		logger.Errorf(ctx, "Project kubernetes resource creation failed - %s", err)
		return nil, errs.ErrKubernetesResourceFailed
	}

	//	resources := make([]entity.KubernetesProjectResource, len(results))
	//	for index, result := range results {
	//		resource := z.client.GetResource(result)
	//		resources[index] = entity.KubernetesProjectResource{KubernetesResource: *resource, Project: project.Name}
	//	}

	//	resources, err := vars.ManagerFactory.GetProjectResourceManager(ctx).CreateProject(ctx, project)
	//	if err != nil {
	//		logger.Errorf(ctx, "Failed to create project - %s", err)
	//		return nil, errs.ErrKubernetesResourceFailed
	//	}

	proj_repo := vars.RepositoryFactory.GetProjectRepository()
	err = proj_repo.CreateProject(ctx, project)
	if err != nil {
		logger.Errorf(ctx, "Failed to save project to repository - %s", err)
		return nil, errs.ErrDBItemInsertFailed
	}

	err = proj_repo.SaveProjectResources(ctx, project.Name, resources)
	if err != nil {
		logger.Errorf(ctx, "Failed to save project resources to repository - %s", err)
		return nil, errs.ErrDBItemInsertFailed
	}

	return project, nil
}

func (z ZBIClient) CreateProjectIngress(ctx context.Context, project *entity.Project, instance *entity.Instance) error {

	data_mgr := vars.ManagerFactory.GetProjectDataManager(ctx)
	proj_repo := vars.RepositoryFactory.GetProjectRepository()

	instances, err := proj_repo.GetInstancesByProject(ctx, project.Name)
	if err != nil {
		logger.Errorf(ctx, "Unable to retrieve existing instances from database - %s", err)
		return errs.ErrDBError
	}

	instances = append(instances, *instance)

	//	project.AddInstance(entity.InstanceSummary{Name: instance.Name, Version: instance.Version, TotalVolumeClaims: 0, TotalStorage: 0})
	//	update_err := proj_repo.UpdateProject(ctx, project)
	//	if update_err != nil {
	//		logger.Errorf(ctx, "Failed to update project - %s", update_err)
	//	}

	spec_arr, ing_err := data_mgr.CreateProjectIngressSpec(ctx, project, instances)
	if ing_err != nil {
		logger.Errorf(ctx, "Failed to create project ingress - %s", ing_err)
		return errs.ErrKubernetesResourceFailed
	}

	objects, err := z.client.GenerateKubernetesObjects(ctx, spec_arr)
	if err != nil {
		logger.Errorf(ctx, "Project kubernetes resource generation failed - %s", err)
		return errs.ErrKubernetesResourceFailed
	}

	resources, err := z.client.ApplyResources(ctx, objects)
	if err != nil {
		logger.Errorf(ctx, "Project kubernetes resource creation failed - %s", err)
		return errs.ErrKubernetesResourceFailed
	}

	err = proj_repo.SaveProjectResources(ctx, project.Name, resources)
	if err != nil {
		logger.Errorf(ctx, "Failed to save project ingress resource to repository - %s", err)
		return errs.ErrDBItemInsertFailed
	}

	return nil
}

func (z ZBIClient) UpdateProject(ctx context.Context, project *entity.Project, request *object.ProjectRequest) (*entity.Project, error) {

	return nil, nil
}

func (z ZBIClient) DeleteProject(ctx context.Context, name string) error {

	proj_repo := vars.RepositoryFactory.GetProjectRepository()

	project, err := proj_repo.GetProject(ctx, name)
	if err != nil {
		return err
	}

	//resources, err := z.repository.GetProjectResources(name)
	// TODO - check permissions from the context

	//err_count, errors := z.projectManager.DeleteProject(ctx, project, resources)

	err = z.client.DeleteNamespace(ctx, project.GetNamespace())
	if err != nil {
		return err
	}

	return proj_repo.UpdateProjectStatus(ctx, project.Name, "DELETED")
}

func (z ZBIClient) GetAllProjects(ctx context.Context) ([]entity.Project, error) {
	// TODO - validate caller's credentials
	proj_repo := vars.RepositoryFactory.GetProjectRepository()

	projects, err := proj_repo.GetProjects(ctx)
	if err != nil {
		return nil, err
	}

	// TODO - check permissions from the context

	//return projects
	log.Printf("Returning - %s", utils.MarshalObject(projects))
	return projects, nil
}

func (z ZBIClient) GetProject(ctx context.Context, name string) (*entity.Project, error) {

	proj_repo := vars.RepositoryFactory.GetProjectRepository()

	project, err := proj_repo.GetProject(ctx, name)
	if err != nil {
		return nil, err
	}

	// TODO - check permissions from the context

	return project, nil
}

func (z ZBIClient) GetProjectsByOwner(ctx context.Context, owner string) ([]entity.Project, error) {

	proj_repo := vars.RepositoryFactory.GetProjectRepository()

	projects, err := proj_repo.GetProjectsByOwner(ctx, owner)
	if err != nil {
		return nil, err
	}

	// TODO - check permissions from the context

	return projects, nil
}

func (z ZBIClient) GetProjectsByTeam(ctx context.Context, team string) ([]entity.Project, error) {
	proj_repo := vars.RepositoryFactory.GetProjectRepository()

	projects, err := proj_repo.GetProjectsByTeam(ctx, team)
	if err != nil {
		return nil, err
	}

	return projects, nil
}

func (z ZBIClient) GetProjectResources(ctx context.Context, name string) ([]entity.KubernetesResource, error) {

	// TODO - check permissions from the context
	proj_repo := vars.RepositoryFactory.GetProjectRepository()

	project, err := proj_repo.GetProject(ctx, name)
	if err != nil {
		return nil, err
	}

	history, err := proj_repo.GetProjectResources(ctx, project.GetName())
	if err != nil {
		return nil, err
	}

	return history, nil
}
