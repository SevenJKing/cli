package command

import (
	"fmt"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(cfgCmd)
}

var cfgCmd = &cobra.Command{
	Use:   "config",
	Short: "Set and get gh settings",
	RunE:  cfg,
	Long: `
	TODO
`,
}

func cfg(cmd *cobra.Command, args []string) error {
	// TODO decide on top level behavior. I'm thinking either just dump the config or go into an
	// interactive mode where config keys can be searched through.

	// otherwise, we'll dispatch on arg length for now.

	// TODO i kind of want set to be its own command.

	// TODO just doing this to force config parsing:
	ctx := contextForCommand(cmd)
	al, err := ctx.AuthLogin()
	if err != nil {
		return err
	}

	fmt.Println(al)

	switch len(args) {
	case 0:
		cfgInteractive(cmd)
	case 1:
		cfgGet(cmd, args[0])
	case 2:
		cfgSet(cmd, args[0], args[1])
	default:
		fmt.Println("i have no idea what you are saying to me")
	}

	return nil
}

func cfgInteractive(cmd *cobra.Command) error {
	fmt.Println("cool interactive thing")
	return nil
}

func cfgGet(cmd *cobra.Command, key string) error {
	fmt.Println("getting", key)
	return nil
}

func cfgSet(cmd *cobra.Command, key, value string) error {
	fmt.Println("setting", key, "to", value)

	return nil
}
