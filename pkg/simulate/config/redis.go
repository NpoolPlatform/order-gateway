package config

import (
	"context"
	"fmt"

	redis2 "github.com/NpoolPlatform/go-service-framework/pkg/redis"
	configmwpb "github.com/NpoolPlatform/message/npool/order/mw/v1/simulate/config"
	configmwcli "github.com/NpoolPlatform/order-middleware/pkg/client/simulate/config"

	cruder "github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	basetypes "github.com/NpoolPlatform/message/npool/basetypes/v1"
)

func (h *Handler) SetSimulateConfigRedis(ctx context.Context) error {
	config, err := configmwcli.GetSimulateConfigOnly(ctx, &configmwpb.Conds{
		AppID:   &basetypes.StringVal{Op: cruder.EQ, Value: *h.AppID},
		Enabled: &basetypes.BoolVal{Op: cruder.EQ, Value: *h.Enabled},
	})
	if err != nil {
		return err
	}
	if config == nil {
		return fmt.Errorf("invalid config")
	}
	cli, err := redis2.GetClient()
	if err != nil {
		return err
	}
	SimulateConfigKey := fmt.Sprintf("%v:%v", basetypes.Prefix_PrefixSimulateConfig, *h.AppID)
	SimulateConfigValue := fmt.Sprintf("%v:%v:%v", config.SendCouponMode, config.SendCouponProbability, config.Units)
	err = cli.Set(ctx, SimulateConfigKey, SimulateConfigValue, 0).Err()
	if err != nil {
		return err
	}

	return nil
}
