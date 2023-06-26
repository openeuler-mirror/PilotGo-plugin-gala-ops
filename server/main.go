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

	engine := gin.Default()

	PluginClient := client.DefaultClient(PluginInfo)
	// 临时给server赋值
	PluginClient.Server = "http://192.168.75.100:8887"
	agentmanager.Galaops = &agentmanager.Opsclient{
		Sdkmethod:   PluginClient,
		PromePlugin: nil,
	}

	// 业务机集群aops组件状态自检
	err := agentmanager.Galaops.DeployStatusCheck()
	if err != nil {
		logger.Error(err.Error())
	}

	// 临时自定义获取prometheus地址方式
	promeplugin, err := agentmanager.Galaops.Getplugininfo(PluginClient.Server, "Prometheus")
	if err != nil {
		logger.Error(err.Error())
	}
	agentmanager.Galaops.PromePlugin = promeplugin

	// 检查prometheus插件是否在运行
	promepluginstatus, _ := agentmanager.Galaops.CheckPrometheusPlugin()
	if !promepluginstatus {
		logger.Error("prometheus plugin is not running")
	}

	// 向prometheus插件发送可视化插件json模板    TODO: prometheus plugin 实现接收jsonmode的接口
	respbody, retcode, err := agentmanager.Galaops.SendJsonMode("/abc")
	if err != nil || retcode != 201 {
		logger.Error("failed to send jsonmode to prometheus plugin: ", respbody, retcode, err)
	}

	// 设置router
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
