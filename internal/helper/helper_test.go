package helper

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	ASSETPATH  = "/Users/johnakinyele/go/src/github.com/zbi/data/etc/zbi"
	KUBECONFIG = "/Users/johnakinyele/.kube/config"
	NginxGVK   = schema.GroupVersionKind{Group: "apps", Kind: "Deployment", Version: "v1"}
	Group      = "apps"
	Kind       = "Deployment"
	Version    = "v1"
	Name       = "nginx"
	NginxYAML  = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
  namespace: default
spec:
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:latest
        ports:
        - containerPort: 80
`
)

func Test_FilterLabels(t *testing.T) {

	tests := []struct {
		Name   string
		Labels map[string]string
		Filter map[string]string
		Want   bool
	}{
		{Name: "Unfiltered", Labels: map[string]string{"type": "zcash", "kind": "deployment"}, Filter: nil, Want: true},
		{Name: "Unfiltered", Labels: map[string]string{"type": "zcash", "kind": "deployment"}, Filter: map[string]string{"type": "zcash"}, Want: true},
		{Name: "Unfiltered", Labels: map[string]string{"type": "lwd", "kind": "deployment"}, Filter: map[string]string{"type": "zcash"}, Want: false},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			exists := FilterLabels(test.Labels, test.Filter)
			if test.Want != exists {
				t.Errorf("got %v, want %v", exists, test.Want)
			}
		})
	}
}

func Test_DecodeFromYaml(t *testing.T) {

	obj, gvk, err := DecodeFromYaml(NginxYAML)
	if err != nil {
		t.Errorf("Got an error %s", err)
	}

	if gvk.Group != NginxGVK.Group || gvk.Kind != NginxGVK.Kind || gvk.Version != NginxGVK.Version {
		t.Logf("got %s.%s/%s, want %s.%s/%s", gvk.Group, gvk.Kind, gvk.Version, NginxGVK.Group, NginxGVK.Version, NginxGVK.Kind)
	}

	if obj == nil {
		t.Errorf("Got nil object")
	}

	if obj.GetName() != Name {
		t.Logf("got %s, want %s", obj.GetName(), Name)
	}

}

func Test_GetResource(t *testing.T) {

	// object, _, err := DecodeFromYaml(NginxYAML)
	// if err != nil {
	// 	t.Errorf("Got an error instead of kubernets object %s", err)
	// }

	// vars.KubernetesKlient, err = NewKlientFactory(context.Background())
	// if err != nil {
	// 	t.Errorf("Got an error instead of kubernetes clint - %s", err)
	// }

	// result, err := vars.KubernetesKlient.ApplyResource(context.Background(), object)
	// if err != nil {
	// 	t.Errorf("Got an error instead of kubernetes deployment result - %s", err)
	// }
	// //t.Logf("Result - %s", fn.MarshalIndentObject(result))

	// gvr, err := GroupVersionResource(vars.KubernetesKlient.GetMapper(), object)
	// if err != nil {
	// 	t.Errorf("Got an error instead of GVR - %s", err)
	// }

	// obj, err := vars.KubernetesKlient.GetDynamicResource(context.Background(), result.GetNamespace(), result.GetName(), *gvr)
	// if err != nil {
	// 	t.Errorf("Got an error instead of kubernetes object - %s", err)
	// }

	// time.Sleep(time.Duration(15) * time.Second)
	// status, _, err := unstructured.NestedStringMap(obj.Object, "status")
	// if err != nil {
	// 	t.Errorf("Got an error while Getting deployment status - %s", err)
	// }
	// t.Logf("Deployment Status - %s", status)

	// dep, err := vars.KubernetesKlient.GetDeploymentByName(context.Background(), "default", Name)
	// if err != nil {
	// 	t.Errorf("Got an error while Getting deployment - %s", err)
	// }
	// t.Logf("Deployment - %s", fn.MarshalIndentObject(dep))

	// err = vars.KubernetesKlient.DeleteDynamicResource(context.Background(), result.GetNamespace(), result.GetName(), *gvr)
	// if err != nil {
	// 	t.Errorf("Got an error while deleting resource - %s", err)
	// }
}

func Test_GroupVersionResource(t *testing.T) {

}

func Test_GetDynamicResourceInterface(t *testing.T) {

}

func Test_GenerateKubernetesObjects(t *testing.T) {

}
