//go:build unit

package config

import (
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func getYamlContent(fileName string) ([]byte, error) {
	yamlContentBytes, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	return yamlContentBytes, nil
}

func TestNewFromYamlContent(t *testing.T) {
	t.Parallel()

	type caseStruct struct {
		name        string
		filepath    string
		expectError bool
	}

	cases := []caseStruct{
		{"Valid", "testdata/valid.yaml", false},
		{"Invalid unknown field", "testdata/invalid-unknown-field.yaml", true},
		{"Invalid missing fields", "testdata/invalid-missing-fields.yaml", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			yamlContent, err := getYamlContent(c.filepath)
			if err != nil {
				t.Error(err)
			}

			_, err = NewFromYamlContent(yamlContent)

			if c.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExpandEnvVars(t *testing.T) {
	randomString := fmt.Sprintf("ran%d", rand.Int())
	t.Setenv("TEST_ENV_VAR", randomString)

	yamlContent, err := getYamlContent("testdata/valid-with-envvar-in-outbox-table-name.yaml")
	if err != nil {
		t.Error(err)
	}

	cfg, _ := NewFromYamlContent(yamlContent)
	assert.Equal(t, randomString, cfg.Outbox.TableName)
}
