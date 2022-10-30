// Package controller Package generate the InboundConfig used by add inbound
package controller

import (
	"encoding/json"
	"fmt"

	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/infra/conf/cfgcommon"
	"github.com/v2fly/v2ray-core/v5/infra/conf/cfgcommon/sniffer"
	"github.com/v2fly/v2ray-core/v5/infra/conf/cfgcommon/socketcfg"
	"github.com/v2fly/v2ray-core/v5/infra/conf/cfgcommon/tlscfg"
	conf "github.com/v2fly/v2ray-core/v5/infra/conf/v4"

	"github.com/thank243/v2rayS/api"
	"github.com/thank243/v2rayS/common/legocmd"
)

// InboundBuilder build Inbound config for different protocol
func InboundBuilder(config *Config, nodeInfo *api.NodeInfo, tag string) (*core.InboundHandlerConfig, error) {
	inboundDetourConfig := &conf.InboundDetourConfig{}
	// Build Listen IP address
	if nodeInfo.NodeType == "Shadowsocks-Plugin" {
		// Shdowsocks listen in 127.0.0.1 for safety
		inboundDetourConfig.ListenOn = &cfgcommon.Address{Address: net.ParseAddress("127.0.0.1")}
	} else if config.ListenIP != "" {
		ipAddress := net.ParseAddress(config.ListenIP)
		inboundDetourConfig.ListenOn = &cfgcommon.Address{Address: ipAddress}
	}

	// Build Port
	inboundDetourConfig.PortRange = &cfgcommon.PortRange{From: nodeInfo.Port, To: nodeInfo.Port}
	// Build Tag
	inboundDetourConfig.Tag = tag
	// SniffingConfig
	sniffingConfig := &sniffer.SniffingConfig{
		Enabled:      true,
		DestOverride: &cfgcommon.StringList{"http", "tls"},
	}
	if config.DisableSniffing {
		sniffingConfig.Enabled = false
	}
	inboundDetourConfig.SniffingConfig = sniffingConfig

	var (
		protocol      string
		streamSetting *conf.StreamConfig
		setting       json.RawMessage
	)

	var proxySetting any
	// Build Protocol and Protocol setting
	switch nodeInfo.NodeType {
	case "V2ray":
		protocol = "vmess"
		proxySetting = &conf.VMessInboundConfig{}
	case "Trojan":
		protocol = "trojan"
		// Enable fallback
		if config.EnableFallback {
			fallbackConfigs, err := buildTrojanFallbacks(config.FallBackConfigs)
			if err == nil {
				proxySetting = struct {
					Fallbacks []*conf.TrojanInboundFallback `json:"fallbacks"`
				}{Fallbacks: fallbackConfigs}
			} else {
				return nil, err
			}
		}
	case "Shadowsocks", "Shadowsocks-Plugin":
		// v2ray official not support single port multi-user

		// protocol = "shadowsocks"
		// proxySetting = &conf.ShadowsocksServerConfig{}
		// randomPasswd := uuid.New()
		// defaultSSUser := &conf.ShadowsocksUserConfig{
		// 	Cipher:   "aes-128-gcm",
		// 	Password: randomPasswd.String(),
		// }
		// proxySetting, _ := proxySetting.(*conf.ShadowsocksServerConfig)
		// proxySetting.Users = append(proxySetting.Users, defaultSSUser)
		// proxySetting.NetworkList = &conf.NetworkList{"tcp", "udp"}
		// proxySetting.IVCheck = true
		// if config.DisableIVCheck {
		// 	proxySetting.IVCheck = false
		// }
		return nil, newError("not support Shadowsocks on single port multi-user")
	case "dokodemo-door":
		protocol = "dokodemo-door"
		proxySetting = struct {
			Host        string   `json:"address"`
			NetworkList []string `json:"network"`
		}{
			Host:        "v1.mux.cool",
			NetworkList: []string{"tcp", "udp"},
		}
	default:
		return nil, newError(fmt.Sprintf("unsupported node type: %s, Only support: V2ray, Trojan, Shadowsocks, and Shadowsocks-Plugin", nodeInfo.NodeType)).AtError()
	}

	setting, err := json.Marshal(proxySetting)
	if err != nil {
		return nil, fmt.Errorf("Marshal proxy %s config fialed: %s", nodeInfo.NodeType, err)
	}

	// Build streamSettings
	streamSetting = new(conf.StreamConfig)
	transportProtocol := conf.TransportProtocol(nodeInfo.TransportProtocol)
	networkType, err := transportProtocol.Build()
	if err != nil {
		return nil, newError(fmt.Sprintf("convert TransportProtocol failed: %s", err)).AtError()
	}

	switch networkType {
	case "tcp":
		streamSetting.TCPSettings = &conf.TCPConfig{
			AcceptProxyProtocol: config.EnableProxyProtocol,
			HeaderConfig:        nodeInfo.Header,
		}
	case "websocket":
		streamSetting.WSSettings = &conf.WebSocketConfig{
			AcceptProxyProtocol: config.EnableProxyProtocol,
			Path:                nodeInfo.Path,
			Headers:             map[string]string{"Host": nodeInfo.Host},
		}
	case "http":
		streamSetting.HTTPSettings = &conf.HTTPConfig{
			Host: &cfgcommon.StringList{nodeInfo.Host},
			Path: nodeInfo.Path,
		}
	case "grpc":
		streamSetting.GRPCSettings = &conf.GunConfig{ServiceName: nodeInfo.ServiceName}
	}

	streamSetting.Network = &transportProtocol
	// Build TLS settings
	if nodeInfo.EnableTLS && config.CertConfig.CertMode != "none" {
		streamSetting.Security = nodeInfo.TLSType
		certFile, keyFile, err := getCertFile(config.CertConfig)
		if err != nil {
			return nil, err
		}
		tlsSettings := &tlscfg.TLSConfig{
			VerifyClientCertificate: config.CertConfig.VerifyClientCertificate,
		}
		tlsSettings.Certs = append(tlsSettings.Certs, &tlscfg.TLSCertConfig{CertFile: certFile, KeyFile: keyFile})

		streamSetting.TLSSettings = tlsSettings

	}
	// Support ProxyProtocol for any transport protocol
	if networkType != "tcp" && networkType != "ws" && config.EnableProxyProtocol {
		sockoptConfig := &socketcfg.SocketConfig{
			AcceptProxyProtocol: config.EnableProxyProtocol,
		}
		streamSetting.SocketSettings = sockoptConfig
	}
	inboundDetourConfig.Protocol = protocol
	inboundDetourConfig.StreamSetting = streamSetting
	inboundDetourConfig.Settings = &setting

	return inboundDetourConfig.Build()
}

func getCertFile(certConfig *CertConfig) (certFile string, keyFile string, err error) {
	if certConfig.CertMode == "file" {
		if certConfig.CertFile == "" || certConfig.KeyFile == "" {
			return "", "", fmt.Errorf("cert file path or key file path not exist")
		}
		return certConfig.CertFile, certConfig.KeyFile, nil
	} else if certConfig.CertMode == "dns" {
		lego, err := legocmd.New()
		if err != nil {
			return "", "", err
		}
		certPath, keyPath, err := lego.DNSCert(certConfig.CertDomain, certConfig.Email, certConfig.Provider, certConfig.DNSEnv)
		if err != nil {
			return "", "", err
		}
		return certPath, keyPath, err
	} else if certConfig.CertMode == "http" {
		lego, err := legocmd.New()
		if err != nil {
			return "", "", err
		}
		certPath, keyPath, err := lego.HTTPCert(certConfig.CertDomain, certConfig.Email)
		if err != nil {
			return "", "", err
		}
		return certPath, keyPath, err
	}

	return "", "", fmt.Errorf("Unsupported certmode: %s", certConfig.CertMode)
}

func buildTrojanFallbacks(fallbackConfigs []*FallBackConfig) ([]*conf.TrojanInboundFallback, error) {
	if fallbackConfigs == nil {
		return nil, fmt.Errorf("you must provide FallBackConfigs")
	}

	trojanFallBacks := make([]*conf.TrojanInboundFallback, len(fallbackConfigs))
	for i, c := range fallbackConfigs {

		if c.Dest == "" {
			return nil, fmt.Errorf("Dest is required for fallback fialed")
		}

		var dest json.RawMessage
		dest, err := json.Marshal(c.Dest)
		if err != nil {
			return nil, fmt.Errorf("Marshal dest %s config fialed: %s", dest, err)
		}
		trojanFallBacks[i] = &conf.TrojanInboundFallback{
			Type: c.Type,
			Alpn: c.Alpn,
			Path: c.Path,
			Dest: dest,
			Xver: c.ProxyProtocolVer,
		}
	}
	return trojanFallBacks, nil
}
