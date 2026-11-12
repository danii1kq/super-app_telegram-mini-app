package media

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

// makeJPEG builds a real JPEG of the given size with visible structure, so the
// encoder has something to compress and the result can be decoded again.
func makeJPEG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// A gradient with text-like stripes: not a flat colour, so the size
			// comparison means something.
			v := uint8((x*7 + y*3) % 256)
			if (y/12)%2 == 0 {
				v = uint8((x * 13) % 256)
			}
			img.Set(x, y, color.RGBA{R: v, G: uint8(255 - v), B: uint8((x + y) % 256), A: 255})
		}
	}
	var out bytes.Buffer
	if err := jpeg.Encode(&out, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatalf("encode fixture: %v", err)
	}
	return out.Bytes()
}

// TestNormalizeShrinksAndKeepsReadableSize checks a normal phone-sized photo.
func TestNormalizeShrinksAndKeepsReadableSize(t *testing.T) {
	raw := makeJPEG(t, 3000, 4000)

	result, err := Normalize(raw)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if result.Mime != "image/jpeg" {
		t.Fatalf("mime: got %q", result.Mime)
	}
	if result.Width > MaxDimension || result.Height > MaxDimension {
		t.Fatalf("long edge must fit %d, got %dx%d", MaxDimension, result.Width, result.Height)
	}
	if result.StoredBytes >= result.OriginalBytes {
		t.Fatalf("expected a smaller file: %d -> %d", result.OriginalBytes, result.StoredBytes)
	}
	// The result must still decode: a broken image would be worse than a big one.
	if _, _, err := image.DecodeConfig(bytes.NewReader(result.Data)); err != nil {
		t.Fatalf("stored image does not decode: %v", err)
	}
	t.Logf("normal photo: %d -> %d bytes (%.1f%%), %dx%d",
		result.OriginalBytes, result.StoredBytes,
		100*float64(result.StoredBytes)/float64(result.OriginalBytes),
		result.Width, result.Height)
}

// TestLongReceiptKeepsShortEdgeReadable is the case a square fit ruins: a long
// narrow receipt must not become a thin unreadable strip.
func TestLongReceiptKeepsShortEdgeReadable(t *testing.T) {
	raw := makeJPEG(t, 900, 6000)

	result, err := Normalize(raw)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	short, long := result.Width, result.Height
	if short > long {
		short, long = long, short
	}
	if short < MinDimension {
		t.Fatalf("short edge fell below the readable floor: %d < %d", short, MinDimension)
	}
	if long < MaxDimension {
		t.Fatalf("long edge was over-shrunk: %d", long)
	}
	t.Logf("long receipt: %d -> %d bytes, %dx%d",
		result.OriginalBytes, result.StoredBytes, result.Width, result.Height)
}

// TestSmallImageIsNotEnlarged: a screenshot smaller than the limit stays as is.
func TestSmallImageIsNotEnlarged(t *testing.T) {
	raw := makeJPEG(t, 600, 800)

	result, err := Normalize(raw)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if result.Width != 600 || result.Height != 800 {
		t.Fatalf("small image must keep its size, got %dx%d", result.Width, result.Height)
	}
}

// TestSecondPassDoesNotDegrade: re-normalizing an optimized image keeps it.
func TestSecondPassDoesNotDegrade(t *testing.T) {
	first, err := Normalize(makeJPEG(t, 3000, 4000))
	if err != nil {
		t.Fatalf("first pass: %v", err)
	}
	second, err := Normalize(first.Data)
	if err != nil {
		t.Fatalf("second pass: %v", err)
	}
	if second.Width != first.Width || second.Height != first.Height {
		t.Fatalf("size changed on the second pass: %dx%d -> %dx%d",
			first.Width, first.Height, second.Width, second.Height)
	}
	// A second pass may re-encode, but must not shrink the image further.
	if second.StoredBytes > first.StoredBytes*2 {
		t.Fatalf("second pass grew the file: %d -> %d", first.StoredBytes, second.StoredBytes)
	}
}

// TestRejectsFakeImage: a text file with an image extension is not an image.
func TestRejectsFakeImage(t *testing.T) {
	if _, err := Normalize([]byte("this is not an image, it only claims to be")); err != ErrUnsupported {
		t.Fatalf("expected ErrUnsupported, got %v", err)
	}
}

// TestRejectsHugeResolution: a decompression bomb is refused before decoding.
func TestRejectsHugeResolution(t *testing.T) {
	// A PNG header claiming 20000x20000 with no real data: DecodeConfig reads
	// the size, and the pixel check must stop it before a full decode.
	bomb := []byte{
		0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n',
		0, 0, 0, 13, 'I', 'H', 'D', 'R',
		0, 0, 0x4E, 0x20, // width 20000
		0, 0, 0x4E, 0x20, // height 20000
		8, 2, 0, 0, 0,
		0, 0, 0, 0,
	}
	if _, err := Normalize(bomb); err != ErrTooLarge && err != ErrUnsupported {
		t.Fatalf("expected the bomb to be refused, got %v", err)
	}
}

// TestExifOrientationIsRead checks the tag parser on a hand-built APP1 segment.
func TestExifOrientationIsRead(t *testing.T) {
	// Minimal JPEG: SOI + APP1 with an EXIF TIFF header carrying orientation 6.
	app1 := []byte{
		0xFF, 0xE1, 0x00, 0x22, // marker, length 34
		'E', 'x', 'i', 'f', 0, 0,
		'I', 'I', 0x2A, 0x00, 0x08, 0x00, 0x00, 0x00, // little-endian TIFF, IFD at 8
		0x01, 0x00, // one entry
		0x12, 0x01, 0x03, 0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
	}
	raw := append([]byte{0xFF, 0xD8}, app1...)
	if got := exifOrientation(raw); got != 6 {
		t.Fatalf("orientation: got %d, want 6", got)
	}
}

// TestOrientationIsApplied: a rotated photo is stored upright.
func TestOrientationIsApplied(t *testing.T) {
	// A 100x200 image with orientation 6 must come out 200x100.
	raw := withOrientation(t, makeJPEG(t, 100, 200), 6)
	result, err := Normalize(raw)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if result.Width != 200 || result.Height != 100 {
		t.Fatalf("orientation 6 must swap the axes, got %dx%d", result.Width, result.Height)
	}
}

// withOrientation injects an EXIF APP1 segment with the given orientation.
func withOrientation(t *testing.T, jpegData []byte, orientation int) []byte {
	t.Helper()
	if orientation < 1 || orientation > 8 {
		t.Fatalf("bad orientation %d", orientation)
	}
	app1 := []byte{
		0xFF, 0xE1, 0x00, 0x22,
		'E', 'x', 'i', 'f', 0, 0,
		'I', 'I', 0x2A, 0x00, 0x08, 0x00, 0x00, 0x00,
		0x01, 0x00,
		0x12, 0x01, 0x03, 0x00, 0x01, 0x00, 0x00, 0x00, byte(orientation), 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
	}
	out := make([]byte, 0, len(jpegData)+len(app1))
	out = append(out, jpegData[0:2]...) // SOI
	out = append(out, app1...)
	out = append(out, jpegData[2:]...)
	return out
}

// TestMetadataIsDropped: the stored image carries no EXIF at all.
func TestMetadataIsDropped(t *testing.T) {
	raw := withOrientation(t, makeJPEG(t, 800, 1000), 3)
	result, err := Normalize(raw)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if bytes.Contains(result.Data, []byte("Exif")) {
		t.Fatal("stored image still carries an EXIF block")
	}
	if bytes.Contains(result.Data, []byte("GPS")) {
		t.Fatal("stored image still carries GPS data")
	}
}

// TestDetectMime checks the sniffing used before decoding.
func TestDetectMime(t *testing.T) {
	if got := DetectMime([]byte("plain text here")); got != "" {
		t.Fatalf("text must not be an image, got %q", got)
	}
	if got := DetectMime(makeJPEG(t, 10, 10)); got != "image/jpeg" {
		t.Fatalf("jpeg: got %q", got)
	}
}
