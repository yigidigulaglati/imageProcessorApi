package handlers

import (
	"context"
	"encoding/json"
	"image"
	"image/color"
	"imageProcessorAPI/utilities"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/gofiber/fiber/v2"
)

type RotateBody struct {
	Angle *int `json:"angle"`
}

func Rotate(c *fiber.Ctx) error {

	ctx, cancelCtx := context.WithTimeout(context.Background(), time.Second*30)
	defer cancelCtx()

	body := c.FormValue(`metadata`)
	if body == `` {
		return c.Status(400).JSON(fiber.Map{`message`: `Angle must be set.`})
	}
	bodyData := RotateBody{}
	err := json.Unmarshal([]byte(body), &bodyData)
	if err != nil {
		slog.Error(`Could not unmarshal rotate body. Error: ` + err.Error())
		return fiber.NewError(500, `Something went wrong.`)
	}

	if bodyData.Angle == nil {
		slog.Info(`Angle is nil in rotate handler.`)
		return c.Status(400).JSON(fiber.Map{`message`: `Angle must be set.`})
	}

	file, err := c.FormFile(`image`)
	if err != nil {
		return fiber.NewError(500, `Could not open image.`)
	}
	mimeType := file.Header.Get("Content-Type")

	if mimeType != "image/jpeg" && mimeType != "image/png" {
		return c.Status(fiber.StatusBadRequest).SendString("Unsupported file type: " + mimeType)
	}

	fileExt := strings.ToLower(filepath.Ext(file.Filename))
	if fileExt != `png` && fileExt != `jpeg` && fileExt != `jpg` {
		return fiber.NewError(400, `Expected png or jpg image.`)
	}

	fileStream, err := file.Open()
	if err != nil {
		return fiber.NewError(500, `Could not open image.`)
	}
	defer fileStream.Close()

	var errChan = make(chan error)
	var imageChan = make(chan image.Image)
	go func() {
		decodedImage, err := imaging.Decode(fileStream)

		if err != nil {
			errChan <- err
			return
		}

		imageChan <- decodedImage
	}()

	var decodedImage image.Image
	select {
	case img := <-imageChan:
		decodedImage = img
	case err := <-errChan:
		slog.Error(`Could not decode image. Error: ` + err.Error())
		return c.Status(500).JSON(fiber.Map{`message`: `Something went wrong.`})
	case <-ctx.Done():
		slog.Info(`Context canceled in rotate handler.`)
		return c.Status(fiber.ErrRequestTimeout.Code).JSON(fiber.Map{`message`: `Timeout.`})
	}

	validImageBounds := utilities.CheckImageBounds(&decodedImage)

	if !validImageBounds {
		return c.Status(400).JSON(fiber.Map{`message`: `Image bound too big.`})
	}

	if ctx.Err() != nil {
		return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{`message`: `Timeout.`})
	}

	rotatedImage := imaging.Rotate(decodedImage, float64(*bodyData.Angle), color.White)

	if ctx.Err() != nil {
		return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{`message`: `Timeout.`})
	}

	writer := c.Response().BodyWriter()
	if fileExt == `png` {
		c.Type(`image/png`)
		err = imaging.Encode(writer, rotatedImage, imaging.PNG)
	} else {
		c.Type(`image/jpeg`)
		err = imaging.Encode(writer, rotatedImage, imaging.JPEG, imaging.JPEGQuality(85))
	}

	if err != nil {
		if ctx.Err() != nil{
			slog.Error(`Encode failed due to context timeout. Error: ` + err.Error());
			return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{`message`:`Timeout.`});
		}

		slog.Error(`Could not encode image in rotate handler. Error: ` + err.Error())
		return c.Status(500).JSON(fiber.Map{`message`: `Something went wrong.`})
	}


	return nil
}

type CropMetaData struct {
	MinX *int `json:"minX"`
	MinY *int `json:"minY"`

	MaxX *int `json:"maxX"`
	MaxY *int `json:"maxY"`
}

func Crop(c *fiber.Ctx) error {

	ctx, cancelCtx := context.WithTimeout(context.Background(), time.Second*30)
	defer cancelCtx()

	metadata := c.FormValue(`metadata`)
	if metadata == `` {
		return c.Status(400).JSON(fiber.Map{`message`: `Must set bounds to crop image.`})
	}
	data := CropMetaData{}
	err := json.Unmarshal([]byte(metadata), &data)
	if err != nil {
		slog.Error(`Could not unmarshal metadata in crop handler. Error: ` + err.Error())
		return fiber.NewError(500, `Something went wrong.`)
	}

	if data.MaxX == nil || data.MinX == nil || data.MinY == nil || data.MaxY == nil {
		return c.Status(400).JSON(fiber.Map{`message`: `Must set bounds to crop the image.`})
	}

	if *data.MaxX < *data.MinX || *data.MaxY < *data.MinY {
		return c.Status(400).JSON(fiber.Map{`message`: `Invalid bounds.`})
	}

	if *data.MaxX < 0 || *data.MinX < 0 || *data.MaxY < 0 || *data.MinY < 0 {
		return c.Status(400).JSON(fiber.Map{`message`: `Invalid bounds.`})
	}

	rec := image.Rectangle{
		Min: image.Point{
			X: *data.MinX,
			Y: *data.MinY,
		},
		Max: image.Point{
			X: *data.MaxX,
			Y: *data.MaxY,
		},
	}

	file, err := c.FormFile(`image`)
	if err != nil {
		return fiber.NewError(500, `Could not open image.`)
	}

	mimeType := file.Header.Get("Content-Type")

	if mimeType != "image/jpeg" && mimeType != "image/png" {
		return c.Status(fiber.StatusBadRequest).SendString("Unsupported file type: " + mimeType)
	}

	fileExt := strings.ToLower(filepath.Ext(file.Filename))
	if fileExt != `png` && fileExt != `jpeg` && fileExt != `jpg` {
		return fiber.NewError(400, `Expected png or jpg image.`)
	}

	fileStream, err := file.Open()
	if err != nil {
		return fiber.NewError(500, `Could not open image.`)
	}
	defer fileStream.Close()

	var errChan = make(chan error)
	var imageChan = make(chan image.Image)
	go func() {
		decodedImage, err := imaging.Decode(fileStream)

		if err != nil {
			errChan <- err
			return
		}

		imageChan <- decodedImage
	}()

	var decodedImage image.Image
	select {
	case img := <-imageChan:
		decodedImage = img
	case err := <-errChan:
		slog.Error(`Could not decode image. Error: ` + err.Error())
		return c.Status(500).JSON(fiber.Map{`message`: `Something went wrong.`})
	case <-ctx.Done():
		slog.Info(`Context canceled in rotate handler.`)
		return c.Status(fiber.ErrRequestTimeout.Code).JSON(fiber.Map{`message`: `Timeout.`})
	}

	validImageBounds := utilities.CheckImageBounds(&decodedImage)

	if !validImageBounds {
		return c.Status(400).JSON(fiber.Map{`message`: `Image bound too big.`})
	}

	if ctx.Err() != nil {
		return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{`message`: `Timeout.`})
	}

	if !rec.In(decodedImage.Bounds()) {
		return c.Status(400).JSON(fiber.Map{`message`: `Invalid bounds.`})
	}

	croppedImage := imaging.Crop(decodedImage, rec)

	if ctx.Err() != nil {
		return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{`message`: `Timeout.`})
	}

	bodyWriter := c.Response().BodyWriter()

	if fileExt == `png` {
		c.Type(`image/png`)
		err = imaging.Encode(bodyWriter, croppedImage, imaging.PNG)
	} else {
		c.Type(`image/jpeg`)
		err = imaging.Encode(bodyWriter, croppedImage, imaging.JPEG, imaging.JPEGQuality(85))
	}
if err != nil {
		if ctx.Err() != nil{
			slog.Error(`Encode failed due to context timeout. Error: ` + err.Error());
			return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{`message`:`Timeout.`});
		}

		slog.Error(`Could not encode image in crop handler. Error: ` + err.Error())
		return c.Status(500).JSON(fiber.Map{`message`: `Something went wrong.`})
	}

	return nil
}

type ResizeMetaData struct {
	Height *int `json:"height"`
	Width  *int `json:"width"`
}

func Resize(c *fiber.Ctx) error {

	ctx, cancelCtx := context.WithTimeout(context.Background(), time.Second*30)
	defer cancelCtx()

	metadata := c.FormValue(`metadata`)

	if metadata == `` {
		return c.Status(400).JSON(fiber.Map{`message`: `Must set height or width to resize image.`})
	}

	data := ResizeMetaData{}
	err := json.Unmarshal([]byte(metadata), &data)
	if err != nil {
		slog.Error(`Could not unmarshal metadata in resize handler. Error: ` + err.Error())
		return c.Status(500).JSON(fiber.Map{`message`: `Something went wrong.`})
	}

	if data.Width == nil && data.Height == nil {
		return c.Status(400).JSON(fiber.Map{`message`: `Must set at least width or height to resize image.`})
	}

	fileHeader, err := c.FormFile(`image`)
	if err != nil {
		slog.Error(`Could not get form file in resize handler. Error: ` + err.Error())
		return c.Status(500).JSON(fiber.Map{`message`: `Something went wrong.`})
	}

	mimeType := fileHeader.Header.Get("Content-Type")

	if mimeType != "image/jpeg" && mimeType != "image/png" {
		return c.Status(fiber.StatusBadRequest).SendString("Unsupported file type: " + mimeType)
	}

	extension := strings.ToLower(filepath.Ext(fileHeader.Filename))

	if extension != `png` && extension != `jpeg` && extension != `jpg` {
		return c.Status(400).JSON(fiber.Map{`message`: `Invalid extension.`})
	}

	fileStream, err := fileHeader.Open()
	if err != nil {
		slog.Error(`Could not open image in resize handler. Error: ` + err.Error())
		return c.Status(500).JSON(fiber.Map{`message`: `Something went wrong.`})
	}
	defer fileStream.Close()

	var errChan = make(chan error)
	var imageChan = make(chan image.Image)
	go func() {
		decodedImage, err := imaging.Decode(fileStream)

		if err != nil {
			errChan <- err
			return
		}

		imageChan <- decodedImage
	}()

	var decodedImage image.Image
	select {
	case img := <-imageChan:
		decodedImage = img
	case err := <-errChan:
		slog.Error(`Could not decode image. Error: ` + err.Error())
		return c.Status(500).JSON(fiber.Map{`message`: `Something went wrong.`})
	case <-ctx.Done():
		slog.Info(`Context canceled in rotate handler.`)
		return c.Status(fiber.ErrRequestTimeout.Code).JSON(fiber.Map{`message`: `Timeout.`})
	}

	validImageBounds := utilities.CheckImageBounds(&decodedImage)

	if !validImageBounds {
		return c.Status(400).JSON(fiber.Map{`message`: `Image bound too big.`})
	}

	if ctx.Err() != nil {
		return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{`message`: `Timeout.`})
	}

	var resizedImage *image.NRGBA
	if data.Width == nil && data.Height != nil && *data.Height > 0 && *data.Height < 2000 {
		resizedImage = imaging.Resize(decodedImage, 0, *data.Height, imaging.Lanczos)
	} else if data.Height == nil && data.Width != nil && *data.Width > 0 && *data.Width < 2000 {
		resizedImage = imaging.Resize(decodedImage, *data.Width, 0, imaging.Lanczos)
	} else if data.Height != nil && *data.Height > 0 && data.Width != nil && *data.Width > 0 && *data.Height < 2000 && *data.Width < 2000 {
		resizedImage = imaging.Resize(decodedImage, *data.Width, *data.Height, imaging.Lanczos)
	} else {
		return c.Status(400).JSON(fiber.Map{`message`: `Invalid parameters.`})
	}

	writer := c.Request().BodyWriter()

	if extension == `png` {
		c.Type(`image/png`)

		err = imaging.Encode(writer, resizedImage, imaging.PNG)
	} else {
		c.Type(`image/jpeg`)

		err = imaging.Encode(writer, resizedImage, imaging.JPEG, imaging.JPEGQuality(85))
	}

	if err != nil {
		if ctx.Err() != nil{
			slog.Error(`Encode failed due to context timeout. Error: ` + err.Error());
			return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{`message`:`Timeout.`});
		}

		slog.Error(`Could not encode image in resize handler. Error: ` + err.Error())
		return c.Status(500).JSON(fiber.Map{`message`: `Something went wrong.`})
	}

	return nil
}

type ChangeFormatMetadata struct {
	FormatName *string `json:"formatName"`
}

func ChangeFormat(c *fiber.Ctx) error {

	ctx, cancelCtx := context.WithTimeout(context.Background(), time.Second*30)
	defer cancelCtx()

	jsonPayload := c.FormValue(`metadata`)
	var metadata = ChangeFormatMetadata{}
	err := json.Unmarshal([]byte(jsonPayload), &metadata)
	if err != nil {
		slog.Error(`Could not unmarshal in change format handler. Error: ` + err.Error())
		return c.Status(500).JSON(fiber.Map{`message`: `Something went wrong.`})
	}

	if metadata.FormatName == nil {
		return c.Status(400).JSON(fiber.Map{`message`: `Must set format name.`})
	}

	formatName := strings.ToLower(*metadata.FormatName)

	if formatName != `png` && formatName != `jpeg` && formatName != `jpg` {
		return c.Status(400).JSON(fiber.Map{`message`: `Invalid format name.`})
	}

	fileheader, err := c.FormFile(`image`)
	if err != nil {
		slog.Error(`Could not get form file image in change format handler. Error: ` + err.Error())
		return c.Status(500).JSON(fiber.Map{`message`: `Something went wrong.`})
	}

	mimeType := fileheader.Header.Get("Content-Type")

	if mimeType != "image/jpeg" && mimeType != "image/png" {
		return c.Status(fiber.StatusBadRequest).SendString("Unsupported file type: " + mimeType)
	}

	extension := strings.ToLower(filepath.Ext(fileheader.Filename))
	if extension != `png` && extension != `jpeg` || extension != `jpeg` {
		return c.Status(400).JSON(fiber.Map{`message`: `Invalid file type.`})
	}

	if extension == formatName {
		return c.Status(400).JSON(fiber.Map{`message`: `Invalid request.`})
	}

	file, err := fileheader.Open()
	if err != nil {
		slog.Error(`Could not open file in change format handler. Error: ` + err.Error())
		return c.Status(500).JSON(fiber.Map{`message`: `Something went wrong.`})
	}
	defer file.Close()

	var errChan = make(chan error)
	var imageChan = make(chan image.Image)
	go func() {
		decodedImage, err := imaging.Decode(file)

		if err != nil {
			errChan <- err
			return
		}

		imageChan <- decodedImage
	}()

	var image image.Image
	select {
	case img := <-imageChan:
		image = img
	case err := <-errChan:
		slog.Error(`Could not decode image. Error: ` + err.Error())
		return c.Status(500).JSON(fiber.Map{`message`: `Something went wrong.`})
	case <-ctx.Done():
		slog.Info(`Context canceled in rotate handler.`)
		return c.Status(fiber.ErrRequestTimeout.Code).JSON(fiber.Map{`message`: `Timeout.`})
	}

	if ctx.Err() != nil {
		return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{`message`: `Timeout.`})
	}

	validImageBounds := utilities.CheckImageBounds(&image)
	if !validImageBounds {
		return c.Status(400).JSON(fiber.Map{`message`: `Image bound too big.`})
	}

	if ctx.Err() != nil {
		return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{`message`: `Timeout.`})
	}

	responseWriter := c.Response().BodyWriter()

	if formatName == `png` {
		c.Type(`image/png`)
		err = imaging.Encode(responseWriter, image, imaging.PNG)
	} else {
		c.Type(`image/jpeg`)

		err = imaging.Encode(responseWriter, image, imaging.JPEG, imaging.JPEGQuality(85))
	}
	if err != nil {
		if ctx.Err() != nil {
			slog.Error(`Encode failed due to context timeout. Error: ` + err.Error())
			return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{`message`: `Timeout.`})
		}

		slog.Error(`Could not encode image in change format handler. Error: ` + err.Error())
		return c.Status(500).JSON(fiber.Map{`message`: `Something went wrong.`})
	}

	return nil
}

type FlipMetadata struct {
	Direction *string `json:"direction"`
}

func Flip(c *fiber.Ctx) error {

	ctx, cancelCtx := context.WithTimeout(context.Background(), time.Second*30)
	defer cancelCtx()

	metadata := c.FormValue(`metadata`)
	if metadata == `` {
		return c.Status(400).JSON(fiber.Map{`message`: `Must give direction to flip image.`})
	}

	data := FlipMetadata{}

	err := json.Unmarshal([]byte(metadata), &data)
	if err != nil {
		slog.Error(`Could not unmarshal in flip handler. Error: ` + err.Error())
		return c.Status(500).JSON(fiber.Map{`message`: `Something went wrong.`})
	}

	if data.Direction == nil {
		return c.Status(400).JSON(fiber.Map{`message`: `Must set direction to flip image.`})
	}

	direction := strings.ToLower(*data.Direction)
	if direction != `horizontal` && direction != `vertical` {
		return c.Status(400).JSON(fiber.Map{`message`: `Direction of flip must be either horizontal or vertical.`})
	}

	fileheader, err := c.FormFile(`image`)
	if err != nil {
		slog.Error(`Could not get form file in flip handler. Error: ` + err.Error())
		return c.Status(500).JSON(fiber.Map{`message`: `Something went wrong.`})
	}

	mimeType := fileheader.Header.Get("Content-Type")

	if mimeType != "image/jpeg" && mimeType != "image/png" {
		return c.Status(fiber.StatusBadRequest).SendString("Unsupported file type: " + mimeType)
	}

	ext := strings.ToLower(filepath.Ext(fileheader.Filename))
	if ext != `png` && ext != `jpeg` && ext != `jpg` {
		return c.Status(400).JSON(fiber.Map{`message`: `Invalid file type.`})
	}

	file, err := fileheader.Open()
	if err != nil {
		slog.Error(`Could not open file in flip handler. Error: ` + err.Error())
		return c.Status(500).JSON(fiber.Map{`message`: `Something went wrong.`})
	}
	defer file.Close()

	var errChan = make(chan error)
	var imageChan = make(chan image.Image)
	go func() {
		decodedImage, err := imaging.Decode(file)

		if err != nil {
			errChan <- err
			return
		}

		imageChan <- decodedImage
	}()

	var image image.Image
	select {
	case img := <-imageChan:
		image = img
	case err := <-errChan:
		slog.Error(`Could not decode image. Error: ` + err.Error())
		return c.Status(500).JSON(fiber.Map{`message`: `Something went wrong.`})
	case <-ctx.Done():
		slog.Info(`Context canceled in rotate handler.`)
		return c.Status(fiber.ErrRequestTimeout.Code).JSON(fiber.Map{`message`: `Timeout.`})
	}

	validImageBounds := utilities.CheckImageBounds(&image)

	if !validImageBounds {
		return c.Status(400).JSON(fiber.Map{`message`: `Image bound too big.`})
	}

	if ctx.Err() != nil {
		return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{`message`: `Timeout.`})
	}
	responseWriter := c.Response().BodyWriter()

	if direction == `horizontal` {

		flippedImage := imaging.FlipH(image)

		if ext == `png` {
			c.Type(`image/png`)

			err = imaging.Encode(responseWriter, flippedImage, imaging.PNG)
		} else {
			c.Type(`image/jpeg`)

			err = imaging.Encode(responseWriter, flippedImage, imaging.JPEG, imaging.JPEGQuality(85))
		}
		if err != nil {
			if ctx.Err() != nil {
				slog.Error(`Encode failed due to context timeout. Error: ` + err.Error())
				return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{`message`: `Timeout.`})
			}

			slog.Error(`Could not encode image in flip handler. Error: ` + err.Error())
			return c.Status(500).JSON(fiber.Map{`message`: `Something went wrong.`})
		}

	} else if direction == `vertical` {
		flippedImage := imaging.FlipV(image)
		if ext == `png` {
			c.Type(`image/png`)

			err = imaging.Encode(responseWriter, flippedImage, imaging.PNG)
		} else {
			c.Type(`image/jpeg`)

			err = imaging.Encode(responseWriter, flippedImage, imaging.JPEG, imaging.JPEGQuality(85))
		}

		if err != nil {
			if ctx.Err() != nil {
				slog.Error(`Encode failed due to context timeout. Error: ` + err.Error())
				return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{`message`: `Timeout.`})
			}

			slog.Error(`Could not encode image in flip handler. Error: ` + err.Error())
			return c.Status(500).JSON(fiber.Map{`message`: `Something went wrong.`})
		}

	}

	return nil
}

func GrayScale(c *fiber.Ctx) error {

	ctx, cancelCtx := context.WithTimeout(context.Background(), time.Second*30)
	defer cancelCtx()

	fileheader, err := c.FormFile(`image`)
	if err != nil {
		slog.Error(`Could not get form file in gray scale handler. Error: ` + err.Error())
		return c.Status(500).JSON(fiber.Map{`message`: `Something went wrong.`})
	}

	mimeType := fileheader.Header.Get("Content-Type")

	if mimeType != "image/jpeg" && mimeType != "image/png" {
		return c.Status(fiber.StatusBadRequest).SendString("Unsupported file type: " + mimeType)
	}

	ext := strings.ToLower(filepath.Ext(fileheader.Filename))

	if ext != `png` && ext != `jpeg` && ext != `jpg` {
		return c.Status(400).JSON(fiber.Map{`message`: `Invalid file format.`})
	}

	file, err := fileheader.Open()
	if err != nil {
		slog.Error(`Could not open file in gray scale handler. Error: ` + err.Error())
		return c.Status(500).JSON(fiber.Map{`message`: `Something went wrong.`})
	}
	defer file.Close()

	var errChan = make(chan error)
	var imageChan = make(chan image.Image)
	go func() {
		decodedImage, err := imaging.Decode(file)

		if err != nil {
			errChan <- err
			return
		}

		imageChan <- decodedImage
	}()

	var image image.Image
	select {
	case img := <-imageChan:
		image = img
	case err := <-errChan:
		slog.Error(`Could not decode image. Error: ` + err.Error())
		return c.Status(500).JSON(fiber.Map{`message`: `Something went wrong.`})
	case <-ctx.Done():
		slog.Info(`Context canceled in rotate handler.`)
		return c.Status(fiber.ErrRequestTimeout.Code).JSON(fiber.Map{`message`: `Timeout.`})
	}

	if ctx.Err() != nil {
		return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{`message`: `Timeout.`})
	}

	validImageBounds := utilities.CheckImageBounds(&image)

	if !validImageBounds {
		return c.Status(400).JSON(fiber.Map{`message`: `Image bound too big.`})
	}

	if ctx.Err() != nil {
		return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{`message`: `Timeout.`})
	}
	grayImage := imaging.Grayscale(image)
	responseWriter := c.Response().BodyWriter()

	if ctx.Err() != nil {
		return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{`message`: `Timeout.`})
	}

	if ext == `png` {
		c.Type(`image/png`)
		err = imaging.Encode(responseWriter, grayImage, imaging.PNG)
	} else {
		c.Type(`image/jpeg`)
		err = imaging.Encode(responseWriter, grayImage, imaging.JPEG, imaging.JPEGQuality(85))
	}

	if err != nil {
		if ctx.Err() != nil {
			slog.Error(`Encode failed due to context timeout. Error: ` + err.Error())
			return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{`message`: `Timeout.`})
		}

		slog.Error(`Could not encode image in gray  scale handler. Error: ` + err.Error())
		return c.Status(500).JSON(fiber.Map{`message`: `Something went wrong.`})
	}

	return nil
}
