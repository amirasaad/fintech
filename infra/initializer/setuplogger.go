package initializer

import (
	"github.com/amirasaad/fintech/pkg/config"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"log/slog"
	"os"
)

func setupLogger(cfg *config.Log) *slog.Logger {
	// Create a new logger with a custom style
	// Define color styles for different log levels
	styles := log.DefaultStyles()
	infoTxtColor := lipgloss.AdaptiveColor{Light: "#04B575", Dark: "#04B575"}
	warnTxtColor := lipgloss.AdaptiveColor{Light: "#EE6FF8", Dark: "#EE6FF8"}
	errorTxtColor := lipgloss.AdaptiveColor{Light: "#FF6B6B", Dark: "#FF6B6B"}
	debugTxtColor := lipgloss.AdaptiveColor{Light: "#7E57C2", Dark: "#7E57C2"}

	// Customize the style for each log level
	// Error level styling
	styles.Levels[log.ErrorLevel] = lipgloss.NewStyle().
		SetString("❌").
		Bold(true).
		Padding(0, 1).
		Foreground(errorTxtColor)

	// Info level styling
	styles.Levels[log.InfoLevel] = lipgloss.NewStyle().
		SetString("ℹ️").
		Bold(true).
		Padding(0, 1).
		Foreground(infoTxtColor)

	// Warn level styling
	styles.Levels[log.WarnLevel] = lipgloss.NewStyle().
		SetString("⚠️").
		Bold(true).
		Padding(0, 1).
		Foreground(warnTxtColor)

	// Debug level styling
	styles.Levels[log.DebugLevel] = lipgloss.NewStyle().
		SetString("🐛").
		Bold(true).
		Padding(0, 1).
		Foreground(debugTxtColor)

	styles.Keys["error"] = lipgloss.NewStyle().Foreground(errorTxtColor)
	styles.Values["error"] = lipgloss.NewStyle().Bold(true)
	styles.Keys["info"] = lipgloss.NewStyle().Foreground(infoTxtColor)
	styles.Values["info"] = lipgloss.NewStyle().Bold(true)
	styles.Keys["warn"] = lipgloss.NewStyle().Foreground(warnTxtColor)
	styles.Values["warn"] = lipgloss.NewStyle().Bold(true)
	styles.Keys["debug"] = lipgloss.NewStyle().Foreground(debugTxtColor)
	styles.Values["debug"] = lipgloss.NewStyle().Bold(true)
	styles.Keys["prefix"] = lipgloss.NewStyle().Foreground(debugTxtColor)
	styles.Values["prefix"] = lipgloss.NewStyle().Bold(true)
	styles.Keys["caller"] = lipgloss.NewStyle().Foreground(debugTxtColor)
	styles.Values["caller"] = lipgloss.NewStyle().Bold(true)
	styles.Keys["time"] = lipgloss.NewStyle().Foreground(debugTxtColor)
	styles.Values["time"] = lipgloss.NewStyle().Bold(true)

	formattersMap := map[string]log.Formatter{
		"json": log.JSONFormatter,
		"text": log.TextFormatter,
	}
	formatter := log.TextFormatter
	if f, ok := formattersMap[cfg.Format]; ok {
		formatter = f
	}

	// Create a new logger with the custom styles
	logger := log.NewWithOptions(os.Stdout, log.Options{
		ReportCaller:    true,
		ReportTimestamp: true,
		TimeFormat:      cfg.TimeFormat,
		Level:           log.Level(cfg.Level),
		Prefix:          cfg.Prefix,
		Formatter:       formatter,
	})

	logger.SetStyles(styles) // Convert to slog.Logger

	slogger := slog.New(logger)
	slog.SetDefault(slogger)

	return slogger
}
