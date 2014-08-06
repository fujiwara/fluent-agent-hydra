package main

import (
	"flag"
	"fmt"
	"github.com/fujiwara/fluent-agent-hydra/fluent"
	"github.com/fujiwara/fluent-agent-hydra/hydra"
	"log"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"
)

const (
	defaultMessageKey = "message"
)

var (
	trapSignals = []os.Signal{
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT}
)

func main() {
	var (
		configFile  string
		help        bool
		fieldName   string
		monitorAddr string
	)
	flag.StringVar(&configFile, "c", "", "configuration file path")
	flag.BoolVar(&help, "h", false, "show help message")
	flag.BoolVar(&help, "help", false, "show help message")
	flag.StringVar(&fieldName, "f", defaultMessageKey, "fieldname of fluentd log message attribute (DEFAULT: message)")
	flag.StringVar(&monitorAddr, "monitor", "127.0.0.1:24223", "monitor HTTP server address")
	flag.Parse()

	if help {
		usage()
	}
	if pprofile := os.Getenv("PPROF"); pprofile != "" {
		f, err := os.Create(pprofile)
		if err != nil {
			log.Fatal("[error] Can't create profiling stat file.", err)
		}
		log.Println("[info] StartCPUProfile() stat file", f.Name())
		pprof.StartCPUProfile(f)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, trapSignals...)
	if configFile != "" {
		config, err := hydra.ReadConfig(configFile)
		if err != nil {
			log.Println("Can't load config", err)
			os.Exit(2)
		}
		runWithConfig(config)
	} else if args := flag.Args(); len(args) >= 3 {
		config := newConfig(args, fieldName, monitorAddr)
		runWithConfig(config)
	} else {
		usage()
	}
	sig := <-done
	log.Println("[info] SIGNAL", sig, ", exit.")
	pprof.StopCPUProfile()
	os.Exit(0)
}

func usage() {
	fmt.Println("Usage of fluent-agent-hydra")
	fmt.Println("")
	fmt.Println("  fluent-agent-hydra -c config.toml")
	fmt.Println("  fluent-agent-hydra [options] TAG TARGET_FILE PRIMARY_SERVER SECONDARY_SERVER")
	fmt.Println("")
	flag.PrintDefaults()
	os.Exit(1)
}

func newConfig(args []string, fieldName string, monitorAddr string) hydra.Config {
	tag := args[0]
	file := args[1]
	servers := args[2:]

	logs := make([]hydra.ConfigLogfile, 1)
	logs[0] = hydra.ConfigLogfile{Tag: tag, File: file}
	return hydra.Config{
		Servers:        servers,
		Logs:           logs,
		FieldName:      fieldName,
		MonitorAddress: monitorAddr,
	}
}

func runWithConfig(config hydra.Config) {
	messageCh, monitorCh := hydra.NewChannel()

	_, err := hydra.NewMonitorServer(monitorCh, config.MonitorAddress)
	if err != nil {
		log.Println("[error] Couldn't start monitor server.", err)
	}

	loggers := make([]*fluent.Fluent, len(config.Servers))
	for i, server := range config.Servers {
		logger, err := fluent.New(fluent.Config{Server: server})
		if err != nil {
			log.Println("[warning] Can't initialize fluentd server.", err)
		} else {
			log.Println("[info] server", server)
		}
		loggers[i] = logger
	}

	for _, logfile := range config.Logs {
		var tag string
		if config.TagPrefix != "" {
			tag = config.TagPrefix + "." + logfile.Tag
		} else {
			tag = logfile.Tag
		}
		go hydra.Trail(logfile.File, tag, messageCh)
	}

	var fieldName string
	if config.FieldName != "" {
		fieldName = config.FieldName
	} else {
		fieldName = defaultMessageKey
	}

	go hydra.Forward(messageCh, monitorCh, fieldName, loggers...)
}
