package thumb

import (
	"encoding/base64"
	"image"
	"image/jpeg"
	"os"

	"github.com/disintegration/imaging"
	"github.com/muesli/smartcrop"
	"github.com/muesli/smartcrop/nfnt"
)

type SubImager interface {
	SubImage(r image.Rectangle) image.Image
}

// SmartCrop64 convert the image to base64 and add prefix "data:image/jpeg;base64,"
func SmartCrop64(imgPath, dstPath string) error {
	img, err := OpenImage(imgPath)
	if err != nil {
		return err
	}
	if img, err = smartCropResize(img); err != nil {
		return err
	}
	return jpegEncodeBase64ToFile(dstPath, img, 0)
}

func SmartCropBytes64(imgBytes []byte, dstPath string) error {
	img, err := ReadImage(imgBytes)
	if err != nil {
		return err
	}
	if img, err = smartCropResize(img); err != nil {
		return err
	}
	return jpegEncodeBase64ToFile(dstPath, img, 0)
}

func OpenImage(imgPath string) (image.Image, error) {
	f, err := os.Open(imgPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return imaging.Decode(f, imaging.AutoOrientation(true))
}

func resizeSquare(img image.Image, cropped image.Rectangle, side uint) image.Image {
	img = img.(SubImager).SubImage(cropped)
	resizer := nfnt.NewDefaultResizer()
	return resizer.Resize(img, side, side)
}

func smartCropResize(img image.Image) (image.Image, error) {
	analyzer := smartcrop.NewAnalyzer(nfnt.NewDefaultResizer())
	side := shortSide(img.Bounds())
	cropped, err := analyzer.FindBestCrop(img, side, side)
	if err != nil {
		return nil, err
	}
	img = resizeSquare(img, cropped, defaultThumbSize)
	return img, nil
}

// ResizeToFile resizes the image if it's long side bigger than limit.
// Use default limit 900 if limit is set to zero.
// Use default quality 85 if quality is set to zero.
func ResizeToFile(dst string, src []byte, limit float64, quality int) error {
	img, err := ReadImage(src)
	if err != nil {
		return err
	}
	w, h := limitWidthHeight(img.Bounds(), limit)
	small := imaging.Resize(img, w, h, imaging.Lanczos)
	return jpegEncodeToFile(dst, small, quality)
}

// Use default quality(85) if quality is set to zero.
// dst is the output file path.
func jpegEncodeToFile(dst string, src image.Image, quality int) error {
	if quality == 0 {
		quality = defaultQuality
	}
	file, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer file.Close()
	return jpeg.Encode(file, src, &jpeg.Options{Quality: quality})
}

// Use default quality(85) if quality is set to zero.
// dst is the output file path.
func jpegEncodeBase64ToFile(dst string, src image.Image, quality int) error {
	if quality == 0 {
		quality = defaultQuality
	}
	file, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer file.Close()

	prefix := []byte("data:image/jpeg;base64,")
	if _, err = file.Write(prefix); err != nil {
		return err
	}

	encoder64 := base64.NewEncoder(base64.StdEncoding, file)
	defer encoder64.Close()
	return jpeg.Encode(encoder64, src, &jpeg.Options{Quality: quality})
}
