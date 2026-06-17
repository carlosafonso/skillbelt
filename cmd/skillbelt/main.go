package main

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/elfonsi/skillbelt/internal/config"
	"github.com/elfonsi/skillbelt/internal/manager"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "skillbelt: %v\n", err)
		os.Exit(1)
	}
	m := manager.New(cfg)

	root := &cobra.Command{
		Use:   "skillbelt",
		Short: "Skill manager for agent harnesses",
		Long: `skillbelt installs agent skills by cloning git repos and symlinking
them into your harness skills directory (~/.agents/skills/ by default).`,
	}

	// install
	root.AddCommand(&cobra.Command{
		Use:   "install <url>",
		Short: "Clone and install a skill",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return m.Install(args[0])
		},
	})

	// remove
	removeCmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a skill (keeps the local clone by default)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			purge, _ := cmd.Flags().GetBool("purge")
			return m.Remove(args[0], purge)
		},
	}
	removeCmd.Flags().Bool("purge", false, "also delete the local clone")
	root.AddCommand(removeCmd)

	// update
	root.AddCommand(&cobra.Command{
		Use:   "update [name]",
		Short: "Pull the latest changes for a skill (or all skills if no name given)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) > 0 {
				name = args[0]
			}
			return m.Update(name)
		},
	})

	// list
	root.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List installed skills",
		RunE: func(cmd *cobra.Command, args []string) error {
			entries, err := m.List()
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				fmt.Println("no skills installed")
				return nil
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tURL\tINSTALLED\tLINKED")
			for _, e := range entries {
				linked := "yes"
				if !e.Linked {
					linked = "BROKEN"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					e.Name, e.URL,
					e.InstalledAt.Local().Format(time.DateOnly),
					linked,
				)
			}
			return w.Flush()
		},
	})

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
