package handlers

import (
	"github.com/torresposso/habi/views/layout"
	"github.com/torresposso/habi/views/pages"

	"github.com/gofiber/fiber/v3"
)

func Landing(c fiber.Ctx) error {
	return Render(c, layout.Base("hello", pages.Landing()))
}
