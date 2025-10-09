package utilities

import (
	"image"
)

func CheckImageBounds(image *image.Image) bool{

	const MaxAllowedDimension = 2000;

	if (*image).Bounds().Dx() > MaxAllowedDimension || (*image).Bounds().Dy() > MaxAllowedDimension{
		return false;
	}

	return true;
}

