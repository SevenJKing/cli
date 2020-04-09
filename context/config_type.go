package context

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

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
