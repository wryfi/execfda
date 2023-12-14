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
		Use:   "rwx",
		Short: "rwx: execute commands allowed by configuration file",
		Long:  `rwx is a wrapper that executes commands specified in a configuration file`,
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
		Short: "configure rwx",
		Long:  `configure avaiable commands for rwx`,
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
		Long:  `adds the specified command to the rwx allowed list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkUserRoot(); err != nil {
				return err
			}
			newCommand := strings.Join(args, " ")
			allowed := viper.GetStringSlice("allowed")
			allowed = append(allowed, newCommand)
			viper.Set("allowed", allowed)
			if err := writeConfig(); err != nil {
				return err
			}
			return nil
		},
	}
	return command
}

func ConfigCreateCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "create",
		Short: "create empty rwx config",
		Long:  `creates an empty rwx configuration`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkUserRoot(); err != nil {
				return err
			}
			return writeConfig()
		},
	}
	return command
}

func ConfigDeleteAllowedCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "delete",
		Short: "delete a command from the allowed list",
		Long:  `deletes the specified command from the rwx allowed list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkUserRoot(); err != nil {
				return err
			}
			toDelete := strings.Join(args, " ")
			allowed := viper.GetStringSlice("allowed")
			for idx, existing := range allowed {
				existingCmd := strings.Join(strings.Fields(existing), " ")
				if toDelete == existingCmd {
					allowed = append(allowed[:idx], allowed[idx+1:]...)
				}
			}
			viper.Set("allowed", allowed)
			if err := writeConfig(); err != nil {
				return err
			}
			return nil
		},
	}
	return command
}

func ConfigGetCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "get",
		Short: "get rwx config",
		Long:  `print the rwx configuration to the screen`,
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

func InitConfig() error {
	viper.SetConfigFile("/private/etc/rwx/config.yml")
	viper.SetDefault("allowed", []string{})
	if err := viper.ReadInConfig(); err != nil {
		if strings.Contains(err.Error(), "no such file or directory") {
			return nil
		} else {
			fmt.Printf("Error: failed to read configuration - %s", err)
			return err
		}
	}
	return nil
}

func Execute(cmd *cobra.Command) error {
	if err := InitConfig(); err != nil {
		return err
	}
	if err := checkFileAccess(viper.ConfigFileUsed()); err != nil {
		fmt.Printf("Error: %s \n", err)
		return err
	}
	cmd.SetOut(os.Stdout)
	return cmd.Execute()
}

func checkUserRoot() error {
	currentUser, err := user.Current()
	if err != nil {
		return errors.New("failure getting info on current user!")
	}
	if currentUser.Username == "root" {
		return nil
	}
	return errors.New("you must be root to perform this operation!")
}

func checkFileAccess(cfgfile string) error {
	stat, err := os.Stat(cfgfile)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	mode, sys := stat.Mode(), stat.Sys().(*syscall.Stat_t)
	if mode != 0644 {
		return errors.New(fmt.Sprintf(
			"invalid permissions on %s (expected 644, got %o)!", cfgfile, mode),
		)
	}
	if sys.Uid != 0 || sys.Gid != 0 {
		return errors.New(fmt.Sprintf("file %s is not owned by root:wheel!", cfgfile))
	}
	return nil
}

func writeConfig() error {
	dir := filepath.Dir(viper.ConfigFileUsed())
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	if err := viper.WriteConfig(); err != nil {
		return err
	}
	if err := os.Chown(viper.ConfigFileUsed(), 0, 0); err != nil {
		return err
	}
	if err := os.Chmod(viper.ConfigFileUsed(), 0644); err != nil {
		return err
	}
	return nil
}
