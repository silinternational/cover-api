package notifications

import (
	"github.com/gobuffalo/buffalo/render"
)

var EmailRenderer = render.New(render.Options{
	HTMLLayout:   "layout.plush.html",
	TemplatesBox: packr.New("app:mailers:templates", "../templates/mail"),
	Helpers:      render.Helpers{},
})

type EmailService interface {
	Send(msg Message) error
}
