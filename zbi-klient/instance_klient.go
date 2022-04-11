package zklient

import (
	"context"
	"errors"

	"github.com/zbitech/common/pkg/errs"
	"github.com/zbitech/common/pkg/logger"
	"github.com/zbitech/common/pkg/model/entity"
	"github.com/zbitech/common/pkg/model/ztypes"
	"github.com/zbitech/common/pkg/vars"
)

func (z ZBIClient) CreateInstance(ctx context.Context, project *entity.Project, request ztypes.InstanceRequestIF) (*entity.Instance, error) {

	// TODO - Validate spec requirements - StorageClass, Namespace, etc
	// ValidateInstanceType()
	// ValidateName()
	// ValidateNamespace()
	// ValidateVersion()
	// ValidateStorageClass()
	// ValidateSubscriptionPlan()

	data_mgr := vars.ManagerFactory.GetProjectDataManager(ctx)

	instance, err := data_mgr.CreateInstance(ctx, project, request)
	if err != nil {
		return nil, err
	}

	spec_arr, err := data_mgr.CreateProjectSpec(ctx, project)
	if err != nil {
		logger.Errorf(ctx, "Instance creation failed - %s", err)
		return nil, errs.ErrKubernetesResourceFailed
	}

	objects, err := z.client.GenerateKubernetesObjects(ctx, spec_arr)
	if err != nil {
		logger.Errorf(ctx, "Instance kubernetes resource generation failed - %s", err)
		return nil, errs.ErrKubernetesResourceFailed
	}

	resources, err := z.client.ApplyResources(ctx, objects)
	if err != nil {
		logger.Errorf(ctx, "Instance kubernetes resource creation failed - %s", err)
		return nil, errs.ErrKubernetesResourceFailed
	}

	proj_repo := vars.RepositoryFactory.GetProjectRepository()
	admin_repo := vars.RepositoryFactory.GetAdminRepository()

	err = proj_repo.CreateInstance(ctx, instance)
	if err != nil {
		return nil, err
	}

	rsc_config, ok := data_mgr.GetInstanceResources(ctx, request.GetInstanceType(), request.GetVersion())
	if !ok {
		return nil, errs.ErrInstanceResourceFailed
	}

	i_policy := entity.NewInstancePolicy(instance.Project, instance.Name, rsc_config.Methods, request.AllowMethods())
	admin_repo.StoreInstancePolicy(ctx, i_policy)

	err = proj_repo.SaveInstanceResources(ctx, instance.Project, instance.Name, resources)
	if err != nil {
		logger.Errorf(ctx, "Failed to create resource - %s", err)
		return nil, errs.ErrKubernetesResourceFailed
	}

	z.CreateProjectIngress(ctx, project, instance)

	//TODO - report instance failure?
	return instance, nil
}

func (z ZBIClient) UpdateInstance(ctx context.Context, project *entity.Project, instance *entity.Instance, request ztypes.InstanceRequestIF) (*entity.Instance, error) {

	return nil, nil
}

func (z ZBIClient) DeleteInstance(ctx context.Context, projectName, instanceName string) error {

	//	rsc_mgr := vars.ManagerFactory.GetProjectResourceManager(ctx)
	proj_repo := vars.RepositoryFactory.GetProjectRepository()

	instance, err := proj_repo.GetInstance(ctx, projectName, instanceName)
	if err != nil {
		return err
	}

	// rsc_manager := factory.GetInstanceResourceManager(ctx, instance.GetInstanceType())
	// if rsc_manager == nil {
	// 	logger.Errorf(ctx, "Instance %s Resource Manager not initialized", instance.GetInstanceType())
	// 	return errors.New("Instance Resource Manager not initialized")
	// }

	resources, err := proj_repo.GetInstanceResources(ctx, instance.Project, instance.Name)
	if err != nil {
		return err
	}

	error_count := 0
	_errors := make([]error, 0)
	failed := make([]string, 0)

	for _, resource := range resources {
		err := z.client.DeleteDynamicResource(ctx, resource.Namespace, resource.Name, *resource.GVR)
		if err != nil {
			error_count++
			failed = append(failed, resource.Name)
			_errors = append(_errors, err)
		}
	}

	if error_count > 0 {
		return errors.New("Delete error")
	}

	return nil
}

func (z ZBIClient) GetAllInstances(ctx context.Context) ([]entity.Instance, error) {

	// TODO - check permissions from the context

	proj_repo := vars.RepositoryFactory.GetProjectRepository()

	instances, err := proj_repo.GetInstances(ctx)
	if err != nil {
		return nil, err
	}

	return instances, nil
}

func (z ZBIClient) GetInstancesByProject(ctx context.Context, projectName string) ([]entity.Instance, error) {

	// TODO - check permissions from the context
	proj_repo := vars.RepositoryFactory.GetProjectRepository()

	instances, err := proj_repo.GetInstancesByProject(ctx, projectName)
	if err != nil {
		return nil, err
	}

	return instances, nil
}

func (z ZBIClient) GetInstancesByOwner(ctx context.Context, owner string) ([]entity.Instance, error) {

	// TODO - check permissions from the context
	proj_repo := vars.RepositoryFactory.GetProjectRepository()

	instances, err := proj_repo.GetInstancesByOwner(ctx, owner)
	if err != nil {
		return nil, err
	}

	return instances, nil
}

func (z ZBIClient) GetInstanceByName(ctx context.Context, projectName, instanceName string) (*entity.Instance, error) {

	proj_repo := vars.RepositoryFactory.GetProjectRepository()

	instance, err := proj_repo.GetInstance(ctx, projectName, instanceName)
	if err != nil {
		return nil, err
	}

	// TODO - check permissions from the context

	return instance, nil
}

// func (z ZBIClient) GetInstanceById(ctx context.Context, id string) (*entity.Instance, error) {

// 	instance, err := z.repository.GetInstanceById(ctx, id)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// TODO - check permissions from the context

// 	return instance, nil
// }

func (z ZBIClient) GetInstanceResources(ctx context.Context, project_name, instance_name string) ([]entity.KubernetesResource, error) {

	return nil, nil
}
