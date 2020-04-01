package context

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const defaultHostname = "github.com"

func parseOrSetupConfigFile(fn string) (*Config, error) {
	config, err := parseConfigFile(fn)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return setupConfigFile(fn)
	}
	return config, err
}

func parseConfigFile(fn string) (*Config, error) {
	f, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return parseConfig(f)
}

// ParseDefaultConfig reads the configuration file
func ParseDefaultConfig() (*Config, error) {
	return parseConfigFile(configFile())
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
	return nil, errors.New("not found")
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

func parseConfig(r io.Reader) (*Config, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var root yaml.Node
	err = yaml.Unmarshal(data, &root)
	if err != nil {
		return nil, err
	}
	if len(root.Content) < 1 {
		return nil, fmt.Errorf("malformed config")
	}
	if root.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("expected a top level map")
	}

	config := defaultConfig()
	config.Root = &root

	// TODO

	// - [x] make hosts: nesting format work with new parsing code and config struct
	// - [ ] support setting editor (or protocol)
	// - [ ] implement new commands
	// - [ ] migration code for old (non-hosts) config

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
