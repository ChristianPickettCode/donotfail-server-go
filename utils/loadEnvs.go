package utils

import (
	"log"
	"os"
)

var DEEPGRAM_API_KEY = ""
var AWS_REGION = ""
var AWS_BUCKET_NAME = ""
var OPENAI_API_KEY = ""

// LoadEnvs loads environment variables from a .env file
func LoadEnvs() error {
	// err := godotenv.Load()
	// if err != nil {
	// 	return fmt.Errorf("failed to load .env file: %w", err)
	// }

	// Load environment variables
	DEEPGRAM_API_KEY = os.Getenv("DEEPGRAM_API_KEY")
	if DEEPGRAM_API_KEY == "" {
		log.Fatal("DEEPGRAM_API_KEY not found in .env file")
	}

	AWS_REGION = os.Getenv("AWS_REGION")
	if AWS_REGION == "" {
		log.Fatal("AWS_REGION not found in .env file")
	}

	AWS_BUCKET_NAME = os.Getenv("AWS_BUCKET_NAME")
	if AWS_BUCKET_NAME == "" {
		log.Fatal("AWS_BUCKET_NAME not found in .env file")
	}

	OPENAI_API_KEY = os.Getenv("OPENAI_API_KEY")
	if OPENAI_API_KEY == "" {
		log.Fatal("OPENAI_API_KEY not found in .env file")
	}

	return nil
}
