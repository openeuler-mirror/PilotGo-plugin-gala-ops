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
	"github.com/mitchellh/mapstructure"
	"openeuler.org/PilotGo/gala-ops-plugin/database"
)

type Middleware struct {
	Kafka         string
	Prometheus    string
	Pyroscope     string
	Arangodb      string
	Elasticsearch string
	Logstash      string
}

type Opsclient struct {
	Sdkmethod        *client.Client
	PromePlugin      map[string]interface{}
	AgentMap         sync.Map
	MiddlewareDeploy *Middleware
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
	promepluginstatus, _ := Galaops.CheckPrometheusPlugin()
	if !promepluginstatus {
		logger.Error("***prometheus plugin is not running***")
	}

	// 向prometheus插件发送可视化插件json模板    TODO: prometheus plugin 实现接收jsonmode的接口
	logger.Debug("***send json mode to prometheus plugin***")
	respbody, retcode, err := Galaops.SendJsonMode("/abc")
	if err != nil || retcode != 201 {
		logger.Error("Err sending jsonmode to prometheus plugin: %s, %d, %s", respbody, retcode, err.Error())
	}

	// 获取业务机集群gala-ops基础组件安装部署运行信息
	logger.Debug("***basic components deploy status check***")
	machines, err = GetPkgDeployInfo(machines, batch, "gala-gopher")
	if err != nil {
		logger.Error("gala-gopher version check failed: %s", err.Error())
	}
	machines, err = GetPkgRunningInfo(machines, batch, "gala-gopher")
	if err != nil {
		logger.Error("gala-gopher running status check failed: %s", err.Error())
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
			logger.Debug("\033[32magent:\033[0m ", value)
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
