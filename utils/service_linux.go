//go:build linux
// +build linux

package utils

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

const (
	ServiceName     = "iptv-aggregator"
	ServiceDir      = "/opt/iptv-aggregator"
	SystemdDir      = "/etc/systemd/system"
	ServiceFile     = ServiceName + ".service"
	ServiceFilePath = SystemdDir + "/" + ServiceFile
)

// ServiceManager handles Linux systemd service operations
type ServiceManager struct {
	execPath   string
	configPath string
	logger     *Logger
}

// NewServiceManager creates a new service manager
func NewServiceManager(execPath, configPath string, logger *Logger) *ServiceManager {
	return &ServiceManager{
		execPath:   execPath,
		configPath: configPath,
		logger:     logger,
	}
}

// checkRoot verifies if running as root
func (sm *ServiceManager) checkRoot() error {
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %v", err)
	}

	if currentUser.Uid != "0" {
		return fmt.Errorf("this operation requires root privileges. Please run with sudo")
	}
	return nil
}

// generateSystemdService generates systemd service file content
func (sm *ServiceManager) generateSystemdService() string {
	configArg := ""
	// Always use config.json in the service directory
	// If a custom config was provided, it will be copied with that name
	if sm.configPath != "" && sm.configPath != "config.json" {
		configArg = fmt.Sprintf(" -config %s", filepath.Base(sm.configPath))
	} else {
		configArg = " -config config.json"
	}

	return fmt.Sprintf(`[Unit]
Description=IPTV M3U Aggregator Service
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=%s
ExecStart=%s/iptv-aggregator%s
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
`, ServiceDir, ServiceDir, configArg)
}

// Install installs the service
func (sm *ServiceManager) Install() error {
	if err := sm.checkRoot(); err != nil {
		return err
	}

	sm.logger.Info("Installing IPTV Aggregator service...")

	// Create service directory
	if err := os.MkdirAll(ServiceDir, 0755); err != nil {
		return fmt.Errorf("failed to create service directory: %v", err)
	}
	sm.logger.Info("Created service directory: %s", ServiceDir)

	// Copy executable
	destExec := filepath.Join(ServiceDir, "iptv-aggregator")
	if err := copyFile(sm.execPath, destExec); err != nil {
		return fmt.Errorf("failed to copy executable: %v", err)
	}
	if err := os.Chmod(destExec, 0755); err != nil {
		return fmt.Errorf("failed to set executable permissions: %v", err)
	}
	sm.logger.Info("Copied executable to: %s", destExec)

	// Copy config file
	var configFile string
	if sm.configPath != "" && sm.configPath != "config.json" {
		// Use custom config file name
		configFile = filepath.Base(sm.configPath)
		destConfig := filepath.Join(ServiceDir, configFile)
		if err := copyFile(sm.configPath, destConfig); err != nil {
			return fmt.Errorf("failed to copy config file: %v", err)
		}
		sm.logger.Info("Copied config file to: %s", destConfig)
	} else {
		// Use default config.json
		configFile = "config.json"
		// Try to copy config.json from current directory if it exists
		if _, err := os.Stat("config.json"); err == nil {
			destConfig := filepath.Join(ServiceDir, "config.json")
			if err := copyFile("config.json", destConfig); err != nil {
				sm.logger.Warn("Failed to copy config.json: %v", err)
			} else {
				sm.logger.Info("Copied config file to: %s", destConfig)
			}
		} else {
			sm.logger.Warn("config.json not found in current directory, service will use default config")
		}
	}

	// Create systemd service file
	serviceContent := sm.generateSystemdService()
	if err := os.WriteFile(ServiceFilePath, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to create systemd service file: %v", err)
	}
	sm.logger.Info("Created systemd service file: %s", ServiceFilePath)

	// Reload systemd daemon
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %v", err)
	}
	sm.logger.Info("Reloaded systemd daemon")

	// Enable service
	if err := exec.Command("systemctl", "enable", ServiceFile).Run(); err != nil {
		return fmt.Errorf("failed to enable service: %v", err)
	}
	sm.logger.Info("Enabled service: %s", ServiceFile)

	fmt.Printf("\n✓ Service installed successfully!\n")
	fmt.Printf("  Service file: %s\n", ServiceFilePath)
	fmt.Printf("  Install path: %s\n", ServiceDir)
	fmt.Printf("  Config file: %s/%s\n", ServiceDir, configFile)
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  Start service:   sudo systemctl start %s\n", ServiceName)
	fmt.Printf("  Check status:    sudo systemctl status %s\n", ServiceName)
	fmt.Printf("  View logs:       sudo journalctl -u %s -f\n", ServiceName)

	return nil
}

// Uninstall uninstalls the service
func (sm *ServiceManager) Uninstall() error {
	if err := sm.checkRoot(); err != nil {
		return err
	}

	sm.logger.Info("Uninstalling IPTV Aggregator service...")

	// Stop service if running
	if err := exec.Command("systemctl", "stop", ServiceFile).Run(); err != nil {
		sm.logger.Warn("Failed to stop service (may not be running): %v", err)
	} else {
		sm.logger.Info("Stopped service")
	}

	// Disable service
	if err := exec.Command("systemctl", "disable", ServiceFile).Run(); err != nil {
		sm.logger.Warn("Failed to disable service: %v", err)
	} else {
		sm.logger.Info("Disabled service")
	}

	// Remove systemd service file
	if err := os.Remove(ServiceFilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove systemd service file: %v", err)
	}
	sm.logger.Info("Removed systemd service file")

	// Reload systemd daemon
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %v", err)
	}
	sm.logger.Info("Reloaded systemd daemon")

	// Remove service directory
	if err := os.RemoveAll(ServiceDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove service directory: %v", err)
	}
	sm.logger.Info("Removed service directory")

	fmt.Printf("\n✓ Service uninstalled successfully!\n")

	return nil
}

// Start starts the service
func (sm *ServiceManager) Start() error {
	if err := sm.checkRoot(); err != nil {
		return err
	}

	sm.logger.Info("Starting service...")
	if err := exec.Command("systemctl", "start", ServiceFile).Run(); err != nil {
		return fmt.Errorf("failed to start service: %v", err)
	}

	fmt.Printf("✓ Service started successfully\n")
	fmt.Printf("  View logs: sudo journalctl -u %s -f\n", ServiceName)

	return nil
}

// Stop stops the service
func (sm *ServiceManager) Stop() error {
	if err := sm.checkRoot(); err != nil {
		return err
	}

	sm.logger.Info("Stopping service...")
	if err := exec.Command("systemctl", "stop", ServiceFile).Run(); err != nil {
		return fmt.Errorf("failed to stop service: %v", err)
	}

	fmt.Printf("✓ Service stopped successfully\n")

	return nil
}

// Restart restarts the service
func (sm *ServiceManager) Restart() error {
	if err := sm.checkRoot(); err != nil {
		return err
	}

	sm.logger.Info("Restarting service...")
	if err := exec.Command("systemctl", "restart", ServiceFile).Run(); err != nil {
		return fmt.Errorf("failed to restart service: %v", err)
	}

	fmt.Printf("✓ Service restarted successfully\n")
	fmt.Printf("  View logs: sudo journalctl -u %s -f\n", ServiceName)

	return nil
}

// Status shows the service status
func (sm *ServiceManager) Status() error {
	cmd := exec.Command("systemctl", "status", ServiceFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, input, 0644)
}

// HandleServiceCommand handles service management commands
func HandleServiceCommand(command, execPath, configPath string, logger *Logger) error {
	sm := NewServiceManager(execPath, configPath, logger)

	command = strings.ToLower(strings.TrimSpace(command))

	switch command {
	case "install":
		return sm.Install()
	case "uninstall":
		return sm.Uninstall()
	case "start":
		return sm.Start()
	case "stop":
		return sm.Stop()
	case "restart":
		return sm.Restart()
	case "status":
		return sm.Status()
	default:
		return fmt.Errorf("unknown service command: %s. Valid commands: install, uninstall, start, stop, restart, status", command)
	}
}
