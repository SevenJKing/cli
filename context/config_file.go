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
const defaultGitProtocol = "https"

type AuthConfig struct {
	User  string
	Token string `yaml:"oauth_token"`
}

type HostConfig struct {
	Host  string
	Auths []*AuthConfig
}

type Config struct {
	Root        *yaml.Node
	hosts       []*HostConfig
	editor      string
	gitProtocol string
}

func (c *Config) Hosts() ([]*HostConfig, error) {
	if len(c.hosts) > 0 {
		return c.hosts, nil
	}
	hostConfigs := []*HostConfig{}

	hostsEntry, err := c.findEntry("hosts")
	if err != nil {
		return nil, fmt.Errorf("could not parse hosts config: %s", err)
	}

	for j, v := range hostsEntry.Content {
		if v.Value == "" {
			continue
		}
		hostConfig := HostConfig{}
		hostConfig.Host = v.Value
		// TODO bound check
		authsRoot := hostsEntry.Content[j+1]
		for _, v := range authsRoot.Content {
			authConfig := AuthConfig{}
			authRoot := v.Content
			for y, v := range authRoot {
				switch v.Value {
				case "user":
					// TODO bound check
					authConfig.User = authRoot[y+1].Value
				case "oauth_token":
					// TODO bound check
					authConfig.Token = authRoot[y+1].Value
				}
			}
			hostConfig.Auths = append(hostConfig.Auths, &authConfig)
		}
		hostConfigs = append(hostConfigs, &hostConfig)
	}

	c.hosts = hostConfigs

	return hostConfigs, nil
}

func (c *Config) ConfigForHost(hostname string) (*HostConfig, error) {
	hosts, err := c.Hosts()
	if err != nil {
		return nil, fmt.Errorf("failed to parse hosts config: %s", err)
	}

	for _, hc := range hosts {
		if hc.Host == hostname {
			return hc, nil
		}
	}
	return nil, fmt.Errorf("could not find config entry for %q", hostname)
}

func (c *Config) DefaultHostConfig() (*HostConfig, error) {
	return c.ConfigForHost(defaultHostname)
}

func (c *Config) findEntry(key string) (*yaml.Node, error) {
	topLevelKeys := c.Root.Content[0].Content
	var entry *yaml.Node
	for i, v := range topLevelKeys {
		if v.Value == key {
			// TODO bound check
			entry = topLevelKeys[i+1]
		}
	}

	if entry == nil {
		return nil, errors.New("not found")
	}

	return entry, nil
}

func (c *Config) Editor() (string, error) {
	if c.editor != "" { // TODO overlap with not found case
		return c.editor, nil
	}

	// if we can't find editor, we don't worry about it. empty string means to respect environment.
	editorEntry, _ := c.findEntry("editor")
	if editorEntry == nil {
		return "", nil
	}

	editorValue := editorEntry.Value

	if !filepath.IsAbs(editorValue) {
		return "", fmt.Errorf("editor should be an absolute path; got: %s", editorValue)
	}
	c.editor = editorValue

	return editorValue, nil
}

func (c *Config) GitProtocol() (string, error) {
	if c.gitProtocol != "" {
		return c.gitProtocol, nil
	}
	gitProtocolEntry, err := c.findEntry("git_protocol")
	gitProtocolValue := ""
	if err != nil {
		// TODO do not warn if merely not found
		fmt.Fprintf(os.Stderr, "malformed gitProtocol config entry: %s. falling back to default\n", err)
		gitProtocolValue = defaultGitProtocol
	} else {
		gitProtocolValue = gitProtocolEntry.Value
	}

	c.gitProtocol = gitProtocolValue

	return gitProtocolValue, nil
}

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

func parseConfig(fn string) (*Config, error) {
	data, root, err := parseConfigFile(fn)
	if err != nil {
		return nil, err
	}

	migrated, err := migrateConfig(configFile(), data, root)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate config: %s", err)
	}

	if migrated {
		data, root, err = parseConfigFile(fn)
		if err != nil {
			return nil, fmt.Errorf("failed to re-read config after migration: %s", err)
		}
	}

	config := Config{}
	config.Root = root

	return &config, nil
}
