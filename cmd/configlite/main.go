package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/dgsb/configlite"

	"github.com/alecthomas/kong"
)

type CommonConfig struct {
	Database string `long:"db" default:"${default_config_file}" description:"the configuration database file to use"`
}

func (cfg *CommonConfig) GetRepo() *configlite.Repository {
	repo, err := configlite.New(cfg.Database)
	if err != nil {
		log.Fatal("cannot open configuration database", err)
	}
	return repo
}

type ListAppCmd struct {
	CommonConfig `embed:""`
}

func (cmd *ListAppCmd) Run() error {
	repo, err := configlite.New(cmd.Database)
	if err != nil {
		return fmt.Errorf("cannot open configuration database: %w", err)
	}

	apps, err := repo.GetApps()
	if err != nil {
		return fmt.Errorf("cannot list registered applications: %w", err)
	}

	for _, a := range apps {
		fmt.Println(a)
	}
	return nil
}

type ListConfigsCmd struct {
	CommonConfig `embed:""`
	Format       string `short:"f" default:"json" enum:"json,text" description:"the format to display the configuration in"`
	Application  string `long:"app" arg:"" description:"the application whose configuration has to be displayed"`
}

func (cmd *ListConfigsCmd) Run() error {
	repo, err := configlite.New(cmd.Database)
	if err != nil {
		return fmt.Errorf("cannot open configuration database: %w", err)
	}

	configs, err := repo.GetConfigs(cmd.Application)
	if err != nil {
		return fmt.Errorf("cannot get configuration for application %s: %w", cmd.Application, err)
	}

	switch cmd.Format {
	case "json":
		jsonConfig, err := json.MarshalIndent(configs, "", "    ")
		if err != nil {
			return fmt.Errorf("json formatting configs: %w", err)
		}
		fmt.Println(string(jsonConfig))
	case "text":
		for k, v := range configs {
			fmt.Println(k, v)
		}
	default:
		return fmt.Errorf("cannot list configs for application: unknow format %s", cmd.Format)
	}
	return nil
}

type UpsertConfigCmd struct {
	CommonConfig  `embed:""`
	Application   string `arg:""`
	Configuration string `arg:""`
	Value         string `arg:""`
}

func (cmd *UpsertConfigCmd) Run() error {
	repo := cmd.GetRepo()
	return repo.UpsertConfig(cmd.Application, cmd.Configuration, cmd.Value)
}

type DeleteConfigCmd struct {
	CommonConfig  `embed:""`
	LikePattern   bool   `long:"like" short:"l" default:"false" help:"the configuration name is going to be used in an sql like clause"`
	Application   string `arg:""`
	Configuration string `arg:""`
}

func (cmd *DeleteConfigCmd) Run() error {
	return cmd.GetRepo().DeleteConfig(cmd.Application, cmd.Configuration, cmd.LikePattern)
}

func main() {
	var cli struct {
		ListApp      ListAppCmd      `cmd:"" aliases:"la"`
		ListConfigs  ListConfigsCmd  `cmd:"" aliases:"lc"`
		UpsertConfig UpsertConfigCmd `cmd:"" aliases:"uc"`
		DeleteConfig DeleteConfigCmd `cmd:"" aliases:"dc"`
	}

	ctx := kong.Parse(&cli, kong.Vars{"default_config_file": configlite.DefaultConfigurationFile()})
	if err := ctx.Run(); err != nil {
		log.Fatal("cannot run command", err)
	}
}
