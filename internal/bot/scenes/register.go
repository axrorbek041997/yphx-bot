package scenes

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	tele "gopkg.in/telebot.v3"
)

// ---- Repo interface (so you can plug your real repo easily)
type UsersRepo interface {
	ExistsByTgUserID(ctx context.Context, telegramID int64) (bool, error)
	UpsertRegister(ctx context.Context, telegramID int64, username, name, phone string) error
}

type registerStep int

const (
	stepAskName registerStep = iota
	stepAskPhone
	stepConfirm
)

type RegisterScene struct {
	users UsersRepo

	mu    sync.RWMutex
	step  map[int64]registerStep
	name  map[int64]string
	phone map[int64]string
}

func NewRegisterScene(users UsersRepo) *RegisterScene {
	return &RegisterScene{
		users: users,
		step:  make(map[int64]registerStep),
		name:  make(map[int64]string),
		phone: make(map[int64]string),
	}
}

func (s *RegisterScene) Start(c tele.Context) error {
	uid := c.Sender().ID

	s.mu.Lock()
	s.step[uid] = stepAskName
	s.mu.Unlock()

	log.Printf("RegisterScene.Start: user=%d step=AskName", uid)

	return c.Send("Ro'yxatdan o'tish.\nIsmingizni kiriting (masalan: Axror):\n/cancel - bekor qilish")
}

func (s *RegisterScene) Handle(c tele.Context) (done bool, err error) {
	uid := c.Sender().ID
	text := strings.TrimSpace(c.Text())

	log.Printf("RegisterScene.Handle: user=%d text=%q", uid, text)

	if strings.EqualFold(text, "/cancel") {
		s.cleanup(uid)
		_ = c.Send("Bekor qilindi ✅")
		return true, nil
	}

	s.mu.RLock()
	st, ok := s.step[uid]
	s.mu.RUnlock()
	if !ok {
		// Scene start qilinmagan bo'lishi mumkin
		return true, nil
	}

	switch st {
	case stepAskName:
		if len([]rune(text)) < 2 {
			return false, c.Send("Ism juda qisqa. Qayta kiriting:")
		}
		s.mu.Lock()
		s.name[uid] = text
		s.step[uid] = stepAskPhone
		s.mu.Unlock()

		return false, c.Send("Telefon raqamingizni kiriting (masalan: +998901234567):")

	case stepAskPhone:
		if !looksLikePhone(text) {
			return false, c.Send("Telefon formati noto'g'ri. Masalan: +998901234567")
		}

		s.mu.Lock()
		s.phone[uid] = text
		s.step[uid] = stepConfirm
		name := s.name[uid]
		phone := s.phone[uid]
		s.mu.Unlock()

		msg := fmt.Sprintf("Tasdiqlaysizmi?\nIsm: %s\nTelefon: %s\n\nHa bo'lsa: yes\nYo'q bo'lsa: no", name, phone)
		return false, c.Send(msg)

	case stepConfirm:
		ans := strings.ToLower(text)
		if ans == "no" || ans == "yo'q" || ans == "yq" {
			s.cleanup(uid)
			_ = c.Send("Bekor qilindi ✅ /register bilan qayta boshlang.")
			return true, nil
		}
		if ans != "yes" && ans != "ha" && ans != "ok" {
			return false, c.Send("Iltimos, 'yes' yoki 'no' deb yozing.")
		}

		s.mu.RLock()
		name := s.name[uid]
		phone := s.phone[uid]
		s.mu.RUnlock()

		username := ""
		if c.Sender() != nil {
			username = c.Sender().Username
		}

		// DB write with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		if err := s.users.UpsertRegister(ctx, uid, username, name, phone); err != nil {
			_ = c.Send("Saqlashda xatolik. Keyinroq urinib ko'ring.")
			return false, err
		}

		s.cleanup(uid)
		_ = c.Send("Ro'yxatdan o'tdingiz ✅ Endi botdan foydalanishingiz mumkin. /help")
		return true, nil
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
