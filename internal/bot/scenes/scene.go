package scenes

import tele "gopkg.in/telebot.v3"

type Scene interface {
	Start(c tele.Context) error
	Handle(c tele.Context) (done bool, err error)
}
