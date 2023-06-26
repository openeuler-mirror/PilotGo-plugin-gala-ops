package database

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"openeuler.org/PilotGo/gala-ops-plugin/config"
)

var GlobalDB *gorm.DB

func MysqlInit(conf *config.MysqlConf) error {
	url := fmt.Sprintf("%s:%s@(%s:%d)/%s?charset=utf8mb4&parseTime=true",
		conf.User,
		conf.Password,
		conf.Host,
		conf.Port,
		conf.Database)

	gdb, err := gorm.Open(mysql.Open(url), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	})
	if err != nil {
		return err
	}
	GlobalDB = gdb

	db, err := gdb.DB()
	if err != nil {
		return err
	}

	db.SetMaxIdleConns(10)
	db.SetMaxOpenConns(100)

	GlobalDB.AutoMigrate(&Agent{})

	return nil
}

func DB() *gorm.DB {
	return GlobalDB
}
