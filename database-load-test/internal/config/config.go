package config

import (
	"fmt"
	"os"
	"time"

	"github.com/AntonChubarov/surrealdb-exporter-utils/database-load-test/internal/domain"
	"gopkg.in/yaml.v3"
)

// Config represents the application configuration loaded from YAML
type Config struct {
	LoadTest LoadTestConfig `yaml:"load_test"`

	MetricsDisplay MetricsDisplayConfig `yaml:"metrics_display"`

	SurrealDB SurrealDBConfig `yaml:"surrealdb"`

	Executables ExecutablesConfig `yaml:"executables"`
}

type LoadTestConfig struct {
	DurationSeconds int `yaml:"duration_seconds"`
}

type MetricsDisplayConfig struct {
	DisplayIntervalMs int `yaml:"display_interval_ms"`
}

type SurrealDBConfig struct {
	URL       string `yaml:"url"`
	Namespace string `yaml:"namespace"`
	Database  string `yaml:"database"`
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
}

type ExecutablesConfig struct {
	Users   UsersConfig   `yaml:"users"`
	Follows FollowsConfig `yaml:"follows"`
}

type UsersConfig struct {
	Create   EventRate `yaml:"create"`
	Read     EventRate `yaml:"read"`
	Update   EventRate `yaml:"update"`
	Delete   EventRate `yaml:"delete"`
	GetAll   EventRate `yaml:"get_all"`
	PageSize int       `yaml:"page_size"`
}

type FollowsConfig struct {
	Create           EventRate `yaml:"create"`
	GetUserFollows   EventRate `yaml:"get_user_follows"`
	GetUserFollowers EventRate `yaml:"get_user_followers"`
	CommonFollows    EventRate `yaml:"common_follows"`
	CommonFollowers  EventRate `yaml:"common_followers"`
	Delete           EventRate `yaml:"delete"`
	PageSize         int       `yaml:"page_size"`
}

type EventRate struct {
	StartDelay       float64 `yaml:"start_delay_seconds"`
	EventsPerMinute  float64 `yaml:"events_per_minute"`
	VariancePercents float64 `yaml:"variance_percents"`
}

func (er *EventRate) ToDomain() domain.EventRate {
	return domain.EventRate{
		StartDelay:       time.Duration(er.StartDelay * float64(time.Second)),
		EventsPerMinute:  er.EventsPerMinute,
		VariancePercents: er.VariancePercents,
	}
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// AppConfig implementation

func (c *Config) LoadTestDuration() time.Duration {
	return time.Duration(c.LoadTest.DurationSeconds) * time.Second
}

// MetricsConfig implementation

func (c *Config) MetricsDisplayInterval() time.Duration {
	return time.Duration(c.MetricsDisplay.DisplayIntervalMs) * time.Millisecond
}

// SurrealDBConfig implementation

func (c *Config) SurrealURL() string {
	return c.SurrealDB.URL
}

func (c *Config) SurrealNamespace() string {
	return c.SurrealDB.Namespace
}

func (c *Config) SurrealDatabase() string {
	return c.SurrealDB.Database
}

func (c *Config) SurrealUsername() string {
	return c.SurrealDB.Username
}

func (c *Config) SurrealPassword() string {
	return c.SurrealDB.Password
}

// UsersConfig implementation

func (c *Config) UsersCreateParams() domain.EventRate {
	return c.Executables.Users.Create.ToDomain()
}

func (c *Config) UsersReadParams() domain.EventRate {
	return c.Executables.Users.Read.ToDomain()
}

func (c *Config) UsersUpdateParams() domain.EventRate {
	return c.Executables.Users.Update.ToDomain()
}

func (c *Config) UsersDeleteParams() domain.EventRate {
	return c.Executables.Users.Delete.ToDomain()
}

func (c *Config) UsersGetAllParams() domain.EventRate {
	return c.Executables.Users.GetAll.ToDomain()
}

func (c *Config) UsersPageSize() int {
	return c.Executables.Users.PageSize
}

// FollowsConfig implementation

func (c *Config) FollowsCreateParams() domain.EventRate {
	return c.Executables.Follows.Create.ToDomain()
}

func (c *Config) FollowsGetUserFollowsParams() domain.EventRate {
	return c.Executables.Follows.GetUserFollows.ToDomain()
}

func (c *Config) FollowsGetUserFollowersParams() domain.EventRate {
	return c.Executables.Follows.GetUserFollowers.ToDomain()
}

func (c *Config) FollowsCommonFollowsParams() domain.EventRate {
	return c.Executables.Follows.CommonFollows.ToDomain()
}

func (c *Config) FollowsCommonFollowersParams() domain.EventRate {
	return c.Executables.Follows.CommonFollowers.ToDomain()
}

func (c *Config) FollowsDeleteParams() domain.EventRate {
	return c.Executables.Follows.Delete.ToDomain()
}

func (c *Config) FollowsPageSize() int {
	return c.Executables.Follows.PageSize
}
