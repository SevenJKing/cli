package context

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

func parseOrSetupConfigFile(fn string) (Config, error) {
	config, err := parseConfig(fn)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return setupConfigFile(fn)
	}
	return config, err
}

func ParseDefaultConfig() (Config, error) {
	return parseConfig(configFile())
}

func migrateConfig(cfgFilename string, data []byte, root *yaml.Node) (bool, error) {
	for _, v := range root.Content[0].Content {
		if v.Value == "hosts" {
			return false, nil
		}
	}

	fmt.Fprintln(os.Stderr, "info: migrating config from old to new format")

	newConfig := "hosts:\n"
	for _, line := range strings.Split(string(data), "\n") {
		newConfig += fmt.Sprintf("  %s\n", line)
	}

	err := os.Rename(cfgFilename, cfgFilename+".bak")
	if err != nil {
		return false, fmt.Errorf("failed to back up existing config: %s", err)
	}

	cfgFile, err := os.OpenFile(cfgFilename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return false, fmt.Errorf("failed to open new config file for writing: %s", err)
	}
	defer cfgFile.Close()

	n, err := cfgFile.WriteString(newConfig)
	if err == nil && n < len(newConfig) {
		err = io.ErrShortWrite
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

var readConfig = func(fn string) ([]byte, error) {
	f, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func parseConfigFile(fn string) ([]byte, *yaml.Node, error) {
	data, err := readConfig(fn)
	if err != nil {
		return nil, nil, err
	}

	var root yaml.Node
	err = yaml.Unmarshal(data, &root)
	if err != nil {
		return data, nil, err
	}
	if len(root.Content) < 1 {
		return data, &root, fmt.Errorf("malformed config")
	}
	if root.Content[0].Kind != yaml.MappingNode {
		return data, &root, fmt.Errorf("expected a top level map")
	}

	return data, &root, nil
}

func parseConfig(fn string) (Config, error) {
	_, root, err := parseConfigFile(fn)
	if err != nil {
		return nil, err
	}

	legacyConfig := true
	for _, v := range root.Content[0].Content {
		if v.Value == "hosts" {
			legacyConfig = false
		}
	}

	if legacyConfig {
		return NewLegacyConfig(root), nil
	}

	return NewConfig(root), nil
}
