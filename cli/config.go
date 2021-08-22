package cli

import (
	"os"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/mcuadros/ofelia/core"
	"github.com/mcuadros/ofelia/middlewares"
	logging "github.com/op/go-logging"

	defaults "github.com/mcuadros/go-defaults"
	gcfg "gopkg.in/gcfg.v1"
)

const (
	logFormat     = "%{time} %{color} %{shortfile} ▶ %{level}%{color:reset} %{message}"
	jobExec       = "job-exec"
	jobRun        = "job-run"
	jobServiceRun = "job-service-run"
	jobLocal      = "job-local"
)

var IsDockerEnv bool

// Config contains the configuration
type Config struct {
	Global struct {
		middlewares.SlackConfig `mapstructure:",squash"`
		middlewares.SaveConfig  `mapstructure:",squash"`
		middlewares.MailConfig  `mapstructure:",squash"`
	}
	ExecJobs    map[string]*ExecJobConfig    `gcfg:"job-exec" mapstructure:"job-exec,squash"`
	RunJobs     map[string]*RunJobConfig     `gcfg:"job-run" mapstructure:"job-run,squash"`
	ServiceJobs map[string]*RunServiceConfig `gcfg:"job-service-run" mapstructure:"job-service-run,squash"`
	LocalJobs   map[string]*LocalJobConfig   `gcfg:"job-local" mapstructure:"job-local,squash"`
}

// BuildFromDockerLabels builds a scheduler using the config from a docker labels
func BuildFromDockerLabels() (*core.Scheduler, error) {
	config := &Config{}

	dockerClient, err := config.buildDockerClient()
	if err != nil {
		return nil, err
	}

	labels, err := getLabels(dockerClient)
	if err != nil {
		return nil, err
	}

	if err := config.buildFromDockerLabels(labels); err != nil {
		return nil, err
	}

	return config.build()
}

// BuildFromFile builds a scheduler using the config from a file
func BuildFromFile(filename string) (*core.Scheduler, error) {
	config := &Config{}
	if err := gcfg.ReadFileInto(config, filename); err != nil {
		return nil, err
	}

	return config.build()
}

// BuildFromString builds a scheduler using the config from a string
func BuildFromString(configString string) (*core.Scheduler, error) {
	config := &Config{}
	if err := gcfg.ReadStringInto(config, configString); err != nil {
		return nil, err
	}

	return config.build()
}

func (config *Config) build() (*core.Scheduler, error) {
	defaults.SetDefaults(config)

	dockerClient, err := config.buildDockerClient()
	if err != nil {
		return nil, err
	}

	sched := core.NewScheduler(config.buildLogger())
	config.buildSchedulerMiddlewares(sched)

	for name, job := range config.ExecJobs {
		defaults.SetDefaults(job)

		job.Client = dockerClient
		job.Name = name
		job.buildMiddlewares()
		sched.AddJob(job)
	}

	for name, job := range config.RunJobs {
		defaults.SetDefaults(job)

		job.Client = dockerClient
		job.Name = name
		job.buildMiddlewares()
		sched.AddJob(job)
	}

	for name, job := range config.LocalJobs {
		defaults.SetDefaults(job)

		job.Name = name
		job.buildMiddlewares()
		sched.AddJob(job)
	}

	for name, job := range config.ServiceJobs {
		defaults.SetDefaults(job)
		job.Name = name
		job.Client = dockerClient
		job.buildMiddlewares()
		sched.AddJob(job)
	}

	return sched, nil
}

func (*Config) buildDockerClient() (*docker.Client, error) {
	dockerClient, err := docker.NewClientFromEnv()
	if err != nil {
		return nil, err
	}

	return dockerClient, nil
}

func (config *Config) buildLogger() core.Logger {
	stdout := logging.NewLogBackend(os.Stdout, "", 0)
	// Set the backends to be used.
	logging.SetBackend(stdout)
	logging.SetFormatter(logging.MustStringFormatter(logFormat))

	return logging.MustGetLogger("ofelia")
}

func (config *Config) buildSchedulerMiddlewares(sched *core.Scheduler) {
	global := &config.Global
	sched.Use(middlewares.NewSlack(&global.SlackConfig))
	sched.Use(middlewares.NewSave(&global.SaveConfig))
	sched.Use(middlewares.NewMail(&global.MailConfig))
}

// ExecJobConfig contains all configuration params needed to build a ExecJob
type ExecJobConfig struct {
	core.ExecJob              `mapstructure:",squash"`
	middlewares.OverlapConfig `mapstructure:",squash"`
	middlewares.SlackConfig   `mapstructure:",squash"`
	middlewares.SaveConfig    `mapstructure:",squash"`
	middlewares.MailConfig    `mapstructure:",squash"`
}

func (config *ExecJobConfig) buildMiddlewares() {
	job := &config.ExecJob
	job.Use(middlewares.NewOverlap(&config.OverlapConfig))
	job.Use(middlewares.NewSlack(&config.SlackConfig))
	job.Use(middlewares.NewSave(&config.SaveConfig))
	job.Use(middlewares.NewMail(&config.MailConfig))
}

// RunServiceConfig contains all configuration params needed to build a RunJob
type RunServiceConfig struct {
	core.RunServiceJob        `mapstructure:",squash"`
	middlewares.OverlapConfig `mapstructure:",squash"`
	middlewares.SlackConfig   `mapstructure:",squash"`
	middlewares.SaveConfig    `mapstructure:",squash"`
	middlewares.MailConfig    `mapstructure:",squash"`
}

type RunJobConfig struct {
	core.RunJob               `mapstructure:",squash"`
	middlewares.OverlapConfig `mapstructure:",squash"`
	middlewares.SlackConfig   `mapstructure:",squash"`
	middlewares.SaveConfig    `mapstructure:",squash"`
	middlewares.MailConfig    `mapstructure:",squash"`
}

func (config *RunJobConfig) buildMiddlewares() {
	job := &config.RunJob
	job.Use(middlewares.NewOverlap(&config.OverlapConfig))
	job.Use(middlewares.NewSlack(&config.SlackConfig))
	job.Use(middlewares.NewSave(&config.SaveConfig))
	job.Use(middlewares.NewMail(&config.MailConfig))
}

// LocalJobConfig contains all configuration params needed to build a RunJob
type LocalJobConfig struct {
	core.LocalJob             `mapstructure:",squash"`
	middlewares.OverlapConfig `mapstructure:",squash"`
	middlewares.SlackConfig   `mapstructure:",squash"`
	middlewares.SaveConfig    `mapstructure:",squash"`
	middlewares.MailConfig    `mapstructure:",squash"`
}

func (config *LocalJobConfig) buildMiddlewares() {
	job := &config.LocalJob
	job.Use(middlewares.NewOverlap(&config.OverlapConfig))
	job.Use(middlewares.NewSlack(&config.SlackConfig))
	job.Use(middlewares.NewSave(&config.SaveConfig))
	job.Use(middlewares.NewMail(&config.MailConfig))
}

func (config *RunServiceConfig) buildMiddlewares() {
	job := &config.RunServiceJob
	job.Use(middlewares.NewOverlap(&config.OverlapConfig))
	job.Use(middlewares.NewSlack(&config.SlackConfig))
	job.Use(middlewares.NewSave(&config.SaveConfig))
	job.Use(middlewares.NewMail(&config.MailConfig))
}
