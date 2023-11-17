package order

import (
	"context"
	"time"

	dtmcli "github.com/NpoolPlatform/dtm-cluster/pkg/dtm"
	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
)

type dtmHandler struct {
	*Handler
}

func (h *dtmHandler) dtmDo(ctx context.Context, dispose *dtmcli.SagaDispose) error {
	start := time.Now()
	_ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	err := dtmcli.WithSaga(_ctx, dispose)
	dtmElapsed := time.Since(start)
	logger.Sugar().Infow(
		"CreateOrder",
		"OrderID", *h.EntID,
		"Start", start,
		"DtmElapsed", dtmElapsed,
		"Error", err,
	)
	return err
}
