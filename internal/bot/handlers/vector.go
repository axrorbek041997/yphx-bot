package handlers

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
	"yphx-bot/internal/ai"
	"yphx-bot/internal/repository"

	tele "gopkg.in/telebot.v3"
)

type VectorHandlers struct {
	aiClient *ai.Client
	vectors  *repository.VectorsRepo
}

func NewVectorHandlers(aiClient *ai.Client, vectors *repository.VectorsRepo) *VectorHandlers {
	return &VectorHandlers{
		aiClient: aiClient,
		vectors:  vectors,
	}
}

func (h *VectorHandlers) HandleText(c tele.Context) error {
	text := strings.TrimSpace(c.Text())
	if text == "" || strings.HasPrefix(text, "/") {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	vector, err := h.aiClient.TextToVector(ctx, text)
	if err != nil {
		log.Printf("text-to-vector error: %v", err)
		return c.Send("Text vector olishda xatolik yuz berdi.")
	}

	_, err = h.vectors.SaveText(ctx, text, "", vector)
	if err != nil {
		log.Printf("save text vector error: %v", err)
		return c.Send("Vector saqlashda xatolik yuz berdi.")
	}

	return c.Send(fmt.Sprintf("Text vector saqlandi ✅ (%d dims)", len(vector)))
}

func (h *VectorHandlers) HandleAudio(c tele.Context) error {
	var fileID string
	msg := c.Message()
	if msg == nil {
		return c.Send("Audio topilmadi.")
	}

	if msg.Audio != nil {
		fileID = msg.Audio.FileID
	} else if msg.Voice != nil {
		fileID = msg.Voice.FileID
	}
	if fileID == "" {
		return c.Send("Audio fayl ID topilmadi.")
	}

	audioURI, err := resolveTelegramFileURL(c, fileID)
	if err != nil {
		log.Printf("resolve audio url error: %v", err)
		return c.Send("Audio linkni olishda xatolik yuz berdi.")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	vector, err := h.aiClient.AudioURIToVector(ctx, audioURI)
	if err != nil {
		log.Printf("audio-to-vector-uri error: %v", err)
		return c.Send("Audio vector olishda xatolik yuz berdi.")
	}

	_, err = h.vectors.SaveImage(ctx, "", "", audioURI, "", vector, nil)
	if err != nil {
		log.Printf("save audio vector error: %v", err)
		return c.Send("Vector saqlashda xatolik yuz berdi.")
	}

	return c.Send(fmt.Sprintf("Audio vector saqlandi ✅ (%d dims)", len(vector)))
}

func resolveTelegramFileURL(c tele.Context, fileID string) (string, error) {
	file, err := c.Bot().FileByID(fileID)
	if err != nil {
		return "", fmt.Errorf("get telegram file by id: %w", err)
	}
	if file.FilePath == "" {
		return "", fmt.Errorf("telegram file path is empty")
	}

	return fmt.Sprintf(
		"%s/file/bot%s/%s",
		strings.TrimRight(c.Bot().URL, "/"),
		c.Bot().Token,
		strings.TrimLeft(file.FilePath, "/"),
	), nil
}
