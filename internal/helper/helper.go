package helper

import (
	"log"

	"github.com/zbitech/common/pkg/model/entity"
	"github.com/zbitech/common/pkg/model/ztypes"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	// dynamicfake "k8s.io/client-go/dynamic/fake"
	// kubernetesfake "k8s.io/client-go/kubernetes/fake"
)

var (
	decUnstructured = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
)

func FilterLabels(objLabels map[string]string, filterLabels map[string]string) bool {
	if filterLabels == nil {
		return true
	}

	found := 0
	for key, value := range filterLabels {
		if objValue, ok := objLabels[key]; ok && objValue == value {
			found++
		}
	}

	return len(filterLabels) == found
}

func DecodeFromYaml(resourceYaml string) (*unstructured.Unstructured, *schema.GroupVersionKind, error) {
	object := &unstructured.Unstructured{}
	_, gvk, err := decUnstructured.Decode([]byte(resourceYaml), nil, object)

	return object, gvk, err
}

func GetResourceStatus(object *unstructured.Unstructured) ztypes.ResourceStateIF {

	objType := ztypes.ResourceObjectType(object.GetKind())

	if objType == ztypes.DEPLOYMENT_RESOURCE || objType == ztypes.POD_RESOURCE ||
		objType == ztypes.PERSISTENT_VOLUME_RESOURCE || objType == ztypes.PERSISTENT_VOLUME_CLAIM_RESOURCE {
		return &entity.ResourceState{Active: false, Deleted: false}
	}

	return &entity.ResourceState{Active: true, Deleted: false}
}

func GetPodState(status corev1.PodStatus) entity.PodState {

	ready := false
	containersReady := false

	var conditions []string
	for _, condition := range status.Conditions {
		if condition.Status == corev1.ConditionTrue {
			conditions = append(conditions, string(condition.Type))
			if condition.Type == corev1.PodReady {
				ready = true
			}

			if condition.Type == corev1.ContainersReady {
				containersReady = true
			}
		}
	}

	var containers []entity.ContainerState
	for _, status := range status.ContainerStatuses {
		state := entity.ContainerState{
			Name:       status.Name,
			Ready:      status.Ready,
			Terminated: status.State.Terminated != nil,
			Started:    *status.Started,
		}
		containers = append(containers, state)
	}

	for _, status := range status.InitContainerStatuses {
		state := entity.ContainerState{
			Name:       status.Name,
			Ready:      status.Ready,
			Terminated: status.State.Terminated != nil,
			Started:    *status.Started,
		}
		containers = append(containers, state)
	}

	return entity.PodState{
		Conditions:        conditions,
		ContainerStatuses: containers,
		Phase:             string(status.Phase),
		PodReady:          ready,
		ContainersReady:   containersReady,
		Deleted:           false,
	}
}

func GetDeploymentStatus(status appsv1.DeploymentStatus) entity.DeploymentState {

	available := false

	var conditions []string
	for _, condition := range status.Conditions {
		if condition.Status == corev1.ConditionStatus(corev1.ConditionTrue) {
			conditions = append(conditions, string(condition.Type))
			if condition.Type == appsv1.DeploymentAvailable {
				available = true
			}
		}
	}

	return entity.DeploymentState{
		Available:           available,
		Deleted:             false,
		Conditions:          conditions,
		Replicas:            status.Replicas,
		ReadyReplicas:       status.ReadyReplicas,
		AvailableReplicas:   status.AvailableReplicas,
		UpdatedReplicas:     status.UpdatedReplicas,
		UnavailableReplicas: status.UnavailableReplicas,
	}
}

func GetDynamicResourceInterface(dynamicClient dynamic.Interface, mapper *restmapper.DeferredDiscoveryRESTMapper, object *unstructured.Unstructured) (dynamic.ResourceInterface, error) {

	gvk := object.GroupVersionKind()
	Mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, err
	}

	gvr := Mapping.Resource
	log.Printf("Created Group: %s, Version: %s, Resource: %s", gvr.Group, gvr.Version, gvr.Resource)
	log.Printf("Namespace Scope: %s, RESTScope: %s", Mapping.Scope.Name(), meta.RESTScopeNameNamespace)

	var dr dynamic.ResourceInterface
	if Mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		log.Printf("Getting resource based on namespace %s", object.GetNamespace())
		dr = dynamicClient.Resource(Mapping.Resource).Namespace(object.GetNamespace())
	} else {
		// for cluster-wide resources
		log.Print("Getting DR for a cluster-wide resource")
		dr = dynamicClient.Resource(Mapping.Resource)
	}

	return dr, nil
}
