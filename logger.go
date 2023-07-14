package gologging

import (
	"database/sql"
	"fmt"
	"io"
	"log/syslog"
	"os"
	"strings"
	"sync"

	_ "github.com/lib/pq"
	"github.com/luraproject/lura/v2/config"
	"github.com/luraproject/lura/v2/logging"
	gologging "github.com/op/go-logging"
)

const Namespace = "github_com/devopsfaith/krakend-gologging"

var (
	// ErrEmptyValue is the error returned when there is no config under the namespace
	ErrWrongConfig = fmt.Errorf("getting the extra config for the krakend-gologging module")
	// DefaultPattern is the pattern to use for rendering the logs
	LogstashPattern          = `{"@timestamp":"%{time:2006-01-02T15:04:05.000+00:00}", "@version": 1, "level": "%{level}", "message": "%{message}", "module": "%{module}"}`
	DefaultPattern           = ` %{time:2006/01/02 - 15:04:05.000} %{color}▶ %{level}%{color:reset} %{message}`
	ActivePattern            = DefaultPattern
	defaultFormatterSelector = func(io.Writer) string { return ActivePattern }
	defaultSyslogFacility    = syslog.LOG_LOCAL3
	defaultSyslogSeverity    = syslog.LOG_CRIT

	DB *sql.DB

	// for initializing DB once
	once sync.Once
)

func SetFormatterSelector(f func(io.Writer) string) {
	defaultFormatterSelector = f
}

type DBLogger struct {
	db     *sql.DB
	logger *gologging.Logger
}

func NewLogger(cfg config.ExtraConfig, ws ...io.Writer) (logging.Logger, error) {
	logConfig, ok := ConfigGetter(cfg).(Config)
	if !ok {
		return nil, ErrWrongConfig
	}
	module := "KRAKEND"
	loggr := gologging.MustGetLogger(module)

	if logConfig.StdOut {
		ws = append(ws, os.Stdout)
	}

	if logConfig.Syslog {
		var err error
		var w *syslog.Writer
		w, err = syslog.New(logConfig.SyslogSeverity|logConfig.SysLogFacility, logConfig.Prefix)
		if err != nil {
			return nil, err
		}
		ws = append(ws, syslogIoWriterWrapper{w})
	}

	if logConfig.Format == "logstash" {
		ActivePattern = LogstashPattern
		logConfig.Prefix = ""
	}

	if logConfig.Format == "custom" {
		ActivePattern = logConfig.CustomFormat
		logConfig.Prefix = ""
	}

	// Initialize DB once
	once.Do(func() {
		var err error
		DB, err = sql.Open("postgres", fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", logConfig.Host, logConfig.Port, logConfig.DBUser, logConfig.DBPass, logConfig.DBName))
		if err != nil {
			loggr.Error("Failed to connect to the database: ", err)
			return
		}

		// Add this line to check if the connection is successful
		err = DB.Ping()
		if err != nil {
			loggr.Error("Failed to establish a connection with the database: ", err)
		}
	})

	backends := []gologging.Backend{}
	for _, w := range ws {
		var pattern string
		var prefix string
		switch w.(type) {
		case syslogIoWriterWrapper:
			pattern = "%{level} > %{message}"
		default:
			prefix = logConfig.Prefix
			pattern = defaultFormatterSelector(w)
		}
		backend := gologging.NewLogBackend(w, prefix, 0)
		format := gologging.MustStringFormatter(pattern)
		backendLeveled := gologging.AddModuleLevel(gologging.NewBackendFormatter(backend, format))
		logLevel, err := gologging.LogLevel(logConfig.Level)
		if err != nil {
			return nil, err
		}
		backendLeveled.SetLevel(logLevel, module)
		backends = append(backends, backendLeveled)
	}

	gologging.SetBackend(backends...)
	return Logger{loggr}, nil
}

func (l DBLogger) Debug(v ...interface{}) {
	msg := fmt.Sprint(v...)
	l.logger.Debug(msg)
	_, err := l.db.Exec("INSERT INTO xebula.logs(level, msg) VALUES('debug', $1)", msg)
	if err != nil {
		l.logger.Error("Failed to write to database: ", err)
	}
}

func ConfigGetter(e config.ExtraConfig) interface{} {
	v, ok := e[Namespace]
	if !ok {
		return nil
	}
	tmp, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}
	cfg := Config{}
	if v, ok := tmp["stdout"]; ok {
		cfg.StdOut = v.(bool)
	}
	if v, ok := tmp["syslog"]; ok {
		cfg.Syslog = v.(bool)
	}

	cfg.SysLogFacility = defaultSyslogFacility
	if v, ok := tmp["syslog_facility"].(string); ok {
		cfg.SysLogFacility = parseSyslogFacility(v)
	}

	if v, ok := tmp["level"].(string); ok {
		cfg.Level = v
	}
	cfg.SyslogSeverity = parseSyslogSeverity(cfg.Level)

	if v, ok := tmp["prefix"].(string); ok {
		cfg.Prefix = v
	}
	if v, ok := tmp["format"].(string); ok {
		cfg.Format = v
	}
	if v, ok := tmp["custom_format"].(string); ok {
		cfg.CustomFormat = v
	}
	if v, ok := tmp["db_host"].(string); ok {
		cfg.Host = v
	}
	if v, ok := tmp["db_port"].(string); ok {
		cfg.Port = v
	}
	if v, ok := tmp["db_user"].(string); ok {
		cfg.DBUser = v
	}
	if v, ok := tmp["db_pass"].(string); ok {
		cfg.DBPass = v
	}
	if v, ok := tmp["db_name"].(string); ok {
		cfg.DBName = v
	}

	return cfg
}

type Config struct {
	Level          string
	StdOut         bool
	Syslog         bool
	SysLogFacility syslog.Priority
	SyslogSeverity syslog.Priority
	Prefix         string
	Format         string
	CustomFormat   string
	Host           string
	Port           string
	DBUser         string
	DBPass         string
	DBName         string
}

// Logger is a wrapper over a github.com/op/go-logging logger
type Logger struct {
	logger *gologging.Logger
}

// Debug implements the logger interface
func (l Logger) Debug(v ...interface{}) {
	l.logger.Debug(v...)
}

// Info implements the logger interface
func (l Logger) Info(v ...interface{}) {
	l.logger.Info(v...)
}

// Warning implements the logger interface
func (l Logger) Warning(v ...interface{}) {
	l.logger.Warning(v...)
}

// Error implements the logger interface
func (l Logger) Error(v ...interface{}) {
	l.logger.Error(v...)
}

// Critical implements the logger interface
func (l Logger) Critical(v ...interface{}) {
	l.logger.Critical(v...)
}

// Fatal implements the logger interface
func (l Logger) Fatal(v ...interface{}) {
	l.logger.Fatal(v...)
}

type syslogIoWriterWrapper struct {
	io.Writer
}

func parseSyslogFacility(name string) syslog.Priority {
	switch strings.ToLower(name) {
	case "local0":
		return syslog.LOG_LOCAL0
	case "local1":
		return syslog.LOG_LOCAL1
	case "local2":
		return syslog.LOG_LOCAL2
	case "local3":
		return syslog.LOG_LOCAL3
	case "local4":
		return syslog.LOG_LOCAL4
	case "local5":
		return syslog.LOG_LOCAL5
	case "local6":
		return syslog.LOG_LOCAL6
	case "local7":
		return syslog.LOG_LOCAL7
	default:
		return defaultSyslogFacility
	}
}

func parseSyslogSeverity(level string) syslog.Priority {
	switch strings.ToLower(level) {
	case "fatal":
		return syslog.LOG_EMERG
	case "critical":
		return syslog.LOG_CRIT
	case "error":
		return syslog.LOG_ERR
	case "warning":
		return syslog.LOG_WARNING
	case "info":
		return syslog.LOG_INFO
	case "debug":
		return syslog.LOG_DEBUG
	default:
		return defaultSyslogSeverity
	}
}
