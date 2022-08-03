package testinit

import (
	"fmt"
	"path"
	"runtime"

	"github.com/NpoolPlatform/go-service-framework/pkg/app"

	servicename "github.com/NpoolPlatform/order-gateway/pkg/servicename"

	mysqlconst "github.com/NpoolPlatform/go-service-framework/pkg/mysql/const"
	rabbitmqconst "github.com/NpoolPlatform/go-service-framework/pkg/rabbitmq/const"
	redisconst "github.com/NpoolPlatform/go-service-framework/pkg/redis/const"
)

func Init() error {
	_, myPath, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("cannot get source file path")
	}

	appName := path.Base(path.Dir(path.Dir(path.Dir(myPath))))
	configPath := fmt.Sprintf("%s/../../cmd/%v", path.Dir(myPath), appName)

	err := app.Init(
		servicename.ServiceName,
		"",
		"",
		"",
		configPath,
		nil,
		nil,
		mysqlconst.MysqlServiceName,
		rabbitmqconst.RabbitMQServiceName,
		redisconst.RedisServiceName,
	)
	if err != nil {
		return fmt.Errorf("cannot init app stub: %v", err)
	}

	return nil
}
