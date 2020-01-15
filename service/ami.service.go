package service

import (
	"ami-reader/conf"
	"bufio"
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"os"
	"os/signal"
	"time"
)

const (
	commandResponseKey = "CommandResponse"
)

type AmiService interface {
	Connect() error
	Login() error
	Listen() error
	Disconnect()
	IsConnected() bool
	IsLoggedIn() bool
}

type amiService struct {
	appConfig               *conf.AppConf
	dialString              string
	con                     net.Conn
	connected               bool
	isLoggedIn              bool
	amiEventConsumerService AmiEventConsumer
}

func NewAmiService(appConfig *conf.AppConf, amiEventConsumerService AmiEventConsumer) AmiService {
	service := amiService{}
	service.appConfig = appConfig
	service.dialString = fmt.Sprintf("%s:%d", *appConfig.AmiHost, *appConfig.AmiPort)
	service.amiEventConsumerService = amiEventConsumerService
	return &service
}

func (service *amiService) Connect() error {
	appConfig := service.appConfig
	var con net.Conn
	var err error
	dialString := service.dialString
	dialRetry := *appConfig.DialRetry
	i := 1
	for ; i <= dialRetry; i++ {
		log.Info("Connecting to ", dialString)
		con, err = net.DialTimeout("tcp", dialString, *appConfig.DialTimeout)
		if err == nil {
			break
		} else {
			if i < dialRetry {
				log.Warnf("[Retry %d] Failed to connect to %s. Reason: %v", i, dialString, err)
				time.Sleep(time.Duration(1 * time.Second))
			} else {
				err = errors.Wrap(err, fmt.Sprintf("Failed to connect to %v. Retry exhausted.", dialString))
			}
		}
	}
	if err == nil {
		service.handleCrtlC()
		service.con = con
		service.connected = true
		err = service.amiEventConsumerService.Initialize()
		if err != nil {
			return err
		}
	}
	return err
}

func (service *amiService) Login() error {
	conn := service.con
	appConfig := service.appConfig
	action := map[string]string{
		"Action":   "Login",
		"ActionID": "1",
		"Username": *appConfig.AmiUsername,
		"Secret":   *appConfig.AmiPassword,
	}
	serialized := serialize(action)
	_, err := conn.Write(serialized)
	if err != nil {
		return err
	}
	reader := bufio.NewReader(conn)
	result, err := readMessage(reader)
	if err != nil {
		return err
	}

	if result["Response"] != "Success" && result["Message"] != "Authentication accepted" {
		return errors.New(result["Message"])
	}
	service.isLoggedIn = true
	return nil
}

func (service *amiService) Listen() error {
	bufReader := bufio.NewReader(service.con)
	var event map[string]string
	var err error
	for {
		// Set a deadline for reading. Read operation will fail if no data is received after deadline.
		err = service.con.SetReadDeadline(time.Now().Add(*service.appConfig.ReadTimeout))
		if err != nil {
			err = errors.Wrap(err, fmt.Sprintf("Failed to set read deadline timeout."))
			break
		}
		event, err = readMessage(bufReader)
		if e, ok := err.(interface{ Timeout() bool }); ok && e.Timeout() {
			// TODO: better handling of timeout and link graceful shutdown
			log.Debug("No data received or timeout reading from ami socket.")
			if !service.connected {
				break
			}
		} else if err != nil {
			break
		} else {
			service.amiEventConsumerService.Consume(event)
		}
	}
	return err
}

func (service *amiService) Disconnect() {
	if service.con != nil {
		if service.isLoggedIn {
			log.Info("Loggin out from AMI.")
			// TODO: Logout logic
		}
		con := service.con
		service.connected = false
		service.con = nil
		err := con.Close()
		if err != nil {
			log.Errorf("Failed to close opened connection in %s. Reason: %v", service.dialString, err)
		}
		// TODO: Graceful shutdown when read message is still in transit to message brokers
		service.amiEventConsumerService.Destroy()
	}

}

func (service *amiService) IsConnected() bool {
	return service.connected
}

func (service *amiService) IsLoggedIn() bool {
	return service.isLoggedIn
}

func (service *amiService) handleCrtlC() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		<-signalChan
		log.Info("Received an interrupt, closing the connection...")
		service.Disconnect()
	}()
}

func serialize(data map[string]string) []byte {
	var outBuf bytes.Buffer
	for key := range data {
		outBuf.WriteString(key)
		outBuf.WriteString(": ")
		outBuf.WriteString(data[key])
		outBuf.WriteString("\r\n")
	}
	outBuf.WriteString("\r\n")
	return outBuf.Bytes()
}

// Copied from https://github.com/ivahaev/amigo/blob/master/ami.go#L306
// TODO: See if we can further optimize this code
func readMessage(r *bufio.Reader) (m map[string]string, err error) {
	m = make(map[string]string)
	var responseFollows bool
	for {
		kv, _, err := r.ReadLine()
		if len(kv) == 0 || err != nil {
			return m, err
		}

		var key string
		i := bytes.IndexByte(kv, ':')
		if i >= 0 {
			endKey := i
			for endKey > 0 && kv[endKey-1] == ' ' {
				endKey--
			}
			key = string(kv[:endKey])
		}

		if key == "" && !responseFollows {
			if err != nil {
				return m, err
			}
			continue
		}

		if responseFollows && key != "Privilege" && key != "ActionID" {
			if string(kv) != "--END COMMAND--" {
				if len(m[commandResponseKey]) == 0 {
					m[commandResponseKey] = string(kv)
				} else {
					m[commandResponseKey] = fmt.Sprintf("%s\n%s", m[commandResponseKey], string(kv))
				}
			}

			if err != nil {
				return m, err
			}

			continue
		}

		i++
		for i < len(kv) && (kv[i] == ' ' || kv[i] == '\t') {
			i++
		}
		value := string(kv[i:])

		if key == "Response" && value == "Follows" {
			responseFollows = true
		}

		m[key] = value

		if err != nil {
			return m, err
		}
	}
}