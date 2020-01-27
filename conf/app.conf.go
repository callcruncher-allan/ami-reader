package conf

import (
    "errors"
    log "github.com/sirupsen/logrus"
    "github.com/spf13/viper"
    "gopkg.in/ini.v1"
    "strconv"
    "time"
)

type AppConf struct {
    AmiUsername     *string
    AmiPassword     *string
    AmiHost         *string
    AmiPort         *int
    HostDeviceId    *string
    DialTimeout     *time.Duration
    ReadTimeout     *time.Duration
    DialRetry       *int
    NumberOfWorkers *int
    NumberOfJobs    *int
    LogEvents       *bool
    AmqpUrl         *string
    AmqpXchName     *string
    AmqpXchType     *string
}

func NewAppConf() (*AppConf, error) {
    managerConfFile := getStringEnv("AMI_CONF_PATH", "/etc/asterisk/manager.conf")
    amiHost := getStringEnv("AMI_HOST", "")
    if amiHost == "" {
        return nil, errors.New("AMI_HOST environment variable not found")
    }
    amiPort := getIntEnv("AMI_PORT", 5038)
    dialTimeout := getDurationEnv("DIAL_TIMEOUT", time.Duration(0)*time.Second)
    readTimeout := getDurationEnv("READ_TIMEOUT", time.Duration(10)*time.Second)
    dialRetry := getIntEnv("DIAL_RETRY", 3)
    numberOfWorkers := getIntEnv("NUMBER_OF_WORKERS", 50)
    numberOfJobs := getIntEnv("NUMBER_OF_JOBS", 60)
    if numberOfJobs < numberOfWorkers {
        return nil, errors.New("NUMBER_OF_WORKERS should be more than or equal to NUMBER_OF_JOBS")
    }
    amiUser := getStringEnv("AMI_USER", "admin")
    amiPassword := getStringEnv("AMI_PASS", "")
    if amiPassword == "" {
        cfg, err := ini.Load(managerConfFile)
        if err != nil {
            log.Errorf("Fail to read %v. Reason: %v", managerConfFile, err)
            return nil, err
        }
        amiPassword = cfg.Section(amiUser).Key("secret").String()
    }

    hostDeviceId := getStringEnv("HOST_DEVICE_ID", "")
    if hostDeviceId == "" {
        return nil, errors.New("HOST_DEVICE_ID environment variable not found")
    }
    logEvents := getBoolEnv("LOG_EVENTS")
    amqpUrl := getStringEnv("AMQP_URL", "")
    if amqpUrl == "" {
        return nil, errors.New("AMQP_URL environment variable not found")
    }
    amqpXchName := getStringEnv("AMQP_EXCHANGE_NAME", "amq.direct")
    amqpXchType := getStringEnv("AMQP_EXCHANGE_TYPE", "direct")
    return &AppConf{
        &amiUser,
        &amiPassword,
        &amiHost,
        &amiPort,
        &hostDeviceId,
        &dialTimeout,
        &readTimeout,
        &dialRetry,
        &numberOfWorkers,
        &numberOfJobs,
        &logEvents,
        &amqpUrl,
        &amqpXchName,
        &amqpXchType,
    }, nil
}

func getStringEnv(name string, defaultVal string) string {
    // viper is already initialized in main
    result := viper.GetString(name)
    if result == "" {
        result = defaultVal
    }

    return result
}

func getDurationEnv(envKey string, defaultDuration time.Duration) time.Duration {
    dialTimeout := getStringEnv(envKey, "0")
    timeout, err := strconv.Atoi(dialTimeout)
    if err == nil {
        return time.Duration(timeout) * time.Second
    } else {
        return defaultDuration
    }
}

func getIntEnv(envKey string, defaultVal int) int {
    str := getStringEnv(envKey, strconv.Itoa(defaultVal))
    i, err := strconv.Atoi(str)
    if err == nil {
        return i
    } else {
        return defaultVal
    }
}

func getBoolEnv(envKey string) bool {
    str := getStringEnv(envKey, "false")
    b, err := strconv.ParseBool(str)
    if err == nil {
        return b
    } else {
        return false
    }
}
