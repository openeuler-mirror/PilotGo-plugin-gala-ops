package database

type Agent struct {
	ID                    uint   `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	UUID                  string `gorm:"not null;unique" json:"uuid"`
	IP                    string `gorm:"not null" json:"ip"`
	Port                  string `gorm:"not null" json:"port"`
	Department            string `json:"department"`
	State                 int    `gorm:"not null" json:"state"` // true:running false:not running
	Gopher_deploy         bool   // ture:installed false:uninstalled
	Gopher_running        bool   // true:running false:not running
	Gopher_version        string `gorm:"type:varchar(20)"`
	Gopher_installtime    string
	Spider_deploy         bool   // ture:installed false:uninstalled
	Spider_running        bool   // true:running false:not running
	Spider_version        string `gorm:"type:varchar(20)"`
	Spider_installtime    string
	Anteater_deploy       bool   // ture:installed false:uninstalled
	Anteater_running      bool   // true:running false:not running
	Anteater_version      string `gorm:"type:varchar(20)"`
	Anteater_installtime  string
	Inference_deploy      bool   // ture:installed false:uninstalled
	Inference_running     bool   // true:running false:not running
	Inference_version     string `gorm:"type:varchar(20)"`
	Inference_installtime string
	Kafka_deploy          bool   // ture:installed false:uninstalled
	Kafka_running         bool   // true:running false:not running
	Kafka_version         string `gorm:"type:varchar(20)"`
}
