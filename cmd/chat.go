package cmd

import (
	"context"
	"fmt"

	"github.com/charmbracelet/glamour"
	"github.com/sashabaranov/go-openai"
)

func main() {
	client := openai.NewClient("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")
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "All output must be in markdown",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: "Write a hello world code in python, golang, nodejs and c#",
				},
			},
		},
	)

	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		return
	}

	in := resp.Choices[0].Message.Content
	fmt.Println(in)
	out, err := glamour.Render(in, "dark")
	fmt.Print(out)
}
