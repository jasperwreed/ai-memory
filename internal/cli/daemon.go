package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/jasperwreed/ai-memory/internal/audit"
	"github.com/jasperwreed/ai-memory/internal/daemon"
)

func NewDaemonCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Manage the auto-capture daemon",
		Long:  `Control the background daemon that automatically captures AI tool sessions.`,
		Example: `  # Start the daemon
  mem daemon start

  # Stop the daemon
  mem daemon stop

  # Check daemon status
  mem daemon status

  # View audit logs
  mem daemon logs

  # Follow audit logs in real-time
  mem daemon logs -f`,
	}

	cmd.AddCommand(
		newDaemonStartCommand(),
		newDaemonStopCommand(),
		newDaemonStatusCommand(),
		newDaemonLogsCommand(),
	)

	return cmd
}

func newDaemonStartCommand() *cobra.Command {
	var background bool
	var configFile string

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the auto-capture daemon",
		Long:  `Start the background daemon that watches for AI tool session files.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			config := daemon.DefaultConfig()

			// Load custom config if provided
			if configFile != "" {
				data, err := os.ReadFile(configFile)
				if err != nil {
					return fmt.Errorf("failed to read config file: %w", err)
				}
				if err := json.Unmarshal(data, config); err != nil {
					return fmt.Errorf("failed to parse config file: %w", err)
				}
			}

			d, err := daemon.NewCaptureDaemon(config)
			if err != nil {
				return fmt.Errorf("failed to create daemon: %w", err)
			}

			if background {
				// Start in background and return
				if err := d.Start(); err != nil {
					return fmt.Errorf("failed to start daemon: %w", err)
				}
				fmt.Println("Daemon started in background")
				fmt.Printf("Audit logs: %s\n", config.AuditDir)
				return nil
			}

			// Run in foreground
			fmt.Println("Starting daemon in foreground (Ctrl+C to stop)...")
			fmt.Printf("Audit logs: %s\n", config.AuditDir)
			return d.Run()
		},
	}

	cmd.Flags().BoolVarP(&background, "background", "b", false, "Run daemon in background")
	cmd.Flags().StringVarP(&configFile, "config", "c", "", "Path to config file")

	return cmd
}

func newDaemonStopCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the auto-capture daemon",
		Long:  `Stop the running auto-capture daemon.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return err
			}

			pidFile := filepath.Join(homeDir, ".ai-memory", "audit", "daemon.pid")

			// Read PID file
			data, err := os.ReadFile(pidFile)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Println("Daemon is not running")
					return nil
				}
				return fmt.Errorf("failed to read PID file: %w", err)
			}

			var pid int
			fmt.Sscanf(string(data), "%d", &pid)

			// Find and terminate process
			process, err := os.FindProcess(pid)
			if err != nil {
				return fmt.Errorf("failed to find process: %w", err)
			}

			if err := process.Signal(os.Interrupt); err != nil {
				return fmt.Errorf("failed to stop daemon: %w", err)
			}

			// Wait a moment for graceful shutdown
			time.Sleep(2 * time.Second)

			// Remove PID file
			os.Remove(pidFile)

			fmt.Println("Daemon stopped")
			return nil
		},
	}
}

func newDaemonStatusCommand() *cobra.Command {
	var detailed bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show daemon status",
		Long:  `Display the current status of the auto-capture daemon.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return err
			}

			auditDir := filepath.Join(homeDir, ".ai-memory", "audit")
			status, err := daemon.GetStatus(auditDir)
			if err != nil {
				return fmt.Errorf("failed to get status: %w", err)
			}

			if status.Status == "stopped" {
				fmt.Println("Daemon status: stopped")
				return nil
			}

			fmt.Printf("Daemon status: %s\n", status.Status)
			fmt.Printf("PID: %d\n", status.PID)
			fmt.Printf("Updated: %s\n", status.UpdatedAt.Format("2006-01-02 15:04:05"))

			if status.Metrics != nil {
				fmt.Println("\nMetrics:")
				fmt.Printf("  Events received: %d\n", status.Metrics.EventsReceived)
				fmt.Printf("  Events processed: %d\n", status.Metrics.EventsProcessed)
				fmt.Printf("  Events dropped: %d\n", status.Metrics.EventsDropped)
				fmt.Printf("  Bytes written: %d\n", status.Metrics.BytesWritten)
				fmt.Printf("  Active sessions: %d\n", status.Metrics.ActiveSessions)
				fmt.Printf("  Running since: %s\n", status.Metrics.StartTime.Format("2006-01-02 15:04:05"))

				if !status.Metrics.LastEventTime.IsZero() {
					fmt.Printf("  Last event: %s\n", status.Metrics.LastEventTime.Format("2006-01-02 15:04:05"))
				}
			}

			if detailed && status.Config != nil {
				fmt.Println("\nConfiguration:")
				fmt.Printf("  Audit directory: %s\n", status.Config.AuditDir)
				fmt.Printf("  Max shard size: %d MB\n", status.Config.MaxShardSize/(1024*1024))
				fmt.Printf("  Compress shards: %v\n", status.Config.CompressShards)
				fmt.Printf("  Batch size: %d\n", status.Config.BatchSize)
				fmt.Printf("  Flush interval: %s\n", status.Config.FlushInterval)

				fmt.Println("\n  Watch directories:")
				for _, dir := range status.Config.WatchDirs {
					fmt.Printf("    - %s\n", dir)
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&detailed, "detailed", "d", false, "Show detailed status including configuration")

	return cmd
}

func newDaemonLogsCommand() *cobra.Command {
	var follow bool
	var tail int
	var sessionID string
	var tool string

	cmd := &cobra.Command{
		Use:   "logs",
		Short: "View audit logs",
		Long:  `Display audit logs captured by the daemon.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return err
			}

			auditDir := filepath.Join(homeDir, ".ai-memory", "audit")

			// Create streamer
			streamer := audit.NewStreamer(auditDir)
			if follow {
				defer streamer.Stop()
			}

			// Create filter if needed
			filter := audit.StreamFilter{
				SessionID: sessionID,
				Tool:      tool,
			}

			// Handler to print logs
			lineCount := 0
			handler := func(line []byte) error {
				// Apply tail limit if not following
				if !follow && tail > 0 {
					lineCount++
					if lineCount > tail {
						return nil
					}
				}

				// Try to pretty-print JSON
				var data map[string]interface{}
				if err := json.Unmarshal(line, &data); err == nil {
					// Format timestamp if present
					if ts, ok := data["timestamp"].(float64); ok {
						timestamp := time.Unix(int64(ts), 0)
						fmt.Printf("[%s] ", timestamp.Format("15:04:05"))
					}

					// Print key fields
					if tool, ok := data["tool"].(string); ok {
						fmt.Printf("<%s> ", tool)
					}
					if eventType, ok := data["type"].(string); ok {
						fmt.Printf("%s ", eventType)
					}
					if session, ok := data["session"].(string); ok && session != "" {
						fmt.Printf("(session: %.8s) ", session)
					}

					// Print raw content if available
					if raw, ok := data["raw"].(string); ok {
						// Truncate long lines
						if len(raw) > 100 {
							raw = raw[:97] + "..."
						}
						fmt.Print(raw)
					} else {
						// Print the whole JSON
						output, _ := json.Marshal(data)
						fmt.Print(string(output))
					}
				} else {
					// Print raw line if not JSON
					fmt.Print(string(line))
				}

				if len(line) == 0 || line[len(line)-1] != '\n' {
					fmt.Println()
				}

				return nil
			}

			// Stream logs
			return streamer.FilteredStream(filter, handler, follow)
		},
	}

	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output")
	cmd.Flags().IntVarP(&tail, "tail", "n", 0, "Number of lines to show from the end")
	cmd.Flags().StringVarP(&sessionID, "session", "s", "", "Filter by session ID")
	cmd.Flags().StringVarP(&tool, "tool", "t", "", "Filter by tool name")

	return cmd
}