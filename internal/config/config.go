package config

import (
	"embed"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/GGP1/adak/internal/logger"
	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/stripe/stripe-go"
)

var (
	configFilename = "config"
	configDir      = filepath.Join(getConfigDir(runtime.GOOS), "adak")
)

// Config constains all the server configurations.
type Config struct {
	Admin     Admin
	Database  Database
	Memcached Memcached
	Server    Server
	Email     Email
	Stripe    Stripe
	Static    Static
}

// Admin contains the admins emails.
type Admin struct {
	Emails []string
}

// Database hols the database attributes.
type Database struct {
	Username string
	Password string
	Host     string
	Port     string
	Name     string
	SSLMode  string
}

// Memcached is the LRU-cache configuration.
type Memcached struct {
	Servers []string
}

// Server holds the server attributes.
type Server struct {
	Host string
	Port string
	TLS  struct {
		KeyFile  string
		CertFile string
	}
	Timeout struct {
		Read     time.Duration
		Write    time.Duration
		Shutdown time.Duration
	}
}

// Email holds email attributes.
type Email struct {
	Host     string
	Port     string
	Sender   string
	Password string
}

// Stripe hold stripe attributes
type Stripe struct {
	SecretKey string
	Logger    struct {
		Level stripe.Level
	}
}

// Static contains the static file system.
type Static struct {
	FS embed.FS
}

// New sets up the configuration with the values the user gave.
// Defaults and env variables are placed at the end to make the config easier to read.
func New() (*Config, error) {
	viper.AddConfigPath(configDir)
	viper.SetConfigName(configFilename)
	viper.SetConfigType("yaml")

	path := os.Getenv("ADAK_CONFIG")
	if path != "" {
		if filepath.Ext(path) == "" {
			path = filepath.Join(path, ".env")
		}

		if err := godotenv.Load(path); err != nil {
			return nil, errors.Wrap(err, "env loading failed")
		}

		logger.Log.Info("Using customized configuration")
	} else {
		logger.Log.Info("Using default configuration")
	}

	// Bind envs
	for k, v := range envVars {
		viper.BindEnv(k, v)
	}

	// Set defaults
	for k, v := range defaults {
		viper.SetDefault(k, v)
	}

	// Read or create configuration file
	if err := loadConfig(); err != nil {
		return nil, errors.Wrap(err, "couldn't read the configuration file")
	}

	// Auto read env variables
	viper.AutomaticEnv()
	config := &Config{}

	if err := viper.Unmarshal(config); err != nil {
		return nil, errors.Wrap(err, "unmarshal configuration failed")
	}

	logger.Log.Info("Configuration created successfully")
	return config, nil
}

// read configuration from file.
func loadConfig() error {
	// Find and read the config file
	if err := viper.ReadInConfig(); err != nil {
		configAbsPath := filepath.Join(configDir, configFilename+".yml")

		// if file does not exist, simply create one
		if _, err := os.Stat(configAbsPath); os.IsNotExist(err) {
			if err := os.MkdirAll(configDir, 0755); err != nil {
				return errors.New("failed creating folder")
			}
			f, err := os.Create(configAbsPath)
			if err != nil {
				return errors.New("failed creating file")
			}
			f.Close()
		} else {
			return err
		}

		if err := viper.WriteConfig(); err != nil {
			return err
		}
	}

	return nil
}

// getConfigDir returns the location of the configuration file.
func getConfigDir(osName string) string {
	if os.Getenv("SV_DIR") != "" {
		return os.Getenv("SV_DIR")
	}

	switch runtime.GOOS {
	case "windows":
		return os.Getenv("APPDATA")
	case "darwin":
		return filepath.Join(os.Getenv("HOME"), "Library/Application Support")
	case "linux":
		return filepath.Join(os.Getenv("HOME"), ".config")
	default:
		dir, _ := os.Getwd()
		return dir
	}
}

// Declared at the end to avoid scrolling
var (
	defaults = map[string]interface{}{
		// Admins
		"admin.emails": []string{},
		// Database
		"database.username": "postgres",
		"database.password": "password",
		"database.host":     "localhost",
		"database.port":     "5432",
		"database.name":     "postgres",
		"database.sslmode":  "disable",
		// Memcached
		"memcached.servers": []string{"localhost:11211"},
		// Server
		"server.host":             "localhost",
		"server.port":             "7070",
		"server.dir":              "../",
		"server.tls.keyfile":      "server.key",
		"server.tls.certfile":     "server.crt",
		"server.timeout.read":     5,
		"server.timeout.write":    5,
		"server.timeout.shutdown": 5,
		// Email
		"email.host":     "smtp.default.com",
		"email.port":     "587",
		"email.sender":   "default@adak.com",
		"email.password": "default",
		"email.admins":   "../pkg/auth/",
		// Stripe
		"stripe.secretkey":    "sk_test_default",
		"stripe.logger.level": "4",
		// Token
		"token.secretkey": "secretkey",
	}

	envVars = map[string]string{
		// Admins
		"admin.emails": "ADMIN_EMAILS",
		// Database
		"database.username": "POSTGRES_USERNAME",
		"database.password": "POSTGRES_PASSWORD",
		"database.host":     "POSTGRES_HOST",
		"database.port":     "POSTGRES_PORT",
		"database.name":     "POSTGRES_DB",
		"database.sslmode":  "POSTGRES_SSL",
		// Memcached
		"memcached.servers": "MEMCACHED_SERVERS",
		// Server
		"server.host":             "SV_HOST",
		"server.port":             "SV_PORT",
		"server.dir":              "SV_DIR",
		"server.tls.keyfile":      "SV_TLS_KEYFILE",
		"server.tls.certfile":     "SV_TLS_CERTFILE",
		"server.timeout.read":     "SV_TIMEOUT_READ",
		"server.timeout.write":    "SV_TIMEOUT_WRITE",
		"server.timeout.shutdown": "SV_TIMEOUT_SHUTDOWN",
		// Email
		"email.host":     "EMAIL_HOST",
		"email.port":     "EMAIL_PORT",
		"email.sender":   "EMAIL_SENDER",
		"email.password": "EMAIL_PASSWORD",
		// Stripe
		"stripe.secretkey":    "STRIPE_SECRET_KEY",
		"stripe.logger.level": "STRIPE_LOGGER_LEVEL",
		// Token
		"token.secretkey": "TOKEN_SECRET_KEY",
		// Google
		"google.client.id":     "GOOGLE_CLIENT_ID",
		"google.client.secret": "GOOGLE_CLIENT_SECRET",
		// Github
		"github.client.id":     "GITHUB_CLIENT_ID",
		"github.client.secret": "GITHUB_CLIENT_SECRET",
	}
)
