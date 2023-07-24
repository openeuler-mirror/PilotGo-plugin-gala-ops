package router

import (
	"github.com/gin-gonic/gin"
	"openeuler.org/PilotGo/gala-ops-plugin/httphandler"
)

func InitRouter(router *gin.Engine) {
	api := router.Group("/plugin/gala-ops/api")
	{
		api.PUT("/install_nginx", httphandler.InstallNginx)
		api.DELETE("/uninstall_nginx", httphandler.UninstallNginx)

		api.PUT("/install_kafka", httphandler.InstallKafka)
		api.DELETE("/uninstall_kafka", httphandler.UninstallKafka)

		api.PUT("/install_arangodb", httphandler.InstallArangodb)
		api.DELETE("/uninstal_arangodb", httphandler.UninstallArangodb)

		api.PUT("/install_pyroscope", httphandler.InstallPyroscope)
		api.DELETE("/uninstall_pyroscope", httphandler.UninstallPyroscope)

		api.PUT("/install_esandlogstash", httphandler.InstallESandLogstash)

		// 安装/升级/卸载gala-gopher监控终端
		api.PUT("/install_gopher", httphandler.InstallGopher)
		api.PUT("/upgrade_gopher", httphandler.UpgradeGopher)
		api.DELETE("/uninstall_gopher", httphandler.UninstallGopher)

		api.PUT("/install_ops", httphandler.InstallOps)
		api.PUT("/upgrade_ops", httphandler.UpgradeOps)
		api.DELETE("/uninstall_ops", httphandler.UninstallOps)
	}

	metrics := router.Group("plugin/gala-ops/api/metrics")
	{
		metrics.GET("/labels_list", httphandler.LabelsList)
		metrics.GET("/targets_list", httphandler.TargetsList)
		metrics.GET("/cpu_usage_rate", httphandler.CPUusagerate) // url?job=gala-gopher host ip
	}
}
