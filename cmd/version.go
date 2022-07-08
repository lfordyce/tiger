package cmd

import (
	"fmt"

	"github.com/lfordyce/tiger/pkg/consts"

	"github.com/spf13/cobra"
)

func getCmdVersion(globalState *globalState) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show application version",
		Long:  `Show the application version and exit.`,
		Run: func(_ *cobra.Command, _ []string) {
			printToStdout(globalState, fmt.Sprintf("tiger v%s\n", consts.FullVersion()))
		},
	}
}
