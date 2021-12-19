package libimage

import (
	"errors"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"path/filepath"

	"github.com/disintegration/imaging"

	"github.com/chai2010/webp"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

type (
	Image struct {
		image.Image
		Quality int
	}
)

var ImageAnchors = map[string]imaging.Anchor{
	"topleft":     imaging.TopLeft,
	"top":         imaging.Top,
	"topright":    imaging.TopRight,
	"left":        imaging.Left,
	"center":      imaging.Center,
	"right":       imaging.Right,
	"bottomleft":  imaging.BottomLeft,
	"bottom":      imaging.Bottom,
	"bottomright": imaging.BottomRight,
}

var ImageModes = map[string]bool{
	"crop": true,
	"fill": true,
	"none": true,
}

func (src Image) Decode(reader io.Reader) (dst Image, err error) {
	if src.Image, _, err = image.Decode(reader); err != nil {
		return
	}
	dst = src
	return
}

func (src Image) Resize(maxWidth int, maxHeight int, anchor string, mode string, enlarge bool) (dst Image, err error) {
	dstW, dstH := maxWidth, maxHeight
	if _, ok := ImageAnchors[anchor]; !ok {
		err = errors.New("image: anchor")
		return
	}
	if maxWidth < 0 || maxHeight < 0 {
		err = errors.New("image: size")
		return
	}

	srcBounds := src.Image.Bounds()
	srcW := srcBounds.Bounds().Dx()
	srcH := srcBounds.Bounds().Dy()

	if dstW == 0 && dstH == 0 {
		// 不裁剪
		dstW = srcW
		dstH = srcH
	} else if dstW == 0 {
		tmpW := float64(dstH) * float64(srcW) / float64(srcH)
		dstW = int(math.Max(1.0, math.Floor(tmpW+0.5)))
	} else if dstH == 0 {
		tmpH := float64(dstW) * float64(srcH) / float64(srcW)
		dstH = int(math.Max(1.0, math.Floor(tmpH+0.5)))
	}

	switch mode {
	case "none":
		// 不裁剪 填补
		if enlarge {
			// 放大
			tmpW := float64(dstW) / float64(srcW)
			tmpH := float64(dstH) / float64(srcH)
			if tmpW > tmpH {
				src.Image = imaging.Resize(src.Image, 0, dstH, imaging.Lanczos)
			} else {
				src.Image = imaging.Resize(src.Image, dstW, 0, imaging.Lanczos)
			}
			src.Image = imaging.Fit(src.Image, dstW, dstH, imaging.Lanczos)
		} else {
			// 不放大
			src.Image = imaging.Fit(src.Image, dstW, dstH, imaging.Lanczos)
		}
	case "crop":
		// 裁剪
		if enlarge {
			// 允许放大
			src.Image = imaging.Fill(src.Image, dstW, dstH, ImageAnchors[anchor], imaging.Lanczos)
		} else {
			// 不允修放大

			if dstW > srcW {
				dstW = srcW
			}
			if dstH > srcH {
				dstH = srcH
			}
			src.Image = imaging.Fill(src.Image, dstW, dstH, ImageAnchors[anchor], imaging.Lanczos)
		}
	case "fill":
		// 填补
		if enlarge {
			// 放大
			tmpW := float64(dstW) / float64(srcW)
			tmpH := float64(dstH) / float64(srcH)
			if tmpW > tmpH {
				src.Image = imaging.Resize(src.Image, 0, dstH, imaging.Lanczos)
			} else {
				src.Image = imaging.Resize(src.Image, dstW, 0, imaging.Lanczos)
			}
			src.Image = imaging.Fit(src.Image, dstW, dstH, imaging.Lanczos)
		} else {
			// 不放大
			src.Image = imaging.Fit(src.Image, dstW, dstH, imaging.Lanczos)
		}
		// 填补计算
		background := imaging.New(dstW, dstH, color.NRGBA{0, 0, 0, 0})
		pt := imageAnchorPt(background.Bounds(), src.Image.Bounds().Dx(), src.Image.Bounds().Dy(), ImageAnchors[anchor])
		src.Image = imaging.Paste(background, src.Image, pt)
	default:
		err = errors.New("image: mode")
		return
	}
	dst = src
	return
}

func (src Image) Save(filename string) (err error) {
	quality := src.Quality
	if quality == 0 {
		quality = 85
	}
	switch filepath.Ext(filename) {
	case ".webp":
		if err = webp.Save(filename, src.Image, &webp.Options{Quality: float32(quality)}); err != nil {
			return
		}
	case ".jpg", ".jpeg":
		err = imaging.Save(src.Image, filename, imaging.JPEGQuality(quality))
	default:
		err = imaging.Save(src.Image, filename)
	}
	return
}

func imageAnchorPt(b image.Rectangle, w, h int, anchor imaging.Anchor) image.Point {
	var x, y int
	switch anchor {
	case imaging.TopLeft:
		x = b.Min.X
		y = b.Min.Y
	case imaging.Top:
		x = b.Min.X + (b.Dx()-w)/2
		y = b.Min.Y
	case imaging.TopRight:
		x = b.Max.X - w
		y = b.Min.Y
	case imaging.Left:
		x = b.Min.X
		y = b.Min.Y + (b.Dy()-h)/2
	case imaging.Right:
		x = b.Max.X - w
		y = b.Min.Y + (b.Dy()-h)/2
	case imaging.BottomLeft:
		x = b.Min.X
		y = b.Max.Y - h
	case imaging.Bottom:
		x = b.Min.X + (b.Dx()-w)/2
		y = b.Max.Y - h
	case imaging.BottomRight:
		x = b.Max.X - w
		y = b.Max.Y - h
	default:
		x = b.Min.X + (b.Dx()-w)/2
		y = b.Min.Y + (b.Dy()-h)/2
	}
	return image.Pt(x, y)
}
