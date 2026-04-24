package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// Config holds the configuration from the YAML file
type Config struct {
	Telegram struct {
		Token  string `yaml:"token"`
		ChatID string `yaml:"chat_id"`
	} `yaml:"telegram"`
	Settings struct {
		Tolerance int    `yaml:"tolerance"`
		Interval  int    `yaml:"interval"`
		LogFile   string `yaml:"log_file"`
	} `yaml:"settings"`
	Targets []string `yaml:"targets"`
	Ping    struct {
		Count   int `yaml:"count"`
		Timeout int `yaml:"timeout"`
	} `yaml:"ping"`
}

// Monitor represents the ping monitor
type Monitor struct {
	config          Config
	tolerance       int
	timer           int
	pingCount       int
	pingTimeout     int
	errCounter       map[string]int
	timeDown         map[string]string
	timeUp           map[string]string
	qDown            []string
	qUp              []string
	mu               sync.Mutex
	apiURL           string
	logFile          *os.File
	logger          *logrus.Logger
}

// PingResult represents the result of a ping operation
type PingResult struct {
	Success bool
	RTT     time.Duration
	Error   error
}

func main() {
	// Parse command line arguments
	configFile := flag.String("config", "hostmonitor.yaml", "Path to YAML config file")
	verbose := flag.Bool("verbose", false, "Enable verbose debug logging")
	flag.Parse()

	// Initialize monitor
	monitor, err := NewMonitor(*configFile)
	if err != nil {
		logrus.Fatalf("Failed to initialize monitor: %v", err)
	}
	defer monitor.logFile.Close()

	// Enable verbose logging if requested
	if *verbose {
		monitor.logger.SetLevel(logrus.DebugLevel)
		monitor.logger.Info("🐛 Verbose debug logging enabled")
	}

	monitor.logger.Info("🚀 Starting ping monitor")
	monitor.logger.WithFields(logrus.Fields{
		"tolerance": monitor.tolerance,
		"interval":   monitor.timer,
		"targets":    len(monitor.config.Targets),
	}).Info("⚙️ Monitor configuration loaded")

	// Main loop
	for {
		monitor.PingMonitor(monitor.config.Targets)
		monitor.ManageQueues()
		time.Sleep(time.Duration(monitor.timer) * time.Second)
	}
}

// NewMonitor creates a new Monitor instance
func NewMonitor(configFile string) (*Monitor, error) {
	// Read config
	config, err := readConfig(configFile)
	if err != nil {
		return nil, err
	}

	// Set defaults if not specified
	if config.Settings.Tolerance == 0 {
		config.Settings.Tolerance = 5
	}
	if config.Settings.Interval == 0 {
		config.Settings.Interval = 30
	}
	if config.Settings.LogFile == "" {
		config.Settings.LogFile = "/var/log/hostmonitor.log"
	}
	if config.Ping.Count == 0 {
		config.Ping.Count = 3
	}
	if config.Ping.Timeout == 0 {
		config.Ping.Timeout = 3
	}

	// Set up logrus logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	logger.SetLevel(logrus.InfoLevel)
	
	// Set up file logging
	logFile, err := os.OpenFile(config.Settings.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}
	logger.SetOutput(logFile)
	
	monitor := &Monitor{
		config:        *config,
		tolerance:     config.Settings.Tolerance,
		timer:         config.Settings.Interval,
		pingCount:     config.Ping.Count,
		pingTimeout:   config.Ping.Timeout,
		errCounter:     make(map[string]int),
		timeDown:       make(map[string]string),
		timeUp:         make(map[string]string),
		qDown:          []string{},
		qUp:            []string{},
		apiURL:         "https://api.telegram.org/bot",
		logFile:        logFile,
		logger:        logger,
	}

	// Test token (skip for testing)
	if config.Telegram.Token != "TEST_TOKEN" {
		err = monitor.testToken()
		if err != nil {
			return nil, err
		}
	}

	monitor.apiURL += config.Telegram.Token
	return monitor, nil
}

// readConfig reads the YAML configuration file
func readConfig(filename string) (*Config, error) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config Config
	err = yaml.Unmarshal(file, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %v", err)
	}

	if config.Telegram.Token == "" || config.Telegram.ChatID == "" {
		return nil, fmt.Errorf("missing required Telegram configuration")
	}

	if len(config.Targets) == 0 {
		return nil, fmt.Errorf("no targets specified in configuration")
	}

	return &config, nil
}

// testToken tests the Telegram bot token
func (m *Monitor) testToken() error {
	apiMethod := m.apiURL + "/getMe"
	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Get(apiMethod)
	if err != nil {
		return fmt.Errorf("failed to verify token: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token verification failed with status: %d", resp.StatusCode)
	}

	m.logger.Info("🔐 Token verified successfully")
	return nil
}

// escapeMarkdownV2 escapes special characters for Telegram MarkdownV2 format
func escapeMarkdownV2(s string) string {
	// Characters that need escaping in MarkdownV2: _ * [ ] ( ) ~ ` > # + - = | { } . !
	specialChars := []string{"_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"}
	result := s
	for _, char := range specialChars {
		result = strings.ReplaceAll(result, char, "\\"+char)
	}
	return result
}

// send sends a message via Telegram API with optional formatting
func (m *Monitor) send(message string, parseMode string) error {
	apiMethod := m.apiURL + "/sendMessage"
	client := &http.Client{Timeout: 1 * time.Second}

	var data string
	if parseMode != "" {
		data = fmt.Sprintf("chat_id=%s&text=%s&parse_mode=%s", m.config.Telegram.ChatID, message, parseMode)
	} else {
		data = fmt.Sprintf("chat_id=%s&text=%s", m.config.Telegram.ChatID, message)
	}

	resp, err := client.Post(apiMethod, "application/x-www-form-urlencoded", strings.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("message sending failed with status: %d", resp.StatusCode)
	}

	return nil
}

// notify prepares and sends notifications with rich MarkdownV2 formatting
func (m *Monitor) notify(target, status string) error {
	var msg string
	targetEscaped := escapeMarkdownV2(target)
	
	if status == "down" {
		msg = fmt.Sprintf("🚨 *🔴 HOST DOWN* 🚨\n\n📍 *Host:* `%s`\n🕒 *Last seen:* `%s`\n\n⚠️ *Status:* Host is unreachable via TCP port 80",
			targetEscaped, m.timeDown[target])
	} else if status == "up" {
		msg = fmt.Sprintf("✅ *🟢 HOST BACK UP* ✅\n\n📍 *Host:* `%s`\n🕒 *Recovered at:* `%s`\n\n✨ *Status:* Connection restored",
			targetEscaped, m.timeUp[target])
	}

	err := m.send(msg, "MarkdownV2")
	if err != nil {
		m.logger.WithFields(logrus.Fields{
		"target": target,
		"status": status,
		"error":   err,
	}).Error("❌ Failed to send notification")
		return err
	}

	m.logger.WithFields(logrus.Fields{
		"target": target,
		"status": status,
	}).Info("🔔 Notification sent")
	return nil
}

// manageQueues processes the notification queues
func (m *Monitor) ManageQueues() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Process down queue
	for i := 0; i < len(m.qDown); i++ {
		target := m.qDown[i]
		if err := m.notify(target, "down"); err == nil {
			// Remove from queue if notification sent successfully
			m.qDown = append(m.qDown[:i], m.qDown[i+1:]...)
			i--
		}
	}

	// Process up queue
	for i := 0; i < len(m.qUp); i++ {
		target := m.qUp[i]
		if err := m.notify(target, "up"); err == nil {
			// Remove from queue if notification sent successfully
			m.qUp = append(m.qUp[:i], m.qUp[i+1:]...)
			i--
		}
	}
}

// handleError handles ping failures
func (m *Monitor) handleError(target string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.errCounter[target]++
	m.logger.WithFields(logrus.Fields{
		"target": target,
		"status": "FAILED",
	}).Error("❌ Ping failed")

	if m.errCounter[target] == 1 {
		m.timeDown[target] = time.Now().Format("2006-01-02 15:04:05")
	}

	if m.errCounter[target] == m.tolerance+1 {
		m.qDown = append(m.qDown, target)
	}
}

// handleRecovery handles successful pings
func (m *Monitor) handleRecovery(target string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.WithFields(logrus.Fields{
		"target": target,
		"status": "OK",
	}).Info("✅ Ping successful")

	if m.errCounter[target] > m.tolerance {
		m.timeUp[target] = time.Now().Format("2006-01-02 15:04:05")
		m.qUp = append(m.qUp, target)
	}

	m.errCounter[target] = 0
}

// pingTarget pings a single target using TCP connect (port 80) as an alternative to ICMP
func (m *Monitor) pingTarget(target string) bool {
	startTime := time.Now()
	m.logger.WithFields(logrus.Fields{
		"target": target,
		"attempt": 1,
		"total_attempts": m.pingCount,
	}).Debug("📡 Starting ping attempt")
	
	// Try to resolve the hostname first
	m.logger.WithFields(logrus.Fields{
		"target": target,
	}).Debug("🔍 Resolving hostname")
	
	_, err := net.ResolveIPAddr("ip4", target)
	if err != nil {
		m.logger.WithFields(logrus.Fields{
			"target": target,
			"error":   err,
		}).Error("❌ DNS resolution failed")
		return false
	}
	
	m.logger.WithFields(logrus.Fields{
		"target": target,
	}).Debug("✅ Hostname resolved successfully")

	successCount := 0
	
	for i := 0; i < m.pingCount; i++ {
		attemptStart := time.Now()
		m.logger.WithFields(logrus.Fields{
			"target": target,
			"attempt": i + 1,
			"timeout": m.pingTimeout,
		}).Debug("🔌 Attempting TCP connection to port 80")
		
		// Try to establish a TCP connection to port 80 (HTTP)
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(target, "80"), time.Duration(m.pingTimeout)*time.Second)
		if err != nil {
			m.logger.WithFields(logrus.Fields{
				"target": target,
				"attempt": i + 1,
				"error":   err,
				"duration": time.Since(attemptStart).Seconds(),
			}).Debug("❌ TCP connection attempt failed")
			continue
		}
		
		// If we can establish a connection, the host is reachable
		successCount++
		conn.Close()
		
		m.logger.WithFields(logrus.Fields{
			"target": target,
			"attempt": i + 1,
			"duration": time.Since(attemptStart).Seconds(),
		}).Debug("✅ TCP connection successful")
		break // One successful connection is enough
	}

	totalDuration := time.Since(startTime).Seconds()
	if successCount > 0 {
		m.logger.WithFields(logrus.Fields{
			"target": target,
			"success": true,
			"attempts": successCount,
			"total_attempts": m.pingCount,
			"duration": totalDuration,
		}).Debug("✅ Ping completed successfully")
	} else {
		m.logger.WithFields(logrus.Fields{
			"target": target,
			"success": false,
			"attempts": m.pingCount,
			"duration": totalDuration,
		}).Debug("❌ Ping completed with all attempts failed")
	}

	return successCount > 0
}

// PingMonitor pings all targets and handles results
func (m *Monitor) PingMonitor(targets []string) {
	for _, target := range targets {
		if m.pingTarget(target) {
			m.handleRecovery(target)
		} else {
			m.handleError(target)
		}
	}
}

