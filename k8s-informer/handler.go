package kinformer

import (
	"github.com/zbitech/common/pkg/model/entity"
	"github.com/zbitech/common/pkg/model/ztypes"
	"github.com/zbitech/common/pkg/vars"
	"github.com/zbitech/klient/internal/helper"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type ObjectStatus struct {
	Type      ztypes.ResourceObjectType
	GVR       *schema.GroupVersionResource
	Project   string
	Instance  string
	Version   string
	Id        string
	Name      string
	Namespace string
	Owner     string
	Network   string
	State     ztypes.ResourceStateIF
	Ignore    bool
}

func NewObjectStatus(oType ztypes.ResourceObjectType, gvk schema.GroupVersionKind,
	id, name, namespace string, labels map[string]string, state ztypes.ResourceStateIF) *ObjectStatus {

	instance := getInstance(labels)
	project := getProject(labels)
	version := getVersion(labels)
	owner := getOwner(labels)
	network := getNetwork(labels)
	gvr, _ := vars.KlientFactory.GetKubernesClient().GetGVR(gvk)

	return &ObjectStatus{
		Type:      ztypes.DEPLOYMENT_RESOURCE,
		GVR:       gvr,
		Project:   project,
		Instance:  instance,
		Version:   version,
		Id:        id,
		Name:      name,
		Namespace: namespace,
		Owner:     owner,
		Network:   network,
		State:     state,
	}
}

func DeploymentEvent(obj *appsv1.Deployment) *ObjectStatus {

	if !isZBIObject(obj.GetLabels()) {
		return &ObjectStatus{Ignore: true}
	}

	id := string(obj.GetUID())
	gvk := obj.GroupVersionKind()
	name := obj.GetName()
	namespace := obj.GetNamespace()
	labels := obj.GetLabels()
	state := helper.GetDeploymentStatus(obj.Status)

	return NewObjectStatus(ztypes.DEPLOYMENT_RESOURCE, gvk, id, name, namespace, labels, &state)
}

func PodEvent(obj *corev1.Pod) *ObjectStatus {

	if !isZBIObject(obj.GetLabels()) {
		return &ObjectStatus{Ignore: true}
	}

	id := string(obj.GetUID())
	gvk := obj.GroupVersionKind()
	name := obj.GetName()
	namespace := obj.GetNamespace()
	labels := obj.GetLabels()

	state := helper.GetPodState(obj.Status)

	return NewObjectStatus(ztypes.POD_RESOURCE, gvk, id, name, namespace, labels, &state)
}

func PersistentVolumeEvent(obj *corev1.PersistentVolume) *ObjectStatus {

	if !isZBIObject(obj.GetLabels()) {
		return &ObjectStatus{Ignore: true}
	}

	id := string(obj.GetUID())
	gvk := obj.GroupVersionKind()
	name := obj.GetName()
	namespace := obj.GetNamespace()
	labels := obj.GetLabels()
	state := entity.ResourceState{}
	return NewObjectStatus(ztypes.PERSISTENT_VOLUME_RESOURCE, gvk, id, name, namespace, labels, &state)
}

func PersistentVolumeClaimEvent(obj *corev1.PersistentVolumeClaim) *ObjectStatus {

	if !isZBIObject(obj.GetLabels()) {
		return &ObjectStatus{Ignore: true}
	}

	id := string(obj.GetUID())
	gvk := obj.GroupVersionKind()
	name := obj.GetName()
	namespace := obj.GetNamespace()
	labels := obj.GetLabels()
	state := entity.ResourceState{}
	return NewObjectStatus(ztypes.PERSISTENT_VOLUME_CLAIM_RESOURCE, gvk, id, name, namespace, labels, &state)
}
