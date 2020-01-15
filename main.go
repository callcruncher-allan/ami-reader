package main

import (
	"ami-reader/conf"
	"ami-reader/service"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"strings"

	//"net"
	"os"
)

func init() {
	// Log as JSON instead of the default ASCII formatter.
	log.SetFormatter(&log.JSONFormatter{})
	// TODO: file logger?
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}

func main() {
	log.Info("Loading app configurations.")
	loadEnv()
	appConfig, err := conf.NewAppConf()
	if err != nil {
		log.Errorf("Failed to initialize app config. Reason: %v", err)
		return
	}
	log.Infof("Loaded Configs:\nAMI Host: %s\nAMI Port: %d\nAMI User: %s\nHost Device ID: %s", *appConfig.AmiHost, *appConfig.AmiPort, *appConfig.AmiUsername, *appConfig.HostDeviceId)
	amiEventConsumer := service.NewDefaultAmiEventConsumerService(appConfig)
	amiService := service.NewAmiService(appConfig, amiEventConsumer)
	log.Infof("Connecting to AMI.")
	if err := amiService.Connect(); err != nil {
		log.Errorf("Failed to connect to Asterisk. Reason: %v.", err)
		return
	}
	log.Info("Logging in to AMI.")
	if err := amiService.Login(); err != nil {
		log.Errorf("Failed to login to Asterisk. Reason: %v.", err)
	}
	if amiService.IsLoggedIn() {
		log.Info("Login Successful.")
		if err := amiService.Listen(); err != nil {
			if !strings.Contains(err.Error(), "use of closed network connection") {
				log.Errorf("Error listening for events. Reason: %v.", err)
			} else if amiService.IsConnected() {
				log.Errorf("Error listening for events. Reason: %v.", err)
			}
		}
	}
	amiService.Disconnect()
}

func loadEnv() {
	viper.AutomaticEnv() // Read from system's environment variable
	if _, err := os.Stat("config.json"); !os.IsNotExist(err) {
		// Read from config.json if exists. Note: system's env variable still takes precedence
		viper.SetConfigName("config")
		viper.SetConfigType("json")
		viper.AddConfigPath(".")
		_ = viper.ReadInConfig()
	}
}
