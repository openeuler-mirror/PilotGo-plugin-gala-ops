package httphandler

import (
	"fmt"
	"net/http"
	"os"

	"gitee.com/openeuler/PilotGo-plugins/sdk/common"
	"gitee.com/openeuler/PilotGo-plugins/sdk/logger"
	"gitee.com/openeuler/PilotGo-plugins/sdk/plugin/client"
	"github.com/gin-gonic/gin"
	"openeuler.org/PilotGo/gala-ops-plugin/agentmanager"
)

func InstallGopher(ctx *gin.Context) {
	// ttcode
	fmt.Println("\033[32mc.request.headers\033[0m: ", ctx.Request.Header)
	fmt.Println("\033[32mc.request.body\033[0m: ", ctx.Request.Body)

	param := &common.Batch{}
	if err := ctx.BindJSON(param); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":   -1,
			"status": "parameter error",
		})
		logger.Error("ctx.bindjson(param) error: ", err)
		return
	}

	// ttcode
	fmt.Println("\033[32mparam\033[0m: ", param)

	workdir, err := os.Getwd()
	if err != nil {
		logger.Error("Err getting current work directory: ", err)
	}

	script, err := os.ReadFile(workdir + "/sh-script/deploy_gopher")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":   -1,
			"status": fmt.Sprintf("read deploy script error:%s", err),
		})
		logger.Error("read deploy script error:%s", err)
		return
	}

	cmdResults, err := agentmanager.Galaops.Sdkmethod.RunScript(param, string(script))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":   -1,
			"status": fmt.Sprintf("run remote script error:%s", err),
		})
		logger.Error("run remote script error:%s", err)
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
	// TODO
	param := &common.Batch{}
	if err := ctx.BindJSON(param); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":   -1,
			"status": "parameter error",
		})
	}

	cmd := "systemctl stop gala-gopher && yum upgrade -y gala-gopher && systemctl start gala-gopher"
	cmdResults, err := client.GetClient().RunCommand(param, cmd)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":   -1,
			"status": fmt.Sprintf("run remote script error:%s", err),
		})
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

func UninstallGopher(ctx *gin.Context) {
	// TODO
	param := &common.Batch{}
	if err := ctx.BindJSON(param); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":   -1,
			"status": "parameter error",
		})
	}

	cmd := "systemctl stop gala-gopher && yum autoremove -y gala-gopher"
	cmdResults, err := client.GetClient().RunCommand(param, cmd)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":   -1,
			"status": fmt.Sprintf("run remote script error:%s", err),
		})
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
