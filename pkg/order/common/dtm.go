package common

import (
	"context"
	"time"

	logger "github.com/NpoolPlatform/go-service-framework/pkg/logger"
	wlog "github.com/NpoolPlatform/go-service-framework/pkg/wlog"

	dtmcli "github.com/NpoolPlatform/dtm-cluster/pkg/dtm"
)

type DtmHandler struct {
	*OrderOpHandler
}

func (h *DtmHandler) DtmDo(ctx context.Context, dispose *dtmcli.SagaDispose) error {
	start := time.Now()
	_ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	err := dtmcli.WithSaga(_ctx, dispose)
	dtmElapsed := time.Since(start)
	logger.Sugar().Infow(
		"CreateOrder(s)",
		"OrderID", *h.OrderID,
		"Start", start,
		"DtmElapsed", dtmElapsed,
		"Error", err,
	)
	return wlog.WrapError(err)
}
