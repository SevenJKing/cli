package context

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const defaultHostname = "github.com"

func parseOrSetupConfigFile(fn string) (*Config, error) {
	config, err := parseConfig(fn)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return setupConfigFile(fn)
	}
	return config, err
}

// ParseDefaultConfig reads the configuration file
func ParseDefaultConfig() (*Config, error) {
	return parseConfig(configFile())
}

type AuthConfig struct {
	User  string
	Token string
}

type HostConfig struct {
	Host  string
	Auths []*AuthConfig
}

type Config struct {
	Root     *yaml.Node
	Hosts    []*HostConfig
	Editor   string
	Protocol string
}

func (c *Config) ConfigForHost(hostname string) (*HostConfig, error) {
	for _, hc := range c.Hosts {
		if hc.Host == hostname {
			return hc, nil
		}
	}
	return nil, fmt.Errorf("could not find config entry for \"%s\"", hostname)
}

func (c *Config) DefaultHostConfig() (*HostConfig, error) {
	return c.ConfigForHost(defaultHostname)
}

func defaultConfig() Config {
	return Config{
		Protocol: "https",
		// we leave editor as empty string to signal that we should use environment variables
	}
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

// TODO this approach is bad; this function does too much and now parseConfig can't be handed some
// content in tests. Refactor to 1) have a better signature and 2) re-enable the testing approach we
// already have.

func readConfig(fn string) ([]byte, *yaml.Node, error) {
	f, err := os.Open(fn)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return data, nil, err
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

func parseConfig(fn string) (*Config, error) {
	data, root, err := readConfig(fn)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %s", err)
	}

	migrated, err := migrateConfig(configFile(), data, root)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate config: %s", err)
	}

	if migrated {
		data, root, err = readConfig(fn)
		if err != nil {
			return nil, fmt.Errorf("failed to re-read config after migration: %s", err)
		}
	}

	config := defaultConfig()
	config.Root = root

	topLevelKeys := root.Content[0].Content

	for i, v := range topLevelKeys {
		switch v.Value {
		case "hosts":
			// hosts is a map of hostname -> arrays of user auth configs
			for j, v := range topLevelKeys[i+1].Content {
				if v.Value == "" {
					continue
				}
				hostConfig := HostConfig{}
				hostConfig.Host = v.Value
				authsRoot := topLevelKeys[i+1].Content[j+1]
				// Each one of these is a map that holds user/oauth_token
				for _, v := range authsRoot.Content {
					authConfig := AuthConfig{}
					authRoot := v.Content
					// This is a map of user/token values
					for y, v := range authRoot {
						switch v.Value {
						case "user":
							authConfig.User = authRoot[y+1].Value
						case "oauth_token":
							authConfig.Token = authRoot[y+1].Value
						}
					}
					hostConfig.Auths = append(hostConfig.Auths, &authConfig)
				}
				config.Hosts = append(config.Hosts, &hostConfig)
			}
		case "protocol":
			protocolValue := topLevelKeys[i+1].Value
			if protocolValue != "ssh" && protocolValue != "https" {
				return nil, fmt.Errorf("got unexpected value for protocol: %s", protocolValue)
			}
			config.Protocol = protocolValue
			// TODO fucking with it to test writing back out
			// root.Content[0].Content[i+1].Value = "LOL"
		case "editor":
			editorValue := topLevelKeys[i+1].Value
			if !filepath.IsAbs(editorValue) {
				return nil, fmt.Errorf("editor should be an absolute path; got: %s", editorValue)
			}
			config.Editor = editorValue
		}
	}
	// TODO writing back out to test comment preservation
	//out, err := yaml.Marshal(&root)
	//if err != nil {
	//	return nil, err
	//}
	//fmt.Println(string(out))

	return &config, nil
}
