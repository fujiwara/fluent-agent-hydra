package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"syscall"
	"time"

	"github.com/fujiwara/fluent-agent-hydra/hydra"
)

var (
	version     string
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
	flag.StringVar(&monitorAddr, "m", "", "monitor HTTP server address")
	flag.BoolVar(&showVersion, "v", false, "show version")
	flag.BoolVar(&showVersion, "version", false, "show version")
	flag.Parse()

	if showVersion {
		fmt.Println("version:", version)
		fmt.Printf("compiler:%s %s\n", runtime.Compiler, runtime.Version())
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

	var (
		config *hydra.Config
		err    error
	)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, trapSignals...)
	if configFile != "" {
		config, err = hydra.ReadConfig(configFile)
		if err != nil {
			log.Println("Can't load config", err)
			os.Exit(2)
		}
	} else if args := flag.Args(); len(args) >= 3 {
		config = hydra.NewConfigByArgs(args, fieldName, monitorAddr)
	} else {
		usage()
	}

	context := hydra.Run(config)
	go func() {
		context.InputProcess.Wait()
		sigCh <- hydra.NewSignal("all input processes terminated")
	}()

	// waiting for all input processes are terminated or got os signal
	sig := <-sigCh

	log.Println("[info] SIGNAL", sig, "shutting down")
	pprof.StopCPUProfile()

	go func() {
		time.Sleep(1 * time.Second) // at least wait 1 sec
		sig, ok := <-sigCh
		if !ok {
			return // closed
		}
		log.Println("[warn] SIGNAL", sig, "before shutdown completed. aborted")
		os.Exit(1)
	}()

	context.Shutdown()
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
