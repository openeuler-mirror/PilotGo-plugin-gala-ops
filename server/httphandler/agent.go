package httphandler

import (
	"fmt"
	"net/http"
	"os"

	"gitee.com/openeuler/PilotGo-plugins/sdk/common"
	"gitee.com/openeuler/PilotGo-plugins/sdk/logger"
	"github.com/gin-gonic/gin"
	"openeuler.org/PilotGo/gala-ops-plugin/agentmanager"
	"openeuler.org/PilotGo/gala-ops-plugin/config"
	"openeuler.org/PilotGo/gala-ops-plugin/database"
)

func InstallGopher(ctx *gin.Context) {
	// ttcode
	fmt.Println("\033[32mc.req.headers\033[0m: ", ctx.Request.Header)
	fmt.Println("\033[32mc.req.body\033[0m: ", ctx.Request.Body)

	batches := &common.Batch{}
	if err := ctx.BindJSON(batches); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":   -1,
			"status": "parameter error",
		})
		logger.Error("ctx.bindjson(batches) error: %s", err.Error())
		return
	}

	// ttcode
	fmt.Println("\033[32minstallgopher batch\033[0m: ", batches)

	workdir, err := os.Getwd()
	if err != nil {
		logger.Error("Err getting current work directory: %s", err.Error())
	}

	script, err := os.ReadFile(workdir + "/script/deploy.sh")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":   -1,
			"status": fmt.Sprintf("Err reading deploy script:%s", err),
		})
		logger.Error("Err reading deploy script: %s", err.Error())
		return
	}

	params := []string{"gopher", "-K", agentmanager.Galaops.MiddlewareDeploy.Kafka, "p", agentmanager.Galaops.MiddlewareDeploy.Pyroscope}
	cmdResults, err := agentmanager.Galaops.Sdkmethod.RunScript(batches, string(script), params)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":   -1,
			"status": fmt.Sprintf("run remote script error:%s", err),
		})
		logger.Error("run remote script error: %s", err.Error())
		return
	}

	ret := []interface{}{}
	for _, result := range cmdResults {
		d := struct {
			MachineUUID   string
			MachineIP     string
			InstallStatus string
			Error         string
		}{
			MachineUUID:   result.MachineUUID,
			InstallStatus: "ok",
			Error:         "",
		}

		if result.RetCode != 0 {
			d.InstallStatus = "error"
			d.Error = result.Stderr
		}

		ret = append(ret, d)
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":   0,
		"status": "ok",
		"data":   ret,
	})
}

func UpgradeGopher(ctx *gin.Context) {
	// ttcode
	fmt.Println("\033[32mc.req.headers\033[0m: ", ctx.Request.Header)
	fmt.Println("\033[32mc.req.body\033[0m: ", ctx.Request.Body)

	batches := &common.Batch{}
	if err := ctx.BindJSON(batches); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":   -1,
			"status": "parameter error",
		})
		logger.Error("ctx.bindjson(param) error: %s", err.Error())
		return
	}

	// ttcode
	fmt.Println("\033[32mupgradegopher batch\033[0m: ", batches)

	cmd := "systemctl stop gala-gopher && yum upgrade -y gala-gopher && systemctl start gala-gopher"
	cmdResults, err := agentmanager.Galaops.Sdkmethod.RunCommand(batches, cmd)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":   -1,
			"status": fmt.Sprintf("run remote script error:%s", err),
		})
		logger.Error("run remote command error: %s", err.Error())
		return
	}

	ret := []interface{}{}
	for _, result := range cmdResults {
		d := struct {
			MachineUUID   string
			UpgradeStatus string
			Error         string
		}{
			MachineUUID:   result.MachineUUID,
			UpgradeStatus: "ok",
			Error:         "",
		}

		if result.RetCode != 0 {
			d.UpgradeStatus = "error"
			d.Error = result.Stderr
		}

		ret = append(ret, d)
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":   0,
		"status": "ok",
		"data":   ret,
	})
}

func UninstallGopher(ctx *gin.Context) {
	// ttcode
	fmt.Println("\033[32mc.req.headers\033[0m: ", ctx.Request.Header)
	fmt.Println("\033[32mc.req.body\033[0m: ", ctx.Request.Body)

	param := &common.Batch{}
	if err := ctx.BindJSON(param); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":   -1,
			"status": "parameter error",
		})
		logger.Error("ctx.bindjson(param) error: %s", err.Error())
		return
	}

	// ttcode
	fmt.Println("\033[32mparam\033[0m: ", param)

	cmd := "systemctl stop gala-gopher && yum autoremove -y gala-gopher"
	cmdResults, err := agentmanager.Galaops.Sdkmethod.RunCommand(param, cmd)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":   -1,
			"status": fmt.Sprintf("run remote script error:%s", err),
		})
		logger.Error("run remote command error: %s", err.Error())
		return
	}

	ret := []interface{}{}
	for _, result := range cmdResults {
		d := struct {
			MachineUUID     string
			UninstallStatus string
			Error           string
		}{
			MachineUUID:     result.MachineUUID,
			UninstallStatus: "ok",
			Error:           "",
		}

		if result.RetCode != 0 {
			d.UninstallStatus = "error"
			d.Error = result.Stderr
		}

		ret = append(ret, d)
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":   0,
		"status": "ok",
		"data":   ret,
	})
}

func InstallOps(ctx *gin.Context) {
	// ttcode
	fmt.Println("\033[32mc.req.headers\033[0m: ", ctx.Request.Header)
	fmt.Println("\033[32mc.req.body\033[0m: ", ctx.Request.Body)

	batches := &common.Batch{}
	var deploy_machine_uuid string
	var deploy_machine_ip string

	switch deploy_machine_uuid = ctx.Query("uuid"); deploy_machine_uuid {
	case "":
		deploy_machine_ip = config.Config().Deploy.ServerBasic
		agentmanager.Galaops.AgentMap.Range(func(key, value any) bool {
			agent := value.(*database.Agent)
			if agent.IP == deploy_machine_ip {
				deploy_machine_uuid = agent.UUID
			}
			return true
		})
	default:
		agentmanager.Galaops.AgentMap.Range(func(key, value any) bool {
			agent := value.(*database.Agent)
			if agent.UUID == deploy_machine_uuid {
				deploy_machine_ip = agent.IP
			}
			return true
		})
	}
	batches.MachineUUIDs = append(batches.MachineUUIDs, deploy_machine_uuid)
	agentmanager.Galaops.BasicDeploy.Spider = deploy_machine_ip
	agentmanager.Galaops.BasicDeploy.Anteater = deploy_machine_ip
	agentmanager.Galaops.BasicDeploy.Inference = deploy_machine_ip

	workdir, err := os.Getwd()
	if err != nil {
		logger.Error("Err getting current work directory: %s", err.Error())
	}

	script, err := os.ReadFile(workdir + "/script/deploy.sh")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":   -1,
			"status": fmt.Sprintf("Err reading deploy script:%s", err),
		})
		logger.Error("Err reading deploy script: %s", err.Error())
		return
	}

	params := []string{"ops", "-K", agentmanager.Galaops.MiddlewareDeploy.Kafka, "-P", agentmanager.Galaops.MiddlewareDeploy.Prometheus, "-A", agentmanager.Galaops.MiddlewareDeploy.Arangodb}
	cmdResults, err := agentmanager.Galaops.Sdkmethod.RunScript(batches, string(script), params)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":   -1,
			"status": fmt.Sprintf("run remote script error:%s", err),
		})
		logger.Error("run remote script error: %s", err.Error())
		return
	}

	ret := []interface{}{}
	for _, result := range cmdResults {
		d := struct {
			MachineUUID   string
			MachineIP     string
			InstallStatus string
			Error         string
		}{
			MachineUUID:   result.MachineUUID,
			InstallStatus: "ok",
			Error:         "",
		}

		if result.RetCode != 0 {
			d.InstallStatus = "error"
			d.Error = result.Stderr
		}

		ret = append(ret, d)
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":   0,
		"status": "ok",
		"data":   ret,
	})
}

func UpgradeOps(ctx *gin.Context) {
	// ttcode
	fmt.Println("\033[32mupgradeops req.headers\033[0m: ", ctx.Request.Header)
	fmt.Println("\033[32mupgradeops req.body\033[0m: ", ctx.Request.Body)

	batches := &common.Batch{}
	var deploy_machine_uuid string

	deploy_machine_ip := agentmanager.Galaops.BasicDeploy.Spider
	agentmanager.Galaops.AgentMap.Range(func(key, value any) bool {
		agent := value.(*database.Agent)
		if agent.IP == deploy_machine_ip {
			deploy_machine_uuid = agent.UUID
		}
		return true
	})
	batches.MachineUUIDs = append(batches.MachineUUIDs, deploy_machine_uuid)

	// ttcode
	fmt.Println("\033[32mupgradeops batch\033[0m: ", batches)

	cmd := "systemctl stop gala-spider && systemctl stop gala-anteater && systemctl stop gala-inference && yum upgrade -y gala-ops && systemctl start gala-spider && systemctl start gala-anteater && systemctl start gala-inference"
	cmdResults, err := agentmanager.Galaops.Sdkmethod.RunCommand(batches, cmd)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":   -1,
			"status": fmt.Sprintf("run remote script error:%s", err),
		})
		logger.Error("run remote command error: %s", err.Error())
		return
	}

	ret := []interface{}{}
	for _, result := range cmdResults {
		d := struct {
			MachineUUID   string
			UpgradeStatus string
			Error         string
		}{
			MachineUUID:   result.MachineUUID,
			UpgradeStatus: "ok",
			Error:         "",
		}

		if result.RetCode != 0 {
			d.UpgradeStatus = "error"
			d.Error = result.Stderr
		}

		ret = append(ret, d)
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":   0,
		"status": "ok",
		"data":   ret,
	})
}

func UninstallOps(ctx *gin.Context) {
	// ttcode
	fmt.Println("\033[32muninstallops req.headers\033[0m: ", ctx.Request.Header)
	fmt.Println("\033[32muninstallops req.body\033[0m: ", ctx.Request.Body)

	batches := &common.Batch{}
	var deploy_machine_uuid string

	deploy_machine_ip := agentmanager.Galaops.BasicDeploy.Spider
	agentmanager.Galaops.AgentMap.Range(func(key, value any) bool {
		agent := value.(*database.Agent)
		if agent.IP == deploy_machine_ip {
			deploy_machine_uuid = agent.UUID
		}
		return true
	})
	batches.MachineUUIDs = append(batches.MachineUUIDs, deploy_machine_uuid)

	// ttcode
	fmt.Println("\033[32muninstallops batch\033[0m: ", batches)

	cmd := "systemctl stop gala-spider && systemctl stop gala-anteater && systemctl stop gala-inference && yum autoremove -y gala-ops"
	cmdResults, err := agentmanager.Galaops.Sdkmethod.RunCommand(batches, cmd)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":   -1,
			"status": fmt.Sprintf("run remote script error:%s", err),
		})
		logger.Error("run remote command error: %s", err.Error())
		return
	}

	ret := []interface{}{}
	for _, result := range cmdResults {
		d := struct {
			MachineUUID   string
			UpgradeStatus string
			Error         string
		}{
			MachineUUID:   result.MachineUUID,
			UpgradeStatus: "ok",
			Error:         "",
		}

		if result.RetCode != 0 {
			d.UpgradeStatus = "error"
			d.Error = result.Stderr
		}

		ret = append(ret, d)
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":   0,
		"status": "ok",
		"data":   ret,
	})
}

func InstallNginx(ctx *gin.Context) {
	agentmanager.Galaops.SingleDeploy(ctx, "nginx", config.Config().Deploy.ServerMeta)
}

func InstallKafka(ctx *gin.Context) {
	agentmanager.Galaops.SingleDeploy(ctx, "kafka", config.Config().Deploy.ServerMeta)
}

func InstallArangodb(ctx *gin.Context) {
	agentmanager.Galaops.SingleDeploy(ctx, "arangodb", config.Config().Deploy.ServerMeta)
}

func InstallPyroscope(ctx *gin.Context) {
	agentmanager.Galaops.SingleDeploy(ctx, "pyroscope", config.Config().Deploy.ServerMeta)
}

func InstallESandLogstash(ctx *gin.Context) {
	agentmanager.Galaops.SingleDeploy(ctx, "elasticandlogstash", config.Config().Deploy.ServerMeta)
}