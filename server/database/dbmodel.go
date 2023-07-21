package database

type Agent struct {
	ID                        uint   `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	UUID                      string `gorm:"not null;unique" json:"UUID"`
	IP                        string `gorm:"not null" json:"IP"`
	Port                      string `gorm:"not null" json:"port"`
	Department                string `json:"Department"`
	State                     int    `gorm:"not null" json:"State"` // true:running false:not running
	Gopher_deploy             bool   // ture:installed false:uninstalled
	Gopher_running            bool   // true:running false:not running
	Gopher_version            string `gorm:"type:varchar(50)"`
	Gopher_installtime        string `gorm:"type:varchar(100)"`
	Spider_deploy             bool   // ture:installed false:uninstalled
	Spider_running            bool   // true:running false:not running
	Spider_version            string `gorm:"type:varchar(50)"`
	Spider_installtime        string `gorm:"type:varchar(100)"`
	Anteater_deploy           bool   // ture:installed false:uninstalled
	Anteater_running          bool   // true:running false:not running
	Anteater_version          string `gorm:"type:varchar(50)"`
	Anteater_installtime      string `gorm:"type:varchar(100)"`
	Inference_deploy          bool   // ture:installed false:uninstalled
	Inference_running         bool   // true:running false:not running
	Inference_version         string `gorm:"type:varchar(50)"`
	Inference_installtime     string `gorm:"type:varchar(100)"`
	Kafka_deploy              bool   // ture:installed false:uninstalled
	Kafka_running             bool   // true:running false:not running
	Kafka_version             string `gorm:"type:varchar(50)"`
	Kafka_installtime         string `gorm:"type:varchar(100)"`
	Prometheus_deploy         bool   // ture:installed false:uninstalled
	Prometheus_running        bool   // true:running false:not running
	Prometheus_version        string `gorm:"type:varchar(50)"`
	Prometheus_installtime    string `gorm:"type:varchar(100)"`
	Pyroscope_deploy          bool   // ture:installed false:uninstalled
	Pyroscope_running         bool   // true:running false:not running
	Pyroscope_version         string `gorm:"type:varchar(50)"`
	Pyroscope_installtime     string `gorm:"type:varchar(100)"`
	Arangodb_deploy           bool   // ture:installed false:uninstalled
	Arangodb_running          bool   // true:running false:not running
	Arangodb_version          string `gorm:"type:varchar(50)"`
	Arangodb_installtime      string `gorm:"type:varchar(100)"`
	Elasticsearch_deploy      bool   // ture:installed false:uninstalled
	Elasticsearch_running     bool   // true:running false:not running
	Elasticsearch_version     string `gorm:"type:varchar(50)"`
	Elasticsearch_installtime string `gorm:"type:varchar(100)"`
	Logstash_deploy           bool
	Logstash_running          bool
	Logstash_version          string `gorm:"type:varchar(50)"`
	Logstash_installtime      string `gorm:"type:varchar(100)"`
	Nginx_deploy              bool
	Nginx_running             bool
	Nginx_version             string `gorm:"type:varchar(50)"`
	Nginx_installtime         string `gorm:"type:varchar(100)"`
}
