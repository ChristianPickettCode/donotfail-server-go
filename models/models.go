package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Slide struct {
	ID             primitive.ObjectID `bson:"_id" json:"id"`
	Name           string             `bson:"name" json:"name"`
	PDFURL         string             `bson:"pdf_url" json:"pdf_url"`
	SpaceID        string             `bson:"space_id" json:"space_id"`
	CreatedAt      time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt      time.Time          `bson:"updated_at" json:"updated_at"`
	GeneratedNotes []string           `bson:"generated_notes" json:"generated_notes"`
}

type SlideImage struct {
	ID            primitive.ObjectID `bson:"_id" json:"id"`
	SlideID       string             `bson:"slide_id" json:"slide_id"`
	ImageURL      string             `bson:"image_url" json:"image_url"`
	Order         int                `bson:"order" json:"order"`
	GeneratedText string             `bson:"generated_text" json:"generated_text"`
	AudioURL      string             `bson:"audio_url" json:"audio_url"`
	CreatedAt     time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time          `bson:"updated_at" json:"updated_at"`
}

type Space struct {
	ID        primitive.ObjectID `bson:"_id" json:"id"`
	Name      string             `bson:"name" json:"name"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

type SlideSpaceRequest struct {
	SlideID   string    `bson:"slide_id" json:"slide_id"`
	SpaceID   string    `bson:"space_id" json:"space_id"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

type AudioRequest struct {
	SlideImageID string    `bson:"slide_image_id" json:"slide_image_id"`
	CreatedAt    time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time `bson:"updated_at" json:"updated_at"`
	Update       bool      `bson:"update" json:"update"`
}

type SearchRequest struct {
	Context  string `json:"context"`
	Question string `json:"question" binding:"required"`
}

type QuizQA struct {
	ID            primitive.ObjectID `bson:"_id" json:"id"`
	Question      string             `bson:"question" json:"question"`
	AnswerChoices []string           `bson:"answer_choices" json:"answer_choices"`
	Answer        string             `bson:"answer" json:"answer"`
	Rationale     string             `bson:"rationale" json:"rationale"`
	SlideID       string             `bson:"slide_id" json:"slide_id"`
	SlideImageID  string             `bson:"slide_image_id" json:"slide_image_id"`
}

type Flashcard struct {
	ID           primitive.ObjectID `bson:"_id" json:"id"`
	Question     string             `bson:"question" json:"question"`
	Answer       string             `bson:"answer" json:"answer"`
	SlideID      string             `bson:"slide_id" json:"slide_id"`
	SlideImageID string             `bson:"slide_image_id" json:"slide_image_id"`
}

type User struct {
	ID         primitive.ObjectID `bson:"_id" json:"id"`
	UserID     string             `bson:"user_id" json:"user_id"`
	FirstName  string             `bson:"first_name" json:"first_name"`
	LastName   string             `bson:"last_name" json:"last_name"`
	Email      string             `bson:"email" json:"email"`
	School     string             `bson:"school" json:"school"`
	CreatedAt  time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt  time.Time          `bson:"updated_at" json:"updated_at"`
	SpaceIDs   []string           `bson:"space_ids" json:"space_ids"`
	AccessCode string             `bson:"access_code" json:"access_code"`
}

type AccessCode struct {
	ID   primitive.ObjectID `bson:"_id" json:"id"`
	Code string             `bson:"code" json:"code"`
	Used bool               `bson:"used" json:"used"`
}
