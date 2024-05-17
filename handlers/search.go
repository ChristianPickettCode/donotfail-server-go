package handlers

import (
	"context"
	"log"

	"main/models"
	"main/utils"
	"net/http"

	"github.com/sashabaranov/go-openai"

	"github.com/gin-gonic/gin"
)

func SearchQuestion(c *gin.Context) {
	var request models.SearchRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		log.Println("Invalid request:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	contextStr := request.Context
	question := request.Question

	response, err := answerQuestion(contextStr, question)
	if err != nil {
		log.Println("Error answering question:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error answering question"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": response})
}

func answerQuestion(contextStr, question string) (string, error) {
	client := openai.NewClient(utils.OPENAI_API_KEY)

	prompt := `
	You are a helpful assistant that can answer questions. If you don't know the answer, you can say 'I don't know'. Or if you don't have all the information, just tell me what you can. If the student asks you to go to a slide or explain a slide use the use the provided functions otherwise just answer their questions.
	`

	result, err := client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
		Model: openai.GPT4o,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: prompt,
			},
			{
				Role: openai.ChatMessageRoleUser,
				MultiContent: []openai.ChatMessagePart{
					{
						Type: openai.ChatMessagePartTypeText,
						Text: question,
					},
				},
			},
		},
		MaxTokens: 300,
	})

	if err != nil {
		return "", err
	}

	return result.Choices[0].Message.Content, nil
}
