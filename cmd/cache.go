package cmd

import (
	"fmt"

	gh "github.com/nickhudkins/gh-stats/github"
	"github.com/nickhudkins/gh-stats/render"
	"github.com/spf13/cobra"
)

var cacheCmd = &cobra.Command{
	Use:         "cache",
	Short:       "Show cache file path or clear it (--clear)",
	Annotations: map[string]string{"group": groupData},
	RunE: func(cmd *cobra.Command, args []string) error {
		clear, _ := cmd.Flags().GetBool("clear")
		if clear {
			if err := gh.ClearCache(); err != nil {
				return err
			}
			render.Bold.Println("Cache cleared.")
			return nil
		}
		fmt.Println(gh.CachePath())
		return nil
	},
}

func init() {
	cacheCmd.Flags().Bool("clear", false, "Delete the cache file")
	rootCmd.AddCommand(cacheCmd)
}
