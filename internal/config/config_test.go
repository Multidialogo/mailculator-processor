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

func TestGetMaxAttachmentsSize_DefaultWhenNotSet(t *testing.T) {
	yamlContent, err := getYamlContent("testdata/valid.yaml")
	if err != nil {
		t.Error(err)
	}

	cfg, err := NewFromYamlContent(yamlContent)
	assert.NoError(t, err)
	assert.Equal(t, DefaultMaxAttachmentsSize, cfg.GetMaxAttachmentsSize())
}

func TestGetMaxAttachmentsSize_CustomValue(t *testing.T) {
	yamlContent, err := getYamlContent("testdata/valid.yaml")
	if err != nil {
		t.Error(err)
	}

	cfg, err := NewFromYamlContent(yamlContent)
	assert.NoError(t, err)
	cfg.Attachments.MaxSize = 10000000
	assert.Equal(t, 10000000, cfg.GetMaxAttachmentsSize())
}

func TestExpandEnvVars(t *testing.T) {
	randomString := fmt.Sprintf("ran%d", rand.Int())
	t.Setenv("TEST_ENV_VAR", randomString)

	yamlContent, err := getYamlContent("testdata/valid-with-envvar-in-outbox-table-name.yaml")
	if err != nil {
		t.Error(err)
	}

	cfg, _ := NewFromYamlContent(yamlContent)
	assert.Equal(t, randomString, cfg.Attachments.BasePath)
}
