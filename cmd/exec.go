package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/embtom/systemd-docker/lib"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Version: "1.0.0",
		Use:     "systemd-docker [flags] -- [docker flags]",
		Short:   "systemd-docker is a wrapper for 'docker run' so that you can sanely run Docker containers under systemd.",
		Long: `systemd-docker is a wrapper for 'docker run' so that you can sanely run Docker containers under systemd.
Using this wrapper you can manage containers through systemctl or the docker CLI.
Additionally you can leverage all the cgroup functionality of systemd and systemd-notify.`,
		Example: `systemd-docker --pid-file=/tmp/registry-pid --networks mqtt_proxy,prometheus_proxy:192.168.98.4 -- 
    --name registry --publish 5000:5000 --env 'REGISTRY_STORAGE_FILESYSTEM_ROOTDIRECTORY=/data' registry:latest`,
		RunE:                  run,
		DisableFlagsInUseLine: true,
	}

	c = &lib.Context{
		Log: lib.NewLogger(),
	}
)

func init() {

	vt := fmt.Sprintf("%s version {{printf \"%%s\" .Version}}\n", "sytemd-docker")
	rootCmd.SetVersionTemplate(vt)
	rootCmd.Flags().StringVarP(&c.PidFile, "pid-file", "p", "", "pipe file")
	rootCmd.Flags().BoolVarP(&c.Logs, "logs", "l", true, "pipe logs")
	rootCmd.Flags().BoolVarP(&c.Notify, "notify", "n", false, "setup systemd notify for container")
	rootCmd.Flags().BoolVarP(&c.Env, "env", "e", false, "inherit environment variable")
	rootCmd.Flags().StringSliceVarP(&c.Cgroups, "cgroups", "c", []string{}, "cgroups to take ownership of or 'all' for all cgroups available")

}

func run(_ *cobra.Command, args []string) error {

	runArgs := make([]string, 0, len(args))
	for i, arg := range args {
		switch {
		case arg == "--rm":
			c.Rm = true
		case strings.HasPrefix(arg, "--name"):
			if strings.Contains(arg, "=") {
				c.Name = strings.SplitN(arg, "=", 2)[1]
			} else if len(args) > i+1 {
				c.Name = args[i+1]
			}
		}
		runArgs = append(runArgs, arg)
	}

	c.Args = runArgs
	for _, val := range c.Cgroups {
		if val == "all" {
			c.Cgroups = nil
			c.AllCgroups = true
			break
		}
	}

	lib.SetupSystemdNotify(c)

	if c.Env {
		for _, val := range os.Environ() {
			if !strings.HasPrefix(val, "HOME=") && !strings.HasPrefix(val, "PATH=") {
				c.Args = append(c.Args, "-e", val)
			}
		}
	}

	err := lib.RunContainer(c)
	if err != nil {
		return err
	}

	err = lib.Notify(c)
	if err != nil {
		return err
	}

	err = lib.PidFile(c)
	if err != nil {
		return err
	}

	return nil
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		c.Log.Fatalf(err.Error())
		os.Exit(1)
	}
}
