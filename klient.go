package klient

import (
	"context"
	"time"

	"github.com/zbitech/common/interfaces"
	"github.com/zbitech/common/pkg/logger"
	"github.com/zbitech/common/pkg/rctx"
	klient "github.com/zbitech/klient/k8s-client"
	zklient "github.com/zbitech/klient/zbi-klient"
)

type KlientFactory struct {
	zbiKlient interfaces.ZBIClientIF
	klient    interfaces.KlientIF
}

func NewKlientFactory() interfaces.KlientFactoryIF {
	return &KlientFactory{}
}

func (k *KlientFactory) Init(ctx context.Context) error {
	ctx = rctx.BuildContext(ctx, rctx.Context(rctx.Component, "KlientFactory"), rctx.Context(rctx.StartTime, time.Now()))
	defer logger.LogComponentTime(ctx)

	var err error

	klient, err := klient.NewKlient(ctx)
	if err != nil {
		return err
	}

	zbiKlient, err := zklient.NewZBIClient(klient)
	if err != nil {
		return err
	}

	k.klient = klient
	k.zbiKlient = zbiKlient

	return nil
}

func (k *KlientFactory) GetZBIClient() interfaces.ZBIClientIF {
	return k.zbiKlient
}

func (k *KlientFactory) GetKubernesClient() interfaces.KlientIF {
	return k.klient
}
