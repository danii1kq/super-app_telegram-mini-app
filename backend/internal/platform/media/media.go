// Package media normalizes a receipt photo before it is encrypted and stored.
//
// The order matters: decode, orient, downscale, re-encode. Re-encoding through
// the standard encoder drops EXIF, GPS and every other metadata block, which is
// what we want. Compressing the ciphertext instead would be pointless.
package media

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png" // register the PNG decoder: a phone screenshot is a PNG

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register the WebP decoder
)

// ErrUnsupported means the bytes are not a decodable image.
var ErrUnsupported = errors.New("media: unsupported or corrupt image")

// ErrTooLarge means the pixel count exceeds the limit.
var ErrTooLarge = errors.New("media: image resolution is too large")

// Limits bound the work a single upload can cause.
const (
	// MaxPixels stops a decompression bomb: 40 MP is far above any phone photo
	// that still reads well, and keeps one decode under a few hundred MB.
	MaxPixels = 40_000_000
	// MaxDimension is the long edge kept for a receipt. A long narrow receipt
	// keeps its length and is limited on the short edge instead.
	MaxDimension = 2000
	// MinDimension is the short edge below which text stops being readable.
	MinDimension = 700
	// JPEGQuality is the encoder setting. 82 keeps small print legible at a
	// fraction of the original size.
	JPEGQuality = 82
)

// Result is the stored image and what it became.
type Result struct {
	Data   []byte
	Mime   string
	Width  int
	Height int
	// OriginalBytes and StoredBytes let the caller report real savings.
	OriginalBytes int
	StoredBytes   int
}

// Normalize decodes the image, applies its EXIF orientation, downscales it to a
// readable size and re-encodes it as JPEG without metadata.
//
// An image that is already small enough is re-encoded once at high quality, so a
// second pass over an already optimized file does not degrade it further: the
// caller checks the stored dimensions and skips the work.
func Normalize(raw []byte) (Result, error) {
	config, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return Result{}, ErrUnsupported
	}
	if config.Width <= 0 || config.Height <= 0 {
		return Result{}, ErrUnsupported
	}
	if int64(config.Width)*int64(config.Height) > MaxPixels {
		return Result{}, ErrTooLarge
	}

	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return Result{}, ErrUnsupported
	}
	img = orient(img, exifOrientation(raw))

	width, height := targetSize(img.Bounds().Dx(), img.Bounds().Dy())
	if width != img.Bounds().Dx() || height != img.Bounds().Dy() {
		img = resize(img, width, height)
	}

	var out bytes.Buffer
	if err := jpeg.Encode(&out, img, &jpeg.Options{Quality: JPEGQuality}); err != nil {
		return Result{}, fmt.Errorf("media: encode: %w", err)
	}

	return Result{
		Data:          out.Bytes(),
		Mime:          "image/jpeg",
		Width:         width,
		Height:        height,
		OriginalBytes: len(raw),
		StoredBytes:   out.Len(),
	}, nil
}

// targetSize keeps the long edge within MaxDimension and the short edge above
// MinDimension.
//
// A long narrow receipt is the case a naive square fit ruins: shrinking the long
// edge to 2000 would leave a 200 px wide strip. The short edge therefore sets a
// floor, and the long edge is allowed to stay longer when the aspect ratio
// demands it.
func targetSize(width, height int) (int, int) {
	long, short := width, height
	if height > width {
		long, short = height, width
	}
	if short == 0 {
		return width, height
	}

	scale := 1.0
	if long > MaxDimension {
		scale = float64(MaxDimension) / float64(long)
	}
	// Do not let the short edge fall below the readable floor.
	if float64(short)*scale < MinDimension {
		scale = float64(MinDimension) / float64(short)
	}
	if scale >= 1.0 {
		// Never enlarge: an already small image stays as it is.
		return width, height
	}
	return int(float64(width) * scale), int(float64(height) * scale)
}

func resize(src image.Image, width, height int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

// orient applies the EXIF orientation so a rotated photo is stored upright.
func orient(src image.Image, orientation int) image.Image {
	switch orientation {
	case 2:
		return flipHorizontal(src)
	case 3:
		return rotate180(src)
	case 4:
		return flipVertical(src)
	case 5:
		return rotate90(flipHorizontal(src))
	case 6:
		return rotate90(src)
	case 7:
		return rotate270(flipHorizontal(src))
	case 8:
		return rotate270(src)
	default:
		return src
	}
}

func rotate90(src image.Image) image.Image {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dy(), b.Dx()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			dst.Set(b.Dy()-1-y, x, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}

func rotate270(src image.Image) image.Image {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dy(), b.Dx()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			dst.Set(y, b.Dx()-1-x, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}

func rotate180(src image.Image) image.Image {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			dst.Set(b.Dx()-1-x, b.Dy()-1-y, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}

func flipHorizontal(src image.Image) image.Image {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			dst.Set(b.Dx()-1-x, y, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}

func flipVertical(src image.Image) image.Image {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			dst.Set(x, b.Dy()-1-y, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}

// exifOrientation reads the orientation tag from a JPEG APP1 segment.
//
// Only the tag is read: the rest of the metadata is dropped by re-encoding, so
// there is no reason to parse it.
func exifOrientation(raw []byte) int {
	if len(raw) < 4 || raw[0] != 0xFF || raw[1] != 0xD8 {
		return 1
	}
	offset := 2
	for offset+4 <= len(raw) {
		if raw[offset] != 0xFF {
			return 1
		}
		marker := raw[offset+1]
		length := int(raw[offset+2])<<8 | int(raw[offset+3])
		if length < 2 || offset+2+length > len(raw) {
			return 1
		}
		if marker == 0xE1 && length > 8 {
			segment := raw[offset+4 : offset+2+length]
			if bytes.HasPrefix(segment, []byte("Exif\x00\x00")) {
				return readOrientation(segment[6:])
			}
		}
		if marker == 0xDA {
			return 1
		}
		offset += 2 + length
	}
	return 1
}

// readOrientation walks the TIFF header for tag 0x0112.
func readOrientation(tiff []byte) int {
	if len(tiff) < 8 {
		return 1
	}
	var order bool // true: little endian
	switch string(tiff[0:2]) {
	case "II":
		order = true
	case "MM":
		order = false
	default:
		return 1
	}
	read16 := func(b []byte) int {
		if order {
			return int(b[0]) | int(b[1])<<8
		}
		return int(b[0])<<8 | int(b[1])
	}
	read32 := func(b []byte) int {
		if order {
			return int(b[0]) | int(b[1])<<8 | int(b[2])<<16 | int(b[3])<<24
		}
		return int(b[0])<<24 | int(b[1])<<16 | int(b[2])<<8 | int(b[3])
	}
	ifdOffset := read32(tiff[4:8])
	if ifdOffset < 8 || ifdOffset+2 > len(tiff) {
		return 1
	}
	count := read16(tiff[ifdOffset : ifdOffset+2])
	entry := ifdOffset + 2
	for i := 0; i < count; i++ {
		if entry+12 > len(tiff) {
			return 1
		}
		tag := read16(tiff[entry : entry+2])
		if tag == 0x0112 {
			value := read16(tiff[entry+8 : entry+10])
			if value >= 1 && value <= 8 {
				return value
			}
			return 1
		}
		entry += 12
	}
	return 1
}

// DetectMime sniffs the container from the magic bytes.
func DetectMime(raw []byte) string {
	if len(raw) < 12 {
		return ""
	}
	switch {
	case raw[0] == 0xFF && raw[1] == 0xD8 && raw[2] == 0xFF:
		return "image/jpeg"
	case string(raw[0:8]) == "\x89PNG\r\n\x1a\n":
		return "image/png"
	case string(raw[0:4]) == "RIFF" && string(raw[8:12]) == "WEBP":
		return "image/webp"
	}
	return ""
}


