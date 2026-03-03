package scenes

import (
	"context"
	"fmt"
	"strings"
	"time"
	"yphx-bot/internal/repository"

	tele "gopkg.in/telebot.v3"
)

const SearchButtonText = "🔎 Search"

type vectorClient interface {
	TextToVector(ctx context.Context, text string) ([]float64, error)
	ImageURIToVector(ctx context.Context, uri string) ([]float64, error)
}

type SearchScene struct {
	ai      vectorClient
	vectors *repository.VectorsRepo
}

func NewSearchScene(ai vectorClient, vectors *repository.VectorsRepo) *SearchScene {
	return &SearchScene{ai: ai, vectors: vectors}
}

func (s *SearchScene) Start(c tele.Context) error {
	return c.Send("Qidiruv rejimi yoqildi.\nMatn yoki rasm yuboring.\n/cancel - chiqish")
}

func (s *SearchScene) Handle(c tele.Context) (done bool, err error) {
	if strings.EqualFold(strings.TrimSpace(c.Text()), "/cancel") {
		return true, c.Send("Qidiruv rejimi yopildi.")
	}

	msg := c.Message()
	if msg == nil {
		return false, c.Send("Matn yoki rasm yuboring.")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	var (
		queryType string
		vector    []float64
	)

	if msg.Photo != nil {
		imageURL, err := resolveTelegramFileURL(c, msg.Photo.FileID)
		if err != nil {
			return false, c.Send("Rasm URL olishda xatolik.")
		}
		vector, err = s.ai.ImageURIToVector(ctx, imageURL)
		if err != nil {
			return false, c.Send("Rasmni vector qilishda xatolik.")
		}
		queryType = "image"
	} else {
		text := strings.TrimSpace(c.Text())
		if text == "" || strings.HasPrefix(text, "/") {
			return false, c.Send("Matn yoki rasm yuboring.")
		}
		vector, err = s.ai.TextToVector(ctx, text)
		if err != nil {
			return false, c.Send("Matnni vector qilishda xatolik.")
		}
		queryType = "text"
	}

	rows, err := s.vectors.SearchSimilar(ctx, queryType, vector, 5)
	if err != nil {
		return false, c.Send("Similarity qidiruvda xatolik.")
	}
	if len(rows) == 0 {
		return false, c.Send("Mos natija topilmadi.")
	}

	var b strings.Builder
	b.WriteString("Top matches:\n")
	for i, row := range rows {
		b.WriteString(fmt.Sprintf("%d) score=%.4f", i+1, row.Score))
		if row.Text.Valid && row.Text.String != "" {
			b.WriteString(" | text=" + row.Text.String)
		}
		if row.ImageURL.Valid && row.ImageURL.String != "" {
			b.WriteString(" | image_url=" + row.ImageURL.String)
		}
		b.WriteString("\n")
	}

	return false, c.Send(strings.TrimSpace(b.String()))
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
