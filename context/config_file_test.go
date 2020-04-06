package context

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func eq(t *testing.T, got interface{}, expected interface{}) {
	t.Helper()
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("expected: %v, got: %v", expected, got)
	}
}

func Test_parseConfig(t *testing.T) {
	c := strings.NewReader(`---
hosts:
  github.com:
  - user: monalisa
    oauth_token: OTOKEN
  - user: wronguser
    oauth_token: NOTTHIS
`)
	config, err := parseConfig(c)
	eq(t, err, nil)
	hostConfig, err := config.DefaultHostConfig()
	eq(t, err, nil)
	eq(t, hostConfig.Auths[0].User, "monalisa")
	eq(t, hostConfig.Auths[0].Token, "OTOKEN")
}

func Test_parseConfig_multipleHosts(t *testing.T) {
	c := strings.NewReader(`---
hosts:
  example.com:
  - user: wronguser
    oauth_token: NOTTHIS
  github.com:
  - user: monalisa
    oauth_token: OTOKEN
`)
	config, err := parseConfig(c)
	eq(t, err, nil)
	hostConfig, err := config.DefaultHostConfig()
	eq(t, err, nil)
	eq(t, hostConfig.Auths[0].User, "monalisa")
	eq(t, hostConfig.Auths[0].Token, "OTOKEN")
}

func Test_parseConfig_notFound(t *testing.T) {
	c := strings.NewReader(`---
hosts:
  example.com:
  - user: wronguser
    oauth_token: NOTTHIS
`)
	config, err := parseConfig(c)
	eq(t, err, nil)
	_, err = config.DefaultHostConfig()
	eq(t, err, errors.New(`could not find config entry for "github.com"`))
}
