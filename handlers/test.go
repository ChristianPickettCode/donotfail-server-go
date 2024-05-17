package handlers

import (
	"fmt"
	"image/jpeg"
	"log"
	"os"
	"path/filepath"

	"github.com/gen2brain/go-fitz"
	"github.com/gin-gonic/gin"
)

func Test(gin *gin.Context) {
	log.Println("Test")
	doc, err := fitz.New("NLP_test.pdf")
	if err != nil {
		panic(err)
	}
	log.Println("Test2")

	defer doc.Close()

	tmpDir, err := os.MkdirTemp(".", "fitz")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpDir)
	log.Println("tmpDir", tmpDir)

	log.Println("Test3")
	// Extract pages as images
	for n := 0; n < doc.NumPage(); n++ {
		// log.Println("Test4")
		img, err := doc.Image(n)
		if err != nil {
			panic(err)
		}

		f, err := os.Create(filepath.Join(tmpDir, fmt.Sprintf("test%03d.jpg", n)))
		if err != nil {
			panic(err)
		}

		err = jpeg.Encode(f, img, &jpeg.Options{Quality: jpeg.DefaultQuality})
		if err != nil {
			panic(err)
		}

		f.Close()
	}

	log.Println("Test5")
}
