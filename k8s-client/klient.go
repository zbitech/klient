package client

import (
	"time"

	"github.com/zbitech/common/pkg/errs"
	"github.com/zbitech/common/pkg/logger"
	"github.com/zbitech/common/pkg/model/entity"
	"github.com/zbitech/common/pkg/vars"
	"github.com/zbitech/klient/internal/helper"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"

	"context"
	"encoding/json"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
)

var (
	decUnstructured = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
)

type Klient struct {
	KubernetesClient kubernetes.Interface
	DynamicClient    dynamic.Interface
	Mapper           *restmapper.DeferredDiscoveryRESTMapper
	restConfg        *rest.Config
}

func NewRestConfig(ctx context.Context) (*rest.Config, error) {
	var restConfig *rest.Config
	var err error
	if vars.AppConfig.Kubernetes.InCluster {
		logger.Infof(ctx, "Connecting to in-cluster kubernetes")
		restConfig, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	} else {
		logger.Infof(ctx, "Connecting to kubernetes with config file - %s", vars.AppConfig.Kubernetes.KubeConfig)
		restConfig, err = clientcmd.BuildConfigFromFlags("", vars.AppConfig.Kubernetes.KubeConfig)
		if err != nil {
			return nil, err
		}
	}
	return restConfig, nil
}

func NewKlient(ctx context.Context) (*Klient, error) {

	cfg, err := NewRestConfig(ctx)
	if err != nil {
		logger.Errorf(ctx, "Unable to create kubernetes configuration - %s", err)
		return nil, errs.ErrKubernetesConnFailed
	}

	kubernetesClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	dynamicClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(kubernetesClient.Discovery()))

	return &Klient{
		KubernetesClient: kubernetesClient,
		DynamicClient:    dynamicClient,
		Mapper:           mapper,
		restConfg:        cfg,
	}, nil
}

func (k *Klient) GetMapper() *restmapper.DeferredDiscoveryRESTMapper {
	return k.Mapper
}

func (k *Klient) GetGVR(gvk schema.GroupVersionKind) (*schema.GroupVersionResource, error) {
	Mapping, err := k.Mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, err
	}

	return &Mapping.Resource, nil
}

func (k *Klient) ApplyResource(ctx context.Context, object *unstructured.Unstructured) (*entity.KubernetesResource, error) {

	data, err := json.Marshal(object)
	if err != nil {
		return nil, err
	}

	dr, err := helper.GetDynamicResourceInterface(k.DynamicClient, k.Mapper, object)
	if err != nil {
		return nil, err
	}

	result, err := dr.Patch(ctx, object.GetName(), types.ApplyPatchType, data, metav1.PatchOptions{FieldManager: "zbi-controller"})
	if err != nil {
		logger.Errorf(ctx, "Failed to create resource - %s", err)
		return nil, errs.ErrKubernetesResourceFailed
	} else {
		logger.Infof(ctx, "Successfully created %s of kind %s", result.GetName(), result.GetKind())
	}

	return k.GetResource(result), err
}

func (k *Klient) ApplyResources(ctx context.Context, objects []*unstructured.Unstructured) ([]entity.KubernetesResource, error) {

	results := make([]entity.KubernetesResource, 0)

	for _, object := range objects {
		res, err := k.ApplyResource(ctx, object)
		if err != nil {
			return nil, err
		}

		results = append(results, *res)
	}

	return results, nil
}

func (k *Klient) DeleteDynamicResource(ctx context.Context, namespace, name string, resource schema.GroupVersionResource) error {
	return k.DynamicClient.Resource(resource).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

func (k *Klient) DeleteNamespace(ctx context.Context, namespace string) error {
	return k.KubernetesClient.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{})
}

func (k *Klient) GetDynamicResource(ctx context.Context, namespace, name string, resource schema.GroupVersionResource) (*unstructured.Unstructured, error) {

	result, err := k.DynamicClient.Resource(resource).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (k *Klient) GetDynamicResourceList(ctx context.Context, namespace string, resource schema.GroupVersionResource) ([]unstructured.Unstructured, error) {
	resultList, err := k.DynamicClient.Resource(resource).Namespace(namespace).List(ctx, metav1.ListOptions{})

	if err != nil {
		return nil, err
	}

	return resultList.Items, nil
}

func (k *Klient) GetNamespaces(ctx context.Context, labels map[string]string) ([]corev1.Namespace, error) {
	results, err := k.KubernetesClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var list []corev1.Namespace
	for _, item := range results.Items {
		if helper.FilterLabels(item.Labels, labels) {
			list = append(list, item)
		}
	}

	return list, nil
}

func (k *Klient) GetStorageClass(ctx context.Context, name string) (*storagev1.StorageClass, error) {
	storageClass, err := k.KubernetesClient.StorageV1().StorageClasses().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return storageClass, nil
}

func (k *Klient) GetStorageClasses(ctx context.Context) ([]storagev1.StorageClass, error) {
	storageClasses, err := k.KubernetesClient.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return storageClasses.Items, nil
}

func (k *Klient) GetNamespace(ctx context.Context, name string) (*corev1.Namespace, error) {
	return k.KubernetesClient.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
}

func (k *Klient) GetDeploymentByName(ctx context.Context, namespace, name string) (*appsv1.Deployment, error) {
	deployment, err := k.KubernetesClient.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})

	if err != nil {
		return nil, err
	}

	return deployment, nil

}

// strategy
// containers
// status
//   availableReplicas
//   replicas
//   readyReplicas
//   updatedReplicas
//   conditions
//     Type (Available), Status, Reason, Message, lastUpdateTime, lastTransitionTime
func (k *Klient) GetDeployments(ctx context.Context, namespace string, labels map[string]string) ([]appsv1.Deployment, error) {
	results, err := k.KubernetesClient.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var list []appsv1.Deployment
	for _, item := range results.Items {
		if helper.FilterLabels(item.Labels, labels) {
			list = append(list, item)
		}
	}

	return list, nil
}

func (k *Klient) GetPodByName(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	result, err := k.KubernetesClient.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (k *Klient) GetPods(ctx context.Context, namespace string, labels map[string]string) ([]corev1.Pod, error) {
	results, err := k.KubernetesClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var list []corev1.Pod
	for _, item := range results.Items {
		if helper.FilterLabels(item.Labels, labels) {
			list = append(list, item)
		}
	}

	return list, nil
}

func (k *Klient) GetServiceByName(ctx context.Context, namespace, name string) (*corev1.Service, error) {
	return k.KubernetesClient.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (k *Klient) GetServices(ctx context.Context, namespace string, labels map[string]string) ([]corev1.Service, error) {
	results, err := k.KubernetesClient.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var list []corev1.Service
	for _, item := range results.Items {
		if helper.FilterLabels(item.Labels, labels) {
			list = append(list, item)
		}
	}

	return list, nil
}

func (k *Klient) GetSecretByName(ctx context.Context, namespace, name string) (*corev1.Secret, error) {
	return k.KubernetesClient.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (k *Klient) GetSecrets(ctx context.Context, namespace string, labels map[string]string) ([]corev1.Secret, error) {
	results, err := k.KubernetesClient.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var list []corev1.Secret
	for _, item := range results.Items {
		if helper.FilterLabels(item.Labels, labels) {
			list = append(list, item)
		}
	}

	return list, nil
}

func (k *Klient) GetConfigMapByName(ctx context.Context, namespace, name string) (*corev1.ConfigMap, error) {
	return k.KubernetesClient.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (k *Klient) GetConfigMaps(ctx context.Context, namespace string, labels map[string]string) ([]corev1.ConfigMap, error) {
	results, err := k.KubernetesClient.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var list []corev1.ConfigMap
	for _, item := range results.Items {
		if helper.FilterLabels(item.Labels, labels) {
			list = append(list, item)
		}
	}

	return list, nil
}

func (k *Klient) GetPersistentVolumeByName(ctx context.Context, name string) (*corev1.PersistentVolume, error) {
	return k.KubernetesClient.CoreV1().PersistentVolumes().Get(ctx, name, metav1.GetOptions{})
}

func (k *Klient) GetPersistentVolumes(ctx context.Context) ([]corev1.PersistentVolume, error) {
	pvs, err := k.KubernetesClient.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return pvs.Items, nil
}

func (k *Klient) GetPersistentVolumeClaimByName(ctx context.Context, namespace, name string) (*corev1.PersistentVolumeClaim, error) {
	return k.KubernetesClient.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (k *Klient) GetPersistentVolumeClaims(ctx context.Context, namespace string, labels map[string]string) ([]corev1.PersistentVolumeClaim, error) {
	results, err := k.KubernetesClient.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var list []corev1.PersistentVolumeClaim
	for _, item := range results.Items {
		if helper.FilterLabels(item.Labels, labels) {
			list = append(list, item)
		}
	}

	return list, nil
}

func (k *Klient) GetResource(object *unstructured.Unstructured) *entity.KubernetesResource {

	gvr, err := k.GetGVR(object.GroupVersionKind())

	if err != nil {
		return nil
	}

	//state := helper.GetResourceStatus(object)
	state := "Active"
	//log.Printf("Object state for %s: %s", object.GetKind(), state)

	resource := entity.KubernetesResource{
		Id:        string(object.GetUID()),
		Name:      object.GetName(),
		Namespace: object.GetNamespace(),
		Type:      object.GetKind(),
		GVR:       gvr,
		State:     state,
		Timestamp: time.Now(),
	}

	return &resource
}

func (k *Klient) GenerateKubernetesObjects(ctx context.Context, specArr []string) ([]*unstructured.Unstructured, error) {

	var objects []*unstructured.Unstructured
	for index, spec := range specArr {
		object, _, err := helper.DecodeFromYaml(spec)
		if err != nil {
			return nil, err
		}
		logger.Infof(ctx, "%d. Generated object %s", index+1, object.GetName())

		objects = append(objects, object)
	}

	return objects, nil
}
