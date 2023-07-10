package agentmanager

import (
	"fmt"
	"strings"

	"gitee.com/openeuler/PilotGo-plugins/sdk/common"
	"gitee.com/openeuler/PilotGo-plugins/sdk/logger"
	"openeuler.org/PilotGo/gala-ops-plugin/database"
	"openeuler.org/PilotGo/gala-ops-plugin/utils"
)

// 获取集群gala-ops组件部署信息
func GetPkgDeployInfo(machines []*database.Agent, batch *common.Batch, pkgname string) ([]*database.Agent, error) {
	cmdresults, err := Galaops.Sdkmethod.RunCommand(batch, "rpm -qi "+pkgname)
	if err == nil {
		for _, result := range cmdresults {
			if result.RetCode == 1 && strings.Contains(result.Stdout, "is not installed") && result.Stderr == "" {
				logger.Error("%s not installed in the process of running getpkgdeployinfo: %s, %s, %s; ", pkgname, result.MachineUUID, result.MachineIP, result.Stderr)
				continue
			} else if result.RetCode == 127 && result.Stdout == "" && strings.Contains(result.Stderr, "command not found") {
				logger.Error("rpm not installed when running getpkgdeployinfo: %s, %s, %s", result.MachineUUID, result.MachineIP, result.Stderr)
				continue
			} else if result.RetCode == 0 && len(result.Stdout) > 0 && result.Stderr == "" {
				reader := strings.NewReader(result.Stdout)
				v, err := utils.ReadInfo(reader, `^Version.*`)
				if err != nil && len(v) != 0 {
					logger.Error("failed to read RPM package version when running getpkgdeployinfo: %s, %s, %s", result.MachineUUID, result.MachineIP, result.Stderr)
					continue
				}

				d, err := utils.ReadInfo(reader, `^Install Date.*`)
				if err != nil && len(d) != 0 {
					logger.Error("failed to read RPM package install date when running getpkgdeployinfo: %s, %s, %s", result.MachineUUID, result.MachineIP, result.Stderr)
				}

				for _, m := range machines {
					if m.UUID == result.MachineUUID {
						switch pkgname {
						case "gala-gopher":
							m.Gopher_version = v
							m.Gopher_deploy = true
							m.Gopher_installtime = d
						case "gala-anteater":
							m.Anteater_version = v
							m.Anteater_deploy = true
							m.Anteater_installtime = d
							Galaops.BasicDeploy.Anteater = m.IP
						case "gala-inference":
							m.Inference_version = v
							m.Inference_deploy = true
							m.Inference_installtime = d
							Galaops.BasicDeploy.Inference = m.IP
						case "gala-spider":
							m.Spider_version = v
							m.Spider_deploy = true
							m.Spider_installtime = d
							Galaops.BasicDeploy.Spider = m.IP
						}
					}
				}
			} else {
				logger.Error("failed to run command: rpm -qi %s in %s, %s, %s when running getpkgdeployinfo", pkgname, result.MachineUUID, result.MachineIP, result.Stderr)
				continue
			}
		}
		return machines, nil
	}
	return nil, fmt.Errorf("runcommand error: %s", err)
}

// 获取集群gala-ops组件运行状态
func GetPkgRunningInfo(machines []*database.Agent, batch *common.Batch, pkgname string) ([]*database.Agent, error) {
	if pkgname == "gala-gopher" {
		for _, m := range machines {
			if !m.Gopher_deploy {
				for i, bm := range batch.MachineUUIDs {
					if m.UUID == bm {
						copy(batch.MachineUUIDs[i:], batch.MachineUUIDs[i+1:])
					}
				}
			}
		}
	}

	cmdresults, err := Galaops.Sdkmethod.RunCommand(batch, "systemctl status "+pkgname)
	if err != nil {
		return nil, fmt.Errorf("runcommand error: %s", err)
	}

	for _, result := range cmdresults {
		// ttcode
		logger.Debug("\033[32mrunning:\033[0m IP: %s, UUID: %s, code: %d, stdo: %s, stde: %s", result.MachineIP, result.MachineUUID, result.RetCode, result.Stdout, result.Stderr)
		if result.RetCode == 3 && strings.Contains(result.Stdout, "Active: inactive (dead)") && result.Stderr == "" {
			for _, m := range machines {
				if m.UUID == result.MachineUUID {
					m.Gopher_running = false
				}
			}
		} else if result.RetCode == 0 && strings.Contains(result.Stdout, "Active: active (running)") && result.Stderr == "" {
			for _, m := range machines {
				if m.UUID == result.MachineUUID {
					m.Gopher_running = true
				}
			}
		} else {
			logger.Error("Err getting running status in getpkgrunninginfo: %s, %s", pkgname, result)
		}
	}

	return machines, nil
}
