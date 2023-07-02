package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/ericklima-ca/lattes/config"
	"github.com/sashabaranov/go-openai"
)

func GetCommitMessage(patch string) (string, error) {
	currenctConfig, err := config.GetConfig()
	if err != nil {
		return "", err
	}
	prompts, err := config.GetContextPrompt()
	if err != nil {
		return "", err
	}
	client := openai.NewClient(currenctConfig.APIConfig.OpenAIAPIKey)
	var messages []openai.ChatCompletionMessage
	for _, prompt := range prompts {
		message := openai.ChatCompletionMessage{
			Role:    prompt.Role,
			Content: prompt.Content,
		}
		messages = append(messages, message)
	}

	patchMessage := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: patch,
	}

	messages = append(messages, patchMessage)

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:       openai.GPT3Dot5Turbo,
			Messages:    messages,
			Temperature: 0,
			TopP:        0.1,
			MaxTokens:   196,
		},
	)

	if err != nil {
		return "", err
	}
	in := resp.Choices[0].Message.Content
	result := fmt.Sprintf(strings.Join(strings.Split(in, "\n\n"), "\n"))
	return result, err

}
