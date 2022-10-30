package panel

//go:generate go run github.com/v2fly/v2ray-core/v5/common/errors/errorgen

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/imdario/mergo"
	"github.com/r3labs/diff/v3"
	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/app/proxyman"
	"github.com/v2fly/v2ray-core/v5/app/stats"
	"github.com/v2fly/v2ray-core/v5/common/serial"
	"github.com/v2fly/v2ray-core/v5/infra/conf/synthetic/dns"
	"github.com/v2fly/v2ray-core/v5/infra/conf/synthetic/log"
	"github.com/v2fly/v2ray-core/v5/infra/conf/synthetic/router"
	conf "github.com/v2fly/v2ray-core/v5/infra/conf/v4"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/thank243/v2rayS/api"
	"github.com/thank243/v2rayS/api/v2board"
	"github.com/thank243/v2rayS/app/mydispatcher"
	_ "github.com/thank243/v2rayS/main/distro/all"
	"github.com/thank243/v2rayS/service"
	"github.com/thank243/v2rayS/service/controller"
)

// Panel Structure
type Panel struct {
	access      sync.Mutex
	panelConfig *Config
	Server      *core.Instance
	Service     []service.Service
	Running     bool
}

func New(panelConfig *Config) *Panel {
	p := &Panel{panelConfig: panelConfig}
	return p
}

func (p *Panel) loadCore(panelConfig *Config) *core.Instance {
	// Log Config
	coreLogConfig := &log.LogConfig{}
	logConfig := getDefaultLogConfig()
	if panelConfig.LogConfig != nil {
		if _, err := diff.Merge(logConfig, panelConfig.LogConfig, logConfig); err != nil {
			panic(fmt.Sprintf("Read Log config failed: %s", err))
		}
	}
	coreLogConfig.LogLevel = logConfig.Level
	coreLogConfig.AccessLog = logConfig.AccessPath
	coreLogConfig.ErrorLog = logConfig.ErrorPath

	// DNS config
	coreDnsConfig := &dns.DNSConfig{}
	if panelConfig.DnsConfigPath != "" {
		if data, err := os.ReadFile(panelConfig.DnsConfigPath); err != nil {
			panic(fmt.Sprintf("Failed to read DNS config file at: %s", panelConfig.DnsConfigPath))
		} else {
			if err = json.Unmarshal(data, coreDnsConfig); err != nil {
				panic(fmt.Sprintf("Failed to unmarshal DNS config: %s", panelConfig.DnsConfigPath))
			}
		}
	}
	dnsConfig, err := coreDnsConfig.Build()
	if err != nil {
		panic(fmt.Sprintf("Failed to understand DNS config, Please check: https://guide.v2fly.org/basics/dns.html for help: %s", err))
	}
	// Routing config
	coreRouterConfig := &router.RouterConfig{}
	if panelConfig.RouteConfigPath != "" {
		if data, err := os.ReadFile(panelConfig.RouteConfigPath); err != nil {
			panic(fmt.Sprintf("Failed to read Routing config file at: %s", panelConfig.RouteConfigPath))
		} else {
			if err = json.Unmarshal(data, coreRouterConfig); err != nil {
				panic(fmt.Sprintf("Failed to unmarshal Routing config: %s", panelConfig.RouteConfigPath))
			}
		}
	}
	routeConfig, err := coreRouterConfig.Build()
	if err != nil {
		panic(fmt.Sprintf("Failed to understand Routing config  Please check: https://guide.v2fly.org/basics/routing/basics_routing.html for help: %s", err))
	}
	// Custom Inbound config
	var coreCustomInboundConfig []conf.InboundDetourConfig
	if panelConfig.InboundConfigPath != "" {
		if data, err := os.ReadFile(panelConfig.InboundConfigPath); err != nil {
			panic(fmt.Sprintf("Failed to read Custom Inbound config file at: %s", panelConfig.OutboundConfigPath))
		} else {
			if err = json.Unmarshal(data, &coreCustomInboundConfig); err != nil {
				panic(fmt.Sprintf("Failed to unmarshal Custom Inbound config: %s", panelConfig.OutboundConfigPath))
			}
		}
	}
	var inBoundConfig []*core.InboundHandlerConfig
	for _, config := range coreCustomInboundConfig {
		oc, err := config.Build()
		if err != nil {
			panic(fmt.Sprintf("Failed to understand Inbound config, Please check: https://www.v2fly.org/v5/config/inbound.html for help: %s", err))
		}
		inBoundConfig = append(inBoundConfig, oc)
	}
	// Custom Outbound config
	var coreCustomOutboundConfig []conf.OutboundDetourConfig
	if panelConfig.OutboundConfigPath != "" {
		if data, err := os.ReadFile(panelConfig.OutboundConfigPath); err != nil {
			panic(fmt.Sprintf("Failed to read Custom Outbound config file at: %s", panelConfig.OutboundConfigPath))
		} else {
			if err = json.Unmarshal(data, &coreCustomOutboundConfig); err != nil {
				panic(fmt.Sprintf("Failed to unmarshal Custom Outbound config: %s", panelConfig.OutboundConfigPath))
			}
		}
	}
	var outBoundConfig []*core.OutboundHandlerConfig
	for _, config := range coreCustomOutboundConfig {
		oc, err := config.Build()
		if err != nil {
			panic(fmt.Sprintf("Failed to understand Outbound config, Please check: https://www.v2fly.org/v5/config/outbound.html for help: %s", err))
		}
		outBoundConfig = append(outBoundConfig, oc)
	}
	// Policy config
	levelPolicyConfig := parseConnectionConfig(panelConfig.ConnectionConfig)
	corePolicyConfig := &conf.PolicyConfig{}
	corePolicyConfig.Levels = map[uint32]*conf.Policy{0: levelPolicyConfig}
	policyConfig, _ := corePolicyConfig.Build()
	// Build Core Config
	config := &core.Config{
		App: []*anypb.Any{
			serial.ToTypedMessage(coreLogConfig.Build()),
			serial.ToTypedMessage(&mydispatcher.Config{}),
			serial.ToTypedMessage(&stats.Config{}),
			serial.ToTypedMessage(&proxyman.InboundConfig{}),
			serial.ToTypedMessage(&proxyman.OutboundConfig{}),
			serial.ToTypedMessage(policyConfig),
			serial.ToTypedMessage(dnsConfig),
			serial.ToTypedMessage(routeConfig),
		},
		Inbound:  inBoundConfig,
		Outbound: outBoundConfig,
	}
	server, err := core.New(config)
	if err != nil {
		panic(fmt.Sprintf("failed to create instance: %s", err))
	}
	newError(fmt.Sprintf("v2ray Core Version: %s", core.Version())).WriteToLog()

	return server
}

// Start the panel
func (p *Panel) Start() {
	p.access.Lock()
	defer p.access.Unlock()
	newError("Starting the panel").AtWarning().WriteToLog()
	// Load Core
	server := p.loadCore(p.panelConfig)
	if err := server.Start(); err != nil {
		panic(fmt.Sprintf("Failed to start instance: %s", err))
	}
	p.Server = server
	// Load Nodes config
	for _, nodeConfig := range p.panelConfig.NodesConfig {
		var apiClient api.API
		switch nodeConfig.PanelType {
		case "V2board":
			apiClient = v2board.New(nodeConfig.ApiConfig)
		default:
			panic(fmt.Sprintf("Unsupport panel type: %s", nodeConfig.PanelType))
		}
		var controllerService service.Service
		// Register controller service
		controllerConfig := getDefaultControllerConfig()
		if nodeConfig.ControllerConfig != nil {
			if err := mergo.Merge(controllerConfig, nodeConfig.ControllerConfig, mergo.WithOverride); err != nil {
				panic(fmt.Sprintf("Read Controller Config Failed"))
			}
		}
		controllerService = controller.New(server, apiClient, controllerConfig, nodeConfig.PanelType)
		p.Service = append(p.Service, controllerService)

	}

	// Start all the service
	for _, s := range p.Service {
		err := s.Start()
		if err != nil {
			panic(fmt.Sprintf("Panel Start fialed: %s", err))
		}
	}
	p.Running = true
	return
}

// Close the panel
func (p *Panel) Close() {
	p.access.Lock()
	defer p.access.Unlock()
	for _, s := range p.Service {
		err := s.Close()
		if err != nil {
			panic(fmt.Sprintf("Panel Close fialed: %s", err))
		}
	}
	p.Service = nil
	p.Server.Close()
	p.Running = false
	return
}

func parseConnectionConfig(c *ConnectionConfig) (policy *conf.Policy) {
	connectionConfig := getDefaultConnectionConfig()
	if c != nil {
		if _, err := diff.Merge(connectionConfig, c, connectionConfig); err != nil {
			panic(fmt.Sprintf("Read ConnectionConfig failed: %s", err))
		}
	}
	policy = &conf.Policy{
		StatsUserUplink:   true,
		StatsUserDownlink: true,
		Handshake:         &connectionConfig.Handshake,
		ConnectionIdle:    &connectionConfig.ConnIdle,
		UplinkOnly:        &connectionConfig.UplinkOnly,
		DownlinkOnly:      &connectionConfig.DownlinkOnly,
		BufferSize:        &connectionConfig.BufferSize,
	}

	return
}
