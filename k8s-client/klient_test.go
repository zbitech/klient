package client

import (
	"context"
	"fmt"
	"github.com/zbitech/common/pkg/logger"
	"github.com/zbitech/common/pkg/model/object"
	"github.com/zbitech/common/pkg/utils"
	"github.com/zbitech/common/pkg/vars"
	"github.com/zbitech/klient/internal/helper"
	"testing"
)

var (
	nginxDefaultDeployment = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
  namespace: default
  labels:
    app: nginx
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
      - name: webserver
        image: nginx
        imagePullPolicy: Always
`

	initialized = InitConfig()
)

func InitConfig() bool {
	ctx := context.Background()

	path := fmt.Sprintf("%s/config.yaml", vars.ASSET_PATH_DIRECTORY)
	if err := utils.ReadConfig(path, nil, &vars.AppConfig); err != nil {
		logger.Fatalf(ctx, "%s", err)
	}

	vars.AppConfig.Kubernetes.KubeConfig = utils.GetEnv("KUBE_CONFIG", "cfg/kubeconfig")
	logger.Infof(ctx, "AppConfig - %s", utils.MarshalObject(vars.AppConfig))
	return true
}

func Test_NewRestConfig(t *testing.T) {
	ctx := context.Background()
	r, err := NewRestConfig(ctx)
	if err != nil {
		t.Fatalf("Failed to create rest config - %s", err)
	}

	if r == nil {
		t.Fatalf("Failed to create rest config - %s", err)
	}
}

func Test_NewKlient(t *testing.T) {
	ctx := context.Background()
	k, err := NewKlient(ctx)
	if err != nil {
		t.Fatalf("Failed to create kubernetes client - %s", err)
	}

	if k == nil {
		t.Fatalf("Failed to create kubernetes client")
	}
}

func Test_GetMapper(t *testing.T) {
	ctx := context.Background()
	k, err := NewKlient(ctx)
	if err != nil {
		t.Fatalf("Failed to create kubernetes client - %s", err)
	}

	if k == nil {
		t.Fatalf("Failed to create kubernetes client")
	}

	mapper := k.GetMapper()
	if mapper == nil {
		t.Fatalf("Failed to create mapper")
	}
}

func Test_GetGVR(t *testing.T) {
	ctx := context.Background()
	k, err := NewKlient(ctx)
	if err != nil {
		t.Fatalf("Failed to create kubernetes client - %s", err)
	}

	if k == nil {
		t.Fatalf("Failed to create kubernetes client - %s", err)
	}

	_, gvk, err := helper.DecodeFromYaml(nginxDefaultDeployment)
	if err != nil {
		t.Fatalf("Failed to generate object and GVK from yaml - %s", err)
	}

	gvr, err := k.GetGVR(*gvk)
	if err != nil {
		t.Fatalf("Failed to get GVR for %s - %s", gvk.String(), err)
	}

	if gvr.Group != "apps" || gvr.Version != "v1" || gvr.Resource != "deployments" {
		t.Fatalf("Expected apps/v1 deployments but got %s/%s %s", gvr.Group, gvr.Version, gvr.Resource)
	}
}

func Test_ApplyResource(t *testing.T) {
	ctx := context.Background()
	k, err := NewKlient(ctx)
	if err != nil {
		t.Fatalf("Failed to create kubernetes client - %s", err)
	}

	if k == nil {
		t.Fatalf("Failed to create kubernetes client - %s", err)
	}

	filePath := fmt.Sprintf("%s/templates/resources.tmpl", vars.ASSET_PATH_DIRECTORY)
	fileTemplate, err := object.NewFileTemplate(filePath)
	if err != nil {
		t.Fatalf("Failed to generate template - %s", err)
	}

	var data interface{}
	deploymentSpec, err := fileTemplate.ExecuteTemplate("DEPLOYMENT", data)
	if err != nil {
		t.Fatalf("Failed to generate template - %s", err)
	}

	obj, gvk, err := helper.DecodeFromYaml(deploymentSpec)
	if err != nil {
		t.Fatalf("Failed to generate object and GVK from yaml - %s", err)
	}

	resource, err := k.ApplyResource(ctx, obj)
	if err != nil {
		t.Fatalf("Failed to create kubernetes resource - %s", resource)
	}
	t.Logf("Created resource - %s", utils.MarshalObject(resource))

	gvr, err := k.GetGVR(*gvk)
	if err != nil {
		t.Fatalf("Failed to get GVR for %s - %s", gvk.String(), err)
	}

	err = k.DeleteDynamicResource(ctx, obj.GetNamespace(), obj.GetName(), *gvr)
	if err != nil {
		t.Fatalf("Failed to delete resource after creation - %s", err)
	}
}

//func Test_ApplyResources(t *testing.T) {
//
//}
//
//func Test_DeleteDynamicResource(t *testing.T) {
//
//}
//
//func Test_DeleteNamespace(t *testing.T) {
//
//}
//
//func Test_GetDynamicResource(t *testing.T) {
//
//}
//
//func Test_GetDynamicResourceList(t *testing.T) {
//
//}
//
//func Test_GetNamespaces(t *testing.T) {
//
//}
//
//func Test_GetStorageClass(t *testing.T) {
//
//}
//
//func Test_GetStorageClasses(t *testing.T) {
//
//}
//
//func Test_GetNamespace(t *testing.T) {
//
//}
//
//func Test_GetDeploymentByName(t *testing.T) {
//
//}
//
//func Test_GetDeployments(t *testing.T) {
//
//}
//
//func Test_GetPodByName(t *testing.T) {
//
//}
//
//func Test_GetPods(t *testing.T) {
//
//}
//
//func Test_GetServiceByName(t *testing.T) {
//
//}
//
//func Test_GetServices(t *testing.T) {
//
//}
//
//func Test_GetSecretByName(t *testing.T) {
//
//}
//
//func Test_GetSecrets(t *testing.T) {
//
//}
//
//func Test_GetConfigMapByName(t *testing.T) {
//
//}
//
//func Test_GetConfigMaps(t *testing.T) {
//
//}
//
//func Test_GetPersistentVolumeByName(t *testing.T) {
//
//}
//
//func Test_GetPersistentVolumes(t *testing.T) {
//
//}
//
//func Test_GetPersistentVolumeClaimByName(t *testing.T) {
//
//}
//
//func Test_GetPersistentVolumeClaims(t *testing.T) {
//
//}
//
//func Test_GetResource(t *testing.T) {
//
//}
//
//func Test_GenerateKubernetesObjects(t *testing.T) {
//
//}
