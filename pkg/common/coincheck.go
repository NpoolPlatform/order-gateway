package common

import (
	"context"

	coinmwcli "github.com/NpoolPlatform/chain-middleware/pkg/client/coin"
	wlog "github.com/NpoolPlatform/go-service-framework/pkg/wlog"
)

type CoinCheckHandler struct {
	CoinTypeID *string
}

func (h *CoinCheckHandler) CheckCoinWithCoinTypeID(ctx context.Context, coinTypeID string) error {
	exist, err := coinmwcli.ExistCoin(ctx, coinTypeID)
	if err != nil {
		return wlog.WrapError(err)
	}
	if !exist {
		return wlog.Errorf("invalid coin")
	}
	return nil
}

func (h *CoinCheckHandler) CheckCoin(ctx context.Context) error {
	return h.CheckCoinWithCoinTypeID(ctx, *h.CoinTypeID)
}
