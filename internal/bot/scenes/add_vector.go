package scenes

import (
	"context"
	"log"
	"strings"
	"time"
	"yphx-bot/internal/repository"

	tele "gopkg.in/telebot.v3"
)

const AddVectorButtonText = "➕ Add Vector"

type AddVectorScene struct {
	ai      vectorClient
	vectors *repository.VectorsRepo
}

func NewAddVectorScene(ai vectorClient, vectors *repository.VectorsRepo) *AddVectorScene {
	return &AddVectorScene{ai: ai, vectors: vectors}
}

func (s *AddVectorScene) Start(c tele.Context) error {
	return c.Send("Add vector rejimi.\nMatn yuboring yoki rasm yuboring.\nRasmga caption yozsangiz text ham saqlanadi.\n/cancel - chiqish")
}

func (s *AddVectorScene) Handle(c tele.Context) (done bool, err error) {
	if strings.EqualFold(strings.TrimSpace(c.Text()), "/cancel") {
		return true, c.Send("Add vector rejimi yopildi.")
	}

	msg := c.Message()
	if msg == nil {
		return false, c.Send("Matn yoki rasm yuboring.")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	if msg.Photo != nil {
		imageURL, err := resolveTelegramFileURL(c, msg.Photo.FileID)
		if err != nil {
			return false, c.Send("Rasm URL olishda xatolik.")
		}

		fileBytes, fileName, err := downloadTelegramFileBytes(c, msg.Photo.FileID)
		if err != nil {
			return false, c.Send("Rasm faylini yuklashda xatolik.")
		}

		imageVector, err := s.ai.ImageUploadToVector(ctx, fileName, fileBytes)
		if err != nil {
			log.Printf("image upload to vector error: %v", err)
			return false, c.Send("Rasmni vector qilishda xatolik.")
		}

		caption := strings.TrimSpace(msg.Caption)
		var textVector []float64
		if caption != "" {
			textVector, err = s.ai.TextToVector(ctx, caption)
			if err != nil {
				return false, c.Send("Caption text vector olishda xatolik.")
			}
		}

		if err := s.vectors.SaveImage(ctx, caption, imageURL, imageVector, textVector); err != nil {
			return false, c.Send("Image vector saqlashda xatolik.")
		}
		return true, c.Send("Image vector saqlandi.")
	}

	text := strings.TrimSpace(c.Text())
	if text == "" || strings.HasPrefix(text, "/") {
		return false, c.Send("Matn yoki rasm yuboring.")
	}

	textVector, err := s.ai.TextToVector(ctx, text)
	if err != nil {
		return false, c.Send("Matn vector olishda xatolik.")
	}
	if err := s.vectors.SaveText(ctx, text, textVector); err != nil {
		return false, c.Send("Text vector saqlashda xatolik.")
	}
	return true, c.Send("Text vector saqlandi.")
}
