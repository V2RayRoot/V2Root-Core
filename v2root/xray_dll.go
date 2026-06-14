package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/xtls/xray-core/app/stats/command"
	"github.com/xtls/xray-core/common/cmdarg"
	"github.com/xtls/xray-core/core"
	_ "github.com/xtls/xray-core/main/distro/all"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	server            core.Server
	serverLock        sync.Mutex
	done              chan struct{}
	serverStatus      string = "STOPPED"
	logConfig                = make(map[string]interface{})
	statsApiPort      int
	lastStatsTime     time.Time
	lastTotalUplink   int64
	lastTotalDownlink int64
	statsMutex        sync.Mutex

	// Build metadata is injected with -ldflags from the release environment.
	buildCodeVersion = "0"
	version          = "development"
	releaseDate      = "unknown"
)

//export FreeCString
func FreeCString(value *C.char) {
	if value != nil {
		C.free(unsafe.Pointer(value))
	}
}

//export GetStatus
func GetStatus() *C.char {
	serverLock.Lock()
	defer serverLock.Unlock()
	return C.CString(serverStatus)
}

//export SetLogOutput
func SetLogOutput(path *C.char) {
	logPath := C.GoString(path)
	if logPath != "" {
		logConfig["access"] = logPath
		logConfig["error"] = logPath
	}
}

//export SetLogLevel
func SetLogLevel(level *C.char) {
	logLevel := C.GoString(level)
	if logLevel != "" {
		logConfig["loglevel"] = logLevel
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func isValidJSON(s string) bool {
	var js map[string]interface{}
	return json.Unmarshal([]byte(s), &js) == nil
}

// normalizeAndValidateConfig normalizes common Xray fields and validates required fields.
// Returns an error string if validation fails, else nil.
func normalizeAndValidateConfig(configMap map[string]interface{}) string {
	// Normalize outbounds
	if outboundsRaw, ok := configMap["outbounds"].([]interface{}); ok {
		for _, outboundRaw := range outboundsRaw {
			if outbound, ok := outboundRaw.(map[string]interface{}); ok {
				if tag, _ := outbound["tag"].(string); tag == "" {
					outbound["tag"] = "Proxy"
				}
				// Validate required fields for common protocols
				if protocol, _ := outbound["protocol"].(string); protocol == "vless" || protocol == "vmess" {
					if settingsRaw, ok := outbound["settings"].(map[string]interface{}); ok {
						if vnextRaw, ok := settingsRaw["vnext"].([]interface{}); ok && len(vnextRaw) > 0 {
							if vnext, ok := vnextRaw[0].(map[string]interface{}); ok {
								if address, ok := vnext["address"].(string); !ok || address == "" {
									return "outbound.settings.vnext[0].address missing or invalid"
								}
								if portRaw, ok := vnext["port"]; !ok {
									return "outbound.settings.vnext[0].port missing"
								} else if port, ok := portRaw.(float64); !ok || port <= 0 {
									return "outbound.settings.vnext[0].port invalid"
								}
								if usersRaw, ok := vnext["users"].([]interface{}); ok && len(usersRaw) > 0 {
									if user, ok := usersRaw[0].(map[string]interface{}); ok {
										if id, ok := user["id"].(string); !ok || id == "" {
											return "outbound.settings.vnext[0].users[0].id missing or invalid"
										}
									} else {
										return "outbound.settings.vnext[0].users[0] invalid"
									}
								} else {
									return "outbound.settings.vnext[0].users missing or empty"
								}
							} else {
								return "outbound.settings.vnext[0] invalid"
							}
						} else {
							return "outbound.settings.vnext missing or empty"
						}
					} else {
						return "outbound.settings missing"
					}
				} else if protocol == "trojan" {
					if settingsRaw, ok := outbound["settings"].(map[string]interface{}); ok {
						if serversRaw, ok := settingsRaw["servers"].([]interface{}); ok && len(serversRaw) > 0 {
							if server, ok := serversRaw[0].(map[string]interface{}); ok {
								if address, ok := server["address"].(string); !ok || address == "" {
									return "outbound.settings.servers[0].address missing or invalid"
								}
								if portRaw, ok := server["port"]; !ok {
									return "outbound.settings.servers[0].port missing"
								} else if port, ok := portRaw.(float64); !ok || port <= 0 {
									return "outbound.settings.servers[0].port invalid"
								}
								if password, ok := server["password"].(string); !ok || password == "" {
									return "outbound.settings.servers[0].password missing or invalid"
								}
							} else {
								return "outbound.settings.servers[0] invalid"
							}
						} else {
							return "outbound.settings.servers missing or empty"
						}
					} else {
						return "outbound.settings missing"
					}
				}
				// Ensure streamSettings exists
				if _, ok := outbound["streamSettings"]; !ok {
					outbound["streamSettings"] = map[string]interface{}{
						"network":  "tcp",
						"security": "none",
					}
				}
			}
		}
	}

	// Normalize routing
	if routingRaw, ok := configMap["routing"].(map[string]interface{}); ok {
		if domainStrategy, _ := routingRaw["domainStrategy"].(string); domainStrategy == "" {
			routingRaw["domainStrategy"] = "IPIfNonMatch"
		}
	} else {
		configMap["routing"] = map[string]interface{}{
			"domainStrategy": "IPIfNonMatch",
			"rules":          []interface{}{},
		}
	}

	return "" // No error
}

//export Start
func Start(configInput_C *C.char, optionsJSON_C *C.char) *C.char {
	serverLock.Lock()
	defer serverLock.Unlock()

	if server != nil {
		if strings.HasPrefix(serverStatus, "RUNNING") {
			return C.CString("server already running")
		}
	}

	serverStatus = "STARTING"
	log.SetOutput(io.Discard) // Xray logs go to file, not here

	configInput := C.GoString(configInput_C)
	optionsJSON := C.GoString(optionsJSON_C)
	if configInput == "" {
		serverStatus = "STOPPED"
		return C.CString("empty config provided")
	}

	// --- Parse Options ---
	// This struct holds options from the JSON string.
	var opts struct {
		GeositeFile string `json:"geositeFile"`
		GeositePath string `json:"geositePath"`
	}
	if optionsJSON != "" {
		if err := json.Unmarshal([]byte(optionsJSON), &opts); err != nil {
			// Not a fatal error, just means options are default
			fmt.Fprintf(os.Stderr, "Warning: failed to parse options JSON: %v\n", err)
		}
	}

	// --- Set Asset Location ---
	var dir string
	if opts.GeositeFile != "" {
		if _, err := os.Stat(opts.GeositeFile); err != nil {
			serverStatus = "STOPPED"
			return C.CString("geosite file not found at: " + opts.GeositeFile)
		}
		dir = opts.GeositeFile
		if idx := strings.LastIndexAny(opts.GeositeFile, "/\\"); idx != -1 {
			dir = opts.GeositeFile[:idx]
		}
		_ = os.Setenv("XRAY_LOCATION_ASSET", dir)
	} else if opts.GeositePath != "" {
		if _, err := os.Stat(opts.GeositePath); err == nil {
			dir = opts.GeositePath
			if idx := strings.LastIndexAny(opts.GeositePath, "/\\"); idx != -1 {
				dir = opts.GeositePath[:idx]
			}
			_ = os.Setenv("XRAY_LOCATION_ASSET", dir)
		}
	}

	// --- Load Base Config ---
	var config string
	trimmed := strings.TrimSpace(configInput)
	if fileExists(trimmed) {
		data, err := os.ReadFile(trimmed)
		if err != nil {
			serverStatus = "STOPPED"
			return C.CString("failed to read config file: " + err.Error())
		}
		config = string(data)
	} else if isValidJSON(trimmed) {
		config = trimmed
	} else if strings.HasPrefix(trimmed, "vless://") || strings.HasPrefix(trimmed, "vmess://") ||
		strings.HasPrefix(trimmed, "trojan://") || strings.HasPrefix(trimmed, "ss://") {
		// This is a URI, so pass the options to the parser.
		parserOpts := make(map[string]interface{})
		if optionsJSON != "" {
			if err := json.Unmarshal([]byte(optionsJSON), &parserOpts); err != nil {
				serverStatus = "STOPPED"
				return C.CString("invalid options JSON: " + err.Error())
			}
		}
		parserOpts["uri"] = trimmed
		optsBytes, _ := json.Marshal(parserOpts)
		parsed_C := Parse(C.CString(string(optsBytes)))

		if parsed_C == nil {
			serverStatus = "STOPPED"
			return C.CString("failed to parse config string")
		}
		config = C.GoString(parsed_C)
	} else {
		serverStatus = "STOPPED"
		return C.CString("invalid config input")
	}

	// --- Post-Process Config Map ---
	configMap := make(map[string]interface{})
	if err := json.Unmarshal([]byte(config), &configMap); err != nil {
		serverStatus = "STOPPED"
		return C.CString("failed to unmarshal generated config for modification: " + err.Error())
	}

	// --- BEGIN DNSConfig to DNS Migration ---
	dnsConfigRaw, dnsConfigExists := configMap["dnsConfig"]
	dnsRaw, dnsExists := configMap["dns"]

	if dnsConfigExists {
		if !dnsExists {
			configMap["dns"] = dnsConfigRaw
			delete(configMap, "dnsConfig")
		} else {
			dnsConfigMap, okConfig := dnsConfigRaw.(map[string]interface{})
			dnsMap, ok := dnsRaw.(map[string]interface{})
			if ok && okConfig {
				for key, value := range dnsConfigMap {
					dnsMap[key] = value
				}
				delete(configMap, "dnsConfig")
			} else {
				configMap["dns"] = dnsConfigRaw
				delete(configMap, "dnsConfig")
			}
		}
	}
	// --- END DNSConfig to DNS Migration ---

	// --- Log Config ---
	if len(logConfig) > 0 {
		configMap["log"] = logConfig
	}

	// --- Stats API Config ---
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		serverStatus = "STOPPED"
		return C.CString("failed to find a free port for stats API: " + err.Error())
	}
	statsApiPort = listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	configMap["stats"] = make(map[string]interface{})
	configMap["api"] = map[string]interface{}{
		"tag":      "api",
		"services": []string{"StatsService"},
	}
	configMap["policy"] = map[string]interface{}{
		"system": map[string]interface{}{
			"statsInboundUplink":    true,
			"statsInboundDownlink":  true,
			"statsOutboundUplink":   true,
			"statsOutboundDownlink": true,
		},
	}

	apiInbound := map[string]interface{}{
		"tag":      "api",
		"protocol": "dokodemo-door",
		"port":     statsApiPort,
		"listen":   "127.0.0.1",
	}
	// Append API inbound
	inbounds, ok := configMap["inbounds"].([]interface{})
	if !ok {
		inbounds = make([]interface{}, 0)
	}
	configMap["inbounds"] = append(inbounds, apiInbound)

	// --- API Outbound Config ---
	apiOutbound := map[string]interface{}{
		"tag":      "api",
		"protocol": "freedom",
		"settings": make(map[string]interface{}),
	}
	outbounds, ok := configMap["outbounds"].([]interface{})
	if !ok {
		outbounds = make([]interface{}, 0)
	}
	configMap["outbounds"] = append(outbounds, apiOutbound)

	// --- Routing Config (API + VPN) ---
	apiRule := map[string]interface{}{
		"type":        "field",
		"inboundTag":  []string{"api"},
		"outboundTag": "api",
	}

	routing, ok := configMap["routing"].(map[string]interface{})
	if !ok {
		routing = make(map[string]interface{})
		configMap["routing"] = routing
	}
	userRules, ok := routing["rules"].([]interface{})
	if !ok {
		userRules = make([]interface{}, 0)
	}

	allRules := []interface{}{apiRule}
	allRules = append(allRules, userRules...)
	routing["rules"] = allRules

	// --- Normalize and Validate Config ---
	if errMsg := normalizeAndValidateConfig(configMap); errMsg != "" {
		serverStatus = "STOPPED"
		return C.CString("config validation failed: " + errMsg)
	}

	// --- Finalize and Start Server ---
	finalConfigBytes, err := json.Marshal(configMap)
	if err != nil {
		serverStatus = "STOPPED"
		return C.CString("failed to marshal final config: " + err.Error())
	}

	tmpDir := "tmp"
	_ = os.MkdirAll(tmpDir, 0755)
	tmpFile, err := os.CreateTemp(tmpDir, "xray-config-*.json")
	if err != nil {
		serverStatus = "STOPPED"
		return C.CString("failed to create temp file: " + err.Error())
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.Write(finalConfigBytes); err != nil {
		tmpFile.Close()
		serverStatus = "STOPPED"
		return C.CString("failed to write config: " + err.Error())
	}
	tmpFile.Close()

	configFiles := cmdarg.Arg{tmpPath}
	c, err := core.LoadConfig("json", configFiles)
	if err != nil {
		serverStatus = "STOPPED"
		return C.CString("failed to load config: " + err.Error())
	}

	srv, err := core.New(c)
	if err != nil {
		serverStatus = "STOPPED"
		return C.CString("failed to create server: " + err.Error())
	}

	server = srv
	done = make(chan struct{})

	statsMutex.Lock()
	lastStatsTime = time.Time{}
	lastTotalUplink = 0
	lastTotalDownlink = 0
	statsMutex.Unlock()

	go func() {
		// srv.Start() blocks until server stops
		if err := srv.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Xray Core Wrapper: server start failed: %v\n", err)
			serverLock.Lock()
			server = nil
			serverStatus = "STOPPED"
			serverLock.Unlock()
			return
		}
		// This runs after server has fully stopped
		fmt.Println("Xray Core Wrapper: server goroutine finished.")
	}()

	// Set status *after* starting the goroutine
	serverStatus = "RUNNING"
	fmt.Println("Xray Core Wrapper: Server Started in Proxy Mode")

	runtime.GC()
	debug.FreeOSMemory()

	return nil // nil means success
}

//export Stop
func Stop() *C.char {
	serverLock.Lock()
	defer serverLock.Unlock()

	if server == nil {
		serverStatus = "STOPPED"
		return C.CString("server not running")
	}

	serverStatus = "STOPPING"
	fmt.Println("Xray Core Wrapper: Stopping server...")

	// Close the server instance
	if err := server.Close(); err != nil {
		server = nil // Force nil even on error
		serverStatus = "STOPPED"
		fmt.Fprintf(os.Stderr, "Xray Core Wrapper: failed to stop server gracefully: %v\n", err)
		// Don't return error string, just ensure state is STOPPED
	}

	// Signal the goroutine to exit
	if done != nil {
		close(done)
		done = nil
	}

	server = nil
	serverStatus = "STOPPED"
	fmt.Println("Xray Core Wrapper: Server stopped.")

	statsMutex.Lock()
	lastStatsTime = time.Time{}
	lastTotalUplink = 0
	lastTotalDownlink = 0
	statsMutex.Unlock()

	return nil // nil means success
}

//export TestLatency
func TestLatency(configsJSON *C.char, testURL *C.char, timeout int) *C.char {
	log.SetOutput(io.Discard)
	configs := C.GoString(configsJSON)
	urlStr := C.GoString(testURL)
	if urlStr == "" {
		urlStr = "https://www.gstatic.com/generate_204"
	}

	var configList []string
	if err := json.Unmarshal([]byte(configs), &configList); err != nil {
		configList = []string{configs}
	}

	results := make([]string, len(configList))
	var wg sync.WaitGroup
	for i, cfg := range configList {
		wg.Add(1)
		go func(idx int, config string) {
			defer wg.Done()
			results[idx] = testSingleConfig(config, urlStr, timeout)
		}(i, cfg)
	}
	wg.Wait()

	resultJSON, _ := json.Marshal(results)
	return C.CString(string(resultJSON))
}

func testSingleConfig(configJSON, testURL string, timeoutSec int) string {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "recovered from panic in testSingleConfig: %v\n", r)
		}
	}()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "[ERROR]failed to create listener"
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	var fullConfig string
	var tempStr string
	if err := json.Unmarshal([]byte(configJSON), &tempStr); err == nil {
		fullConfig = tempStr
	} else {
		fullConfig = configJSON
	}

	trimmedConfig := strings.TrimSpace(fullConfig)
	if strings.HasPrefix(trimmedConfig, "vless://") || strings.HasPrefix(trimmedConfig, "vmess://") ||
		strings.HasPrefix(trimmedConfig, "trojan://") || strings.HasPrefix(trimmedConfig, "ss://") {
		// TestLatency should *not* enable VPN mode.
		opts := map[string]interface{}{"uri": trimmedConfig}
		optsBytes, _ := json.Marshal(opts)
		parsed_C := Parse(C.CString(string(optsBytes)))
		if parsed_C == nil {
			return "[ERROR]URI parse failed in latency test"
		}
		fullConfig = C.GoString(parsed_C)
	}

	configMap := make(map[string]interface{})
	if err := json.Unmarshal([]byte(fullConfig), &configMap); err != nil {
		return "[ERROR]failed to parse config"
	}

	// --- BEGIN DNSConfig to DNS Migration (for TestLatency) ---
	dnsConfigRaw, dnsConfigExists := configMap["dnsConfig"]
	dnsRaw, dnsExists := configMap["dns"]
	if dnsConfigExists {
		if !dnsExists {
			configMap["dns"] = dnsConfigRaw
			delete(configMap, "dnsConfig")
		} else {
			dnsConfigMap, okConfig := dnsConfigRaw.(map[string]interface{})
			dnsMap, ok := dnsRaw.(map[string]interface{})
			if ok && okConfig {
				for key, value := range dnsConfigMap {
					dnsMap[key] = value
				}
				delete(configMap, "dnsConfig")
			} else {
				configMap["dns"] = dnsConfigRaw
				delete(configMap, "dnsConfig")
			}
		}
	}
	// --- END DNSConfig to DNS Migration (for TestLatency) ---

	inbounds, ok := configMap["inbounds"].([]interface{})
	if !ok || len(inbounds) == 0 {
		if len(inbounds) == 0 {
			return "[ERROR]no inbounds found in config"
		}
	}

	var targetInbound map[string]interface{}
	for _, rawInbound := range inbounds {
		in, ok := rawInbound.(map[string]interface{})
		if !ok {
			continue
		}
		proto, _ := in["protocol"].(string)
		if proto == "socks" || proto == "http" {
			targetInbound = in
			break
		}
	}

	if targetInbound == nil {
		return "[ERROR]no suitable inbound (socks/http) to use for test"
	}
	targetInbound["port"] = float64(port)
	targetInbound["listen"] = "127.0.0.1"
	modifiedConfig, err := json.Marshal(configMap)
	if err != nil {
		return "[ERROR]failed to marshal config"
	}

	tmpFile, err := os.CreateTemp("", fmt.Sprintf("xray-test-%d-*.json", port))
	if err != nil {
		return "[ERROR]failed to create temp file"
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(modifiedConfig); err != nil {
		tmpFile.Close()
		return "[ERROR]failed to write config"
	}
	tmpFile.Close()

	configFiles := cmdarg.Arg{tmpFile.Name()}
	c, err := core.LoadConfig("json", configFiles)
	if err != nil {
		return "[ERROR]failed to load config: " + err.Error()
	}
	srv, err := core.New(c)
	if err != nil {
		return "[ERROR]failed to create server: " + err.Error()
	}

	go func() {
		_ = srv.Start()
	}()
	defer srv.Close()

	proxyAddr := fmt.Sprintf("127.0.0.1:%d", port)
	if !waitForServerReady(proxyAddr, 3*time.Second) {
		return "[ERROR]waitForServer"
	}

	for i := 0; i < 3; i++ {
		latency := performLatencyTest(proxyAddr, testURL, timeoutSec)
		if latency != "-1" {
			return latency
		}
		time.Sleep(100 * time.Millisecond)
	}

	return "ERROR"
}

func performLatencyTest(proxyAddr, testURL string, timeoutSec int) string {
	proxyURL, err := url.Parse("http://" + proxyAddr)
	if err != nil {
		return "-1"
	}
	transport := &http.Transport{
		Proxy:                 http.ProxyURL(proxyURL),
		DialContext:           (&net.Dialer{Timeout: 3 * time.Second}).DialContext,
		TLSHandshakeTimeout:   3 * time.Second,
		ResponseHeaderTimeout: time.Duration(timeoutSec) * time.Second,
		MaxIdleConns:          1,
		MaxIdleConnsPerHost:   1,
		IdleConnTimeout:       10 * time.Second,
	}
	client := &http.Client{
		Timeout:   time.Duration(timeoutSec) * time.Second,
		Transport: transport,
	}
	warmupResp, err := client.Get(testURL)
	if err != nil {
		return "-1"
	}
	io.Copy(io.Discard, warmupResp.Body)
	warmupResp.Body.Close()
	if warmupResp.StatusCode != http.StatusNoContent && warmupResp.StatusCode != http.StatusOK {
		return "-1"
	}
	start := time.Now()
	resp, err := client.Get(testURL)
	if err != nil {
		return "-1"
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return "-1"
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	latency := time.Since(start).Milliseconds()
	return fmt.Sprintf("%d", latency)
}

func waitForServerReady(addr string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			conn.Close()
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

//export ValidateConfig
func ValidateConfig(configInput_C *C.char, optionsJSON_C *C.char) *C.char {
	configStr := C.GoString(configInput_C)
	optionsJSON := C.GoString(optionsJSON_C)
	if configStr == "" {
		return C.CString(`{"error":"empty config"}`)
	}

	// --- Parse Options ---
	var opts struct {
		GeositeFile string `json:"geositeFile"`
		GeositePath string `json:"geositePath"`
	}
	if optionsJSON != "" {
		_ = json.Unmarshal([]byte(optionsJSON), &opts)
	}

	// --- Set Asset Location ---
	var dir string
	if opts.GeositeFile != "" {
		if _, err := os.Stat(opts.GeositeFile); err != nil {
			return C.CString(fmt.Sprintf(`{"error":"geosite file not found at: %s"}`, opts.GeositeFile))
		}
		dir = opts.GeositeFile
		if idx := strings.LastIndexAny(opts.GeositeFile, "/\\"); idx != -1 {
			dir = opts.GeositeFile[:idx]
		}
		_ = os.Setenv("XRAY_LOCATION_ASSET", dir)
	} else if opts.GeositePath != "" {
		if _, err := os.Stat(opts.GeositePath); err == nil {
			dir = opts.GeositePath
			if idx := strings.LastIndexAny(opts.GeositePath, "/\\"); idx != -1 {
				dir = opts.GeositePath[:idx]
			}
			_ = os.Setenv("XRAY_LOCATION_ASSET", dir)
		}
	}

	// --- Load Base Config ---
	var config string
	trimmed := strings.TrimSpace(configStr)
	if fileExists(trimmed) {
		data, err := os.ReadFile(trimmed)
		if err != nil {
			return C.CString(fmt.Sprintf(`{"error":"failed to read file: %s"}`, err.Error()))
		}
		config = string(data)
	} else if isValidJSON(trimmed) {
		config = trimmed
	} else if strings.HasPrefix(trimmed, "vless://") || strings.HasPrefix(trimmed, "vmess://") ||
		strings.HasPrefix(trimmed, "trojan://") || strings.HasPrefix(trimmed, "ss://") {
		parserOpts := make(map[string]interface{})
		if optionsJSON != "" {
			_ = json.Unmarshal([]byte(optionsJSON), &parserOpts)
		}
		parserOpts["uri"] = trimmed
		optsBytes, _ := json.Marshal(parserOpts)
		parsed_C := Parse(C.CString(string(optsBytes)))
		if parsed_C == nil {
			return C.CString(`{"error":"failed to parse config string"}`)
		}
		config = C.GoString(parsed_C)
	} else {
		return C.CString(`{"error":"invalid config input"}`)
	}

	// --- Post-Process Config Map ---
	configMap := make(map[string]interface{})
	if err := json.Unmarshal([]byte(config), &configMap); err != nil {
		return C.CString(fmt.Sprintf(`{"error":"invalid JSON: %s"}`, err.Error()))
	}

	// --- DNS Fix ---
	dnsConfigRaw, dnsConfigExists := configMap["dnsConfig"]
	dnsRaw, dnsExists := configMap["dns"]
	if dnsConfigExists {
		if !dnsExists {
			configMap["dns"] = dnsConfigRaw
			delete(configMap, "dnsConfig")
		} else {
			dnsConfigMap, okConfig := dnsConfigRaw.(map[string]interface{})
			dnsMap, ok := dnsRaw.(map[string]interface{})
			if ok && okConfig {
				for key, value := range dnsConfigMap {
					dnsMap[key] = value
				}
				delete(configMap, "dnsConfig")
			} else {
				configMap["dns"] = dnsConfigRaw
				delete(configMap, "dnsConfig")
			}
		}
	}

	// --- Normalize and Validate Config ---
	if errMsg := normalizeAndValidateConfig(configMap); errMsg != "" {
		return C.CString(fmt.Sprintf(`{"error":"config validation failed: %s"}`, errMsg))
	}

	// Re-marshal the config after fixing DNS and injecting VPN
	finalConfigBytes, err := json.Marshal(configMap)
	if err != nil {
		return C.CString(fmt.Sprintf(`{"error":"failed to re-marshal config for validation: %s"}`, err.Error()))
	}
	config = string(finalConfigBytes)

	// --- Final Validation by Loading Config ---
	var js json.RawMessage
	if err := json.Unmarshal([]byte(config), &js); err != nil {
		return C.CString(fmt.Sprintf(`{"error":"invalid final JSON: %s"}`, err.Error()))
	}

	tmpDir := "tmp"
	_ = os.MkdirAll(tmpDir, 0755)
	tmpFile, err := os.CreateTemp(tmpDir, "xray-validate-*.json")
	if err != nil {
		return C.CString(fmt.Sprintf(`{"error":"failed to create temp file: %s"}`, err.Error()))
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.Write([]byte(config)); err != nil {
		tmpFile.Close()
		return C.CString(fmt.Sprintf(`{"error":"failed to write config: %s"}`, err.Error()))
	}
	tmpFile.Close()

	configFiles := cmdarg.Arg{tmpPath}
	_, err = core.LoadConfig("json", configFiles)
	if err != nil {
		return C.CString(fmt.Sprintf(`{"error":"failed to load config: %s"}`, err.Error()))
	}

	return C.CString(`{"result":"valid"}`)
}

func queryStatsFromAPI() (int64, int64, error) {
	addr := fmt.Sprintf("127.0.0.1:%d", statsApiPort)
	dialCtx, dialCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer dialCancel()

	conn, err := grpc.DialContext(dialCtx, addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return 0, 0, fmt.Errorf("stats API connection failed: %w", err)
	}
	defer conn.Close()

	client := command.NewStatsServiceClient(conn)
	req := &command.QueryStatsRequest{Reset_: false}
	rpcCtx, rpcCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer rpcCancel()

	resp, err := client.QueryStats(rpcCtx, req)
	if err != nil {
		return 0, 0, fmt.Errorf("QueryStats RPC failed: %w", err)
	}

	var totalUplink, totalDownlink int64
	for _, stat := range resp.Stat {
		// Outbound stats are consistent across supported proxy configurations.
		if strings.HasPrefix(stat.Name, "outbound") && !strings.Contains(stat.Name, "api") {
			if strings.HasSuffix(stat.Name, "uplink") {
				totalUplink += stat.Value
			} else if strings.HasSuffix(stat.Name, "downlink") {
				totalDownlink += stat.Value
			}
		}
	}
	return totalUplink, totalDownlink, nil
}

//export GetTotalTraffics
func GetTotalTraffics() *C.char {
	serverLock.Lock()
	if !strings.HasPrefix(serverStatus, "RUNNING") || server == nil {
		serverLock.Unlock()
		return C.CString(`{"error": "server is not running"}`)
	}
	serverLock.Unlock()

	totalUplink, totalDownlink, err := queryStatsFromAPI()
	if err != nil {
		return C.CString(fmt.Sprintf(`{"error": "%s"}`, err.Error()))
	}

	stats := map[string]int64{
		"uplink":   totalUplink,
		"downlink": totalDownlink,
	}

	resultJSON, _ := json.Marshal(stats)
	return C.CString(string(resultJSON))
}

//export GetRealtimeSpeed
func GetRealtimeSpeed() *C.char {
	serverLock.Lock()
	if !strings.HasPrefix(serverStatus, "RUNNING") || server == nil {
		serverLock.Unlock()
		return C.CString(`{"error": "server is not running", "uplinkSpeed": 0, "downlinkSpeed": 0}`)
	}
	serverLock.Unlock()

	statsMutex.Lock()
	defer statsMutex.Unlock()

	currentUplink, currentDownlink, err := queryStatsFromAPI()
	if err != nil {
		return C.CString(fmt.Sprintf(`{"error": "%s", "uplinkSpeed": 0, "downlinkSpeed": 0}`, err.Error()))
	}

	currentTime := time.Now()
	var uplinkSpeed, downlinkSpeed float64

	if !lastStatsTime.IsZero() {
		timeDiff := currentTime.Sub(lastStatsTime).Seconds()
		if timeDiff > 0.1 {
			uplinkDiff := currentUplink - lastTotalUplink
			downlinkDiff := currentDownlink - lastTotalDownlink

			if uplinkDiff < 0 {
				uplinkDiff = 0
			}
			if downlinkDiff < 0 {
				downlinkDiff = 0
			}

			uplinkSpeed = float64(uplinkDiff) / timeDiff
			downlinkSpeed = float64(downlinkDiff) / timeDiff
		}
	}

	lastStatsTime = currentTime
	lastTotalUplink = currentUplink
	lastTotalDownlink = currentDownlink

	speeds := map[string]float64{
		"uplinkSpeed":   uplinkSpeed,
		"downlinkSpeed": downlinkSpeed,
	}
	resultJSON, _ := json.Marshal(speeds)
	return C.CString(string(resultJSON))
}

//export UpdateGeoAssets
func UpdateGeoAssets(assetPath_C *C.char) *C.char {
	assetPath := C.GoString(assetPath_C)
	targetDir := assetPath

	if targetDir == "" {
		exePath, err := os.Executable()
		if err != nil {
			return C.CString(fmt.Sprintf(`{"error": "failed to get executable path: %s"}`, err.Error()))
		}
		targetDir = filepath.Join(filepath.Dir(exePath), "Resources")
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return C.CString(fmt.Sprintf(`{"error": "failed to create resources directory: %s"}`, err.Error()))
	}

	urls := map[string]string{
		"geosite.dat": "https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geosite.dat",
		"geoip.dat":   "https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geoip.dat",
	}

	results := make(map[string]string)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for filename, url := range urls {
		wg.Add(1)
		go func(fn, u string) {
			defer wg.Done()
			destPath := filepath.Join(targetDir, fn)
			err := downloadFile(u, destPath)
			mu.Lock()
			if err != nil {
				results[fn] = "failed: " + err.Error()
			} else {
				results[fn] = fn
			}
			mu.Unlock()
		}(filename, url)
	}

	wg.Wait()
	resultJSON, _ := json.Marshal(results)
	return C.CString(string(resultJSON))
}

func downloadFile(url, destPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

//export GetVersionInfo
func GetVersionInfo() *C.char {
	codeVersion, err := strconv.Atoi(buildCodeVersion)
	if err != nil {
		codeVersion = 0
	}
	info := map[string]interface{}{
		"codeVersion": codeVersion,
		"version":     version,
		"releaseDate": releaseDate,
	}
	b, _ := json.Marshal(info)
	return C.CString(string(b))
}

func main() {}
