package agentmanager

import (
	"fmt"
	"strings"

	"gitee.com/openeuler/PilotGo-plugins/sdk/common"
	"gitee.com/openeuler/PilotGo-plugins/sdk/logger"
	"openeuler.org/PilotGo/gala-ops-plugin/database"
	"openeuler.org/PilotGo/gala-ops-plugin/utils"
)

// 获取业务机集群gala-gopher版本信息
func GetPkgVersion(machines []*database.Agent, batch *common.Batch, pkgname string) ([]*database.Agent, error) {
	cmdresults, err := Galaops.Sdkmethod.RunCommand(batch, "rpm -qi"+pkgname)
	if err == nil {
		for _, result := range cmdresults {
			if result.RetCode == 1 && strings.Contains(result.Stdout, "is not installed") && result.Stderr == "" {
				logger.Error("%s not installed happened when getpackageversion: %s, %s, %s; ", pkgname, result.MachineUUID, result.MachineIP, result.Stderr)
				continue
			} else if result.RetCode == 127 && result.Stdout == "" && strings.Contains(result.Stderr, "command not found") {
				logger.Error("rpm not installed when getpackageversion: %s, %s, %s", result.MachineUUID, result.MachineIP, result.Stderr)
				continue
			} else if result.RetCode == 0 && len(result.Stdout) > 0 && result.Stderr == "" {
				reader := strings.NewReader(result.Stdout)
				v, err := utils.ReadInfo(reader, `^Version.*`)
				if err != nil && len(v) != 0 {
					logger.Error("failed to read RPM package version when getpackageversion: %s, %s, %s", result.MachineUUID, result.MachineIP, result.Stderr)
					continue
				}
				for _, m := range machines {
					if m.UUID == result.MachineUUID {
						switch pkgname {
						case "gala-gopher":
							m.Gopher_version = v
						case "gala-anteater":
							m.Anteater_version = v
						case "gala-inference":
							m.Inference_version = v
						case "gala-spider":
							m.Spider_version = v
						}
					}
				}
			} else {
				logger.Error("failed to run command: rpm -qi %s in %s, %s, %s when getpackageversion", pkgname, result.MachineUUID, result.MachineIP, result.Stderr)
				continue
			}
		}
		return machines, nil
	}
	return nil, fmt.Errorf("runcommand error: %s", err)
}
