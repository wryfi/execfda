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
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			cfgfile, _ := cmd.Flags().GetString("cfgfile")
			if err := initConfig(cfgfile); err != nil {
				fmt.Printf("Error: initialization failed - %s \n", err)
				os.Exit(1)
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmd.Help(); err != nil {
				return err
			}
			return nil
		},
	}
	command.PersistentFlags().StringP("cfgfile", "c", "/private/etc/rwx/config.yml", "configuration file")
	command.AddCommand(ExecCommand())
	command.AddCommand(ConfigCommand())
	return command
}

func ExecCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "exec",
		Aliases: []string{"ex", "run", "x"},
		Short:   "execute an allowed command",
		Args:    cobra.ArbitraryArgs,
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
		Use:     "configure",
		Aliases: []string{"cf", "cfg", "config"},
		Short:   "configure rwx",
		Long:    `configure available commands for rwx`,
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
		Use:     "delete",
		Aliases: []string{"del", "rm"},
		Short:   "delete a command from the allowed list",
		Long:    `deletes the specified command from the rwx allowed list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkUserRoot(); err != nil {
				return err
			}
			toDelete := strings.Join(args, " ")
			toAllow := []string{}
			allowed := viper.GetStringSlice("allowed")
			for _, existing := range allowed {
				existingCmd := strings.Join(strings.Fields(existing), " ")
				if toDelete != existingCmd {
					toAllow = append(toAllow, existingCmd)
				}
			}
			viper.Set("allowed", toAllow)
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

func initConfig(cfgfile string) error {
	viper.SetDefault("allowed", []string{})
	viper.SetConfigFile(cfgfile)
	if _, err := os.Stat(cfgfile); os.IsNotExist(err) {
		return nil
	}
	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Error reading config - %s", err)
		return err
	}
	if err := checkFileAccess(cfgfile); err != nil {
		return err
	}
	return nil
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
			"invalid permissions on %s (expected 0644, got 0%o)!", cfgfile, mode),
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
