package main

import (
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/torresposso/habi/handlers"
)

func main() {

	app := fiber.New(fiber.Config{
		AppName: "Habi",
	})

	app.Use("/static", static.New("./static"))

	app.Get("/", func(c fiber.Ctx) error {
		return handlers.Landing(c)
	})

	log.Printf("Server starting on port 8080")
	if err := app.Listen(":8080"); err != nil {
		log.Fatal(err)
	}
}
