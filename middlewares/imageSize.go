package middlewares

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
)


func CheckImageSize(c *fiber.Ctx) error{
	
	const MaxAllowedFileSize = 30 * 1024 * 1024;

	fileheader,err  := c.FormFile(`image`);

	if err != nil {
		
		slog.Error(`Could not get image file in check image size middleware. Error: ` + err.Error())
		return c.Status(500).JSON(fiber.Map{`message`:`Something went wrong.`})
	}

	if fileheader.Size > MaxAllowedFileSize{
		return c.Status(400).JSON(fiber.Map{`message`:`File size too big.`})
	}

	return c.Next();
}


