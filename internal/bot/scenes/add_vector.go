package scenes

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
	"yphx-bot/internal/repository"

	tele "gopkg.in/telebot.v3"
)

const AddVectorButtonText = "➕ Add Vector"

type AddVectorScene struct {
	ai       vectorClient
	vectors  *repository.VectorsRepo
	filesDir string
}

func NewAddVectorScene(ai vectorClient, vectors *repository.VectorsRepo, filesDir string) *AddVectorScene {
	return &AddVectorScene{ai: ai, vectors: vectors, filesDir: filesDir}
}

func (s *AddVectorScene) Start(c tele.Context) error {
	return c.Send("Add vector rejimi.\nMatn yuboring yoki rasm yuboring.\nFormat: text || info (info ixtiyoriy)\n/cancel - chiqish")
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
		fileBytes, fileName, err := downloadTelegramFileBytes(c, msg.Photo.FileID)
		if err != nil {
			return false, c.Send("Rasm faylini yuklashda xatolik.")
		}
		localImagePath, err := s.saveImageLocally(fileName, fileBytes)
		if err != nil {
			return false, c.Send("Rasmni local saqlashda xatolik.")
		}

		imageVector, err := s.ai.ImageUploadToVector(ctx, fileName, fileBytes)
		if err != nil {
			log.Printf("image upload to vector error: %v", err)
			return false, c.Send("Rasmni vector qilishda xatolik.")
		}

		textValue, infoValue := splitTextAndInfo(strings.TrimSpace(msg.Caption))
		combinedText := strings.TrimSpace(joinTextInfoForVector(textValue, infoValue))
		var textVector []float64
		if combinedText != "" {
			textVector, err = s.ai.TextToVector(ctx, combinedText)
			if err != nil {
				return false, c.Send("Caption text vector olishda xatolik.")
			}
		}

		imageHash := hashBytes(fileBytes)
		saved, err := s.vectors.SaveImage(ctx, textValue, infoValue, localImagePath, imageHash, imageVector, textVector)
		if err != nil {
			return false, c.Send("Image vector saqlashda xatolik.")
		}
		if !saved {
			return true, c.Send("Bu rasm allaqachon mavjud (duplicate).")
		}
		return true, c.Send("Image vector saqlandi.")
	}

	textValue, infoValue := splitTextAndInfo(strings.TrimSpace(c.Text()))
	if textValue == "" || strings.HasPrefix(textValue, "/") {
		return false, c.Send("Matn yoki rasm yuboring.")
	}
	combinedText := joinTextInfoForVector(textValue, infoValue)

	textVector, err := s.ai.TextToVector(ctx, combinedText)
	if err != nil {
		return false, c.Send("Matn vector olishda xatolik.")
	}
	saved, err := s.vectors.SaveText(ctx, textValue, infoValue, textVector)
	if err != nil {
		return false, c.Send("Text vector saqlashda xatolik.")
	}
	if !saved {
		return true, c.Send("Bu text allaqachon mavjud (duplicate).")
	}
	return true, c.Send("Text vector saqlandi.")
}

func (s *AddVectorScene) saveImageLocally(fileName string, data []byte) (string, error) {
	if err := os.MkdirAll(s.filesDir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir files dir: %w", err)
	}

	base := filepath.Base(fileName)
	if base == "" || base == "." {
		base = fmt.Sprintf("image_%d.bin", time.Now().UnixNano())
	}
	localPath := filepath.Join(s.filesDir, fmt.Sprintf("%d_%s", time.Now().UnixNano(), base))
	if err := os.WriteFile(localPath, data, 0o644); err != nil {
		return "", fmt.Errorf("write image file: %w", err)
	}
	return localPath, nil
}

func hashBytes(data []byte) string {
	sum := md5.Sum(data)
	return hex.EncodeToString(sum[:])
}

func splitTextAndInfo(input string) (string, string) {
	parts := strings.SplitN(input, "||", 2)
	text := strings.TrimSpace(parts[0])
	if len(parts) == 1 {
		return text, ""
	}
	info := strings.TrimSpace(parts[1])
	return text, info
}

func joinTextInfoForVector(text, info string) string {
	text = strings.TrimSpace(text)
	info = strings.TrimSpace(info)
	if info == "" {
		return text
	}
	if text == "" {
		return info
	}
	return text + " " + info
}
