package main

import (
	"fmt"
	"os"

	"gitee.com/openeuler/PilotGo-plugins/sdk/logger"
	"gitee.com/openeuler/PilotGo-plugins/sdk/plugin/client"
	"github.com/gin-gonic/gin"
	"openeuler.org/PilotGo/gala-ops-plugin/agentmanager"
	"openeuler.org/PilotGo/gala-ops-plugin/config"
	"openeuler.org/PilotGo/gala-ops-plugin/database"
	"openeuler.org/PilotGo/gala-ops-plugin/router"
)

const Version = "0.0.1"

var PluginInfo = &client.PluginInfo{
	Name:        "gala-ops",
	Version:     Version,
	Description: "gala-ops智能运维工具",
	Author:      "guozhengxin",
	Email:       "guozhengxin@kylinos.cn",
	Url:         "http://192.168.75.100:9999/plugin/gala-ops",
	// ReverseDest: "http://192.168.48.163:3000/",
}

func main() {
	fmt.Println("hello gala-ops")

	if err := database.MysqlInit(config.Config().Mysql); err != nil {
		fmt.Println("failed to initialize database")
		os.Exit(1)
	}

	InitLogger()

	PluginClient := client.DefaultClient(PluginInfo)
	// 临时给server赋值
	PluginClient.Server = "http://192.168.75.100:8887"
	agentmanager.Galaops = &agentmanager.Opsclient{
		Sdkmethod:   PluginClient,
		PromePlugin: nil,
		MiddlewareDeploy: &agentmanager.Middleware{
			Kafka:      config.Config().Deploy.ServerMeta,
			Prometheus: config.Config().Deploy.ServerMeta,
			Pyroscope:  config.Config().Deploy.ServerMeta,
			Arangodb:   config.Config().Deploy.ServerMeta,
			Logstash:   config.Config().Deploy.ServerMeta,
			Elasticsearch: config.Config().Deploy.ServerMeta,
		},
	}

	// 业务机集群aops组件状态自检
	err := agentmanager.Galaops.DeployStatusCheck()
	if err != nil {
		logger.Error(err.Error())
	}

	// 设置router
	engine := gin.Default()
	agentmanager.Galaops.Sdkmethod.RegisterHandlers(engine)
	router.InitRouter(engine)
	if err := engine.Run(config.Config().Http.Addr); err != nil {
		logger.Fatal("failed to run server")
	}
}

func InitLogger() {
	if err := logger.Init(config.Config().Logopts); err != nil {
		fmt.Printf("logger init failed, please check the config file: %s", err)
		os.Exit(1)
	}
}
