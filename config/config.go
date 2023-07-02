package config

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"text/template"

	"embed"

	"gopkg.in/yaml.v3"
)

//go:embed i18n
var i18n embed.FS

//go:embed prompt.yaml
var promptFS []byte

type APIConfig struct {
	OpenAIAPIKey string `yaml:"openai_api_key"`
}

type CommitConfig struct {
	Language     string `yaml:"language"`
	Emoji        bool   `yaml:"emoji"`
	Description  bool   `yaml:"description"`
	OpenAIAPIKey string `yaml:"openai_api_key"`
}
type Config struct {
	APIConfig    APIConfig    `yaml:"api"`
	CommitConfig CommitConfig `yaml:"commit"`
}

func defaultConfig() (*Config, error) {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("`OPENAI_API_KEY` env var not set")
	}
	return &Config{APIConfig{
		OpenAIAPIKey: key,
	},
		CommitConfig{
			Language:     "en-US",
			Emoji:        false,
			Description:  false,
			OpenAIAPIKey: key,
		}}, nil
}

func GetConfig() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	configPath := path.Join(homeDir, ".commiteiro.yaml")
	_, err = os.Stat(configPath)
	if err != nil {
		config, err := defaultConfig()
		return config, err
	}
	configFile, err := os.Open(configPath)
	defer configFile.Close()
	confiData, err := io.ReadAll(configFile)
	if err != nil {
		return nil, err
	}
	var config Config
	err = yaml.Unmarshal(confiData, &config)
	if config.APIConfig.OpenAIAPIKey == "" {
		key := os.Getenv("OPENAI_API_KEY")
		if key == "" {
			return nil, fmt.Errorf("`OPENAI_API_KEY` env not set")
		}
		config.APIConfig.OpenAIAPIKey = key
	}
	return &config, err

}

type ContextPrompt struct {
	Language          string `yaml:"language"`
	CommitFix         string `yaml:"commit_fix"`
	CommitFeat        string `yaml:"commit_feat"`
	CommitDescription string `yaml:"commit_description"`
}

func getContextPrompt() (*ContextPrompt, error) {
	config, err := GetConfig()
	if err != nil {
		return nil, err
	}
	langFilePath := config.CommitConfig.Language + ".yaml"
	langFile, err := i18n.Open("i18n/" + langFilePath)
	if err != nil {
		return nil, err
	}
	langFileData, err := io.ReadAll(langFile)
	if err != nil {
		return nil, err
	}
	var contextPrompt ContextPrompt
	err = yaml.Unmarshal(langFileData, &contextPrompt)
	return &contextPrompt, err
}

type Prompt struct {
	Role    string `yaml:"role"`
	Content string `yaml:"content"`
}

type PromptFile struct {
	Prompts []Prompt `yaml:"prompts"`
}

func GetContextPrompt() ([]Prompt, error) {
	contextPrompt, err := getContextPrompt()
	if err != nil {
		return nil, err
	}
	config, err := GetConfig()
	if err != nil {
		return nil, err
	}
	tmpl := template.Must(template.New("prompt").Parse(string(promptFS)))
	var buffer bytes.Buffer

	tmpl.Execute(&buffer, map[string]interface{}{
		"language":          contextPrompt.Language,
		"description":       config.CommitConfig.Description,
		"emoji":             config.CommitConfig.Emoji,
		"commitFix":         contextPrompt.CommitFix,
		"commitDescription": contextPrompt.CommitDescription,
		"commitFeat":        contextPrompt.CommitFeat,
	})
	var promptFile PromptFile
	err = yaml.Unmarshal(buffer.Bytes(), &promptFile)
	if err != nil {
		panic(err)
	}
	return promptFile.Prompts, nil
}
