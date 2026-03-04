package scenes

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"yphx-bot/internal/repository"

	tele "gopkg.in/telebot.v3"
)

const SearchButtonText = "🔎 Search"
const defaultSimilarityThreshold = 0.7

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
	mu         sync.RWMutex
	pages      map[int64][]repository.VectorSearchResult // key: search_log_id
	adminInfo  map[int64]int64                           // key: admin_tg_user_id -> target_user_tg_user_id
}

func NewSearchScene(ai vectorClient, vectors *repository.VectorsRepo, searchLogs *repository.SearchLogsRepo, users *repository.UsersRepo) *SearchScene {
	return &SearchScene{
		ai:         ai,
		vectors:    vectors,
		searchLogs: searchLogs,
		users:      users,
		pages:      make(map[int64][]repository.VectorSearchResult),
		adminInfo:  make(map[int64]int64),
	}
}

func (s *SearchScene) Start(c tele.Context) error {
	return c.Send("Qidiruv rejimi yoqildi.\nMatn yoki rasm yuboring.\n/cancel - chiqish")
}

func (s *SearchScene) Handle(c tele.Context) (done bool, err error) {
	if cb := c.Callback(); cb != nil && strings.HasPrefix(cb.Data, "search_react:") {
		return false, s.handleReactionCallback(c)
	}
	if cb := c.Callback(); cb != nil && strings.HasPrefix(cb.Data, "search_page:") {
		return false, s.handlePageCallback(c)
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
		rows, err = s.vectors.SearchSimilarImage(ctx, vector, defaultSimilarityThreshold, 30)
	} else {
		rows, err = s.vectors.SearchSimilarText(ctx, vector, defaultSimilarityThreshold, 30)
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

	resultText := fmt.Sprintf(
		"Top matches (score >= %.2f)\nFound: %d",
		defaultSimilarityThreshold,
		len(rows),
	)
	logID, err := s.createSuccessLog(c, queryType, resultText)
	if err != nil {
		log.Printf("create success log error: %v", err)
		return true, c.Send(resultText)
	}

	s.mu.Lock()
	s.pages[logID] = rows
	s.mu.Unlock()

	return true, s.sendPage(c, logID, 0)
}

func (s *SearchScene) HandleReactionCallback(c tele.Context) error {
	return s.handleReactionCallback(c)
}

func (s *SearchScene) HandlePageCallback(c tele.Context) error {
	return s.handlePageCallback(c)
}

func (s *SearchScene) HandleAdminNotFoundCallback(c tele.Context) error {
	cb := c.Callback()
	if cb == nil {
		return nil
	}
	_ = c.Respond()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	role, err := s.users.GetRoleByTgUserID(ctx, c.Sender().ID)
	if err != nil {
		return c.Send("Admin tekshirishda xatolik.")
	}
	if role != "admin" {
		return c.Send("Faqat admin bu amalni bajara oladi.")
	}

	parts := strings.Split(cb.Data, ":")
	if len(parts) != 2 {
		return c.Send("Noto'g'ri callback.")
	}
	logID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return c.Send("Noto'g'ri log id.")
	}

	notFoundLog, err := s.searchLogs.GetByID(ctx, logID)
	if err != nil {
		return c.Send("Log olishda xatolik.")
	}
	if notFoundLog == nil {
		return c.Send("Log topilmadi.")
	}

	switch parts[0] {
	case "nf_retry":
		return s.retryNotFound(c, notFoundLog)
	case "nf_ignore":
		if err := s.searchLogs.SetStatus(ctx, logID, repository.SearchStatusIgnored); err != nil {
			return c.Send("Statusni yangilashda xatolik.")
		}
		_, err := c.Bot().Send(&tele.User{ID: notFoundLog.TgUserID}, "Sizning so'rovingiz yo'l harakati qoidalariga mos deb topilmadi.")
		if err != nil {
			return c.Send("Userga xabar yuborib bo'lmadi.")
		}
		return c.Send("Ignore qilindi va userga xabar yuborildi.")
	case "nf_info":
		s.mu.Lock()
		s.adminInfo[c.Sender().ID] = notFoundLog.TgUserID
		s.mu.Unlock()
		return c.Send("Userga yuboriladigan matnni yozing. Bekor qilish: /cancel")
	default:
		return c.Send("Noma'lum amal.")
	}
}

func (s *SearchScene) HandlePendingAdminInfo(c tele.Context) (bool, error) {
	adminID := c.Sender().ID

	s.mu.RLock()
	targetUserID, ok := s.adminInfo[adminID]
	s.mu.RUnlock()
	if !ok {
		return false, nil
	}

	text := strings.TrimSpace(c.Text())
	if strings.EqualFold(text, "/cancel") {
		s.mu.Lock()
		delete(s.adminInfo, adminID)
		s.mu.Unlock()
		return true, c.Send("Info yuborish bekor qilindi.")
	}
	if text == "" || strings.HasPrefix(text, "/") {
		return true, c.Send("Iltimos, oddiy matn yuboring. Bekor qilish: /cancel")
	}

	_, err := c.Bot().Send(&tele.User{ID: targetUserID}, text)
	if err != nil {
		return true, c.Send("Userga info yuborishda xatolik.")
	}

	s.mu.Lock()
	delete(s.adminInfo, adminID)
	s.mu.Unlock()
	return true, c.Send("Info userga yuborildi.")
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

func (s *SearchScene) handlePageCallback(c tele.Context) error {
	cb := c.Callback()
	if cb == nil {
		return nil
	}
	_ = c.Respond()

	parts := strings.Split(cb.Data, ":")
	if len(parts) != 3 || parts[0] != "search_page" {
		return nil
	}

	logID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return c.Send("Pagination formati xato.")
	}
	page, err := strconv.Atoi(parts[2])
	if err != nil {
		return c.Send("Pagination sahifa xato.")
	}

	return s.sendPage(c, logID, page)
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
	markup := &tele.ReplyMarkup{
		InlineKeyboard: [][]tele.InlineButton{
			{{Text: "Retry", Data: fmt.Sprintf("nf_retry:%d", logID)}},
			{{Text: "Ignore", Data: fmt.Sprintf("nf_ignore:%d", logID)}},
			{{Text: "Info", Data: fmt.Sprintf("nf_info:%d", logID)}},
		},
	}
	for _, adminID := range adminIDs {
		_, err := c.Bot().Send(&tele.User{ID: adminID}, alert, markup)
		if err != nil {
			log.Printf("notify admin %d error: %v", adminID, err)
		}
	}
}

func (s *SearchScene) retryNotFound(c tele.Context, notFoundLog *repository.SearchLog) error {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	var (
		rows []repository.VectorSearchResult
		err  error
	)

	if notFoundLog.QueryType == "image" {
		imgBytes, fileName, err := readImageForRetry(notFoundLog.QueryImageURL)
		if err != nil {
			return c.Send("Retry image olishda xatolik.")
		}
		vector, err := s.ai.ImageUploadToVector(ctx, fileName, imgBytes)
		if err != nil {
			return c.Send("Retry image vector xatolik.")
		}
		rows, err = s.vectors.SearchSimilarImage(ctx, vector, defaultSimilarityThreshold, 1)
	} else {
		vector, err := s.ai.TextToVector(ctx, strings.TrimSpace(notFoundLog.QueryText))
		if err != nil {
			return c.Send("Retry text vector xatolik.")
		}
		rows, err = s.vectors.SearchSimilarText(ctx, vector, defaultSimilarityThreshold, 1)
	}
	if err != nil {
		return c.Send("Retry search xatolik.")
	}
	if len(rows) == 0 {
		return c.Send("Retry not found.")
	}

	row := rows[0]
	textValue := ""
	if row.Text.Valid {
		textValue = strings.TrimSpace(row.Text.String)
	}
	if textValue == "" {
		textValue = "(caption/text yo'q)"
	}
	reply := fmt.Sprintf("Top result\nType: %s\nScore: %.4f\nText: %s", row.Type, row.Score, textValue)

	if row.Type == "image" && row.ImageURL.Valid {
		imagePath := strings.TrimSpace(row.ImageURL.String)
		if st, statErr := os.Stat(imagePath); statErr == nil && !st.IsDir() {
			_, sendErr := c.Bot().Send(&tele.User{ID: notFoundLog.TgUserID}, &tele.Photo{
				File:    tele.FromDisk(imagePath),
				Caption: reply,
			})
			if sendErr == nil {
				return c.Send("Retry topildi va userga yuborildi.")
			}
		}
	}

	_, sendErr := c.Bot().Send(&tele.User{ID: notFoundLog.TgUserID}, reply)
	if sendErr != nil {
		return c.Send("Userga natija yuborishda xatolik.")
	}
	return c.Send("Retry topildi va userga yuborildi.")
}

func (s *SearchScene) sendPage(c tele.Context, logID int64, page int) error {
	s.mu.RLock()
	rows, ok := s.pages[logID]
	s.mu.RUnlock()
	if !ok || len(rows) == 0 {
		return c.Send("Pagination session topilmadi.")
	}
	if page < 0 {
		page = 0
	}
	if page >= len(rows) {
		page = len(rows) - 1
	}

	row := rows[page]
	textValue := ""
	if row.Text.Valid {
		textValue = strings.TrimSpace(row.Text.String)
	}
	if textValue == "" {
		if row.Type == "image" {
			textValue = "(caption yo'q)"
		} else {
			textValue = "(text yo'q)"
		}
	}

	resultText := fmt.Sprintf(
		"Result %d/%d\nType: %s\nScore: %.4f\nText: %s",
		page+1, len(rows), row.Type, row.Score, textValue,
	)

	buttons := make([][]tele.InlineButton, 0, 4)
	if page > 0 {
		buttons = append(buttons, []tele.InlineButton{
			{Text: "⬅️ Prev", Data: fmt.Sprintf("search_page:%d:%d", logID, page-1)},
		})
	}
	if page < len(rows)-1 {
		buttons = append(buttons, []tele.InlineButton{
			{Text: "➡️ Next", Data: fmt.Sprintf("search_page:%d:%d", logID, page+1)},
		})
	}
	buttons = append(buttons, []tele.InlineButton{
		{Text: "👍 Like", Data: fmt.Sprintf("search_react:%d:like", logID)},
	})
	buttons = append(buttons, []tele.InlineButton{
		{Text: "👎 Dislike", Data: fmt.Sprintf("search_react:%d:dislike", logID)},
	})

	markup := &tele.ReplyMarkup{InlineKeyboard: buttons}
	if row.Type == "image" && row.ImageURL.Valid && strings.TrimSpace(row.ImageURL.String) != "" {
		imagePath := strings.TrimSpace(row.ImageURL.String)
		if st, err := os.Stat(imagePath); err == nil && !st.IsDir() {
			return c.Send(&tele.Photo{
				File:    tele.FromDisk(imagePath),
				Caption: resultText,
			}, markup)
		}
	}

	return c.Send(resultText, markup)
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

func readImageForRetry(source string) ([]byte, string, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil, "", fmt.Errorf("empty image source")
	}

	if st, err := os.Stat(source); err == nil && !st.IsDir() {
		data, readErr := os.ReadFile(source)
		if readErr != nil {
			return nil, "", readErr
		}
		return data, filepath.Base(source), nil
	}

	resp, err := http.Get(source)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, "", fmt.Errorf("http status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 20<<20))
	if err != nil {
		return nil, "", err
	}
	fileName := filepath.Base(source)
	if fileName == "" || fileName == "." || strings.Contains(fileName, "?") {
		fileName = fmt.Sprintf("retry_%d.bin", time.Now().UnixNano())
	}
	return data, fileName, nil
}
