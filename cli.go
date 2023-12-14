package main

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"
)

func MainCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "execfda",
		Short: "execfda: execute command with full disk access",
		Long:  `execfda executes configured commands with full disk access`,
		Args:  cobra.ArbitraryArgs,
	}
	command.AddCommand(ExecCommand())
	command.AddCommand(ConfigCommand())
	return command
}

func ExecCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "exec",
		Short: "execute an allowed command",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			allowed := viper.GetStringSlice("allowed")
			for _, cmd := range allowed {
				cmdNormalized := strings.Join(strings.Fields(cmd), " ")
				if cmdNormalized == strings.Join(args, " ") {
					binary, lookErr := exec.LookPath(args[0])
					if lookErr != nil {
						return errors.New("error finding command on path!")
					}
					env := os.Environ()
					execErr := syscall.Exec(binary, args, env)
					if execErr != nil {
						return errors.New("error running command!")
					}
				}
			}
			return errors.New("command not allowed!")
		},
	}
	return command
}

func ConfigCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "config",
		Short: "configure execfda",
		Long:  `configure avaiable commands for execfda`,
	}
	command.AddCommand(ConfigAddAllowedCommand())
	command.AddCommand(ConfigCreateCommand())
	command.AddCommand(ConfigDeleteAllowedCommand())
	command.AddCommand(ConfigGetCommand())
	return command
}

func ConfigAddAllowedCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "add",
		Short: "add a command to the allowed list",
		Long:  `adds the specified command to the execfda allowed list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if checkRoot() != true {
				return errors.New("you must be root to manage the configuration!")
			}
			newCommand := strings.Join(args, " ")
			allowed := viper.GetStringSlice("allowed")
			allowed = append(allowed, newCommand)
			viper.Set("allowed", allowed)
			err := viper.WriteConfig()
			if err != nil {
				return errors.New(fmt.Sprintf("error writing config - %s", err))
			}
			return nil
		},
	}
	return command
}

func ConfigCreateCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "create",
		Short: "create empty execfda config",
		Long:  `creates an empty execfda configuration`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if checkRoot() != true {
				return errors.New("you must be root to manage the configuration!")
			}
			dir := filepath.Dir(viper.ConfigFileUsed())
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				err := os.MkdirAll(dir, 0755)
				if err != nil {
					return err
				}
			}
			if _, err := os.Stat(viper.ConfigFileUsed()); os.IsNotExist(err) {
				err := viper.WriteConfig()
				if err != nil {
					return err
				}
			}
			err := os.Chown(viper.ConfigFileUsed(), 0, 0)
			if err != nil {
				return err
			}
			err = os.Chmod(viper.ConfigFileUsed(), 0644)
			if err != nil {
				return err
			}
			return nil
		},
	}
	return command
}

func ConfigDeleteAllowedCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "delete",
		Short: "delete a command from the allowed list",
		Long:  `deletes the specified command from the execfda allowed list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if checkRoot() != true {
				return errors.New("you must be root to manage the configuration!")
			}
			del := strings.Join(args, " ")
			allowed := viper.GetStringSlice("allowed")
			for count, cmd := range allowed {
				normal := strings.Join(strings.Fields(cmd), " ")
				if del == normal {
					allowed = append(allowed[:count], allowed[count+1:]...)
				}
			}
			viper.Set("allowed", allowed)
			err := viper.WriteConfig()
			if err != nil {
				return errors.New(fmt.Sprintf("error writing config - %s", err))
			}
			return nil
		},
	}
	return command
}

func ConfigGetCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "get",
		Short: "get execfda config",
		Long:  `print the execfda configuration to the screen`,
		RunE: func(cmd *cobra.Command, args []string) error {
			config := viper.AllSettings()
			out, err := yaml.Marshal(config)
			if err != nil {
				return errors.New("error marshalling yaml!")
			}
			cmd.Printf("%s \n", out)
			return nil
		},
	}
	return command
}

func InitConfig() {
	viper.SetConfigFile("/private/etc/execfda/config.yml")
	viper.SetDefault("allowed", []string{})
	viper.ReadInConfig()
}

func Execute(cmd *cobra.Command) error {
	InitConfig()
	cmd.SetOut(os.Stdout)
	return cmd.Execute()
}

func checkRoot() bool {
	user, err := user.Current()
	if err != nil {
		return false
	}
	if user.Username == "root" {
		return true
	}
	return false
}
