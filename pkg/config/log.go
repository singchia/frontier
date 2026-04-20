package config

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jumboframes/armorigo/log"
	"gopkg.in/natefinch/lumberjack.v2"
	"k8s.io/klog/v2"
)

// Log holds unified logging configuration for both frontier and frontlas.
type Log struct {
	// Level controls log verbosity for both klog and armorigo.
	// Options: "debug", "info", "warn", "error". Default: "info".
	Level string `yaml:"level,omitempty" json:"level"`

	// Output controls where logs are written.
	// Options: "stdout", "stderr", "file", "both" (stdout+file). Default: "stdout".
	Output string `yaml:"output,omitempty" json:"output"`

	// Format controls the log output format.
	// Options: "text", "json". Default: "text".
	Format string `yaml:"format,omitempty" json:"format"`

	// File holds file-based logging config, used when Output is "file" or "both".
	File LogFile `yaml:"file,omitempty" json:"file"`
}

// LogFile configures file-based log output with rotation via lumberjack.
type LogFile struct {
	// Path is the log file path. Default: "/var/log/frontier/<component>.log".
	Path string `yaml:"path,omitempty" json:"path"`

	// MaxSize is the max size in MB before rotation. Default: 100.
	MaxSize int `yaml:"max_size,omitempty" json:"max_size"`

	// MaxBackups is the max number of old log files to keep. Default: 5.
	MaxBackups int `yaml:"max_backups,omitempty" json:"max_backups"`

	// MaxAge is the max days to retain old log files. 0 means no age limit. Default: 30.
	MaxAge int `yaml:"max_age,omitempty" json:"max_age"`

	// Compress enables gzip compression for rotated log files. Default: false.
	Compress bool `yaml:"compress,omitempty" json:"compress"`
}

// level mapping: user-facing level -> klog verbosity
var levelToKlogVerbosity = map[string]int{
	"debug": 4,
	"info":  2,
	"warn":  0,
	"error": 0,
}

// level mapping: user-facing level -> armorigo level
var levelToArmorigo = map[string]log.Level{
	"debug": log.LevelDebug,
	"info":  log.LevelInfo,
	"warn":  log.LevelWarn,
	"error": log.LevelError,
}

// SetupLogging initializes both klog and armorigo logging based on the unified
// Log config. The component parameter ("frontier" or "frontlas") is used for
// the default log file path.
func SetupLogging(cfg *Log, component string) error {
	applyDefaults(cfg, component)

	if err := validateConfig(cfg); err != nil {
		return err
	}

	// build the writer(s) for the chosen output mode
	writer, err := buildWriter(cfg)
	if err != nil {
		return err
	}

	// configure klog
	setupKlog(cfg, writer)

	// configure armorigo
	setupArmorigo(cfg, writer)

	return nil
}

func applyDefaults(cfg *Log, component string) {
	if cfg.Level == "" {
		cfg.Level = "info"
	}
	cfg.Level = strings.ToLower(cfg.Level)

	if cfg.Output == "" {
		cfg.Output = "stdout"
	}
	cfg.Output = strings.ToLower(cfg.Output)

	if cfg.Format == "" {
		cfg.Format = "text"
	}
	cfg.Format = strings.ToLower(cfg.Format)

	if cfg.File.Path == "" {
		cfg.File.Path = "/var/log/frontier/" + component + ".log"
	}
	if cfg.File.MaxSize <= 0 {
		cfg.File.MaxSize = 100
	}
	if cfg.File.MaxBackups <= 0 {
		cfg.File.MaxBackups = 5
	}
	if cfg.File.MaxAge <= 0 {
		cfg.File.MaxAge = 30
	}
}

func validateConfig(cfg *Log) error {
	if _, ok := levelToKlogVerbosity[cfg.Level]; !ok {
		return fmt.Errorf("unsupported log level %q, options: debug, info, warn, error", cfg.Level)
	}
	switch cfg.Output {
	case "stdout", "stderr", "file", "both":
	default:
		return fmt.Errorf("unsupported log output %q, options: stdout, stderr, file, both", cfg.Output)
	}
	switch cfg.Format {
	case "text", "json":
	default:
		return fmt.Errorf("unsupported log format %q, options: text, json", cfg.Format)
	}
	return nil
}

func buildWriter(cfg *Log) (io.Writer, error) {
	var fileWriter io.Writer
	if cfg.Output == "file" || cfg.Output == "both" {
		fileWriter = &lumberjack.Logger{
			Filename:   cfg.File.Path,
			MaxSize:    cfg.File.MaxSize,
			MaxBackups: cfg.File.MaxBackups,
			MaxAge:     cfg.File.MaxAge,
			Compress:   cfg.File.Compress,
		}
	}

	switch cfg.Output {
	case "stdout":
		return os.Stdout, nil
	case "stderr":
		return os.Stderr, nil
	case "file":
		return fileWriter, nil
	case "both":
		return io.MultiWriter(os.Stdout, fileWriter), nil
	}
	return os.Stdout, nil
}

func setupKlog(cfg *Log, writer io.Writer) {
	// ensure klog flags are initialized
	fs := flag.NewFlagSet("klog", flag.ContinueOnError)
	klog.InitFlags(fs)

	// set verbosity
	verbosity := levelToKlogVerbosity[cfg.Level]
	fs.Set("v", fmt.Sprintf("%d", verbosity))

	// for "error" level, raise the stderr threshold so only warnings+ go through
	if cfg.Level == "error" {
		fs.Set("stderrthreshold", "WARNING")
	}

	// direct all klog output to our unified writer
	// disable klog's own stderr/file logic — we handle it
	fs.Set("logtostderr", "false")
	fs.Set("alsologtostderr", "false")
	klog.SetOutput(writer)
}

func setupArmorigo(cfg *Log, writer io.Writer) {
	log.SetLevel(levelToArmorigo[cfg.Level])
	log.SetOutput(writer)
}

// ApplyLogEnvOverrides applies environment variable overrides to the Log config.
// Priority: env > yaml (caller should call this after loading yaml but before SetupLogging).
func ApplyLogEnvOverrides(cfg *Log) {
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.Level = v
	}
	if v := os.Getenv("LOG_OUTPUT"); v != "" {
		cfg.Output = v
	}
	if v := os.Getenv("LOG_FORMAT"); v != "" {
		cfg.Format = v
	}
	if v := os.Getenv("LOG_FILE"); v != "" {
		cfg.File.Path = v
	}
}
