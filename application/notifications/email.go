package notifications

import (
	"github.com/gobuffalo/buffalo/render"
	"github.com/silinternational/cover-api/templates"
)

var EmailRenderer = render.New(render.Options{
	HTMLLayout:  "mail/layout.plush.html",
	TemplatesFS: templates.FS(),
	Helpers:     render.Helpers{},
})

type EmailService interface {
	Send(msg Message) error
}
