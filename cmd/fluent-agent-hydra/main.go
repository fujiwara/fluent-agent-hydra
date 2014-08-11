package main

import (
	"flag"
	"fmt"
	"github.com/fujiwara/fluent-agent-hydra/hydra"
	"log"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"
)

var (
	version     string
	revision    string
	buildDate   string
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
		showVersion bool
	)
	flag.StringVar(&configFile, "c", "", "configuration file path")
	flag.BoolVar(&help, "h", false, "show help message")
	flag.BoolVar(&help, "help", false, "show help message")
	flag.StringVar(&fieldName, "f", hydra.DefaultFieldName, "fieldname of fluentd log message attribute (DEFAULT: message)")
	flag.StringVar(&monitorAddr, "monitor", "", "monitor HTTP server address")
	flag.BoolVar(&showVersion, "v", false, "show version")
	flag.BoolVar(&showVersion, "version", false, "show version")
	flag.Parse()

	if showVersion {
		fmt.Println("version:", version)
		fmt.Println("revision:", revision)
		fmt.Println("build:", buildDate)
		os.Exit(0)
	}
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
		run(config)
	} else if args := flag.Args(); len(args) >= 3 {
		config := hydra.NewConfigByArgs(args, fieldName, monitorAddr)
		run(config)
	} else {
		usage()
	}
	sig := <-done
	log.Println("[info] SIGNAL", sig, "exit.")
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

func run(config *hydra.Config) {
	messageCh, monitorCh := hydra.NewChannel()

	// start monitor server
	monitor, err := hydra.NewMonitor(config, monitorCh)
	if err != nil {
		log.Println("[error] Couldn't start monitor server.", err)
	} else {
		go monitor.Run()
	}

	// start out_forward
	outForward, err := hydra.NewOutForward(config.Servers, messageCh, monitorCh)
	if err != nil {
		log.Println("[error]", err)
	} else {
		go outForward.Run()
	}

	// start watcher && in_tail
	if len(config.Logs) > 0 {
		watcher, err := hydra.NewWatcher()
		if err != nil {
			log.Println("[error]", err)
		}
		for _, configLogfile := range config.Logs {
			tail, err := hydra.NewInTail(configLogfile, watcher, messageCh, monitorCh)
			if err != nil {
				log.Println("[error]", err)
			} else {
				go tail.Run()
			}
		}
		go watcher.Run()
	}

	// start in_forward
	for _, configReceiver := range config.Receivers {
		inForward, err := hydra.NewInForward(configReceiver, messageCh, monitorCh)
		if err != nil {
			log.Println("[error]", err)
		} else {
			go inForward.Run()
		}
	}
}
