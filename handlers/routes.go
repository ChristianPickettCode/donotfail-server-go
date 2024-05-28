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
	r.POST("/generate-audio", GenerateAudio2)

	// generate text routes
	// Generate text for a slide image
	r.POST("/generate-image-text/:slide_image_id", GenerateText)

	// convert pdf to image routes
	// Convert PDF to images for a slide
	r.POST("/convert-pdf-to-images/:slide_id", ConvertPDFToImages)

	// test
	r.GET("/test", Test)

	// generate all image text
	r.POST("/generate-all-image-text/:slide_id", GenerateAllImageText)

	// search
	r.POST("/search", SearchQuestion)

	// generate notes
	r.POST("/generate-notes/:slide_id", GenerateNotes)

	// generate all audio
	r.POST("/generate-all-audio/:slide_id", GenerateAllAudioForSlide)

	// generate quiz
	r.POST("/generate-quiz/:slide_id", GenerateQuizQuestions)

	// GetSlidesWithQuizQuestions
	r.GET("/slides-with-quiz-questions", GetSlidesWithQuizQuestions)

	// Get all quiz questions for a slide
	r.GET("/quiz-questions/:slide_id", GetQuizQuestions)

	// Delete quiz question by quiz id
	r.DELETE("/quiz-question/:quiz_id", DeleteQuizQuestion)

}
