package handlers

import (
	"encoding/json"
	"image"
	"image/color"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/gofiber/fiber/v2"
)


type RotateBody struct{
	Angle *int `json:"angle"`
}

func Rotate(c *fiber.Ctx) error{

	body := c.FormValue(`metadata`);
	if body == ``{
		return c.Status(400).JSON(fiber.Map{`message`:`Angle must be set.`});
	}
	bodyData := RotateBody{};
	err := json.Unmarshal([]byte(body), &bodyData);
	if err != nil {
		slog.Error(`Could not unmarshal rotate body. Error: ` + err.Error())
		return fiber.NewError(500, `Something went wrong.`); 
	}

	if bodyData.Angle == nil{
		slog.Info(`Angle is nil in rotate handler.`);
		return c.Status(400).JSON(fiber.Map{`message`:`Angle must be set.`})
	}

	file, err := c.FormFile(`image`);
	if err != nil {
		return fiber.NewError(500, `Could not open image.`);	 
	}
	
	fileExt := strings.ToLower(filepath.Ext(file.Filename));
	if fileExt != `png` && fileExt != `jpeg` && fileExt != `jpg`{
		return fiber.NewError(400, `Expected png or jpg image.`); 
	}

	fileStream ,err := file.Open();
	if err != nil {
		return fiber.NewError(500, `Could not open image.`); 
	}
	defer fileStream.Close();
	
	var decodedImage image.Image;
	decodedImage, err = imaging.Decode(fileStream);

	if err != nil {
		slog.Error(`Could noe decode filestream in rotate handler. Error: ` + err.Error())
		return fiber.NewError(500, `Something went wrong.`); 
	}

	rotatedImage := imaging.Rotate(decodedImage, float64(*bodyData.Angle), color.White)
	
	writer := c.Response().BodyWriter();
	if fileExt == `png`{
		err = imaging.Encode(writer, rotatedImage, imaging.PNG);
	}else{
		err = imaging.Encode(writer, rotatedImage, imaging.JPEG, imaging.JPEGQuality(85));
	}

	if err != nil {
		slog.Error(`Could not encode in rotate handler. Error: ` + err.Error())
		return fiber.NewError(500, `Something went wrong.`); 

	}

	return nil;
}


type CropMetaData struct{
	MinX *int `json:"minX"`
	MinY *int `json:"minY"`

	MaxX *int `json:"maxX"`
	MaxY *int `json:"maxY"`
}

func Crop(c *fiber.Ctx) error{
	metadata := c.FormValue(`metadata`);
	if metadata == ``{
		return c.Status(400).JSON(fiber.Map{`message`:`Must set bounds to crop image.`})
	}
	data := CropMetaData{};
	err := json.Unmarshal([]byte(metadata), &data);
	if err != nil {
		slog.Error(`Could not unmarshal metadata in crop handler. Error: ` + err.Error())
		return fiber.NewError(500, `Something went wrong.`); 
	}
	
	if data.MaxX == nil || data.MinX == nil || data.MinY == nil || data.MaxY == nil{
		return c.Status(400).JSON(fiber.Map{`message`:`Must set bounds to crop the image.`})
	}

	if *data.MaxX < *data.MinX || *data.MaxY < *data.MinY{
		return c.Status(400).JSON(fiber.Map{`message`:`Invalid bounds.`})
	}

	if *data.MaxX <0  || *data.MinX < 0 || *data.MaxY < 0 || *data.MinY <0 {
		return c.Status(400).JSON(fiber.Map{`message`:`Invalid bounds.`})
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
	
	file, err := c.FormFile(`image`);
	if err != nil {
		return fiber.NewError(500, `Could not open image.`);	 
	}

	fileExt := strings.ToLower(filepath.Ext(file.Filename));
	if fileExt != `png` && fileExt != `jpeg` && fileExt != `jpg`{
		return fiber.NewError(400, `Expected png or jpg image.`); 
	}
	
	fileStream ,err := file.Open();
	if err != nil {
		return fiber.NewError(500, `Could not open image.`); 
	}
	defer fileStream.Close();
	
	var decodedImage image.Image;
	decodedImage, err = imaging.Decode(fileStream);
	
	if err != nil {
		slog.Error(`Could not decode image in crop handler. Error: ` + err.Error())
		return fiber.NewError(500, `Something went wrong.`); 
	}

	if !rec.In(decodedImage.Bounds()){
		return c.Status(400).JSON(fiber.Map{`message`:`Invalid bounds.`})
	}

	croppedImage := imaging.Crop(decodedImage, rec);	
	
	bodyWriter := c.Response().BodyWriter();

	if fileExt == `png`{
		err = imaging.Encode(bodyWriter, croppedImage, imaging.PNG);
	}else{
		err = imaging.Encode(bodyWriter, croppedImage, imaging.JPEG, imaging.JPEGQuality(85));
	}
	
	if err != nil {
		slog.Error(`Could not encode image in crop handler. Error: ` + err.Error())
		return fiber.NewError(500, `Something went wrong.`); 
	}

	return nil;
}

type ResizeMetaData struct{
	Height *int `json:"height"`
	Width *int `json:"width"`
}

func Resize(c *fiber.Ctx) error{
	metadata := c.FormValue(`metadata`);

	if metadata == ``{
		return c.Status(400).JSON(fiber.Map{`message`:`Must set height or width to resize image.`})
	}

	data := ResizeMetaData{};
	err := json.Unmarshal([]byte(metadata), &data);
	if err != nil {
		slog.Error(`Could not unmarshal metadata in resize handler. Error: ` + err.Error())
		return c.Status(500).JSON(fiber.Map{`message`:`Something went wrong.`})
	}

	if data.Width == nil && data.Height == nil{
		return c.Status(400).JSON(fiber.Map{`message`:`Must set at least width or height to resize image.`})
	}

	fileHeader, err := c.FormFile(`image`);
	if err != nil {
		slog.Error(`Could not get form file in resize handler. Error: ` + err.Error())
		return c.Status(500).JSON(fiber.Map{`message`:`Something went wrong.`})
	}

	extension := strings.ToLower(filepath.Ext(fileHeader.Filename));

	if extension != `png` && extension != `jpeg` && extension != `jpg`{
		return c.Status(400).JSON(fiber.Map{`message`:`Invalid extension.`});
	}

	fileStream, err := fileHeader.Open();
	if err != nil {
		slog.Error(`Could not open image in resize handler. Error: ` + err.Error())
		return c.Status(500).JSON(fiber.Map{`message`:`Something went wrong.`})
	}

	decodedImage, err := imaging.Decode(fileStream);
	if err != nil {
		slog.Error(`Could not decode image in resize handler. Error: ` + err.Error())
		return c.Status(500).JSON(fiber.Map{`message`:`Something went wrong.`})
	}

	var resizedImage *image.NRGBA;
	if data.Width == nil && data.Height != nil && *data.Height > 0 && *data.Height < 2000{
		resizedImage = imaging.Resize(decodedImage, 0, *data.Height, imaging.Lanczos);
	}else if data.Height == nil && data.Width != nil && *data.Width > 0 && *data.Width < 2000{
		resizedImage = imaging.Resize(decodedImage, *data.Width, 0, imaging.Lanczos);
	}else if data.Height != nil && *data.Height > 0 && data.Width != nil && *data.Width > 0 && *data.Height < 2000 && *data.Width < 2000{
		resizedImage = imaging.Resize(decodedImage, *data.Width, *data.Height, imaging.Lanczos);
	}else{
		return c.Status(400).JSON(fiber.Map{`message`:`Invalid parameters.`})
	}

	writer := c.Request().BodyWriter()

	if extension == `png`{
		err = imaging.Encode(writer, resizedImage, imaging.PNG);
	}else{
		err = imaging.Encode(writer, resizedImage, imaging.JPEG, imaging.JPEGQuality(85));
	}
	if err != nil {
		slog.Error(`Could node encode image in resize handler. Error: ` + err.Error())
		return c.Status(500).JSON(fiber.Map{`message`:`Something went wrong.`})
	}

	return nil;
}


