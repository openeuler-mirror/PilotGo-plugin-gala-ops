package router

import (
	"github.com/gin-gonic/gin"
	"openeuler.org/PilotGo/gala-ops-plugin/httphandler"
)

func InitRouter(router *gin.Engine) {
	api := router.Group("/plugin/gala-ops/api")
	{
		// 脚本执行结果接口
		// api.PUT("/run_script_result", httphandler.RunScriptResult)

		// 安装/升级/卸载gala-gopher监控终端
		api.PUT("/install_gopher", httphandler.InstallGopher)
		api.PUT("/upgrade_gopher", httphandler.UpgradeGopher)
		api.DELETE("/uninstall_gopher", httphandler.UninstallGopher)

		api.PUT("/install_ops", httphandler.InstallOps)
		api.PUT("/upgrade_ops", httphandler.UpgradeOps)
	}

	metrics := router.Group("plugin/gala-ops/api/metrics")
	{
		metrics.GET("/labels_list", httphandler.LabelsList)
		metrics.GET("/targets_list", httphandler.TargetsList)
		metrics.GET("/cpu_usage_rate", httphandler.CPUusagerate) // url?job=gala-gopher host ip
	}
}
