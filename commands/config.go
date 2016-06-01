package commands

import (
	"errors"
	"fmt"
	"github.com/ContainX/depcon/cliconfig"
	"github.com/ContainX/depcon/pkg/cli"
	"github.com/ContainX/depcon/utils"
	"github.com/spf13/cobra"
	"io"
	"text/template"
)

const (
	T_CONFIG_ENV = `
{{ "NAME" }}	{{ "TYPE" }}	{{ "ENDPOINT" }}	{{ "AUTH" }}	{{ "DEFAULT" }}
{{ range . }}{{ .Name }}	{{ .EnvType }}	{{ .HostURL }}	{{ .Auth | boolToYesNo }}	{{ .Default | defaultEnvToStr }}
{{end}}`
)

type ConfigEnvironments struct {
	DefaultEnv string
	Envs       map[string]*cliconfig.ConfigEnvironment
}

type EnvironmentSummary struct {
	Name    string
	EnvType string
	HostURL string
	Auth    bool
	Default bool
}

var ValidOutputs []string = []string{"json", "yaml", "column"}
var ErrInvalidOutputFormat = errors.New("Invalid Output specified. Must be 'json','yaml' or 'column'")
var ErrInvalidRootOption = errors.New("Invalid chroot option specified. Must be 'true' or 'false'")

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "DepCon configuration",
	Long: `Manage DepCon configuration (eg. default, list, adding and removing of environments, default output, service rooting)

See config's subcommands for available choices`,
}

var configEnvCmd = &cobra.Command{
	Use:   "env",
	Short: "Configuration environments define remote Marathon and other supported services by a name. ",
	Long: `Manage configuration environments (eg. default, list, adding and removing of environments)

See env's subcommands for available choices`,
}

var configRemoveCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Remove a defined environment by it's [name]",
	Run: func(cmd *cobra.Command, args []string) {
		if cli.EvalPrintUsage(cmd.Usage, args, 1) {
			return
		}
		_, err := configFile.GetEnvironment(args[0])
		if err != nil {
			cli.Output(nil, err)
		} else {
			err := configFile.RemoveEnvironment(args[0], false)
			if err != nil {
				cli.Output(nil, err)
			}
		}
	},
}

var configAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Adds a new environment",
	Run: func(cmd *cobra.Command, args []string) {
		configFile.AddEnvironment()
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List current environments",
	Run: func(cmd *cobra.Command, args []string) {
		ce := &ConfigEnvironments{DefaultEnv: configFile.DefaultEnv, Envs: configFile.Environments}
		template := templateFor(T_CONFIG_ENV, ce.toEnvironmentMap())
		cli.Output(template, nil)
	},
}

var configDefaultCmd = &cobra.Command{
	Use:   "default [name]",
	Short: "Sets the default environment [name] to use (eg. -e envname can be eliminated when set and using default)",
	Run: func(cmd *cobra.Command, args []string) {
		if cli.EvalPrintUsage(cmd.Usage, args, 1) {
			return
		}
		err := configFile.SetDefaultEnvironment(args[0])
		if err != nil {
			cli.Output(nil, err)
		} else {
			fmt.Printf("\nDefault environment is now '%s'\n\n", args[0])
		}
	},
}

var configOutputCmd = &cobra.Command{
	Use:   "output [json | column]",
	Short: "Sets the default output to use when -o flag is not specified.  Values are 'json, 'yaml' or 'column'",
	Run: func(cmd *cobra.Command, args []string) {
		if cli.EvalPrintUsage(cmd.Usage, args, 1) {
			return
		}
		format := args[0]

		if utils.Contains(ValidOutputs, format) {
			configFile.Format = format
			configFile.Save()
			fmt.Printf("\nDefault cli.Output is now '%s'\n\n", format)
		} else {
			cli.Output(nil, ErrInvalidOutputFormat)
		}
	},
}

var configRootServiceCmd = &cobra.Command{
	Use:   "chroot [true | false]",
	Short: "If true DepCon will root the service based on the current configuration environment. (eg. ./depcon mar app would be ./depcon app)",
	Run: func(cmd *cobra.Command, args []string) {
		if cli.EvalPrintUsage(cmd.Usage, args, 1) {
			return
		}
		chroot := args[0]
		if chroot == "true" || chroot == "false" {
			rootBool := chroot == "true"
			configFile.RootService = rootBool
			configFile.Save()
			if rootBool {
				fmt.Println("\nService rooting is now enabled\n")
			} else {
				fmt.Println("\nService rooting is now disabled\n")
			}
		} else {
			cli.Output(nil, ErrInvalidRootOption)
		}
	},
}

var configRenameCmd = &cobra.Command{
	Use:   "rename [oldName] [newName]",
	Short: "Renames an environment from specified [oldName] to the [newName]",
	Run: func(cmd *cobra.Command, args []string) {
		if cli.EvalPrintUsage(cmd.Usage, args, 2) {
			return
		}
		err := configFile.RenameEnvironment(args[0], args[1])
		if err != nil {
			cli.Output(nil, err)
		} else {
			fmt.Printf("\nEnvironment '%s' has been renamed to '%s'\n\n", args[0], args[1])
		}
	},
}

func init() {
	configEnvCmd.AddCommand(configListCmd, configDefaultCmd, configRenameCmd, configAddCmd, configRemoveCmd)
	configCmd.AddCommand(configEnvCmd, configOutputCmd, configRootServiceCmd)
}

type ConfigTemplate struct {
	cli.FormatData
}

func templateFor(template string, data interface{}) ConfigTemplate {
	return ConfigTemplate{cli.FormatData{Template: template, Data: data, Funcs: buildFuncMap()}}
}

func (d ConfigTemplate) ToColumns(output io.Writer) error {
	return d.FormatData.ToColumns(output)
}

func (d ConfigTemplate) Data() cli.FormatData {
	return d.FormatData
}

func (e ConfigEnvironments) toEnvironmentMap() []*EnvironmentSummary {

	arr := []*EnvironmentSummary{}

	for k, v := range e.Envs {
		var sc cliconfig.ServiceConfig
		switch v.EnvironmentType() {
		case cliconfig.TypeMarathon:
			sc = *v.Marathon
		}
		arr = append(arr, &EnvironmentSummary{
			Name:    k,
			EnvType: v.EnvironmentType(),
			HostURL: sc.HostUrl,
			Auth:    sc.Username != "",
			Default: k == e.DefaultEnv,
		})
	}
	return arr
}

func buildFuncMap() template.FuncMap {
	funcMap := template.FuncMap{
		"defaultEnvToStr": defaultEnvToStr,
	}
	return funcMap
}

func defaultEnvToStr(b bool) string {
	if b {
		return "true"
	}
	return "-"
}
