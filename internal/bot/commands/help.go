package commands

import tele "gopkg.in/telebot.v3"

func Help(c tele.Context) error {
	text := "Komandalar:\n/start\n/help\n/ping"
	_, err := c.Bot().Send(c.Chat(), text)
	return err
}
