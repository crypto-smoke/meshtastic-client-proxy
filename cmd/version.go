package cmd

import (
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"runtime/debug"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print current version",
	Long:  `Print the current version, commit hash, and build date of the executable`,
	Run: func(cmd *cobra.Command, args []string) {
		if v, _ := cmd.Flags().GetBool("verbose"); v {
			fmt.Printf("%s %s %s", version, commit, date)
			return
		}
		if v, _ := cmd.Flags().GetBool("debug"); v {
			dbg, ok := debug.ReadBuildInfo()
			if !ok {
				log.Error("failed getting debug build info")
				return
			}
			fmt.Println(dbg.String())
			return
		}
		fmt.Printf("%s", version)

	},
}

func init() {
	rootCmd.AddCommand(versionCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	versionCmd.PersistentFlags().BoolP("verbose", "v", false, "print additional version information")
	versionCmd.PersistentFlags().BoolP("debug", "d", false, "print go build debug information")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// versionCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
