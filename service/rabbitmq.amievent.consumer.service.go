package service

import (
    "ami-reader/conf"
    "encoding/json"
    "fmt"
    "github.com/pkg/errors"
    log "github.com/sirupsen/logrus"
    "github.com/streadway/amqp"
    "os"
    "time"
)

type rabbitMQAmiEventConsumer struct {
    appConfig    *conf.AppConf
    eventJobChan chan map[string]string
    eventFile    *os.File
    eventLogger  *log.Logger
    amqpConn     *amqp.Connection
    amqpChannel  *amqp.Channel
}

func NewRabbitMQAmiEventConsumerService(appConfig *conf.AppConf) AmiEventConsumer {
    return &rabbitMQAmiEventConsumer{appConfig, nil, nil, nil, nil, nil}
}

func (service *rabbitMQAmiEventConsumer) Initialize() error {
    // Initialize unsent event logger
    currentTime := time.Now()
    fileName := currentTime.Format("2006-01-02") + "_events.log"
    file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
    if err == nil {
        eventLogger := log.New()
        eventLogger.SetFormatter(&log.JSONFormatter{})
        eventLogger.SetOutput(file)
        eventLogger.SetLevel(log.InfoLevel)
        service.eventFile = file
        service.eventLogger = eventLogger
    } else {
        errMsg := fmt.Sprintf("Failed to open file %s", fileName)
        log.Error(errMsg)
        return errors.Wrap(err, errMsg)
    }

    // Initial AMQP
    appConfig := service.appConfig
    amqpUrl := *appConfig.AmqpUrl
    log.Infof("Connecting to MQ %s", amqpUrl)
    conn, err := amqp.Dial(amqpUrl)
    if err != nil {
        errMsg := fmt.Sprintf("Failed to connect MQ %s.", amqpUrl)
        log.Error(errMsg)
        return errors.Wrap(err, errMsg)
    }
    log.Info("Successfully connected to MQ")
    service.amqpConn = conn
    ch, err := conn.Channel()
    if err != nil {
        log.Error("Failed to open a channel.")
        return errors.Wrap(err, "Failed to open a channel.")
    }
    service.amqpChannel = ch
    err = ch.ExchangeDeclarePassive(
        *appConfig.AmqpXchName, // name
        *appConfig.AmqpXchType, // type
        true,                   // durable
        false,                  // auto-deleted
        false,                  // internal
        false,                  // no-wait
        nil,                    // arguments
    )
    if err != nil {
        log.Errorf("Failed to declare amq.direct exchange. Reason %v", err)
        return errors.Wrap(err, "Failed to declare amq.direct exchange.")
    }
    log.Info("Done initializing MQ")
    // Initialize workers
    eventJobChan := make(chan map[string]string, *appConfig.NumberOfJobs)
    numberOfWorkers := *appConfig.NumberOfWorkers
    for w := 1; w <= numberOfWorkers; w++ {
        go service.worker(w, eventJobChan)
    }
    service.eventJobChan = eventJobChan
    return nil
}

func (service *rabbitMQAmiEventConsumer) Destroy() {
    log.Info("Closing unseen event log file.")
    if service.eventFile != nil {
        if err := service.eventFile.Close(); err != nil {
            log.Errorf("Failed to close file %s. Reason: %v.", service.eventFile.Name(), err)
        }
    }
    log.Info("Closing MQ channel.")
    if service.amqpChannel != nil {
        if err := service.amqpChannel.Close(); err != nil {
            log.Errorf("Failed to close amqp channel. Reason: %v.", err)
        }
    }
    log.Info("Closing MQ connection.")
    if service.amqpConn != nil {
        if err := service.amqpConn.Close(); err != nil {
            log.Errorf("Failed to close amqp connection. Reason: %v.", err)
        }
    }
    log.Info("Closing workers.")
    if service.eventJobChan != nil {
        close(service.eventJobChan)
    }
}

func (service *rabbitMQAmiEventConsumer) Consume(event map[string]string) {
    service.eventJobChan <- event
}

func (service *rabbitMQAmiEventConsumer) worker(id int, eventJobChan <-chan map[string]string) {
    appConfig := service.appConfig
    logEvents := *appConfig.LogEvents
    for event := range eventJobChan {
        eventJsonB, _ := json.Marshal(event)
        eventJson := string(eventJsonB)
        fmt.Println("worker", id, "event:", eventJson)
        err := service.amqpChannel.Publish(
            *appConfig.AmqpXchName,   // exchange
            * appConfig.HostDeviceId, // routing key
            true,                     // undelivered when no queue is bound that matches the routing key
            false,                    // deliver even if no consumer on the matched queue is ready to accept the delivery
            amqp.Publishing{
                ContentType: "application/json",
                Body:        eventJsonB,
            })
        if logEvents || err != nil {
            if err != nil {
                log.Errorf("Failed to send event: %s", eventJson)
            }
            service.eventLogger.Info(eventJson)
        }
    }
}
