package vision

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"testing"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// Opt-in only: the default test suite never spends money or sends user images.
func TestLiveDeepSeekFlash(t *testing.T) {
	if os.Getenv("SRUNAL_VISION_LIVE_TEST") != "1" {
		t.Skip("set SRUNAL_VISION_LIVE_TEST=1 for a synthetic live-provider check")
	}
	client := NewClient("", os.Getenv("SRUNAL_DEEPSEEK_API_KEY"), "deepseek-flash")
	if !client.Configured() {
		t.Fatal("live provider key is not configured")
	}
	fixture := image.NewRGBA(image.Rect(0, 0, 320, 150))
	draw.Draw(fixture, fixture.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	for i, line := range []string{"DEMO RECEIPT", "Tea 1000 KZT", "Milk 500 KZT", "TOTAL 1500 KZT"} {
		writer := font.Drawer{Dst: fixture, Src: image.NewUniform(color.Black), Face: basicfont.Face7x13, Dot: fixed.P(15, 25+i*28)}
		writer.DrawString(line)
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, fixture); err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	raw, err := client.Extract(context.Background(), encoded.Bytes(), "image/png", `Read this synthetic receipt. Return JSON {"total":1500,"currency":"KZT"}, replacing total with the integer KZT amount printed after TOTAL. Do not multiply by 100.`)
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Total    int    `json:"total"`
		Currency string `json:"currency"`
	}
	if json.Unmarshal(raw, &result) != nil || result.Total != 1500 || result.Currency != "KZT" {
		t.Fatal("live provider did not read the synthetic total correctly")
	}
	t.Logf("DeepSeek Flash read the synthetic image with thinking disabled in %s", time.Since(started).Round(time.Millisecond))
}
