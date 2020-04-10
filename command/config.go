package command

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Set and get gh settings",
	Long: `
	TODO
`,
}

var configGetCmd = &cobra.Command{
	Use:   "get",
	Short: "TODO",
	RunE:  configGet,
}

var configSetCmd = &cobra.Command{
	Use:   "set",
	Short: "TODO",
	RunE:  configSet,
}

func configGet(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("need to pass a key")
	}
	key := args[0]
	ctx := contextForCommand(cmd)

	cfg, err := ctx.Config()
	if err != nil {
		return err
	}

	val, err := cfg.Get(key)
	if err != nil {
		return err
	}

	if val != "" {
		fmt.Println(val)
	}

	return nil
}

func configSet(cmd *cobra.Command, args []string) error {
	if len(args) != 2 {
		return errors.New("need to pass a key and a value")
	}

	key := args[0]
	value := args[1]

	ctx := contextForCommand(cmd)

	cfg, err := ctx.Config()
	if err != nil {
		return err
	}
	// TODO NOTES
	// to write a config back out you can do yaml.Marshal(&root). it will serialize the root itself if
	// you don't pass a pointer. to mutate the parsed yaml, root.Content[0].Content[i+1].Value = "LOL"

	err = cfg.Set(key, value)
	if err != nil {
		return fmt.Errorf("failed to set %q to %q: %s", key, value, err)
	}

	return nil
}
