package httphandler

import (
	"fmt"
	"net/http"

	"gitee.com/openeuler/PilotGo-plugins/sdk/logger"
	"github.com/gin-gonic/gin"
	"openeuler.org/PilotGo/gala-ops-plugin/agentmanager"
)

// prometheus plugin add /api/v1/labels handler
func LabelsList(ctx *gin.Context) {
	promurl := agentmanager.Galaops.PromePlugin["url"].(string)

	data, err := agentmanager.Galaops.QueryMetric(promurl, "labels", "")
	if err != nil {
		logger.Error("faild to querymetric from prometheus: ", err)
	}
	ctx.JSON(http.StatusOK, data)
}

func TargetsList(ctx *gin.Context) {
	// 查询prometheus监控对象列表
	promurl := agentmanager.Galaops.PromePlugin["url"].(string)

	param := map[string]string{
		"query": "up",
	}

	urlparam := fmt.Sprintf("?query=%v", param["query"])
	data, err := agentmanager.Galaops.QueryMetric(promurl, "query", urlparam)
	if err != nil {
		logger.Error("faild to querymetric from prometheus: ", err)
	}
	ctx.JSON(http.StatusOK, data)
}

func CPUusagerate(ctx *gin.Context) {
	promurl := agentmanager.Galaops.PromePlugin["url"].(string)
	start, end := agentmanager.Galaops.UnixTimeStartandEnd(-5)
	job := ctx.Query("job")
	if job == "" {
		logger.Error("need job parameter in url: cpuusagerate")
		ctx.JSON(http.StatusOK, "need job parameter in url: cpuusagerate")
	}

	param := map[string]string{
		"query": fmt.Sprintf("avg by(job) (gala_gopher_cpu_total_used_per{job=~\"%s\"})", job),
		"start": fmt.Sprint(start),
		"end":   fmt.Sprint(end),
		"step":  "15s",
	}

	urlparam := fmt.Sprintf("?query=%v&start=%v&end=%v&step=%v", param["query"], param["start"], param["end"], param["step"])
	data, err := agentmanager.Galaops.QueryMetric(promurl, "query_range", urlparam)
	if err != nil {
		logger.Error("faild to querymetric from prometheus: ", err)
	}
	ctx.JSON(http.StatusOK, data)
}
