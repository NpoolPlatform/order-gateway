package migrator

import (
	"context"
	"fmt"

	"github.com/NpoolPlatform/go-service-framework/pkg/config"
	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	redis2 "github.com/NpoolPlatform/go-service-framework/pkg/redis"
	constant1 "github.com/NpoolPlatform/order-gateway/pkg/message/const"
	"github.com/NpoolPlatform/order-middleware/pkg/db"
)

const (
	keyServiceID = "serviceid"
)

func lockKey() string {
	serviceID := config.GetStringValueWithNameSpace(constant1.ServiceName, keyServiceID)
	return fmt.Sprintf("migrator:%v", serviceID)
}

func Migrate(ctx context.Context) error {
	if err := redis2.TryLock(lockKey(), 0); err != nil {
		return err
	}
	defer func() {
		_ = redis2.Unlock(lockKey())
	}()

	logger.Sugar().Infow("Migrate", "Start", "...")

	if err := db.Init(); err != nil {
		logger.Sugar().Errorw("Migrate", "error", err)
		return err
	}

	logger.Sugar().Infow("Migrate", "Done", "success")

	return nil
}
