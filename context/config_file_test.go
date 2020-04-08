package context

import (
	"errors"
	"reflect"
	"testing"
)

func stubbedReadConfig(content string) func(fn string) ([]byte, error) {
	return func(fn string) ([]byte, error) {
		return []byte(content), nil
	}
}

func stubConfig(content string) func() {
	orig := readConfig
	readConfig = stubbedReadConfig(content)
	return func() {
		readConfig = orig
	}
}

func eq(t *testing.T, got interface{}, expected interface{}) {
	t.Helper()
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("expected: %v, got: %v", expected, got)
	}
}

func Test_parseConfig(t *testing.T) {
	defer stubConfig(`---
hosts:
  github.com:
  - user: monalisa
    oauth_token: OTOKEN
  - user: wronguser
    oauth_token: NOTTHIS
`)()
	config, err := parseConfig("filename")
	eq(t, err, nil)
	hostConfig, err := config.DefaultHostConfig()
	eq(t, err, nil)
	eq(t, hostConfig.Auths[0].User, "monalisa")
	eq(t, hostConfig.Auths[0].Token, "OTOKEN")
}

func Test_parseConfig_multipleHosts(t *testing.T) {
	defer stubConfig(`---
hosts:
  example.com:
  - user: wronguser
    oauth_token: NOTTHIS
  github.com:
  - user: monalisa
    oauth_token: OTOKEN
`)()
	config, err := parseConfig("filename")
	eq(t, err, nil)
	hostConfig, err := config.DefaultHostConfig()
	eq(t, err, nil)
	eq(t, hostConfig.Auths[0].User, "monalisa")
	eq(t, hostConfig.Auths[0].Token, "OTOKEN")
}

func Test_parseConfig_notFound(t *testing.T) {
	defer stubConfig(`---
hosts:
  example.com:
  - user: wronguser
    oauth_token: NOTTHIS
`)()
	config, err := parseConfig("filename")
	eq(t, err, nil)
	_, err = config.DefaultHostConfig()
	eq(t, err, errors.New(`could not find config entry for "github.com"`))
}
