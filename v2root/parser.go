package main

import "C"

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type OutputConfig struct {
	Inbounds  []InboundConfig  `json:"inbounds"`
	Outbounds []OutboundConfig `json:"outbounds"`
	DNS       *DNSConfig       `json:"dns,omitempty"`
	Routing   *RoutingConfig   `json:"routing,omitempty"`
}

type OutboundConfig struct {
	Protocol       string          `json:"protocol"`
	Settings       interface{}     `json:"settings"`
	StreamSettings *StreamSettings `json:"streamSettings,omitempty"`
	Remark         string          `json:"remark,omitempty"`
	Tag            string          `json:"tag,omitempty"`
}

type DNSConfig struct {
	Servers  []interface{}     `json:"servers"`
	Hosts    map[string]string `json:"hosts,omitempty"`
	ClientIP string            `json:"clientIp,omitempty"`
	Tag      string            `json:"tag,omitempty"`
	Strategy string            `json:"strategy,omitempty"`
}

type VlessSettings struct {
	Vnext []VlessVnext `json:"vnext"`
}

type TrojanSettings struct {
	Servers []TrojanServer `json:"servers"`
}

type VmessSettings struct {
	Vnext []VmessVnext `json:"vnext"`
}

type ShadowsocksSettings struct {
	Servers []ShadowsocksServer `json:"servers"`
}

type InboundConfig struct {
	Port     uint16                 `json:"port,omitempty"`
	Protocol string                 `json:"protocol,omitempty"`
	Settings map[string]interface{} `json:"settings,omitempty"`
	Tag      string                 `json:"tag,omitempty"`
	Type     string                 `json:"type,omitempty"`
	Sniffing *SniffingSettings      `json:"sniffing,omitempty"`
}

type SniffingSettings struct {
	Enabled      bool     `json:"enabled"`
	DestOverride []string `json:"destOverride,omitempty"`
}

type StreamSettings struct {
	Network         string           `json:"network"`
	Security        string           `json:"security"`
	TLSSettings     *TLSSettings     `json:"tlsSettings,omitempty"`
	RealitySettings *RealitySettings `json:"realitySettings,omitempty"`
	TCPSettings     *TCPSettings     `json:"tcpSettings,omitempty"`
	WSSettings      *WSSettings      `json:"wsSettings,omitempty"`
	HTTPSettings    *HTTPSettings    `json:"httpSettings,omitempty"`
	QUICSettings    *QUICSettings    `json:"quicSettings,omitempty"`
	KCPSettings     *KCPSettings     `json:"kcpSettings,omitempty"`
	GRPCSettings    *GRPCSettings    `json:"grpcSettings,omitempty"`
}

type VlessVnext struct {
	Address string      `json:"address"`
	Port    uint16      `json:"port"`
	Users   []VlessUser `json:"users"`
}

type VlessUser struct {
	ID         string `json:"id"`
	Encryption string `json:"encryption"`
	Flow       string `json:"flow,omitempty"`
}

type TrojanServer struct {
	Address  string `json:"address"`
	Port     uint16 `json:"port"`
	Password string `json:"password"`
	Flow     string `json:"flow,omitempty"`
}

type VmessVnext struct {
	Address string      `json:"address"`
	Port    uint16      `json:"port"`
	Users   []VmessUser `json:"users"`
}

type VmessUser struct {
	ID       string `json:"id"`
	AlterID  uint16 `json:"alterId"`
	Security string `json:"security"`
}

type ShadowsocksServer struct {
	Address  string `json:"address"`
	Port     uint16 `json:"port"`
	Method   string `json:"method"`
	Password string `json:"password"`
}

type RealitySettings struct {
	SNI         string   `json:"serverName,omitempty"`
	Fingerprint string   `json:"fingerprint,omitempty"`
	PublicKey   string   `json:"publicKey"`
	ShortID     string   `json:"shortId,omitempty"`
	SpiderX     string   `json:"spiderX,omitempty"`
	ServerNames []string `json:"-"` // Not used in output
	ALPN        string   `json:"-"` // Not used in output
	Xver        string   `json:"-"` // Not used in output
}

type TLSSettings struct {
	SNI           string   `json:"serverName,omitempty"`
	ALPN          []string `json:"alpn,omitempty"`
	Fingerprint   string   `json:"fingerprint,omitempty"`
	AllowInsecure bool     `json:"allowInsecure,omitempty"`
}

type TCPSettings struct {
	Header *HeaderConfig `json:"header,omitempty"`
}

type HeaderConfig struct {
	Type    string         `json:"type"`
	Request *RequestConfig `json:"request,omitempty"`
}

type RequestConfig struct {
	Path    []string            `json:"path"`
	Headers map[string][]string `json:"headers"`
}

type WSSettings struct {
	Path    string            `json:"path,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

type HTTPSettings struct {
	Path string   `json:"path,omitempty"`
	Host []string `json:"host,omitempty"`
}

type QUICSettings struct {
	Security string        `json:"security,omitempty"`
	Key      string        `json:"key,omitempty"`
	Header   *HeaderConfig `json:"header,omitempty"`
}

type KCPSettings struct {
	Seed   string        `json:"seed,omitempty"`
	Header *HeaderConfig `json:"header,omitempty"`
}

type GRPCSettings struct {
	ServiceName string `json:"serviceName,omitempty"`
	MultiMode   bool   `json:"multiMode,omitempty"`
}

type RoutingConfig struct {
	DomainStrategy string        `json:"domainStrategy,omitempty"`
	Rules          []RoutingRule `json:"rules"`
}

type RoutingRule struct {
	Type        string   `json:"type"`
	Domain      []string `json:"domain,omitempty"`
	Ip          []string `json:"ip,omitempty"`
	Port        string   `json:"port,omitempty"` // Added port support for catch-all
	InboundTag  []string `json:"inboundTag,omitempty"`
	OutboundTag string   `json:"outboundTag"`
}

type ParseOptions struct {
	GeositePath   string        `json:"geositePath"`
	GeositeFile   string        `json:"geositeFile"`
	DNSConfig     *DNSConfig    `json:"dnsConfig,omitempty"`
	GeositeDomain string        `json:"geositeDomain,omitempty"`
	GeositeDNS    string        `json:"geositeDNS,omitempty"`
	HTTPPort      int           `json:"httpPort,omitempty"`
	SOCKSPort     int           `json:"socksPort,omitempty"`
	RoutingMode   string        `json:"routingMode,omitempty"`
	URI           string        `json:"uri"`
	GeositeRules  []GeositeRule `json:"geositeRules,omitempty"`
	GeoipRules    []GeoipRule   `json:"geoipRules,omitempty"`
}

type GeositeRule struct {
	Domain      string `json:"domain"`
	Action      string `json:"action"`
	OutboundTag string `json:"outboundTag,omitempty"`
}

type GeoipRule struct {
	Country     string `json:"country"`
	Action      string `json:"action"`
	OutboundTag string `json:"outboundTag,omitempty"`
}

type GeositeDNSRule struct {
	Domain string
	DNS    string
}

// validateOutbound checks required fields for common protocols and returns error if missing.
func validateOutbound(ob OutboundConfig) error {
	switch ob.Protocol {
	case "vless", "vmess", "trojan":
		settingsBytes, _ := json.Marshal(ob.Settings)
		if ob.Protocol == "vless" {
			var settings VlessSettings
			if err := json.Unmarshal(settingsBytes, &settings); err != nil {
				return fmt.Errorf("invalid %s settings format", ob.Protocol)
			}
			if len(settings.Vnext) == 0 || len(settings.Vnext[0].Users) == 0 {
				return fmt.Errorf("%s: missing vnext or users", ob.Protocol)
			}
			vnext := settings.Vnext[0]
			if vnext.Address == "" {
				return fmt.Errorf("%s: missing address", ob.Protocol)
			}
			if vnext.Port == 0 {
				return fmt.Errorf("%s: missing or invalid port", ob.Protocol)
			}
			if settings.Vnext[0].Users[0].ID == "" {
				return fmt.Errorf("%s: missing user id", ob.Protocol)
			}
		} else if ob.Protocol == "trojan" {
			var settings TrojanSettings
			if err := json.Unmarshal(settingsBytes, &settings); err != nil {
				return fmt.Errorf("invalid %s settings format", ob.Protocol)
			}
			if len(settings.Servers) == 0 {
				return fmt.Errorf("%s: missing servers", ob.Protocol)
			}
			server := settings.Servers[0]
			if server.Address == "" {
				return fmt.Errorf("%s: missing address", ob.Protocol)
			}
			if server.Port == 0 {
				return fmt.Errorf("%s: missing or invalid port", ob.Protocol)
			}
			if server.Password == "" {
				return fmt.Errorf("%s: missing password", ob.Protocol)
			}
		} else if ob.Protocol == "vmess" {
			var settings VmessSettings
			if err := json.Unmarshal(settingsBytes, &settings); err != nil {
				return fmt.Errorf("invalid %s settings format", ob.Protocol)
			}
			if len(settings.Vnext) == 0 || len(settings.Vnext[0].Users) == 0 {
				return fmt.Errorf("%s: missing vnext or users", ob.Protocol)
			}
			vnext := settings.Vnext[0]
			if vnext.Address == "" {
				return fmt.Errorf("%s: missing address", ob.Protocol)
			}
			if vnext.Port == 0 {
				return fmt.Errorf("%s: missing or invalid port", ob.Protocol)
			}
			if settings.Vnext[0].Users[0].ID == "" {
				return fmt.Errorf("%s: missing user id", ob.Protocol)
			}
		}
	case "shadowsocks":
		settingsBytes, _ := json.Marshal(ob.Settings)
		var settings ShadowsocksSettings
		if err := json.Unmarshal(settingsBytes, &settings); err != nil {
			return fmt.Errorf("invalid shadowsocks settings format")
		}
		if len(settings.Servers) == 0 {
			return fmt.Errorf("shadowsocks: missing servers")
		}
		server := settings.Servers[0]
		if server.Address == "" {
			return fmt.Errorf("shadowsocks: missing address")
		}
		if server.Port == 0 {
			return fmt.Errorf("shadowsocks: missing or invalid port")
		}
		if server.Method == "" || server.Password == "" {
			return fmt.Errorf("shadowsocks: missing method or password")
		}
	}
	return nil
}

func getOutboundTag(action, customTag string) string {
	if customTag != "" {
		return customTag
	}
	switch strings.ToLower(action) {
	case "block", "reject":
		return "Reject"
	case "direct":
		return "Direct"
	case "proxy":
		return "Proxy"
	default:
		return "Proxy"
	}
}

func addGeositeRuleIfExists(dnsConfig *DNSConfig, geositePath string, rule *GeositeDNSRule) {
	if rule == nil || rule.Domain == "" || rule.DNS == "" {
		return
	}
	if geositePath != "" {
		if _, err := os.Stat(geositePath); err == nil {
			dnsRule := map[string]interface{}{
				"address": rule.DNS,
				"domains": []string{"geosite:" + rule.Domain},
			}
			exists := false
			for _, s := range dnsConfig.Servers {
				if m, ok := s.(map[string]interface{}); ok {
					if m["address"] == rule.DNS && fmt.Sprintf("%v", m["domains"]) == fmt.Sprintf("%v", dnsRule["domains"]) {
						exists = true
						break
					}
				}
			}
			if !exists {
				dnsConfig.Servers = append([]interface{}{dnsRule}, dnsConfig.Servers...)
			}
		}
	}
}

func createDefaultInbounds(httpPort, socksPort uint16) []InboundConfig {
	sniffing := &SniffingSettings{
		Enabled:      true,
		DestOverride: []string{"http", "tls", "quic"},
	}

	return []InboundConfig{
		{
			Port:     httpPort,
			Protocol: "http",
			Settings: map[string]interface{}{},
			Sniffing: sniffing,
		},
		{
			Port:     socksPort,
			Protocol: "socks",
			Settings: map[string]interface{}{"udp": true},
			Sniffing: sniffing,
		},
	}
}

func buildRoutingConfig(geositeRules []GeositeRule, geoipRules []GeoipRule, defaultAction string) *RoutingConfig {
	rules := []RoutingRule{}

	for _, gr := range geositeRules {
		domain := gr.Domain
		if predefined, exists := predefinedGeositeCategories[domain]; exists {
			domain = predefined
		}
		rules = append(rules, RoutingRule{
			Type:        "field",
			Domain:      []string{"geosite:" + domain},
			OutboundTag: getOutboundTag(gr.Action, gr.OutboundTag),
		})
	}

	for _, gr := range geoipRules {
		rules = append(rules, RoutingRule{
			Type:        "field",
			Ip:          []string{"geoip:" + gr.Country},
			OutboundTag: getOutboundTag(gr.Action, gr.OutboundTag),
		})
	}

	// Default Fallback Rule
	if defaultAction != "" {
		rules = append(rules, RoutingRule{
			Type:        "field",
			Port:        "0-65535",
			OutboundTag: getOutboundTag(defaultAction, ""),
		})
	}

	return &RoutingConfig{
		DomainStrategy: "AsIs",
		Rules:          rules,
	}
}

//export Parse
func Parse(optionsJSON *C.char) *C.char {
	var opts ParseOptions
	if err := json.Unmarshal([]byte(C.GoString(optionsJSON)), &opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error unmarshaling options: %v\n", err)
		return C.CString("")
	}
	if opts.GeositeFile != "" {
		if _, err := os.Stat(opts.GeositeFile); err == nil {
			dir := opts.GeositeFile
			if idx := strings.LastIndexAny(opts.GeositeFile, "/\\"); idx != -1 {
				dir = opts.GeositeFile[:idx]
			}
			_ = os.Setenv("XRAY_LOCATION_ASSET", dir)
			opts.GeositePath = opts.GeositeFile
		}
	} else if opts.GeositePath != "" {
		if _, err := os.Stat(opts.GeositePath); err == nil {
			dir := opts.GeositePath
			if idx := strings.LastIndexAny(opts.GeositePath, "/\\"); idx != -1 {
				dir = opts.GeositePath[:idx]
			}
			_ = os.Setenv("XRAY_LOCATION_ASSET", dir)
		}
	}
	var geositeDNSRule *GeositeDNSRule
	if opts.GeositeDomain != "" && opts.GeositeDNS != "" {
		geositeDNSRule = &GeositeDNSRule{
			Domain: opts.GeositeDomain,
			DNS:    opts.GeositeDNS,
		}
	}
	httpPort := uint16(10809)
	socksPort := uint16(10808)
	if opts.HTTPPort > 0 {
		httpPort = uint16(opts.HTTPPort)
	}
	if opts.SOCKSPort > 0 {
		socksPort = uint16(opts.SOCKSPort)
	}

	routingMode := opts.RoutingMode
	if routingMode == "" {
		routingMode = "proxy"
	}

	var config *OutputConfig
	var err error
	if strings.HasPrefix(opts.URI, "vless://") {
		config, err = parseVless(opts.URI, httpPort, socksPort, opts.GeositePath, opts.DNSConfig, geositeDNSRule, opts.GeositeRules, opts.GeoipRules, routingMode)
	} else if strings.HasPrefix(opts.URI, "vmess://") {
		config, err = parseVmess(opts.URI, httpPort, socksPort, opts.GeositePath, opts.DNSConfig, geositeDNSRule, opts.GeositeRules, opts.GeoipRules, routingMode)
	} else if strings.HasPrefix(opts.URI, "trojan://") {
		config, err = parseTrojan(opts.URI, httpPort, socksPort, opts.GeositePath, opts.DNSConfig, geositeDNSRule, opts.GeositeRules, opts.GeoipRules, routingMode)
	} else if strings.HasPrefix(opts.URI, "ss://") {
		config, err = parseShadowsocks(opts.URI, httpPort, socksPort, opts.GeositePath, opts.DNSConfig, geositeDNSRule, opts.GeositeRules, opts.GeoipRules, routingMode)
	} else {
		return C.CString("")
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing URI '%s': %v\n", opts.URI, err)
		return C.CString("")
	}
	// Validate outbounds
	for _, ob := range config.Outbounds {
		if err := validateOutbound(ob); err != nil {
			fmt.Fprintf(os.Stderr, "Validation error: %v\n", err)
			return C.CString("")
		}
	}
	b, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling config: %v\n", err)
		return C.CString("")
	}

	// Debug: print config
	// fmt.Fprintf(os.Stderr, "[DEBUG] Generated Config:\n%s\n", string(b))

	return C.CString(string(b))
}

//export ParseVless
func ParseVless(optionsJSON *C.char) *C.char {
	var opts ParseOptions
	if err := json.Unmarshal([]byte(C.GoString(optionsJSON)), &opts); err != nil {
		return C.CString("")
	}
	if opts.GeositeFile != "" {
		if _, err := os.Stat(opts.GeositeFile); err == nil {
			dir := opts.GeositeFile
			if idx := strings.LastIndexAny(opts.GeositeFile, "/\\"); idx != -1 {
				dir = opts.GeositeFile[:idx]
			}
			_ = os.Setenv("XRAY_LOCATION_ASSET", dir)
			opts.GeositePath = opts.GeositeFile
		}
	} else if opts.GeositePath != "" {
		if _, err := os.Stat(opts.GeositePath); err == nil {
			dir := opts.GeositePath
			if idx := strings.LastIndexAny(opts.GeositePath, "/\\"); idx != -1 {
				dir = opts.GeositePath[:idx]
			}
			_ = os.Setenv("XRAY_LOCATION_ASSET", dir)
		}
	}
	var geositeDNSRule *GeositeDNSRule
	if opts.GeositeDomain != "" && opts.GeositeDNS != "" {
		geositeDNSRule = &GeositeDNSRule{
			Domain: opts.GeositeDomain,
			DNS:    opts.GeositeDNS,
		}
	}
	httpPort := uint16(10809)
	socksPort := uint16(10808)
	if opts.HTTPPort > 0 {
		httpPort = uint16(opts.HTTPPort)
	}
	if opts.SOCKSPort > 0 {
		socksPort = uint16(opts.SOCKSPort)
	}
	config, err := parseVless(opts.URI, httpPort, socksPort, opts.GeositePath, opts.DNSConfig, geositeDNSRule, opts.GeositeRules, opts.GeoipRules, opts.RoutingMode)
	if err != nil {
		return C.CString("")
	}
	b, _ := json.Marshal(config)
	return C.CString(string(b))
}

//export ParseTrojan
func ParseTrojan(optionsJSON *C.char) *C.char {
	var opts ParseOptions
	if err := json.Unmarshal([]byte(C.GoString(optionsJSON)), &opts); err != nil {
		return C.CString("")
	}
	if opts.GeositeFile != "" {
		if _, err := os.Stat(opts.GeositeFile); err == nil {
			dir := opts.GeositeFile
			if idx := strings.LastIndexAny(opts.GeositeFile, "/\\"); idx != -1 {
				dir = opts.GeositeFile[:idx]
			}
			_ = os.Setenv("XRAY_LOCATION_ASSET", dir)
			opts.GeositePath = opts.GeositeFile
		}
	} else if opts.GeositePath != "" {
		if _, err := os.Stat(opts.GeositePath); err == nil {
			dir := opts.GeositePath
			if idx := strings.LastIndexAny(opts.GeositePath, "/\\"); idx != -1 {
				dir = opts.GeositePath[:idx]
			}
			_ = os.Setenv("XRAY_LOCATION_ASSET", dir)
		}
	}
	var geositeDNSRule *GeositeDNSRule
	if opts.GeositeDomain != "" && opts.GeositeDNS != "" {
		geositeDNSRule = &GeositeDNSRule{
			Domain: opts.GeositeDomain,
			DNS:    opts.GeositeDNS,
		}
	}
	httpPort := uint16(10809)
	socksPort := uint16(10808)
	if opts.HTTPPort > 0 {
		httpPort = uint16(opts.HTTPPort)
	}
	if opts.SOCKSPort > 0 {
		socksPort = uint16(opts.SOCKSPort)
	}
	config, err := parseTrojan(opts.URI, httpPort, socksPort, opts.GeositePath, opts.DNSConfig, geositeDNSRule, opts.GeositeRules, opts.GeoipRules, opts.RoutingMode)
	if err != nil {
		return C.CString("")
	}
	b, _ := json.Marshal(config)
	return C.CString(string(b))
}

//export ParseVmess
func ParseVmess(optionsJSON *C.char) *C.char {
	var opts ParseOptions
	if err := json.Unmarshal([]byte(C.GoString(optionsJSON)), &opts); err != nil {
		return C.CString("")
	}
	if opts.GeositeFile != "" {
		if _, err := os.Stat(opts.GeositeFile); err == nil {
			dir := opts.GeositeFile
			if idx := strings.LastIndexAny(opts.GeositeFile, "/\\"); idx != -1 {
				dir = opts.GeositeFile[:idx]
			}
			_ = os.Setenv("XRAY_LOCATION_ASSET", dir)
			opts.GeositePath = opts.GeositeFile
		}
	} else if opts.GeositePath != "" {
		if _, err := os.Stat(opts.GeositePath); err == nil {
			dir := opts.GeositePath
			if idx := strings.LastIndexAny(opts.GeositePath, "/\\"); idx != -1 {
				dir = opts.GeositePath[:idx]
			}
			_ = os.Setenv("XRAY_LOCATION_ASSET", dir)
		}
	}
	var geositeDNSRule *GeositeDNSRule
	if opts.GeositeDomain != "" && opts.GeositeDNS != "" {
		geositeDNSRule = &GeositeDNSRule{
			Domain: opts.GeositeDomain,
			DNS:    opts.GeositeDNS,
		}
	}
	httpPort := uint16(10809)
	socksPort := uint16(10808)
	if opts.HTTPPort > 0 {
		httpPort = uint16(opts.HTTPPort)
	}
	if opts.SOCKSPort > 0 {
		socksPort = uint16(opts.SOCKSPort)
	}
	config, err := parseVmess(opts.URI, httpPort, socksPort, opts.GeositePath, opts.DNSConfig, geositeDNSRule, opts.GeositeRules, opts.GeoipRules, opts.RoutingMode)
	if err != nil {
		return C.CString("")
	}
	b, _ := json.Marshal(config)
	return C.CString(string(b))
}

//export ParseShadowsocks
func ParseShadowsocks(optionsJSON *C.char) *C.char {
	var opts ParseOptions
	if err := json.Unmarshal([]byte(C.GoString(optionsJSON)), &opts); err != nil {
		return C.CString("")
	}
	if opts.GeositeFile != "" {
		if _, err := os.Stat(opts.GeositeFile); err == nil {
			dir := opts.GeositeFile
			if idx := strings.LastIndexAny(opts.GeositeFile, "/\\"); idx != -1 {
				dir = opts.GeositeFile[:idx]
			}
			_ = os.Setenv("XRAY_LOCATION_ASSET", dir)
			opts.GeositePath = opts.GeositeFile
		}
	} else if opts.GeositePath != "" {
		if _, err := os.Stat(opts.GeositePath); err == nil {
			dir := opts.GeositePath
			if idx := strings.LastIndexAny(opts.GeositePath, "/\\"); idx != -1 {
				dir = opts.GeositePath[:idx]
			}
			_ = os.Setenv("XRAY_LOCATION_ASSET", dir)
		}
	}
	var geositeDNSRule *GeositeDNSRule
	if opts.GeositeDomain != "" && opts.GeositeDNS != "" {
		geositeDNSRule = &GeositeDNSRule{
			Domain: opts.GeositeDomain,
			DNS:    opts.GeositeDNS,
		}
	}
	httpPort := uint16(10809)
	socksPort := uint16(10808)
	if opts.HTTPPort > 0 {
		httpPort = uint16(opts.HTTPPort)
	}
	if opts.SOCKSPort > 0 {
		socksPort = uint16(opts.SOCKSPort)
	}
	config, err := parseShadowsocks(opts.URI, httpPort, socksPort, opts.GeositePath, opts.DNSConfig, geositeDNSRule, opts.GeositeRules, opts.GeoipRules, opts.RoutingMode)
	if err != nil {
		return C.CString("")
	}
	b, _ := json.Marshal(config)
	return C.CString(string(b))
}

//export JSONToConfigString
func JSONToConfigString(configJSON *C.char) *C.char {
	configStr := C.GoString(configJSON)
	if configStr == "" {
		return C.CString("")
	}

	// --- BEGIN DNSConfig to DNS Migration ---
	// Unmarshal to a generic map first to apply the dnsConfig -> dns fix.
	// This ensures consistency if a full config with "dnsConfig" is passed here.
	var configMap map[string]interface{}
	if err := json.Unmarshal([]byte(configStr), &configMap); err != nil {
		// If it's not a valid map, it might be invalid JSON.
		return C.CString("")
	}

	dnsConfigRaw, dnsConfigExists := configMap["dnsConfig"]
	dnsRaw, dnsExists := configMap["dns"]

	if dnsConfigExists {
		if !dnsExists {
			// Case 1: Only "dnsConfig" exists. Rename it to "dns".
			configMap["dns"] = dnsConfigRaw
			delete(configMap, "dnsConfig")
		} else {
			// Case 2: Both "dns" and "dnsConfig" exist. Merge them.
			dnsConfigMap, okConfig := dnsConfigRaw.(map[string]interface{})
			dnsMap, ok := dnsRaw.(map[string]interface{})

			if ok && okConfig {
				// Both are valid maps. Merge "dnsConfig" into "dns", with "dnsConfig" values taking priority.
				for key, value := range dnsConfigMap {
					dnsMap[key] = value
				}
				delete(configMap, "dnsConfig")
			} else {
				// One is not a map. Overwrite "dns" with "dnsConfig" to respect priority.
				configMap["dns"] = dnsConfigRaw
				delete(configMap, "dnsConfig")
			}
		}

		// Re-marshal the string with the fix applied.
		fixedConfigBytes, err := json.Marshal(configMap)
		if err != nil {
			return C.CString("") // Should not happen, but good to check.
		}
		configStr = string(fixedConfigBytes)
	}
	// --- END DNSConfig to DNS Migration ---

	var config OutputConfig
	if err := json.Unmarshal([]byte(configStr), &config); err != nil {
		// If unmarshal to OutputConfig fails, re-check the map for outbounds (original logic).
		// Note: configMap is already populated from above.
		outboundsRaw, ok := configMap["outbounds"]
		if !ok {
			return C.CString("")
		}
		outboundsBytes, _ := json.Marshal(outboundsRaw)
		var outbounds []OutboundConfig
		if err := json.Unmarshal(outboundsBytes, &outbounds); err != nil {
			return C.CString("")
		}
		config.Outbounds = outbounds
	}

	if len(config.Outbounds) == 0 {
		return C.CString("")
	}

	var firstOutbound OutboundConfig
	for _, ob := range config.Outbounds {
		if ob.Tag == "Proxy" || (ob.Tag != "Direct" && ob.Tag != "Reject") {
			firstOutbound = ob
			break
		}
	}
	if firstOutbound.Protocol == "" {
		firstOutbound = config.Outbounds[0]
	}
	if firstOutbound.StreamSettings == nil {
		firstOutbound.StreamSettings = &StreamSettings{
			Network:  "tcp",
			Security: "none",
		}
	}

	var uri string
	var err error

	switch firstOutbound.Protocol {
	case "vless":
		uri, err = buildVlessURI(firstOutbound)
	case "trojan":
		uri, err = buildTrojanURI(firstOutbound)
	case "vmess":
		uri, err = buildVmessURI(firstOutbound)
	case "shadowsocks":
		uri, err = buildShadowsocksURI(firstOutbound)
	default:
		return C.CString("")
	}

	if err != nil {
		return C.CString("")
	}

	return C.CString(uri)
}

func buildVlessURI(ob OutboundConfig) (string, error) {
	settingsBytes, _ := json.Marshal(ob.Settings)
	var settings VlessSettings
	if err := json.Unmarshal(settingsBytes, &settings); err != nil {
		return "", fmt.Errorf("invalid vless settings format")
	}

	if len(settings.Vnext) == 0 || len(settings.Vnext[0].Users) == 0 {
		return "", fmt.Errorf("invalid vless settings content")
	}
	vnext := settings.Vnext[0]
	user := vnext.Users[0]
	uri := fmt.Sprintf("vless://%s@%s:%d", user.ID, vnext.Address, vnext.Port)
	query := url.Values{}

	if user.Encryption != "" && user.Encryption != "none" {
		query.Set("encryption", user.Encryption)
	}
	if user.Flow != "" {
		query.Set("flow", user.Flow)
	}
	if ob.StreamSettings.Network != "" && ob.StreamSettings.Network != "tcp" {
		query.Set("type", ob.StreamSettings.Network)
	}
	if ob.StreamSettings.Security != "" && ob.StreamSettings.Security != "none" {
		query.Set("security", ob.StreamSettings.Security)
	}

	if ob.StreamSettings.TCPSettings != nil && ob.StreamSettings.TCPSettings.Header != nil {
		header := ob.StreamSettings.TCPSettings.Header
		query.Set("headerType", header.Type)
		if header.Request != nil {
			if len(header.Request.Path) > 0 {
				query.Set("path", strings.Join(header.Request.Path, ","))
			}
			if hosts, ok := header.Request.Headers["Host"]; ok && len(hosts) > 0 {
				query.Set("host", hosts[0])
			}
		}
	}

	if ob.StreamSettings.WSSettings != nil {
		if ob.StreamSettings.WSSettings.Path != "" {
			query.Set("path", ob.StreamSettings.WSSettings.Path)
		}
		if ob.StreamSettings.WSSettings.Headers != nil {
			if host, ok := ob.StreamSettings.WSSettings.Headers["Host"]; ok && host != "" {
				query.Set("host", host)
			}
		}
	}
	if ob.StreamSettings.HTTPSettings != nil {
		if ob.StreamSettings.HTTPSettings.Path != "" {
			query.Set("path", ob.StreamSettings.HTTPSettings.Path)
		}
		if len(ob.StreamSettings.HTTPSettings.Host) > 0 {
			query.Set("host", strings.Join(ob.StreamSettings.HTTPSettings.Host, ","))
		}
	}
	if ob.StreamSettings.QUICSettings != nil {
		if ob.StreamSettings.QUICSettings.Security != "" {
			query.Set("quicSecurity", ob.StreamSettings.QUICSettings.Security)
		}
		if ob.StreamSettings.QUICSettings.Key != "" {
			query.Set("key", ob.StreamSettings.QUICSettings.Key)
		}
	}
	if ob.StreamSettings.KCPSettings != nil {
		if ob.StreamSettings.KCPSettings.Seed != "" {
			query.Set("seed", ob.StreamSettings.KCPSettings.Seed)
		}
		if ob.StreamSettings.KCPSettings.Header != nil && ob.StreamSettings.KCPSettings.Header.Type != "" {
			query.Set("headerType", ob.StreamSettings.KCPSettings.Header.Type)
		}
	}
	if ob.StreamSettings.GRPCSettings != nil {
		if ob.StreamSettings.GRPCSettings.ServiceName != "" {
			query.Set("serviceName", ob.StreamSettings.GRPCSettings.ServiceName)
		}
		if ob.StreamSettings.GRPCSettings.MultiMode {
			query.Set("mode", "multi")
		}
	}

	if ob.StreamSettings.TLSSettings != nil {
		if ob.StreamSettings.TLSSettings.SNI != "" {
			query.Set("sni", ob.StreamSettings.TLSSettings.SNI)
		}
		if len(ob.StreamSettings.TLSSettings.ALPN) > 0 {
			query.Set("alpn", strings.Join(ob.StreamSettings.TLSSettings.ALPN, ","))
		}
		if ob.StreamSettings.TLSSettings.Fingerprint != "" {
			query.Set("fp", ob.StreamSettings.TLSSettings.Fingerprint)
		}
	}

	if ob.StreamSettings.Security == "reality" && ob.StreamSettings.RealitySettings != nil {
		rs := ob.StreamSettings.RealitySettings
		if rs.SNI != "" {
			query.Set("sni", rs.SNI)
		}
		if rs.Fingerprint != "" {
			query.Set("fp", rs.Fingerprint)
		}
		if rs.PublicKey != "" {
			query.Set("pbk", rs.PublicKey)
		}
		if rs.ShortID != "" {
			query.Set("sid", rs.ShortID)
		}
		if rs.SpiderX != "" {
			query.Set("spx", rs.SpiderX)
		}
	}

	if len(query) > 0 {
		uri += "?" + query.Encode()
	}
	if ob.Remark != "" {
		uri += "#" + url.PathEscape(ob.Remark)
	}

	return uri, nil
}

func buildTrojanURI(ob OutboundConfig) (string, error) {
	settingsBytes, _ := json.Marshal(ob.Settings)
	var settings TrojanSettings
	if err := json.Unmarshal(settingsBytes, &settings); err != nil {
		return "", fmt.Errorf("invalid trojan settings format")
	}

	if len(settings.Servers) == 0 {
		return "", fmt.Errorf("invalid trojan settings")
	}
	server := settings.Servers[0]
	uri := fmt.Sprintf("trojan://%s@%s:%d", server.Password, server.Address, server.Port)
	query := url.Values{}
	if server.Flow != "" {
		query.Set("flow", server.Flow)
	}
	if ob.StreamSettings.Network != "" && ob.StreamSettings.Network != "tcp" {
		query.Set("type", ob.StreamSettings.Network)
	}
	if ob.StreamSettings.Security != "" {
		query.Set("security", ob.StreamSettings.Security)
	}
	if ob.StreamSettings.TLSSettings != nil {
		if ob.StreamSettings.TLSSettings.SNI != "" {
			query.Set("sni", ob.StreamSettings.TLSSettings.SNI)
		}
		if len(ob.StreamSettings.TLSSettings.ALPN) > 0 {
			query.Set("alpn", strings.Join(ob.StreamSettings.TLSSettings.ALPN, ","))
		}
		if ob.StreamSettings.TLSSettings.Fingerprint != "" {
			query.Set("fp", ob.StreamSettings.TLSSettings.Fingerprint)
		}
	}

	if ob.StreamSettings.Security == "reality" && ob.StreamSettings.RealitySettings != nil {
		rs := ob.StreamSettings.RealitySettings
		if rs.SNI != "" {
			query.Set("sni", rs.SNI)
		}
		if rs.Fingerprint != "" {
			query.Set("fp", rs.Fingerprint)
		}
		if rs.PublicKey != "" {
			query.Set("pbk", rs.PublicKey)
		}
		if rs.ShortID != "" {
			query.Set("sid", rs.ShortID)
		}
		if rs.SpiderX != "" {
			query.Set("spx", rs.SpiderX)
		}
	}

	if ob.StreamSettings.TCPSettings != nil && ob.StreamSettings.TCPSettings.Header != nil {
		header := ob.StreamSettings.TCPSettings.Header
		query.Set("headerType", header.Type)
		if header.Request != nil {
			if len(header.Request.Path) > 0 {
				query.Set("path", strings.Join(header.Request.Path, ","))
			}
			if hosts, ok := header.Request.Headers["Host"]; ok && len(hosts) > 0 {
				query.Set("host", hosts[0])
			}
		}
	}

	switch ob.StreamSettings.Network {
	case "ws":
		if ob.StreamSettings.WSSettings != nil {
			if ob.StreamSettings.WSSettings.Path != "" {
				query.Set("path", ob.StreamSettings.WSSettings.Path)
			}
			if host, ok := ob.StreamSettings.WSSettings.Headers["Host"]; ok && host != "" {
				query.Set("host", host)
			}
		}
	case "http":
		if ob.StreamSettings.HTTPSettings != nil {
			if ob.StreamSettings.HTTPSettings.Path != "" {
				query.Set("path", ob.StreamSettings.HTTPSettings.Path)
			}
			if len(ob.StreamSettings.HTTPSettings.Host) > 0 {
				query.Set("host", strings.Join(ob.StreamSettings.HTTPSettings.Host, ","))
			}
		}
	case "quic":
		if ob.StreamSettings.QUICSettings != nil {
			if ob.StreamSettings.QUICSettings.Security != "" {
				query.Set("quicSecurity", ob.StreamSettings.QUICSettings.Security)
			}
			if ob.StreamSettings.QUICSettings.Key != "" {
				query.Set("key", ob.StreamSettings.QUICSettings.Key)
			}
		}
	case "kcp":
		if ob.StreamSettings.KCPSettings != nil {
			if ob.StreamSettings.KCPSettings.Seed != "" {
				query.Set("seed", ob.StreamSettings.KCPSettings.Seed)
			}
			if ob.StreamSettings.KCPSettings.Header != nil && ob.StreamSettings.KCPSettings.Header.Type != "" {
				query.Set("headerType", ob.StreamSettings.KCPSettings.Header.Type)
			}
		}
	case "grpc":
		if ob.StreamSettings.GRPCSettings != nil {
			if ob.StreamSettings.GRPCSettings.ServiceName != "" {
				query.Set("serviceName", ob.StreamSettings.GRPCSettings.ServiceName)
			}
			if ob.StreamSettings.GRPCSettings.MultiMode {
				query.Set("mode", "multi")
			}
		}
	}
	if len(query) > 0 {
		uri += "?" + query.Encode()
	}
	if ob.Remark != "" {
		uri += "#" + url.PathEscape(ob.Remark)
	}

	return uri, nil
}

func buildVmessURI(ob OutboundConfig) (string, error) {
	settingsBytes, _ := json.Marshal(ob.Settings)
	var settings VmessSettings
	if err := json.Unmarshal(settingsBytes, &settings); err != nil {
		return "", fmt.Errorf("invalid vmess settings format")
	}

	if len(settings.Vnext) == 0 || len(settings.Vnext[0].Users) == 0 {
		return "", fmt.Errorf("invalid vmess settings")
	}
	vnext := settings.Vnext[0]
	user := vnext.Users[0]
	data := map[string]interface{}{
		"v":    "2",
		"ps":   ob.Remark,
		"add":  vnext.Address,
		"port": vnext.Port,
		"id":   user.ID,
		"aid":  user.AlterID,
		"scy":  user.Security,
		"net":  ob.StreamSettings.Network,
		"type": "none",
	}
	if ob.StreamSettings.Security == "tls" {
		data["tls"] = "tls"
		if ob.StreamSettings.TLSSettings != nil {
			if ob.StreamSettings.TLSSettings.SNI != "" {
				data["sni"] = ob.StreamSettings.TLSSettings.SNI
			}
			if len(ob.StreamSettings.TLSSettings.ALPN) > 0 {
				data["alpn"] = strings.Join(ob.StreamSettings.TLSSettings.ALPN, ",")
			}
			if ob.StreamSettings.TLSSettings.Fingerprint != "" {
				data["fp"] = ob.StreamSettings.TLSSettings.Fingerprint
			}
		}
	}

	if ob.StreamSettings.TCPSettings != nil && ob.StreamSettings.TCPSettings.Header != nil {
		header := ob.StreamSettings.TCPSettings.Header
		if header.Type == "http" {
			data["type"] = "http"
			if header.Request != nil {
				if len(header.Request.Path) > 0 {
					data["path"] = strings.Join(header.Request.Path, ",")
				}
				if hosts, ok := header.Request.Headers["Host"]; ok && len(hosts) > 0 {
					data["host"] = hosts[0]
				}
			}
		}
	}

	switch ob.StreamSettings.Network {
	case "ws":
		if ob.StreamSettings.WSSettings != nil {
			data["path"] = ob.StreamSettings.WSSettings.Path
			if host, ok := ob.StreamSettings.WSSettings.Headers["Host"]; ok && host != "" {
				data["host"] = host
			}
		}
	case "http":
		if ob.StreamSettings.HTTPSettings != nil {
			data["path"] = ob.StreamSettings.HTTPSettings.Path
			if len(ob.StreamSettings.HTTPSettings.Host) > 0 {
				data["host"] = strings.Join(ob.StreamSettings.HTTPSettings.Host, ",")
			}
		}
	case "quic":
		if ob.StreamSettings.QUICSettings != nil {
			data["host"] = ob.StreamSettings.QUICSettings.Security
			data["path"] = ob.StreamSettings.QUICSettings.Key
			if ob.StreamSettings.QUICSettings.Header != nil {
				data["type"] = ob.StreamSettings.QUICSettings.Header.Type
			}
		}
	case "kcp":
		if ob.StreamSettings.KCPSettings != nil {
			data["path"] = ob.StreamSettings.KCPSettings.Seed
			if ob.StreamSettings.KCPSettings.Header != nil {
				data["type"] = ob.StreamSettings.KCPSettings.Header.Type
			}
		}
	case "grpc":
		if ob.StreamSettings.GRPCSettings != nil {
			data["path"] = ob.StreamSettings.GRPCSettings.ServiceName
			if ob.StreamSettings.GRPCSettings.MultiMode {
				data["type"] = "multi"
			}
		}
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("vmess: failed to marshal json: %w", err)
	}
	encoded := base64.StdEncoding.EncodeToString(jsonData)
	return "vmess://" + encoded, nil
}

func buildShadowsocksURI(ob OutboundConfig) (string, error) {
	settingsBytes, _ := json.Marshal(ob.Settings)
	var settings ShadowsocksSettings
	if err := json.Unmarshal(settingsBytes, &settings); err != nil {
		return "", fmt.Errorf("invalid shadowsocks settings format")
	}

	if len(settings.Servers) == 0 {
		return "", fmt.Errorf("invalid shadowsocks settings")
	}
	server := settings.Servers[0]
	auth := fmt.Sprintf("%s:%s", server.Method, server.Password)
	encodedAuth := base64.URLEncoding.EncodeToString([]byte(auth))
	uri := fmt.Sprintf("ss://%s@%s:%d", encodedAuth, server.Address, server.Port)

	if ob.StreamSettings.Network != "tcp" {
		plugin, ok := getPluginInfo(ob)
		if ok {
			uri += "?plugin=" + url.QueryEscape(plugin)
		}
	}

	if ob.Remark != "" {
		uri += "#" + url.PathEscape(ob.Remark)
	}

	return uri, nil
}

func getPluginInfo(ob OutboundConfig) (string, bool) {
	var plugin string
	switch ob.StreamSettings.Network {
	case "ws":
		opts := []string{}
		if ob.StreamSettings.Security == "tls" {
			opts = append(opts, "tls")
		}
		if ob.StreamSettings.WSSettings != nil {
			if host, ok := ob.StreamSettings.WSSettings.Headers["Host"]; ok && host != "" {
				opts = append(opts, "host="+host)
			}
			if ob.StreamSettings.WSSettings.Path != "" {
				opts = append(opts, "path="+ob.StreamSettings.WSSettings.Path)
			}
		}
		if ob.StreamSettings.TLSSettings != nil {
			if ob.StreamSettings.TLSSettings.SNI != "" {
				opts = append(opts, "sni="+ob.StreamSettings.TLSSettings.SNI)
			}
		}
		plugin = "v2ray-plugin;" + strings.Join(opts, ";")
		return plugin, true
	default:
		return "", false
	}
}

func parseVless(rawURL string, httpPort, socksPort uint16, geositePath string, customDNS *DNSConfig, geositeDNSRule *GeositeDNSRule, geositeRules []GeositeRule, geoipRules []GeoipRule, routingMode string) (*OutputConfig, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("vless: invalid url format: %w", err)
	}

	port, err := strconv.ParseUint(parsedURL.Port(), 10, 16)
	if err != nil {
		return nil, fmt.Errorf("vless: invalid port: %s", parsedURL.Port())
	}

	query := parsedURL.Query()

	encryption := query.Get("encryption")
	if encryption == "" {
		encryption = "none"
	}
	if encryption != "none" {
		return nil, fmt.Errorf("vless: encryption must be 'none'")
	}

	remark, _ := url.PathUnescape(parsedURL.Fragment)

	config := &OutputConfig{}
	config.Inbounds = createDefaultInbounds(httpPort, socksPort)

	if parsedURL.User != nil && parsedURL.User.Username() != "" {
		mainOutbound := OutboundConfig{
			Tag:      "Proxy",
			Protocol: "vless",
			Remark:   remark,
			Settings: VlessSettings{
				Vnext: []VlessVnext{
					{
						Address: parsedURL.Hostname(),
						Port:    uint16(port),
						Users: []VlessUser{
							{
								ID:         parsedURL.User.Username(),
								Encryption: encryption,
								Flow:       query.Get("flow"),
							},
						},
					},
				},
			},
		}
		config.Outbounds = []OutboundConfig{
			mainOutbound,
			{Protocol: "freedom", Tag: "Direct"},
			{Protocol: "blackhole", Tag: "Reject"},
		}
	}

	config.DNS = defaultDNSConfigWithCustom(geositePath, customDNS, geositeDNSRule)
	config.Routing = buildRoutingConfig(geositeRules, geoipRules, routingMode)

	security := query.Get("security")
	if security == "" {
		security = "none"
	}

	network := query.Get("type")
	if network == "" {
		network = "tcp"
	}

	if len(config.Outbounds) > 0 {
		config.Outbounds[0].StreamSettings = &StreamSettings{
			Network:  network,
			Security: security,
		}
	}

	headerType := query.Get("headerType")

	switch network {
	case "tcp":
		if headerType == "http" && len(config.Outbounds) > 0 {
			tcpSettings := &TCPSettings{
				Header: &HeaderConfig{
					Type: "http",
					Request: &RequestConfig{
						Headers: make(map[string][]string),
					},
				},
			}
			if path := query.Get("path"); path != "" {
				tcpSettings.Header.Request.Path = strings.Split(path, ",")
			}
			if host := query.Get("host"); host != "" {
				tcpSettings.Header.Request.Headers["Host"] = []string{host}
			}
			config.Outbounds[0].StreamSettings.TCPSettings = tcpSettings
		}
	case "ws":
		if len(config.Outbounds) > 0 {
			config.Outbounds[0].StreamSettings.WSSettings = &WSSettings{
				Path:    query.Get("path"),
				Headers: map[string]string{},
			}
			if host := query.Get("host"); host != "" {
				config.Outbounds[0].StreamSettings.WSSettings.Headers["Host"] = host
			}
		}
	case "http":
		if len(config.Outbounds) > 0 {
			hostStr := query.Get("host")
			hosts := []string{}
			if hostStr != "" {
				hosts = strings.Split(hostStr, ",")
			}
			config.Outbounds[0].StreamSettings.HTTPSettings = &HTTPSettings{
				Path: query.Get("path"),
				Host: hosts,
			}
		}
	case "quic":
		if len(config.Outbounds) > 0 {
			config.Outbounds[0].StreamSettings.QUICSettings = &QUICSettings{
				Security: query.Get("quicSecurity"),
				Key:      query.Get("key"),
				Header:   &HeaderConfig{Type: headerType},
			}
		}
	case "kcp":
		if len(config.Outbounds) > 0 {
			config.Outbounds[0].StreamSettings.KCPSettings = &KCPSettings{
				Seed:   query.Get("seed"),
				Header: &HeaderConfig{Type: headerType},
			}
		}
	case "grpc":
		if len(config.Outbounds) > 0 {
			config.Outbounds[0].StreamSettings.GRPCSettings = &GRPCSettings{
				ServiceName: query.Get("serviceName"),
				MultiMode:   query.Get("mode") == "multi",
			}
		}
	}
	if security == "tls" || security == "reality" {
		if len(config.Outbounds) > 0 {
			var alpn []string
			if alpnStr := query.Get("alpn"); alpnStr != "" {
				alpn = strings.Split(alpnStr, ",")
			}
			config.Outbounds[0].StreamSettings.TLSSettings = &TLSSettings{
				SNI:           query.Get("sni"),
				ALPN:          alpn,
				Fingerprint:   query.Get("fp"),
				AllowInsecure: query.Get("allowInsecure") == "1",
			}
		}
	}

	if security == "reality" {
		if len(config.Outbounds) > 0 {
			if query.Get("pbk") == "" {
				return nil, fmt.Errorf("vless: reality requires 'pbk' parameter")
			}
			config.Outbounds[0].StreamSettings.RealitySettings = &RealitySettings{
				SNI:         query.Get("sni"),
				Fingerprint: query.Get("fp"),
				PublicKey:   query.Get("pbk"),
				ShortID:     query.Get("sid"),
				SpiderX:     query.Get("spx"),
			}
			if config.Outbounds[0].StreamSettings.TLSSettings != nil {
				config.Outbounds[0].StreamSettings.RealitySettings.SNI = config.Outbounds[0].StreamSettings.TLSSettings.SNI
			}
		}
	}

	return config, nil
}

func parseTrojan(rawURL string, httpPort, socksPort uint16, geositePath string, customDNS *DNSConfig, geositeDNSRule *GeositeDNSRule, geositeRules []GeositeRule, geoipRules []GeoipRule, routingMode string) (*OutputConfig, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("trojan: invalid url format: %w", err)
	}

	port, err := strconv.ParseUint(parsedURL.Port(), 10, 16)
	if err != nil {
		return nil, fmt.Errorf("trojan: invalid port: %s", parsedURL.Port())
	}

	remark, _ := url.PathUnescape(parsedURL.Fragment)
	query := parsedURL.Query()
	config := &OutputConfig{}
	config.Inbounds = createDefaultInbounds(httpPort, socksPort)
	mainOutbound := OutboundConfig{
		Tag:      "Proxy",
		Protocol: "trojan",
		Remark:   remark,
		Settings: TrojanSettings{
			Servers: []TrojanServer{
				{
					Address:  parsedURL.Hostname(),
					Port:     uint16(port),
					Password: parsedURL.User.Username(),
					Flow:     query.Get("flow"),
				},
			},
		},
	}

	config.Outbounds = []OutboundConfig{
		mainOutbound,
		{Protocol: "freedom", Tag: "Direct"},
		{Protocol: "blackhole", Tag: "Reject"},
	}

	config.DNS = defaultDNSConfigWithCustom(geositePath, customDNS, geositeDNSRule)
	config.Routing = buildRoutingConfig(geositeRules, geoipRules, routingMode)

	security := query.Get("security")
	if security == "" {
		security = "tls"
	}

	network := query.Get("type")
	if network == "" {
		network = "tcp"
	}

	config.Outbounds[0].StreamSettings = &StreamSettings{
		Network:  network,
		Security: security,
	}

	headerType := query.Get("headerType")

	switch network {
	case "tcp":
		if headerType == "http" {
			tcpSettings := &TCPSettings{
				Header: &HeaderConfig{
					Type: "http",
					Request: &RequestConfig{
						Path:    strings.Split(query.Get("path"), ","),
						Headers: make(map[string][]string),
					},
				},
			}
			if host := query.Get("host"); host != "" {
				tcpSettings.Header.Request.Headers["Host"] = []string{host}
			}
			config.Outbounds[0].StreamSettings.TCPSettings = tcpSettings
		}
	case "ws":
		config.Outbounds[0].StreamSettings.WSSettings = &WSSettings{
			Path:    query.Get("path"),
			Headers: map[string]string{},
		}
		if host := query.Get("host"); host != "" {
			config.Outbounds[0].StreamSettings.WSSettings.Headers["Host"] = host
		}
	case "http":
		hostStr := query.Get("host")
		hosts := []string{}
		if hostStr != "" {
			hosts = strings.Split(hostStr, ",")
		}
		config.Outbounds[0].StreamSettings.HTTPSettings = &HTTPSettings{
			Path: query.Get("path"),
			Host: hosts,
		}
	case "quic":
		config.Outbounds[0].StreamSettings.QUICSettings = &QUICSettings{
			Security: query.Get("quicSecurity"),
			Key:      query.Get("key"),
			Header:   &HeaderConfig{Type: headerType},
		}
	case "kcp":
		config.Outbounds[0].StreamSettings.KCPSettings = &KCPSettings{
			Seed:   query.Get("seed"),
			Header: &HeaderConfig{Type: headerType},
		}
	case "grpc":
		config.Outbounds[0].StreamSettings.GRPCSettings = &GRPCSettings{
			ServiceName: query.Get("serviceName"),
			MultiMode:   query.Get("mode") == "multi",
		}
	}

	if security == "tls" || security == "reality" {
		var alpn []string
		if alpnStr := query.Get("alpn"); alpnStr != "" {
			alpn = strings.Split(alpnStr, ",")
		}
		config.Outbounds[0].StreamSettings.TLSSettings = &TLSSettings{
			SNI:           query.Get("sni"),
			ALPN:          alpn,
			Fingerprint:   query.Get("fp"),
			AllowInsecure: query.Get("allowInsecure") == "1",
		}
	}

	if security == "reality" {
		if query.Get("pbk") == "" {
			return nil, fmt.Errorf("trojan: reality requires 'pbk' parameter")
		}
		rs := &RealitySettings{
			PublicKey:   query.Get("pbk"),
			ShortID:     query.Get("sid"),
			SpiderX:     query.Get("spx"),
			Fingerprint: query.Get("fp"),
		}
		if tlsSettings := config.Outbounds[0].StreamSettings.TLSSettings; tlsSettings != nil {
			rs.SNI = tlsSettings.SNI
		}
		config.Outbounds[0].StreamSettings.RealitySettings = rs
	}
	return config, nil
}

func parseVmess(rawURL string, httpPort, socksPort uint16, geositePath string, customDNS *DNSConfig, geositeDNSRule *GeositeDNSRule, geositeRules []GeositeRule, geoipRules []GeoipRule, routingMode string) (*OutputConfig, error) {
	if !strings.HasPrefix(rawURL, "vmess://") {
		return nil, fmt.Errorf("vmess: invalid prefix")
	}
	base64Data := strings.TrimPrefix(rawURL, "vmess://")
	jsonData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return nil, fmt.Errorf("vmess: invalid base64 data: %w", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("vmess: invalid json data: %w", err)
	}

	remark, _ := data["ps"].(string)

	config := &OutputConfig{}
	config.Inbounds = createDefaultInbounds(httpPort, socksPort)
	mainOutbound := OutboundConfig{
		Tag:      "Proxy",
		Protocol: "vmess",
		Remark:   remark,
	}
	config.Outbounds = []OutboundConfig{
		mainOutbound,
		{Protocol: "freedom", Tag: "Direct"},
		{Protocol: "blackhole", Tag: "Reject"},
	}

	config.DNS = defaultDNSConfigWithCustom(geositePath, customDNS, geositeDNSRule)
	config.Routing = buildRoutingConfig(geositeRules, geoipRules, routingMode)

	vnext := VmessVnext{}
	if add, ok := data["add"].(string); ok {
		vnext.Address = add
	} else {
		return nil, fmt.Errorf("vmess: missing 'add'")
	}

	portStr := fmt.Sprintf("%v", data["port"])
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return nil, fmt.Errorf("vmess: invalid port: %v", data["port"])
	}
	vnext.Port = uint16(port)

	user := VmessUser{}
	if id, ok := data["id"].(string); ok {
		user.ID = id
	} else {
		return nil, fmt.Errorf("vmess: missing 'id'")
	}

	aidStr := fmt.Sprintf("%v", data["aid"])
	aid, err := strconv.ParseUint(aidStr, 10, 16)
	user.AlterID = uint16(aid)
	if err != nil {
		user.AlterID = 0
	}

	if scy, ok := data["scy"].(string); ok {
		user.Security = scy
	} else {
		user.Security = "auto"
	}
	vnext.Users = []VmessUser{user}
	config.Outbounds[0].Settings = VmessSettings{Vnext: []VmessVnext{vnext}}

	net, _ := data["net"].(string)
	if net == "" {
		net = "tcp"
	}

	config.Outbounds[0].StreamSettings = &StreamSettings{Network: net}

	if tls, ok := data["tls"].(string); ok && tls == "tls" {
		config.Outbounds[0].StreamSettings.Security = "tls"
		var alpn []string
		if alpnStr, ok := data["alpn"].(string); ok && alpnStr != "" {
			alpn = strings.Split(alpnStr, ",")
		}
		tlsSettings := &TLSSettings{ALPN: alpn}
		if sni, ok := data["sni"].(string); ok {
			tlsSettings.SNI = sni
		}
		if fp, ok := data["fp"].(string); ok {
			tlsSettings.Fingerprint = fp
		}
		config.Outbounds[0].StreamSettings.TLSSettings = tlsSettings
	}

	hType, _ := data["type"].(string)
	if net == "tcp" && hType == "http" {
		tcpSettings := &TCPSettings{
			Header: &HeaderConfig{
				Type: "http",
				Request: &RequestConfig{
					Headers: make(map[string][]string),
				},
			},
		}
		if path, ok := data["path"].(string); ok {
			tcpSettings.Header.Request.Path = strings.Split(path, ",")
		}
		if host, ok := data["host"].(string); ok {
			tcpSettings.Header.Request.Headers["Host"] = []string{host}
		}
		config.Outbounds[0].StreamSettings.TCPSettings = tcpSettings
	}

	switch net {
	case "ws":
		wsSettings := &WSSettings{Headers: map[string]string{}}
		if path, ok := data["path"].(string); ok {
			wsSettings.Path = path
		}
		if host, ok := data["host"].(string); ok {
			wsSettings.Headers["Host"] = host
		}
		config.Outbounds[0].StreamSettings.WSSettings = wsSettings
	case "http":
		httpSettings := &HTTPSettings{}
		if path, ok := data["path"].(string); ok {
			httpSettings.Path = path
		}
		if host, ok := data["host"].(string); ok {
			httpSettings.Host = strings.Split(host, ",")
		}
		config.Outbounds[0].StreamSettings.HTTPSettings = httpSettings
	case "quic":
		quicSettings := &QUICSettings{}
		if sec, ok := data["host"].(string); ok {
			quicSettings.Security = sec
		}
		if key, ok := data["path"].(string); ok {
			quicSettings.Key = key
		}
		if hType != "" {
			quicSettings.Header = &HeaderConfig{Type: hType}
		}
		config.Outbounds[0].StreamSettings.QUICSettings = quicSettings
	case "kcp":
		kcpSettings := &KCPSettings{}
		if seed, ok := data["path"].(string); ok {
			kcpSettings.Seed = seed
		}
		if hType != "" {
			kcpSettings.Header = &HeaderConfig{Type: hType}
		}
		config.Outbounds[0].StreamSettings.KCPSettings = kcpSettings
	case "grpc":
		grpcSettings := &GRPCSettings{}
		if serviceName, ok := data["path"].(string); ok {
			grpcSettings.ServiceName = serviceName
		}
		if hType == "multi" {
			grpcSettings.MultiMode = true
		}
		config.Outbounds[0].StreamSettings.GRPCSettings = grpcSettings
	}

	return config, nil
}

func parseShadowsocks(rawURL string, httpPort, socksPort uint16, geositePath string, customDNS *DNSConfig, geositeDNSRule *GeositeDNSRule, geositeRules []GeositeRule, geoipRules []GeoipRule, routingMode string) (*OutputConfig, error) {
	if strings.Contains(rawURL, "plugin=") {
		return parseShadowsocksPlugin(rawURL, httpPort, socksPort, geositePath, customDNS, geositeDNSRule, geositeRules, geoipRules, routingMode)
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("shadowsocks: invalid url format: %w", err)
	}

	remark, _ := url.PathUnescape(parsedURL.Fragment)

	var encodedPart string
	if parsedURL.User != nil && parsedURL.User.Username() != "" {
		encodedPart = parsedURL.User.Username()
	} else {
		atIndex := strings.Index(rawURL, "@")
		if atIndex == -1 {
			return nil, fmt.Errorf("shadowsocks: invalid format, missing @")
		}
		prefixIndex := strings.Index(rawURL, "ss://") + 5
		encodedPart = rawURL[prefixIndex:atIndex]
	}

	decoded, err := base64.RawURLEncoding.DecodeString(encodedPart)
	if err != nil {
		decoded, err = base64.StdEncoding.DecodeString(encodedPart)
		if err != nil {
			return nil, fmt.Errorf("shadowsocks: invalid base64 data: %w", err)
		}
	}

	authParts := strings.SplitN(string(decoded), ":", 2)
	if len(authParts) != 2 {
		return nil, fmt.Errorf("shadowsocks: invalid auth format")
	}
	method := authParts[0]
	password := authParts[1]

	config := &OutputConfig{}
	config.Inbounds = []InboundConfig{
		{Port: httpPort, Protocol: "http", Settings: make(map[string]interface{})},
		{Port: socksPort, Protocol: "socks", Settings: map[string]interface{}{"udp": true}},
	}

	mainOutbound := OutboundConfig{
		Tag:      "Proxy",
		Protocol: "shadowsocks",
		Remark:   remark,
		Settings: ShadowsocksSettings{
			Servers: []ShadowsocksServer{
				{
					Address:  parsedURL.Hostname(),
					Port:     mustParsePort(parsedURL.Port()),
					Method:   method,
					Password: password,
				},
			},
		},
		StreamSettings: &StreamSettings{Network: "tcp"},
	}
	config.Outbounds = []OutboundConfig{
		mainOutbound,
		{Protocol: "freedom", Tag: "Direct"},
		{Protocol: "blackhole", Tag: "Reject"},
	}
	config.DNS = defaultDNSConfigWithCustom(geositePath, customDNS, geositeDNSRule)
	config.Routing = buildRoutingConfig(geositeRules, geoipRules, routingMode)

	return config, nil
}

func parseShadowsocksPlugin(rawURL string, httpPort, socksPort uint16, geositePath string, customDNS *DNSConfig, geositeDNSRule *GeositeDNSRule, geositeRules []GeositeRule, geoipRules []GeoipRule, routingMode string) (*OutputConfig, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("shadowsocks: invalid plugin url format: %w", err)
	}

	remark, _ := url.PathUnescape(parsedURL.Fragment)
	query := parsedURL.Query()

	pluginStr := query.Get("plugin")
	pluginParts := strings.Split(pluginStr, ";")
	pluginName := pluginParts[0]

	var method, password string
	if parsedURL.User != nil {
		encodedPart := parsedURL.User.String()
		decoded, err := base64.RawURLEncoding.DecodeString(encodedPart)
		if err != nil {
			decoded, err = base64.StdEncoding.DecodeString(encodedPart)
			if err != nil {
				return nil, fmt.Errorf("shadowsocks plugin: invalid base64 user info: %w", err)
			}
		}
		authParts := strings.SplitN(string(decoded), ":", 2)
		if len(authParts) == 2 {
			method, password = authParts[0], authParts[1]
		}
	}

	if method == "" || password == "" {
		return nil, fmt.Errorf("shadowsocks plugin: could not parse method and password from user info")
	}

	config := &OutputConfig{}
	config.Inbounds = []InboundConfig{
		{Port: httpPort, Protocol: "http", Settings: make(map[string]interface{})},
		{Port: socksPort, Protocol: "socks", Settings: map[string]interface{}{"udp": true}},
	}

	mainOutbound := OutboundConfig{
		Tag:            "Proxy",
		Protocol:       "shadowsocks",
		Remark:         remark,
		StreamSettings: &StreamSettings{Network: "tcp"},
		Settings: ShadowsocksSettings{
			Servers: []ShadowsocksServer{
				{
					Address:  parsedURL.Hostname(),
					Port:     mustParsePort(parsedURL.Port()),
					Method:   method,
					Password: password,
				},
			},
		},
	}
	config.Outbounds = []OutboundConfig{
		mainOutbound,
		{Protocol: "freedom", Tag: "Direct"},
		{Protocol: "blackhole", Tag: "Reject"},
	}
	config.DNS = defaultDNSConfigWithCustom(geositePath, customDNS, geositeDNSRule)
	config.Routing = buildRoutingConfig(geositeRules, geoipRules, routingMode)

	if pluginName == "v2ray-plugin" {
		config.Outbounds[0].StreamSettings.Network = "ws"
		wsSettings := &WSSettings{Headers: make(map[string]string)}
		tlsSettings := &TLSSettings{}
		isTLS := false

		optsStr := query.Get("plugin")
		actualOpts := strings.Split(optsStr, ";")
		for _, opt := range actualOpts {
			switch {
			case opt == "tls":
				isTLS = true
			case strings.HasPrefix(opt, "host="):
				wsSettings.Headers["Host"] = strings.TrimPrefix(opt, "host=")
			case strings.HasPrefix(opt, "path="):
				wsSettings.Path = strings.TrimPrefix(opt, "path=")
			}
		}

		if sec := query.Get("security"); sec == "tls" {
			isTLS = true
		}
		if sni := query.Get("sni"); sni != "" {
			tlsSettings.SNI = sni
		}
		if fp := query.Get("fp"); fp != "" {
			tlsSettings.Fingerprint = fp
		}
		if path := query.Get("path"); path != "" {
			wsSettings.Path = path
		}
		if host := query.Get("host"); host != "" {
			wsSettings.Headers["Host"] = host
		}

		config.Outbounds[0].StreamSettings.WSSettings = wsSettings
		if isTLS {
			config.Outbounds[0].StreamSettings.Security = "tls"
			config.Outbounds[0].StreamSettings.TLSSettings = tlsSettings
		}
	}

	return config, nil
}

func mustParsePort(portStr string) uint16 {
	if portStr == "" {
		return 443
	}
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return 443
	}
	return uint16(port)
}

func defaultDNSConfigWithCustom(geositePath string, customDNS *DNSConfig, geositeDNSRule *GeositeDNSRule) *DNSConfig {
	if customDNS == nil && geositeDNSRule == nil {
		return nil
	}

	if customDNS != nil && geositeDNSRule == nil {
		// Remove hosts (DNS host mappings) entirely
		return &DNSConfig{
			Servers:  customDNS.Servers,
			Hosts:    nil, // <-- REMOVE HOSTS
			ClientIP: customDNS.ClientIP,
			Tag:      customDNS.Tag,
			Strategy: customDNS.Strategy,
		}
	}

	servers := []interface{}{}
	// hosts := map[string]string // <-- REMOVE HOSTS
	clientIP := ""
	tag := ""
	strategy := ""

	if customDNS != nil {
		servers = append(servers, customDNS.Servers...)
		// Do not copy hosts at all
		clientIP = customDNS.ClientIP
		tag = customDNS.Tag
		strategy = customDNS.Strategy
	}

	if geositePath != "" && geositeDNSRule != nil && geositeDNSRule.Domain != "" && geositeDNSRule.DNS != "" {
		if _, err := os.Stat(geositePath); err == nil {
			dnsRule := map[string]interface{}{
				"address": geositeDNSRule.DNS,
				"domains": []string{"geosite:" + geositeDNSRule.Domain},
			}
			servers = append([]interface{}{dnsRule}, servers...)
		}
	}

	// Always return a DNSConfig with NO hosts field
	return &DNSConfig{
		Servers:  servers,
		Hosts:    nil, // <-- REMOVE HOSTS
		ClientIP: clientIP,
		Tag:      tag,
		Strategy: strategy,
	}
}

var predefinedGeositeCategories = map[string]string{
	"ads":            "category-ads-all",
	"porn":           "category-porn",
	"media":          "category-media",
	"anticensorship": "category-anticensorship",
	"vpn":            "category-vpnservices",
	"games":          "category-games",
	"dev":            "category-dev",
	"ai":             "category-ai",
	"malware":        "category-malware",
	"phishing":       "category-phishing",
	"messaging":      "category-messaging",
	"cn":             "cn",
	"not-cn":         "geolocation-!cn",
	"private":        "private",
	"win-spy":        "win-spy",
	"win-update":     "win-update",
}
