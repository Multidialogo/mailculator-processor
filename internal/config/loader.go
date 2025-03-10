package config

import (
	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
	"os"
	"strings"
)

type Loader struct {
	filePath string
}

func NewLoader(filePath string) *Loader {
	return &Loader{filePath: filePath}
}

func (c *Loader) Load(cfg any) error {
	yamlData, err := os.ReadFile(c.filePath)
	if err != nil {
		return err
	}

	yamlString := os.ExpandEnv(string(yamlData))

	decoder := yaml.NewDecoder(strings.NewReader(yamlString))
	decoder.KnownFields(true)

	if err := decoder.Decode(&cfg); err != nil {
		return err
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	return validate.Struct(&cfg)
}
