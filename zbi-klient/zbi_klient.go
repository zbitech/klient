package zklient

import (
	"github.com/zbitech/common/pkg/model/ztypes"
	client "github.com/zbitech/klient/k8s-client"
	kinformer "github.com/zbitech/klient/k8s-informer"

	"github.com/zbitech/common/interfaces"
)

type ZBIClient struct {
	client   interfaces.KlientIF
	informer *kinformer.KlientInformerController
}

func NewZBIClient(client *client.Klient) (*ZBIClient, error) {

	return &ZBIClient{
		client:   client,
		informer: kinformer.NewKlientInformerController(client.KubernetesClient, client.DynamicClient),
	}, nil
}

func (z *ZBIClient) RunInformer() {

	z.informer.AddInformer(ztypes.DEPLOYMENT_RESOURCE)
	z.informer.AddInformer(ztypes.POD_RESOURCE)
	z.informer.AddInformer(ztypes.PERSISTENT_VOLUME_RESOURCE)
	z.informer.AddInformer(ztypes.PERSISTENT_VOLUME_CLAIM_RESOURCE)
	z.informer.AddInformer(ztypes.VOLUME_SNAPHOT_RESOURCE)
	z.informer.AddInformer(ztypes.NAMESPACE_RESOURCE)
	//	z.informer.Run()
}
