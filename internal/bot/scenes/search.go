package scenes

import (
	"context"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"yphx-bot/internal/repository"

	tele "gopkg.in/telebot.v3"
)

const SearchButtonText = "🔎 Search"

type vectorClient interface {
	TextToVector(ctx context.Context, text string) ([]float64, error)
	ImageURIToVector(ctx context.Context, uri string) ([]float64, error)
	ImageUploadToVector(ctx context.Context, fileName string, data []byte) ([]float64, error)
}

type SearchScene struct {
	ai         vectorClient
	vectors    *repository.VectorsRepo
	searchLogs *repository.SearchLogsRepo
	users      *repository.UsersRepo
}

func NewSearchScene(ai vectorClient, vectors *repository.VectorsRepo, searchLogs *repository.SearchLogsRepo, users *repository.UsersRepo) *SearchScene {
	return &SearchScene{ai: ai, vectors: vectors, searchLogs: searchLogs, users: users}
}

func (s *SearchScene) Start(c tele.Context) error {
	return c.Send("Qidiruv rejimi yoqildi.\nMatn yoki rasm yuboring.\n/cancel - chiqish")
}

func (s *SearchScene) Handle(c tele.Context) (done bool, err error) {
	if cb := c.Callback(); cb != nil && strings.HasPrefix(cb.Data, "search_react:") {
		return false, s.handleReactionCallback(c)
	}

	if strings.EqualFold(strings.TrimSpace(c.Text()), "/cancel") {
		return true, c.Send("Qidiruv rejimi yopildi.")
	}

	msg := c.Message()
	if msg == nil {
		_, _ = s.createIgnoredLog(c, "", "")
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
			_, _ = s.createIgnoredLog(c, "image", "")
			return false, c.Send("Rasm URL olishda xatolik.")
		}
		fileBytes, fileName, err := downloadTelegramFileBytes(c, msg.Photo.FileID)
		if err != nil {
			_, _ = s.createIgnoredLog(c, "image", imageURL)
			return false, c.Send("Rasm faylini yuklashda xatolik.")
		}
		vector, err = s.ai.ImageUploadToVector(ctx, fileName, fileBytes)
		if err != nil {
			_, _ = s.createIgnoredLog(c, "image", imageURL)
			return false, c.Send("Rasmni vector qilishda xatolik.")
		}
		queryType = "image"
	} else {
		text := strings.TrimSpace(c.Text())
		if text == "" || strings.HasPrefix(text, "/") {
			_, _ = s.createIgnoredLog(c, "text", text)
			return false, c.Send("Matn yoki rasm yuboring.")
		}
		vector, err = s.ai.TextToVector(ctx, text)
		if err != nil {
			_, _ = s.createIgnoredLog(c, "text", text)
			return false, c.Send("Matnni vector qilishda xatolik.")
		}
		queryType = "text"
	}

	var rows []repository.VectorSearchResult
	if queryType == "image" {
		rows, err = s.vectors.SearchSimilarImage(ctx, vector, 5)
	} else {
		rows, err = s.vectors.SearchSimilarText(ctx, vector, 5)
	}
	if err != nil {
		_, _ = s.createIgnoredLog(c, queryType, strings.TrimSpace(c.Text()))
		return false, c.Send("Similarity qidiruvda xatolik.")
	}
	if len(rows) == 0 {
		logID, logErr := s.logNotFound(c, queryType)
		if logErr != nil {
			log.Printf("log not_found error: %v", logErr)
		} else {
			s.notifyAdminsAboutNotFound(c, logID, queryType)
		}
		return true, c.Send("Mos natija topilmadi.")
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

	resultText := strings.TrimSpace(b.String())
	logID, err := s.createSuccessLog(c, queryType, resultText)
	if err != nil {
		log.Printf("create success log error: %v", err)
		return true, c.Send(resultText)
	}

	markup := &tele.ReplyMarkup{
		InlineKeyboard: [][]tele.InlineButton{
			{
				{Text: "👍 Like", Data: fmt.Sprintf("search_react:%d:like", logID)},
				{Text: "👎 Dislike", Data: fmt.Sprintf("search_react:%d:dislike", logID)},
			},
		},
	}

	return true, c.Send(resultText, markup)
}

func (s *SearchScene) HandleReactionCallback(c tele.Context) error {
	return s.handleReactionCallback(c)
}

func (s *SearchScene) handleReactionCallback(c tele.Context) error {
	cb := c.Callback()
	if cb == nil {
		return nil
	}
	_ = c.Respond()

	parts := strings.Split(cb.Data, ":")
	if len(parts) != 3 || parts[0] != "search_react" {
		return nil
	}
	logID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return c.Send("Reaction formati xato.")
	}
	reaction := parts[2]
	if reaction != "like" && reaction != "dislike" {
		return c.Send("Reaction noto'g'ri.")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ok, err := s.searchLogs.SetReaction(ctx, logID, c.Sender().ID, reaction)
	if err != nil {
		return c.Send("Reaction saqlashda xatolik.")
	}
	if !ok {
		return c.Send("Bu javobga reaction qo'yib bo'lmadi.")
	}

	return c.Send("Fikringiz uchun rahmat.")
}

func (s *SearchScene) createSuccessLog(c tele.Context, queryType, resultText string) (int64, error) {
	msg := c.Message()
	logInput := repository.SearchLog{
		TgUserID:   c.Sender().ID,
		QueryType:  queryType,
		ResultText: resultText,
		Status:     repository.SearchStatusSuccess,
	}
	if msg != nil && msg.Photo != nil {
		imageURL, _ := resolveTelegramFileURL(c, msg.Photo.FileID)
		logInput.QueryImageURL = imageURL
	} else {
		logInput.QueryText = strings.TrimSpace(c.Text())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.searchLogs.Create(ctx, logInput)
}

func (s *SearchScene) logNotFound(c tele.Context, queryType string) (int64, error) {
	msg := c.Message()
	logInput := repository.SearchLog{
		TgUserID:   c.Sender().ID,
		QueryType:  queryType,
		ResultText: "not found",
		Status:     repository.SearchStatusNotFound,
	}
	if msg != nil && msg.Photo != nil {
		imageURL, _ := resolveTelegramFileURL(c, msg.Photo.FileID)
		logInput.QueryImageURL = imageURL
	} else {
		logInput.QueryText = strings.TrimSpace(c.Text())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.searchLogs.Create(ctx, logInput)
}

func (s *SearchScene) createIgnoredLog(c tele.Context, queryType, raw string) (int64, error) {
	logInput := repository.SearchLog{
		TgUserID:   c.Sender().ID,
		QueryType:  queryType,
		Status:     repository.SearchStatusIgnored,
		ResultText: "ignored",
	}
	if queryType == "image" {
		logInput.QueryImageURL = raw
	} else {
		logInput.QueryText = raw
		if queryType == "" {
			logInput.QueryType = "unknown"
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.searchLogs.Create(ctx, logInput)
}

func (s *SearchScene) notifyAdminsAboutNotFound(c tele.Context, logID int64, queryType string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	adminIDs, err := s.users.ListAdminTgIDs(ctx)
	if err != nil {
		log.Printf("list admins error: %v", err)
		return
	}
	if len(adminIDs) == 0 {
		return
	}

	var queryValue string
	msg := c.Message()
	if msg != nil && msg.Photo != nil {
		queryValue, _ = resolveTelegramFileURL(c, msg.Photo.FileID)
	} else {
		queryValue = strings.TrimSpace(c.Text())
	}

	alert := fmt.Sprintf(
		"NOT_FOUND search\nlog_id=%d\nuser_id=%d\ntype=%s\nquery=%s",
		logID, c.Sender().ID, queryType, queryValue,
	)
	for _, adminID := range adminIDs {
		_, err := c.Bot().Send(&tele.User{ID: adminID}, alert)
		if err != nil {
			log.Printf("notify admin %d error: %v", adminID, err)
		}
	}
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

func downloadTelegramFileBytes(c tele.Context, fileID string) ([]byte, string, error) {
	fileMeta, err := c.Bot().FileByID(fileID)
	if err != nil {
		return nil, "", fmt.Errorf("get file meta: %w", err)
	}

	reader, err := c.Bot().File(&tele.File{FileID: fileID})
	if err != nil {
		return nil, "", fmt.Errorf("download file: %w", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(io.LimitReader(reader, 20<<20))
	if err != nil {
		return nil, "", fmt.Errorf("read file bytes: %w", err)
	}

	fileName := filepath.Base(fileMeta.FilePath)
	if fileName == "" || fileName == "." {
		fileName = fileID + ".bin"
	}
	return data, fileName, nil
}
