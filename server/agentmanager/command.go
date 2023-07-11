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
		switch pkgname {
		case "gala-gopher":
			for _, result := range cmdresults {
				if result.RetCode == 1 && strings.Contains(result.Stdout, "is not installed") && result.Stderr == "" {
					logger.Error("%s not installed in the process of running getpkgdeployinfo: %s, %s, %s; ", pkgname, result.MachineUUID, result.MachineIP, result.Stderr)
					continue
				} else if result.RetCode == 127 && result.Stdout == "" && strings.Contains(result.Stderr, "command not found") {
					logger.Error("rpm not installed when running getpkgdeployinfo: %s, %s, %s", result.MachineUUID, result.MachineIP, result.Stderr)
					continue
				} else if result.RetCode == 0 && len(result.Stdout) > 0 && result.Stderr == "" {
					reader1 := strings.NewReader(result.Stdout)
					v, err := utils.ReadInfo(reader1, `^Version.*`)
					if err != nil && len(v) != 0 {
						logger.Error("failed to read RPM package version when running getpkgdeployinfo: %s, %s, %s", result.MachineUUID, result.MachineIP, result.Stderr)
						continue
					}
					reader2 := strings.NewReader(result.Stdout)
					d, err := utils.ReadInfo(reader2, `^Install Date.*`)
					if err != nil && len(d) != 0 {
						logger.Error("failed to read RPM package install date when running getpkgdeployinfo: %s, %s, %s", result.MachineUUID, result.MachineIP, result.Stderr)
					}

					for _, m := range machines {
						if m.UUID == result.MachineUUID {
							m.Gopher_version = v
							m.Gopher_deploy = true
							m.Gopher_installtime = d
						}
					}
				} else {
					logger.Error("failed to run command: rpm -qi %s in %s, %s, %s when running getpkgdeployinfo", pkgname, result.MachineUUID, result.MachineIP, result.Stderr)
					continue
				}
			}
			return machines, nil
		case "gala-spider", "gala-anteater", "gala-inference":
			deploy_machine := []map[string]string{}
			for _, result := range cmdresults {
				if result.RetCode == 1 && strings.Contains(result.Stdout, "is not installed") && result.Stderr == "" {
					// logger.Error("%s not installed in the process of running getpkgdeployinfo: %s, %s, %s; ", pkgname, result.MachineUUID, result.MachineIP, result.Stderr)
					continue
				} else if result.RetCode == 127 && result.Stdout == "" && strings.Contains(result.Stderr, "command not found") {
					// logger.Error("rpm not installed when running getpkgdeployinfo: %s, %s, %s", result.MachineUUID, result.MachineIP, result.Stderr)
					continue
				} else if result.RetCode == 0 && len(result.Stdout) > 0 && result.Stderr == "" {
					reader1 := strings.NewReader(result.Stdout)
					v, err := utils.ReadInfo(reader1, `^Version.*`)
					if err != nil && len(v) != 0 {
						logger.Error("failed to read RPM package version when running getpkgdeployinfo: %s, %s, %s", result.MachineUUID, result.MachineIP, result.Stderr)
						continue
					}
					reader2 := strings.NewReader(result.Stdout)
					d, err := utils.ReadInfo(reader2, `^Install Date.*`)
					if err != nil && len(d) != 0 {
						logger.Error("failed to read RPM package install date when running getpkgdeployinfo: %s, %s, %s", result.MachineUUID, result.MachineIP, result.Stderr)
					}

					for _, m := range machines {
						if m.UUID == result.MachineUUID {
							switch pkgname {
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
					deploy_machine = append(deploy_machine, map[string]string{"ip": result.MachineIP, "uuid": result.MachineUUID})
				} else {
					logger.Error("failed to run command: rpm -qi %s in %s, %s, %s when running getpkgdeployinfo", pkgname, result.MachineUUID, result.MachineIP, result.Stderr)
					continue
				}
			}
			if len(deploy_machine) == 0 {
				logger.Error("%s not deployed in any machine", pkgname)
				return machines, nil
			}
			logger.Debug("%s is deployed on %v", pkgname, deploy_machine)
			return machines, nil
		case "kafka", "arangodb", "prometheus", "pyroscope", "elasticsearch", "logstash":
			return machines, nil
		}
	}
	return nil, fmt.Errorf("runcommand error: %s", err)
}

// 获取集群gala-ops组件运行状态
func GetPkgRunningInfo(machines []*database.Agent, batch *common.Batch, pkgname string) ([]*database.Agent, error) {
	// 运行状态检测自检时将未部署pkgname的机器从batch.machinesuuids数组中移除
	batch_deployed := batch
	delete_from_batch := func(mgopherdeploy bool, muuid string, b *common.Batch) *common.Batch {
		if !mgopherdeploy {
			for i, bm := range b.MachineUUIDs {
				if muuid == bm {
					copy(b.MachineUUIDs[i:], b.MachineUUIDs[i+1:])
				}
			}
		}
		return b
	}
	for _, m := range machines {
		switch pkgname {
		case "gala-gopher":
			batch_deployed = delete_from_batch(m.Gopher_deploy, m.UUID, batch_deployed)
		case "gala-spider":
			batch_deployed = delete_from_batch(m.Spider_deploy, m.UUID, batch_deployed)
		case "gala-anteater":
			batch_deployed = delete_from_batch(m.Anteater_deploy, m.UUID, batch_deployed)
		case "gala-inference":
			batch_deployed = delete_from_batch(m.Inference_deploy, m.UUID, batch_deployed)
		}
	}

	cmdresults, err := Galaops.Sdkmethod.RunCommand(batch_deployed, "systemctl status "+pkgname)
	if err != nil {
		return nil, fmt.Errorf("runcommand error: %s", err)
	}

	for _, result := range cmdresults {
		// ttcode
		// logger.Debug("\033[32mrunning:\033[0m IP: %s, UUID: %s, code: %d, stdo: %s, stde: %s", result.MachineIP, result.MachineUUID, result.RetCode, result.Stdout, result.Stderr)
		if result.RetCode == 3 && strings.Contains(result.Stdout, "Active: inactive (dead)") && result.Stderr == "" {
			for _, m := range machines {
				if m.UUID == result.MachineUUID {
					switch pkgname {
					case "gala-gopher":
						m.Gopher_running = false
					case "gala-anteater":
						m.Anteater_running = false
					case "gala-inference":
						m.Inference_running = false
					case "gala-spider":
						m.Spider_running = false
					}
				}
			}
		} else if result.RetCode == 0 && strings.Contains(result.Stdout, "Active: active (running)") && result.Stderr == "" {
			for _, m := range machines {
				if m.UUID == result.MachineUUID {
					switch pkgname {
					case "gala-gopher":
						m.Gopher_running = true
					case "gala-anteater":
						m.Anteater_running = true
					case "gala-inference":
						m.Inference_running = true
					case "gala-spider":
						m.Spider_running = true
					}
				}
			}
		} else {
			logger.Error("Err getting running status in getpkgrunninginfo: %s, %s", pkgname, result)
		}
	}

	return machines, nil
}
