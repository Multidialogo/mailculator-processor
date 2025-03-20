package config

import (
	"fmt"
	"github.com/stretchr/testify/suite"
	"testing"
)

type config struct {
	Pippo    int    `yaml:"pippo" validate:"required"`
	Pluto    bool   `yaml:"pluto" validate:"required"`
	Paperino string `yaml:"paperino" validate:"required"`
}

func TestLoaderTestSuite(t *testing.T) {
	suite.Run(t, &LoaderTestSuite{})
}

type LoaderTestSuite struct {
	suite.Suite
	sut            *Loader
	configFilePath string
}

func (suite *LoaderTestSuite) SetupTest() {
	suite.configFilePath = "testdata/valid.yaml"
	suite.sut = NewLoader(suite.configFilePath)
}

func (suite *LoaderTestSuite) TestLoadAndValidateConfig() {
	cfg := config{}
	err := suite.sut.Load(&cfg)
	suite.Require().NoError(err)

	suite.Assert().Equal(3, cfg.Pippo)
	suite.Assert().Equal(true, cfg.Pluto)
	suite.Assert().Equal("paperina", cfg.Paperino)
}

func (suite *LoaderTestSuite) TestNonexistentConfigFile() {
	fakeFilePath := fmt.Sprintf("%s.fake", suite.configFilePath)
	cfg := config{}
	err := NewLoader(fakeFilePath).Load(&cfg)
	suite.Require().EqualError(err, fmt.Sprintf("open %s: no such file or directory", fakeFilePath))
}

func (suite *LoaderTestSuite) TestConfigFileWithOnlyAnUnknownField() {
	cfg := config{}
	err := NewLoader("testdata/only-unknown-field.yaml").Load(&cfg)
	suite.Require().EqualError(err, "Key: 'config.Pippo' Error:Field validation for 'Pippo' failed on the 'required' tag\nKey: 'config.Pluto' Error:Field validation for 'Pluto' failed on the 'required' tag\nKey: 'config.Paperino' Error:Field validation for 'Paperino' failed on the 'required' tag\nyaml: unmarshal errors:\n  line 1: field chuck not found in type config.config")
}
