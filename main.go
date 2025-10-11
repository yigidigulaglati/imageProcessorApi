package main

import (
	"imageProcessorAPI/handlers"
	"imageProcessorAPI/middlewares"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)


func main(){
	
	app := fiber.New();

	app.Use(limiter.New(limiter.Config{
		Next: func(c *fiber.Ctx) bool{
			return c.IP() == `127.0.0.1`;
		},
		KeyGenerator: func(c *fiber.Ctx) string{
			return c.IP();
		},
		Max: 10,
		Expiration: time.Second * 30,
		LimitReached: func (c *fiber.Ctx) error{
			return c.Status(fiber.StatusTooManyRequests).SendString(`Limit reached.`);
		},
	}))


	app.Use(middlewares.CheckImageSize);

	app.Post(`/resize`, handlers.Resize);
	app.Post(`/crop`, handlers.Crop);
	app.Post(`/flip`, handlers.Flip);
	app.Post(`/grayscale`, handlers.GrayScale);
	app.Post(`/changeformat`, handlers.ChangeFormat);
	app.Post(`/rotate`, handlers.Rotate);


	err := app.Listen(`:8000`);
	if err != nil {
		log.Fatal(err.Error());
	}
}