package handlers

import "github.com/gin-gonic/gin"

func SetUpRoutes(r *gin.Engine) {

	// Spaces routes
	spaceRoutes := r.Group("/space")
	{
		spaceRoutes.GET("/:id", GetSpace)
		spaceRoutes.POST("/", CreateSpace)
		// spaceRoutes.PUT("/:id", updateSpace)
		spaceRoutes.DELETE("/:id", DeleteSpace)
	}

	// Get a space by ID
	r.GET("/spaces", GetSpaces)

	// Get all space slides
	r.GET("/space-slides/:space_id", GetSpaceSlides)

	// Slides
	slideRoutes := r.Group("/slide")
	{
		slideRoutes.GET("/:id", GetSlide)
		slideRoutes.POST("/", CreateSlide)
		slideRoutes.PUT("/:id", UpdateSlide)
		slideRoutes.DELETE("/:id", DeleteSlide)

		// 	// Slide Images
		slideImageRoutes := slideRoutes.Group("/images/:slide_id")
		{
			slideImageRoutes.GET("/", GetSlideImages)
		}
	}

	// Get all slides
	r.GET("/slides", GetSlides)

	// generate audio routes
	// Generate audio for a slide
	r.GET("/generate-audio/:slide_image_id", GenerateAudio2)

	// generate text routes
	// Generate text for a slide image
	r.GET("/generate-image-text/:slide_image_id", GenerateText)

	// convert pdf to image routes
	// Convert PDF to images for a slide
	r.GET("/convert-pdf-to-images/:slide_id", ConvertPDFToImages)

	// generate all image text
	r.GET("/generate-all-image-text/:slide_id", GenerateAllImageText)

	// search
	r.POST("/search", SearchQuestion)

	// generate notes
	r.POST("/generate-notes/:slide_id", GenerateNotes)

	// generate all audio
	r.POST("/generate-all-audio/:slide_id", GenerateAllAudioForSlide)

	// generate quiz
	r.POST("/generate-quiz/:slide_id", GenerateQuizQuestions)

	r.POST("/generate-quiz/:slide_id/:slide_image_id", GenerateQuizQuestionsForSlideImage)

	r.GET("/quiz-questions/:slide_id/:slide_image_id", GetQuizQuestionsForSlideImage)

	// GetSlidesWithQuizQuestions
	r.GET("/slides-with-quiz-questions", GetSlidesWithQuizQuestions)

	// Get all quiz questions for a slide
	r.GET("/quiz-questions/:slide_id", GetQuizQuestions)

	// Delete quiz question by quiz id
	r.DELETE("/quiz-question/:quiz_id", DeleteQuizQuestion)

	// Get all flashcards for a slide
	r.GET("/generate-flashcards/:slide_id", GenerateFlashCards)

	// Generate flashcards for a slide image
	r.GET("/generate-flashcards/:slide_id/:slide_image_id", GenerateFlashcardsForSlideImage)

	// Get all flashcards for a slide image
	r.GET("/flashcards/:slide_id/:slide_image_id", GetFlashcardsForSlideImage)

	// Get all slides with flashcards
	r.GET("/slides-with-flashcards", GetSlidesWithFlashcards)

	// Get all flashcards for a slide
	r.GET("/flashcards/:slide_id", GetFlashcards)

	// Delete flashcard by flashcard id
	r.DELETE("/flashcard/:flashcard_id", DeleteFlashcard)

	// Users
	userRoutes := r.Group("/user")
	{
		userRoutes.GET("/:user_id", GetUser)
		userRoutes.POST("/", CreateUser)
		// userRoutes.PUT("/:id", updateUser)
		// userRoutes.DELETE("/:id", deleteUser)

		// User spaces
		userSpaceRoutes := userRoutes.Group("/:user_id/space")
		{
			userSpaceRoutes.GET("/", GetUserSpaces)
			userSpaceRoutes.PUT("/:space_id", AddSpaceToUser)
			userSpaceRoutes.DELETE("/:space_id", RemoveSpaceFromUser)
		}
	}

	// Credits
	creditRoutes := r.Group("/credits")
	{
		creditRoutes.GET("/:user_id", GetUserCredits)
		creditRoutes.POST("/:user_id/add/:amount", AddCredits)
		creditRoutes.POST("/:user_id/remove/:amount", RemoveCredits)
	}

	// Access codes
	accessCodeRoutes := r.Group("/access-code")
	{
		accessCodeRoutes.POST("/", CreateAccessCode)
		accessCodeRoutes.GET("/:id", GetAccessCode)
		accessCodeRoutes.PUT("/:id", UpdateAccessCode)
		accessCodeRoutes.DELETE("/:id", DeleteAccessCode)
	}

	// Verify access code
	r.POST("/verify-access-code", VerifyAccessCode)

}
