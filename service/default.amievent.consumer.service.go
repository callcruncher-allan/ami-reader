package service

import (
	"ami-reader/conf"
	"encoding/json"
	"fmt"
	"time"
)

type defaultAmiEventConsumer struct {
	appConfig    *conf.AppConf
	eventJobChan chan map[string]string
}

func NewDefaultAmiEventConsumerService(appConfig *conf.AppConf) AmiEventConsumer {
	return &defaultAmiEventConsumer{appConfig, nil}
}

func (service *defaultAmiEventConsumer) Initialize() error {
	appConfig := service.appConfig
	eventJobChan := make(chan map[string]string, *appConfig.NumberOfJobs)
	numberOfWorkers := *appConfig.NumberOfWorkers
	for w := 1; w <= numberOfWorkers; w++ {
		go service.worker(w, eventJobChan)
	}
	service.eventJobChan = eventJobChan
	return nil
}

func (service *defaultAmiEventConsumer) Destroy() {
	close(service.eventJobChan)
}

func (service *defaultAmiEventConsumer) Consume(event map[string]string) {
	event["timestamp"] = time.Now().Format(time.RFC3339Nano)
	service.eventJobChan <- event
}

func (service *defaultAmiEventConsumer) worker(id int, eventJobChan <-chan map[string]string) {
	for event := range eventJobChan {
		eventJsonB, _ := json.Marshal(event)
		eventJson := string(eventJsonB)
		fmt.Println("worker", id, "event:", eventJson)
	}
}
