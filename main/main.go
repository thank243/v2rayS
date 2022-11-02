package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"

	_ "github.com/thank243/v2rayS/main/distro/all"
	"github.com/thank243/v2rayS/panel"
)

var (
	configFile   = flag.String("config", "", "Config file for v2rayS.")
	printVersion = flag.Bool("version", false, "show version")
)

var (
	version  = "0.3.1"
	codename = "v2rayS"
	intro    = "A v2ray backend that supports many panels"
)

func showVersion() {
	fmt.Printf("%s %s (%s) \n", codename, version, intro)
}

func getConfig() *viper.Viper {
	config := viper.New()

	// Set custom path and name
	if *configFile != "" {
		configName := path.Base(*configFile)
		configFileExt := path.Ext(*configFile)
		configNameOnly := strings.TrimSuffix(configName, configFileExt)
		configPath := path.Dir(*configFile)
		config.SetConfigName(configNameOnly)
		config.SetConfigType(strings.TrimPrefix(configFileExt, "."))
		config.AddConfigPath(configPath)
		// Set ASSET Path and Config Path for v2rayS
		os.Setenv("V2RAY_LOCATION_ASSET", configPath)
		os.Setenv("V2RAY_LOCATION_CONFIG", configPath)
	} else {
		// Set default config path
		config.SetConfigName("config")
		config.SetConfigType("yml")
		config.AddConfigPath(".")
	}

	if err := config.ReadInConfig(); err != nil {
		panic(fmt.Sprintf("Config file error: %s \n", err))
	}

	config.WatchConfig() // Watch the config

	return config
}

func main() {
	flag.Parse()
	showVersion()
	if *printVersion {
		return
	}

	config := getConfig()
	panelConfig := &panel.Config{}
	if err := config.Unmarshal(panelConfig); err != nil {
		log.Panicf("Parse config file %s failed: %s \n", *configFile, err)
	}
	p := panel.New(panelConfig)
	lastTime := time.Now()
	config.OnConfigChange(func(e fsnotify.Event) {
		// Discarding event received within a short period of time after receiving an event.
		if time.Now().After(lastTime.Add(3 * time.Second)) {
			// Hot reload function
			fmt.Println("Config file changed:", e.Name)
			p.Close()
			if err := config.Unmarshal(panelConfig); err != nil {
				log.Panicf("Parse config file %s failed: %s \n", *configFile, err)
			}
			p.Start()
			lastTime = time.Now()
		}
	})
	p.Start()
	defer p.Close()

	// Running backend
	{
		osSignals := make(chan os.Signal, 1)
		signal.Notify(osSignals, os.Interrupt, os.Kill, syscall.SIGTERM)
		<-osSignals
	}
}
