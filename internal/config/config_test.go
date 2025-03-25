package config

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewFromYaml(t *testing.T) {
	t.Parallel()

	type caseStruct struct {
		filepath    string
		expectError bool
	}

	cases := []caseStruct{
		{"testdata/valid.yaml", false},
		{"testdata/invalid-unknown-field.yaml", true},
		{"testdata/invalid-missing-host.yaml", true},
	}

	for _, c := range cases {
		_, err := NewFromYaml(c.filepath)

		if c.expectError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	}
}

func TestExpandEnvVars(t *testing.T) {
	randomString := fmt.Sprintf("ran%d", rand.Int())
	t.Setenv("TEST_ENV_VAR", randomString)

	cfg, _ := NewFromYaml("testdata/valid-with-envvar-in-aws-secret.yaml")

	assert.Equal(t, randomString, cfg.Aws.Secret)
}
