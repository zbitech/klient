package kinformer

import (
	"encoding/json"
	"github.com/zbitech/common/pkg/utils"
	"log"

	"github.com/zbitech/common/pkg/model/ztypes"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func isZBIObject(labels map[string]string) bool {
	platform, ok := labels["platform"]
	if ok && platform == "zbi" {
		return true
	}

	return false
}

func getValue(labels map[string]string, name string) string {
	value, ok := labels[name]
	if ok {
		return value
	}

	return ""
}

func getPlatform(labels map[string]string) string {
	return getValue(labels, "platform")
}

func getInstance(labels map[string]string) string {
	return getValue(labels, "instance")
}

func getProject(labels map[string]string) string {
	return getValue(labels, "project")
}

func getVersion(labels map[string]string) string {
	return getValue(labels, "version")
}

func getOwner(labels map[string]string) string {
	return getValue(labels, "owner")
}

func getNetwork(labels map[string]string) string {
	return getValue(labels, "network")
}

func getObjectId(labels map[string]string) string {
	id, ok := labels["id"]
	if ok {
		return id
	}

	return ""
}

func getObjectProjectId(labels map[string]string) string {
	pid, ok := labels["pid"]
	if ok {
		return pid
	}

	return ""
}

func getObjectOwner(labels map[string]string) string {
	owner, ok := labels["owner"]
	if ok {
		return owner
	}

	return ""
}

func getObjectNetwork(labels map[string]string) ztypes.NetworkType {
	network, ok := labels["network"]
	if ok {
		return ztypes.NetworkType(network)
	}

	return ztypes.TESTNET_TYPE
}

func getObjectLevel(labels map[string]string) ztypes.ResourceLevelType {
	resource, ok := labels["resource"]
	if ok {
		return ztypes.ResourceLevelType(resource)
	}

	return ztypes.ResourceLevelType("project")
}

func marshallCondition(condition interface{}) string {
	if condition != nil {
		c, err := json.Marshal(condition)
		if err != nil {
			log.Printf("Unable to marshall condition - %s", err)
			return err.Error()
		}

		return string(c)
	}
	return ""
}

func getNamespaceConditions(obj *corev1.Namespace) interface{} {

	conditions := []interface{}{}
	for _, condition := range obj.Status.Conditions {
		conditions = append(conditions, condition)
	}
	return conditions
}

func getPersistentVolumeClaimConditions(obj *corev1.PersistentVolumeClaim) interface{} {

	conditions := []interface{}{}
	for _, condition := range obj.Status.Conditions {
		conditions = append(conditions, condition)
	}

	return conditions
}

func getPodConditions(obj *corev1.Pod) interface{} {

	conditions := []interface{}{}
	//TODO - add phase to the message

	for _, condition := range obj.Status.Conditions {
		conditions = append(conditions, condition)
	}

	for _, condition := range obj.Status.ContainerStatuses {
		conditions = append(conditions, condition)
	}

	return conditions
}

func getDeploymentConditions(obj *appsv1.Deployment) interface{} {

	conditions := []interface{}{}
	for _, condition := range obj.Status.Conditions {
		conditions = append(conditions, condition)
	}
	return conditions
}

func getServiceConditions(obj *corev1.Service) []string {

	conditions := []string{}
	for _, condition := range obj.Status.Conditions {
		conditions = append(conditions, marshallCondition(condition))
	}

	return conditions
}

func isObjectModified(old, new interface{}) bool {
	old_m := utils.MarshalObject(old)
	new_m := utils.MarshalObject(new)

	modified := old_m != new_m

	//	log.Printf("Comparing %s and %s = %s", old_m, new_m, modified)
	return modified
}
