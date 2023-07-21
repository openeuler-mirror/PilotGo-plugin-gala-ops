package agentmanager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"gitee.com/openeuler/PilotGo-plugins/sdk/common"
	"gitee.com/openeuler/PilotGo-plugins/sdk/logger"
	"gitee.com/openeuler/PilotGo-plugins/sdk/plugin/client"
	"gitee.com/openeuler/PilotGo-plugins/sdk/utils/httputils"
	"github.com/gin-gonic/gin"
	"github.com/mitchellh/mapstructure"
	"openeuler.org/PilotGo/gala-ops-plugin/database"
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

type Middleware struct {
	Nginx         string
	Kafka         string
	Prometheus    string
	Pyroscope     string
	Arangodb      string
	Elasticsearch string
	Logstash      string
}

type BasicComponents struct {
	Spider    string
	Anteater  string
	Inference string
}

type Opsclient struct {
	Sdkmethod        *client.Client
	PromePlugin      map[string]interface{}
	AgentMap         sync.Map
	MiddlewareDeploy *Middleware
	BasicDeploy      *BasicComponents
}

var Galaops *Opsclient

/*******************************************************访问prometheus数据库*******************************************************/

func (o *Opsclient) UnixTimeStartandEnd(timerange time.Duration) (int64, int64) {
	now := time.Now()
	past5Minutes := now.Add(timerange * time.Minute)
	startOfPast5Minutes := time.Date(past5Minutes.Year(), past5Minutes.Month(), past5Minutes.Day(),
		past5Minutes.Hour(), past5Minutes.Minute(), 0, 0, past5Minutes.Location())
	timestamp := startOfPast5Minutes.Unix()
	return timestamp, now.Unix()
}

func (o *Opsclient) QueryMetric(endpoint string, querymethod string, param string) (interface{}, error) {
	ustr := endpoint + "/api/v1/" + querymethod + param
	u, err := url.Parse(ustr)
	if err != nil {
		return nil, err
	}
	u.RawQuery = u.Query().Encode()

	httpClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Get(u.String())
	if err != nil {
		return nil, err
	}
	bs, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var data interface{}

	err = json.Unmarshal(bs, &data)
	if err != nil {
		return nil, fmt.Errorf("unmarshal cpu usage rate error:%s", err.Error())
	}
	return data, nil
}

/*******************************************************prometheus插件相关*******************************************************/

func (o *Opsclient) Getplugininfo(pilotgoserver string, pluginname string) (map[string]interface{}, error) {
	resp, err := http.Get(pilotgoserver + "/api/v1/plugins")
	if err != nil {
		return nil, fmt.Errorf("faild to get plugin list: %s", err.Error())
	}
	defer resp.Body.Close()

	var buf bytes.Buffer
	_, erriocopy := io.Copy(&buf, resp.Body)
	if erriocopy != nil {
		return nil, erriocopy
	}

	data := map[string]interface{}{
		"code": nil,
		"data": nil,
		"msg":  nil,
	}
	err = json.Unmarshal(buf.Bytes(), &data)
	if err != nil {
		return nil, fmt.Errorf("unmarshal request plugin info error:%s", err.Error())
	}

	var PromePlugin map[string]interface{}
	for _, p := range data["data"].([]interface{}) {
		if p.(map[string]interface{})["name"] == pluginname {
			PromePlugin = p.(map[string]interface{})
		}
	}
	if len(PromePlugin) == 0 {
		return nil, fmt.Errorf("pilotgo server not add %s plugin", pluginname)
	}
	return PromePlugin, nil
}

func (o *Opsclient) SendJsonMode(jsonmodeURL string) (string, int, error) {
	url := Galaops.PromePlugin["url"].(string) + jsonmodeURL

	_, thisfile, _, _ := runtime.Caller(0)
	dirpath := filepath.Dir(thisfile)
	files := make(map[string]string)
	err := filepath.Walk(path.Join(dirpath, "..", "gui-json-mode"), func(jsonfilepath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		data, err := os.ReadFile(jsonfilepath)
		if err != nil {
			return err
		}
		_, jsonfilename := filepath.Split(jsonfilepath)
		files[strings.Split(jsonfilename, ".")[0]] = string(data)
		return nil
	})
	if err != nil {
		return "", -1, err
	}

	resp, err := httputils.Post(url, &httputils.Params{
		Body: files,
	})
	if resp != nil {
		if err != nil || resp.StatusCode != 201 {
			return "", resp.StatusCode, err
		}
		return string(resp.Body), resp.StatusCode, nil
	}
	return "the target web server does not exist", -1, err
}

func (o *Opsclient) CheckPrometheusPlugin() (bool, error) {
	url := Galaops.PromePlugin["url"].(string) + "aaa"
	resp, err := httputils.Get(url, nil)
	if resp == nil {
		return false, err
	}
	return true, err
}

/*******************************************************agentmanager*******************************************************/

func (o *Opsclient) AddAgent(a *database.Agent) {
	o.AgentMap.Store(a.UUID, a)
}

func (o *Opsclient) GetAgent(uuid string) *database.Agent {
	agent, ok := o.AgentMap.Load(uuid)
	if ok {
		return agent.(*database.Agent)
	}
	return nil
}

func (o *Opsclient) DeleteAgent(uuid string) {
	if _, ok := o.AgentMap.LoadAndDelete(uuid); !ok {
		logger.Warn("delete known agent:%s", uuid)
	}
}

/*******************************************************插件启动自检*******************************************************/

func (o *Opsclient) GetMachineList() ([]*database.Agent, error) {
	url := Galaops.Sdkmethod.Server + "/api/v1/pluginapi/machine_list"
	r, err := httputils.Get(url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get machine list: %s", err.Error())
	}

	results := &struct {
		Code int         `json:"code"`
		Data interface{} `json:"data"`
	}{}
	if err := json.Unmarshal(r.Body, &results); err != nil {
		return nil, fmt.Errorf("failed to unmarshal in getmachinelist(): %s", err.Error())
	}

	machinelist := []*database.Agent{}
	for _, m := range results.Data.([]interface{}) {
		p := &database.Agent{}
		mapstructure.Decode(m, p)
		machinelist = append(machinelist, p)
	}

	return machinelist, nil
}

func (o *Opsclient) DeployStatusCheck() error {
	// 临时自定义获取prometheus地址方式
	promeplugin, err := Galaops.Getplugininfo(Galaops.Sdkmethod.Server, "Prometheus")
	if err != nil {
		logger.Error(err.Error())
	}
	Galaops.PromePlugin = promeplugin

	// 获取业务机集群机器列表
	machines, err := Galaops.GetMachineList()
	if err != nil {
		return err
	}

	batch := &common.Batch{}
	for _, m := range machines {
		batch.MachineUUIDs = append(batch.MachineUUIDs, m.UUID)
	}

	logger.Debug("***plugin self-check***")

	// 检查prometheus插件是否在运行
	logger.Debug("***prometheus plugin running check***")
	promepluginstatus, _ := o.CheckPrometheusPlugin()
	if !promepluginstatus {
		logger.Error("***prometheus plugin is not running***")
	}

	// 向prometheus插件发送可视化插件json模板    TODO: prometheus plugin 实现接收jsonmode的接口
	logger.Debug("***send json mode to prometheus plugin***")
	respbody, retcode, err := Galaops.SendJsonMode("/abc")
	if err != nil || retcode != 201 {
		logger.Error("Err sending jsonmode to prometheus plugin: %s, %d, %s", respbody, retcode, err.Error())
	}

	// TODO: 自检部分添加各组件部署情况检测，更新opsclient中的middlewaredeploy和basicdeploy
	// 获取业务机集群gala-ops基础组件安装部署运行信息
	logger.Debug("***basic components deploy status check***")

	machines, err = GetPkgDeployInfo(machines, batch, "gala-gopher")
	if err != nil {
		logger.Error("gala-gopher deploy check failed: %s", err.Error())
	}
	machines, err = GetPkgRunningInfo(machines, batch, "gala-gopher")
	if err != nil {
		logger.Error("gala-gopher running status check failed: %s", err.Error())
	}
	machines, err = GetPkgDeployInfo(machines, batch, "gala-spider")
	if err != nil {
		logger.Error("gala-spider deploy check failed: %s", err.Error())
	}
	machines, err = GetPkgRunningInfo(machines, batch, "gala-spider")
	if err != nil {
		logger.Error("gala-spider running status check failed: %s", err.Error())
	}
	machines, err = GetPkgDeployInfo(machines, batch, "gala-anteater")
	if err != nil {
		logger.Error("gala-anteater deploy check failed: %s", err.Error())
	}
	machines, err = GetPkgRunningInfo(machines, batch, "gala-anteater")
	if err != nil {
		logger.Error("gala-anteater running status check failed: %s", err.Error())
	}
	machines, err = GetPkgDeployInfo(machines, batch, "gala-inference")
	if err != nil {
		logger.Error("gala-inference deploy check failed: %s", err.Error())
	}
	machines, err = GetPkgRunningInfo(machines, batch, "gala-inference")
	if err != nil {
		logger.Error("gala-inference running status check failed: %s", err.Error())
	}
	logger.Debug("***basic components deploy status check down***")

	logger.Debug("***plugin self-check down***")

	// 添加业务机集群信息至opsclient.AgentMap
	for _, m := range machines {
		o.AddAgent(m)
	}

	// ttcode
	Galaops.AgentMap.Range(
		func(key, value any) bool {
			logger.Debug("\033[32magent:\033[0m %v", value)
			return true
		},
	)

	// 更新DB中业务机集群的信息
	dberr := database.GlobalDB.Save(&machines).Error
	if dberr != nil {
		return fmt.Errorf("failed to update table: %s", dberr.Error())
	}

	return nil
}

/*******************************************************单机组件部署卸载handler*******************************************************/

func (o *Opsclient) SingleDeploy(c *gin.Context, pkgname string, defaultIP string) {
	// ttcode
	fmt.Println("\033[32mc.req.headers\033[0m: ", c.Request.Header)
	fmt.Println("\033[32mc.req.body\033[0m: ", c.Request.Body)

	batches := &common.Batch{}
	var deploy_machine_uuid string
	var deploy_machine_ip string
	var params []string

	switch deploy_machine_uuid = c.Query("uuid"); deploy_machine_uuid {
	case "":
		deploy_machine_ip = defaultIP
		Galaops.AgentMap.Range(func(key, value any) bool {
			agent := value.(*database.Agent)
			if agent.IP == deploy_machine_ip {
				deploy_machine_uuid = agent.UUID
			}
			return true
		})
	default:
		Galaops.AgentMap.Range(func(key, value any) bool {
			agent := value.(*database.Agent)
			if agent.UUID == deploy_machine_uuid {
				deploy_machine_ip = agent.IP
			}
			return true
		})

		switch pkgname {
		case "ops":
			Galaops.BasicDeploy.Spider = deploy_machine_ip
			Galaops.BasicDeploy.Anteater = deploy_machine_ip
			Galaops.BasicDeploy.Inference = deploy_machine_ip
		case "nginx":
			Galaops.MiddlewareDeploy.Nginx = deploy_machine_ip
		case "kafka":
			Galaops.MiddlewareDeploy.Kafka = deploy_machine_ip
		case "arangodb":
			Galaops.MiddlewareDeploy.Arangodb = deploy_machine_ip
		case "prometheus":
			Galaops.MiddlewareDeploy.Prometheus = deploy_machine_ip
		case "pyroscope":
			Galaops.MiddlewareDeploy.Pyroscope = deploy_machine_ip
		case "elasticandlogstash":
			Galaops.MiddlewareDeploy.Elasticsearch = deploy_machine_ip
		}
	}

	batches.MachineUUIDs = append(batches.MachineUUIDs, deploy_machine_uuid)

	workdir, err := os.Getwd()
	if err != nil {
		logger.Error("Err getting current work directory: %s", err.Error())
	}

	script, err := os.ReadFile(workdir + "/script/deploy.sh")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":   -1,
			"status": fmt.Sprintf("Err reading deploy script:%s", err),
		})
		logger.Error("Err reading deploy script: %s", err.Error())
		return
	}

	switch pkgname {
	case "ops":
		params = []string{"ops", "-K", Galaops.MiddlewareDeploy.Kafka, "-P", Galaops.MiddlewareDeploy.Prometheus, "-A", Galaops.MiddlewareDeploy.Arangodb}
	case "nginx":
		params = []string{"nginx", Galaops.MiddlewareDeploy.Nginx}
	case "kafka":
		params = []string{"middleware", "-K", Galaops.MiddlewareDeploy.Kafka, "-N", Galaops.MiddlewareDeploy.Nginx}
	case "arangodb":
		params = []string{"middleware", "-A", "-N", Galaops.MiddlewareDeploy.Nginx}
	case "prometheus":

	case "pyroscope":
		params = []string{"middleware", "-p", "-N", Galaops.MiddlewareDeploy.Nginx}
	case "elasticandlogstash":
		params = []string{"middleware", "-E", Galaops.MiddlewareDeploy.Elasticsearch, "-N", Galaops.MiddlewareDeploy.Nginx}
	}

	cmdResults, err := Galaops.Sdkmethod.RunScript(batches, string(script), params)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
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

	c.JSON(http.StatusOK, gin.H{
		"code":   0,
		"status": "ok",
		"data":   ret,
	})
}

func (o *Opsclient) SingleUninstall(c *gin.Context, pkgname string) {
	batches := &common.Batch{}
	var deploy_machine_uuid string
	var deploy_machine_ip string
	var cmd string

	switch pkgname {
	case "ops":
		deploy_machine_ip = Galaops.BasicDeploy.Spider
	case "nginx":
		deploy_machine_ip = Galaops.MiddlewareDeploy.Nginx
	case "kafka":
		deploy_machine_ip = Galaops.MiddlewareDeploy.Kafka
	case "arangodb":
		deploy_machine_ip = Galaops.MiddlewareDeploy.Arangodb
	case "prometheus":
		deploy_machine_ip = Galaops.MiddlewareDeploy.Prometheus
	case "pyroscope":
		deploy_machine_ip = Galaops.MiddlewareDeploy.Pyroscope
	case "elasticandlogstash":
		deploy_machine_ip = Galaops.MiddlewareDeploy.Elasticsearch
	}
	Galaops.AgentMap.Range(func(key, value any) bool {
		agent := value.(*database.Agent)
		if agent.IP == deploy_machine_ip {
			deploy_machine_uuid = agent.UUID
		}
		return true
	})

	batches.MachineUUIDs = append(batches.MachineUUIDs, deploy_machine_uuid)

	switch pkgname {
	case "ops":
		cmd = "systemctl stop gala-spider && systemctl stop gala-anteater && systemctl stop gala-inference && yum autoremove -y gala-ops"
	case "nginx":
		cmd = "yum autoremove -y nginx"
	case "kafka":
		cmd = "/bin/bash /opt/kafka_2.13-2.8.2/bin/kafka-server-stop.sh && /bin/bash /opt/kafka_2.13-2.8.2/bin/zookeeper-server-stop.sh && rm -rf /opt/kafka_2.13-2.8.2"
	case "arangodb":
		cmd = "systemctl stop arangodb3.service && yum autoremove -y arangodb3"
	case "prometheus":

	case "pyroscope":
		cmd = "kill -9 $(ps -ef | grep pyroscope | awk '{print $2}') 2> /dev/null ; yum autoremove -y gala-ops"
	case "elasticandlogstash":
		cmd = "kill -9 $(ps -ef | grep elasticsearch-8.5.3 | awk '{if($3==1) print $2}')  2>/dev/null ; rm -rf /home/elastic"
	}

	cmdResults, err := Galaops.Sdkmethod.RunCommand(batches, cmd)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
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

	c.JSON(http.StatusOK, gin.H{
		"code":   0,
		"status": "ok",
		"data":   ret,
	})
}
