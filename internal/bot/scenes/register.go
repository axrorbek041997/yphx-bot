package scenes

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"yphx-bot/internal/repository"
)

type registerStep int

const (
	stepAskName registerStep = iota
	stepAskPhone
	stepConfirm
)

type RegisterScene struct {
	users *repository.UsersRepo

	mu    sync.RWMutex
	step  map[int64]registerStep
	name  map[int64]string
	phone map[int64]string
}

func NewRegisterScene(users *repository.UsersRepo) *RegisterScene {
	return &RegisterScene{
		users: users,
		step:  make(map[int64]registerStep),
		name:  make(map[int64]string),
		phone: make(map[int64]string),
	}
}

func (s *RegisterScene) Start(api *tgbotapi.BotAPI, m *tgbotapi.Message) error {
	uid := int64(m.From.ID)
	s.mu.Lock()
	s.step[uid] = stepAskName
	s.mu.Unlock()

	log.Printf("RegisterScene.Start: step=%v", s.step[uid])

	_, err := api.Send(tgbotapi.NewMessage(m.Chat.ID, "Ro'yxatdan o'tish.\nIsmingizni kiriting (masalan: Axror):\n/cancel - bekor qilish"))
	return err
}

func (s *RegisterScene) Handle(api *tgbotapi.BotAPI, m *tgbotapi.Message) (done bool, err error) {
	log.Printf("RegisterScene.Handle: user=%d text=%q", m.From.ID, m.Text)

	uid := int64(m.From.ID)
	text := strings.TrimSpace(m.Text)

	if strings.EqualFold(text, "/cancel") {
		s.cleanup(uid)
		_, _ = api.Send(tgbotapi.NewMessage(m.Chat.ID, "Bekor qilindi ✅"))
		return true, nil
	}

	s.mu.RLock()
	st, ok := s.step[uid]
	s.mu.RUnlock()
	if !ok {
		// Step yo'q bo'lsa, scene start qilinmagan
		return true, nil
	}

	switch st {
	case stepAskName:
		if len(text) < 2 {
			_, err = api.Send(tgbotapi.NewMessage(m.Chat.ID, "Ism juda qisqa. Qayta kiriting:"))
			return false, err
		}
		s.mu.Lock()
		s.name[uid] = text
		s.step[uid] = stepAskPhone
		s.mu.Unlock()

		_, err = api.Send(tgbotapi.NewMessage(m.Chat.ID, "Telefon raqamingizni kiriting (masalan: +998901234567):"))
		return false, err

	case stepAskPhone:
		if !looksLikePhone(text) {
			_, err = api.Send(tgbotapi.NewMessage(m.Chat.ID, "Telefon formati noto'g'ri. Masalan: +998901234567"))
			return false, err
		}
		s.mu.Lock()
		s.phone[uid] = text
		s.step[uid] = stepConfirm
		name := s.name[uid]
		phone := s.phone[uid]
		s.mu.Unlock()

		msg := fmt.Sprintf("Tasdiqlaysizmi?\nIsm: %s\nTelefon: %s\n\nHa bo'lsa: yes\nYo'q bo'lsa: no", name, phone)
		_, err = api.Send(tgbotapi.NewMessage(m.Chat.ID, msg))
		return false, err

	case stepConfirm:
		ans := strings.ToLower(text)
		if ans == "no" || ans == "yo'q" || ans == "yq" {
			s.cleanup(uid)
			_, err = api.Send(tgbotapi.NewMessage(m.Chat.ID, "Bekor qilindi ✅ /register bilan qayta boshlang."))
			return true, err
		}
		if ans != "yes" && ans != "ha" && ans != "ok" {
			_, err = api.Send(tgbotapi.NewMessage(m.Chat.ID, "Iltimos, 'yes' yoki 'no' deb yozing."))
			return false, err
		}

		s.mu.RLock()
		name := s.name[uid]
		phone := s.phone[uid]
		s.mu.RUnlock()

		username := ""
		if m.From != nil {
			username = m.From.UserName
		}

		if err := s.users.UpsertRegister(context.Background(), uid, username, name, phone); err != nil {
			_, _ = api.Send(tgbotapi.NewMessage(m.Chat.ID, "Saqlashda xatolik. Keyinroq urinib ko'ring."))
			return false, err
		}

		s.cleanup(uid)
		_, err = api.Send(tgbotapi.NewMessage(m.Chat.ID, "Ro'yxatdan o'tdingiz ✅ Endi botdan foydalanishingiz mumkin. /help"))
		return true, err
	}

	return false, nil
}

func (s *RegisterScene) cleanup(uid int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.step, uid)
	delete(s.name, uid)
	delete(s.phone, uid)
}

func looksLikePhone(x string) bool {
	x = strings.ReplaceAll(x, " ", "")
	if strings.HasPrefix(x, "+") && len(x) >= 12 {
		return true
	}
	if strings.HasPrefix(x, "998") && len(x) >= 12 {
		return true
	}
	return false
}
