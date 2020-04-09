package context

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config interface {
	Editor() (string, error)
	GitProtocol() (string, error)
	Hosts() ([]*HostConfig, error)
	ConfigForHost(string) (*HostConfig, error)
	DefaultHostConfig() (*HostConfig, error)
}

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

func NewConfig(root *yaml.Node) Config {
	return &fileConfig{Root: root}
}

type fileConfig struct {
	Root        *yaml.Node
	hosts       []*HostConfig
	editor      string
	gitProtocol string
}

func (c *fileConfig) ConfigForHost(hostname string) (*HostConfig, error) {
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

func (c *fileConfig) DefaultHostConfig() (*HostConfig, error) {
	return c.ConfigForHost(defaultHostname)
}

func parseHosts(hostsEntry *yaml.Node) ([]*HostConfig, error) {
	hostConfigs := []*HostConfig{}

	for i, v := range hostsEntry.Content {
		if v.Value == "" {
			continue
		}
		hostConfig := HostConfig{}
		hostConfig.Host = v.Value
		// TODO bound check
		authsRoot := hostsEntry.Content[i+1]
		for _, v := range authsRoot.Content {
			authConfig := AuthConfig{}
			authRoot := v.Content
			for j, v := range authRoot {
				switch v.Value {
				case "user":
					// TODO bound check
					authConfig.User = authRoot[j+1].Value
				case "oauth_token":
					// TODO bound check
					authConfig.Token = authRoot[j+1].Value
				}
			}
			hostConfig.Auths = append(hostConfig.Auths, &authConfig)
		}
		hostConfigs = append(hostConfigs, &hostConfig)
	}

	return hostConfigs, nil
}

func (c *fileConfig) findEntry(key string) (*yaml.Node, error) {
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

func (c *fileConfig) Hosts() ([]*HostConfig, error) {
	if len(c.hosts) > 0 {
		return c.hosts, nil
	}

	hostsEntry, err := c.findEntry("hosts")
	if err != nil {
		return nil, fmt.Errorf("could not find hosts config: %s", err)
	}

	hostConfigs, err := parseHosts(hostsEntry)
	if err != nil {
		return nil, fmt.Errorf("could not parse hosts config: %s", err)
	}

	c.hosts = hostConfigs

	return hostConfigs, nil
}

func (c *fileConfig) Editor() (string, error) {
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

func (c *fileConfig) GitProtocol() (string, error) {
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

func NewLegacyConfig(root *yaml.Node) Config {
	return &LegacyConfig{Root: root}
}

type LegacyConfig struct {
	Root  *yaml.Node
	hosts []*HostConfig
}

func (lc *LegacyConfig) Editor() (string, error) {
	return "", nil
}

func (lc *LegacyConfig) GitProtocol() (string, error) {
	return "https", nil
}

func (lc *LegacyConfig) ConfigForHost(hostname string) (*HostConfig, error) {
	hosts, err := lc.Hosts()
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

func (lc *LegacyConfig) DefaultHostConfig() (*HostConfig, error) {
	return lc.ConfigForHost(defaultHostname)
}

func (lc *LegacyConfig) Hosts() ([]*HostConfig, error) {
	if len(lc.hosts) > 0 {
		return lc.hosts, nil
	}

	hostConfigs, err := parseHosts(lc.Root.Content[0])
	if err != nil {
		return nil, fmt.Errorf("could not parse hosts config: %s", err)
	}

	lc.hosts = hostConfigs

	return hostConfigs, nil
}
