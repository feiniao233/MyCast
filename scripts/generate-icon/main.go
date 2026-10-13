// SPDX-License-Identifier: AGPL-3.0-or-later
// A deliberately small SVG renderer/resource writer for our checked-in artwork.
// Supports only this source's absolute M/L/Q/Z paths and vertical gradients;
// rejects other geometry rather than silently producing an inaccurate icon.
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

var sizes = []int{16, 20, 24, 32, 48, 64, 128, 256}

type stop struct {
	Offset string `xml:"offset,attr"`
	Color  string `xml:"stop-color,attr"`
}
type gradient struct {
	ID    string `xml:"id,attr"`
	X1    string `xml:"x1,attr"`
	X2    string `xml:"x2,attr"`
	Y1    string `xml:"y1,attr"`
	Y2    string `xml:"y2,attr"`
	Units string `xml:"gradientUnits,attr"`
	Stops []stop `xml:"stop"`
}
type path struct {
	Fill string `xml:"fill,attr"`
	D    string `xml:"d,attr"`
}
type source struct {
	ViewBox   string     `xml:"viewBox,attr"`
	Gradients []gradient `xml:"defs>linearGradient"`
	Paths     []path     `xml:"path"`
}
type vertical struct {
	a, b  color.NRGBA
	scale float64
}

type point struct{ x, y float32 }
type polygon []point

func (p *polygon) MoveTo(x, y float32) { *p = append(*p, point{x, y}) }
func (p *polygon) LineTo(x, y float32) { p.MoveTo(x, y) }
func (p *polygon) QuadTo(cx, cy, x, y float32) {
	a := (*p)[len(*p)-1]
	for i := 1; i <= 32; i++ {
		t := float32(i) / 32
		u := 1 - t
		p.LineTo(u*u*a.x+2*u*t*cx+t*t*x, u*u*a.y+2*u*t*cy+t*t*y)
	}
}
func (p *polygon) ClosePath() {}
func (p polygon) contains(x, y float32) bool {
	inside := false
	for i, a := range p {
		b := p[(i+1)%len(p)]
		if (a.y > y) != (b.y > y) && x < (b.x-a.x)*(y-a.y)/(b.y-a.y)+a.x {
			inside = !inside
		}
	}
	return inside
}
func (p polygon) Draw(dst *image.NRGBA, bounds image.Rectangle, src vertical, _ image.Point) {
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if p.contains(float32(x)+.5, float32(y)+.5) {
				dst.SetNRGBA(x, y, src.At(x, y).(color.NRGBA))
			}
		}
	}
}

func (v vertical) ColorModel() color.Model { return color.NRGBAModel }
func (v vertical) Bounds() image.Rectangle { return image.Rect(-1e6, -1e6, 1e6, 1e6) }
func (v vertical) At(x, y int) color.Color {
	t := max(0., min(1., (float64(y)+.5)/v.scale))
	lerp := func(a, b uint8) uint8 { return uint8(float64(a)*(1-t) + float64(b)*t + .5) }
	return color.NRGBA{lerp(v.a.R, v.b.R), lerp(v.a.G, v.b.G), lerp(v.a.B, v.b.B), 255}
}
func hex(s string) (color.NRGBA, error) {
	if len(s) != 7 || s[0] != '#' {
		return color.NRGBA{}, fmt.Errorf("invalid color %q", s)
	}
	n, e := strconv.ParseUint(s[1:], 16, 24)
	return color.NRGBA{uint8(n >> 16), uint8(n >> 8), uint8(n), 255}, e
}
func raster(svg []byte, n int) ([]byte, error) {
	allowed := map[string]string{"svg": "xmlns viewBox", "defs": "", "linearGradient": "id x1 y1 x2 y2 gradientUnits", "stop": "offset stop-color", "path": "fill d"}
	decoder := xml.NewDecoder(bytes.NewReader(svg))
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if tag, ok := token.(xml.StartElement); ok {
			attrs, known := allowed[tag.Name.Local]
			if !known || tag.Name.Space != "http://www.w3.org/2000/svg" {
				return nil, fmt.Errorf("unsupported SVG element %s", tag.Name.Local)
			}
			for _, a := range tag.Attr {
				if !strings.Contains(" "+attrs+" ", " "+a.Name.Local+" ") {
					return nil, fmt.Errorf("unsupported SVG attribute %s", a.Name.Local)
				}
			}
		}
	}
	var s source
	if err := xml.Unmarshal(svg, &s); err != nil {
		return nil, err
	}
	if s.ViewBox != "0 0 32 32" || len(s.Paths) != 2 || len(s.Gradients) != 2 {
		return nil, fmt.Errorf("unsupported SVG structure")
	}
	// Eightfold rasterization gives subpixel coverage without changing geometry.
	scale := float32(n*8) / 32
	canvas := image.NewNRGBA(image.Rect(0, 0, n*8, n*8))
	gradients := map[string]vertical{}
	for _, g := range s.Gradients {
		if g.X1 != "0" || g.X2 != "0" || g.Y1 != "0" || g.Y2 != "32" || g.Units != "userSpaceOnUse" || len(g.Stops) != 2 || g.Stops[0].Offset != "0" || g.Stops[1].Offset != "1" {
			return nil, fmt.Errorf("unsupported gradient")
		}
		a, e := hex(g.Stops[0].Color)
		if e != nil {
			return nil, e
		}
		b, e := hex(g.Stops[1].Color)
		if e != nil {
			return nil, e
		}
		gradients["url(#"+g.ID+")"] = vertical{a, b, float64(n * 8)}
	}
	for _, p := range s.Paths {
		g, ok := gradients[p.Fill]
		if !ok {
			return nil, fmt.Errorf("unknown fill")
		}
		r := polygon{}
		tokens := strings.Fields(p.D)
		for i := 0; i < len(tokens); {
			command := tokens[i]
			i++
			if (len(r) == 0 && command != "M") || (len(r) > 0 && command == "M") {
				return nil, fmt.Errorf("unsupported SVG subpath")
			}
			count := map[string]int{"M": 2, "L": 2, "Q": 4, "Z": 0}[command]
			if command != "M" && command != "L" && command != "Q" && command != "Z" {
				return nil, fmt.Errorf("unsupported path command %s", command)
			}
			if i+count > len(tokens) {
				return nil, fmt.Errorf("truncated path")
			}
			v := make([]float32, count)
			for j := range v {
				n, e := strconv.ParseFloat(tokens[i+j], 32)
				if e != nil || n < 0 || n > 32 {
					return nil, fmt.Errorf("invalid coordinate")
				}
				v[j] = float32(n) * scale
			}
			i += count
			switch command {
			case "M":
				r.MoveTo(v[0], v[1])
			case "L":
				r.LineTo(v[0], v[1])
			case "Q":
				r.QuadTo(v[0], v[1], v[2], v[3])
			case "Z":
				r.ClosePath()
			}
		}
		r.Draw(canvas, canvas.Bounds(), g, image.Point{})
	}
	out := image.NewNRGBA(image.Rect(0, 0, n, n))
	xdraw.CatmullRom.Scale(out, out.Bounds(), canvas, canvas.Bounds(), draw.Src, nil)
	var b bytes.Buffer
	err := png.Encode(&b, out)
	return b.Bytes(), err
}
func ico(images [][]byte) []byte {
	b := make([]byte, 6+16*len(images))
	binary.LittleEndian.PutUint16(b[2:], 1)
	binary.LittleEndian.PutUint16(b[4:], uint16(len(images)))
	for i, data := range images {
		e := b[6+16*i:]
		e[0] = byte(sizes[i])
		e[1] = byte(sizes[i])
		binary.LittleEndian.PutUint16(e[4:], 1)
		binary.LittleEndian.PutUint16(e[6:], 32)
		binary.LittleEndian.PutUint32(e[8:], uint32(len(data)))
		binary.LittleEndian.PutUint32(e[12:], uint32(len(b)))
		b = append(b, data...)
	}
	return b
}

// coff writes a single read-only .rsrc COFF object. Only payload RVA fields
// have IMAGE_REL_AMD64_ADDR32NB relocations against the section symbol.
// All timestamps, padding, IDs and language are deterministic (neutral).
func coff(images [][]byte) []byte {
	group := make([]byte, 6+14*len(images))
	binary.LittleEndian.PutUint16(group[2:], 1)
	binary.LittleEndian.PutUint16(group[4:], uint16(len(images)))
	for i, data := range images {
		e := group[6+14*i:]
		e[0] = byte(sizes[i])
		e[1] = byte(sizes[i])
		binary.LittleEndian.PutUint16(e[4:], 1)
		binary.LittleEndian.PutUint16(e[6:], 32)
		binary.LittleEndian.PutUint32(e[8:], uint32(len(data)))
		binary.LittleEndian.PutUint16(e[12:], uint16(i+1))
	}
	data := make([]byte, 32)
	u16 := func(off int, n uint16) { binary.LittleEndian.PutUint16(data[off:], n) }
	u32 := func(off int, n uint32) { binary.LittleEndian.PutUint32(data[off:], n) }
	u16(14, 2)
	type block struct {
		entry   int
		payload []byte
	}
	var blocks []block
	for i, typ := range []int{3, 14} {
		payloads := images
		if typ == 14 {
			payloads = [][]byte{group}
		}
		off := len(data)
		data = append(data, make([]byte, 16+8*len(payloads))...)
		u16(off+14, uint16(len(payloads)))
		u32(16+i*8, uint32(typ))
		u32(20+i*8, uint32(off)|0x80000000)
		for j, payload := range payloads {
			lang := len(data)
			data = append(data, make([]byte, 24)...)
			u16(lang+14, 1)
			u32(off+16+j*8, uint32(j+1))
			u32(off+20+j*8, uint32(lang)|0x80000000)
			entry := len(data)
			data = append(data, make([]byte, 16)...)
			u32(lang+20, uint32(entry))
			blocks = append(blocks, block{entry, payload})
		}
	}
	for _, block := range blocks {
		for len(data)%4 != 0 {
			data = append(data, 0)
		}
		u32(block.entry, uint32(len(data)))
		u32(block.entry+4, uint32(len(block.payload)))
		data = append(data, block.payload...)
	}
	object := make([]byte, 60)
	binary.LittleEndian.PutUint16(object, 0x8664)
	binary.LittleEndian.PutUint16(object[2:], 1)
	relocOffset := 60 + len(data)
	symbolOffset := relocOffset + 10*len(blocks)
	binary.LittleEndian.PutUint32(object[8:], uint32(symbolOffset))
	binary.LittleEndian.PutUint32(object[12:], 1)
	copy(object[20:], ".rsrc")
	binary.LittleEndian.PutUint32(object[36:], uint32(len(data)))
	binary.LittleEndian.PutUint32(object[40:], 60)
	binary.LittleEndian.PutUint32(object[44:], uint32(relocOffset))
	binary.LittleEndian.PutUint16(object[52:], uint16(len(blocks)))
	binary.LittleEndian.PutUint32(object[56:], 0x40000040)
	object = append(object, data...)
	for _, b := range blocks {
		r := make([]byte, 10)
		binary.LittleEndian.PutUint32(r, uint32(b.entry))
		binary.LittleEndian.PutUint16(r[8:], 3)
		object = append(object, r...)
	}
	symbol := make([]byte, 18)
	copy(symbol, ".rsrc")
	binary.LittleEndian.PutUint16(symbol[12:], 1)
	symbol[16] = 3
	object = append(object, symbol...)
	object = binary.LittleEndian.AppendUint32(object, 4)
	return object
}

func preview(images [][]byte) []byte {
	out := image.NewNRGBA(image.Rect(0, 0, 1120, 940))
	draw.Draw(out, out.Bounds(), image.NewUniform(color.NRGBA{245, 247, 252, 255}), image.Point{}, draw.Src)
	label := func(x, y int, text string, c color.Color) {
		d := font.Drawer{Dst: out, Src: image.NewUniform(c), Face: basicfont.Face7x13, Dot: fixed.P(x, y)}
		d.DrawString(text)
	}
	ink := color.NRGBA{30, 43, 67, 255}
	label(28, 28, "MYCAST / ORIGINAL VECTOR ICON / SYNTHETIC PREVIEW", ink)
	label(28, 50, "No native taskbar capture. Icons below use the generated PNG pixels.", ink)
	for row, bg := range []color.NRGBA{{238, 241, 247, 255}, {23, 28, 39, 255}} {
		y := 72 + row*340
		draw.Draw(out, image.Rect(20, y, 1100, y+332), image.NewUniform(bg), image.Point{}, draw.Src)
		text := ink
		if row == 1 {
			text = color.NRGBA{220, 228, 242, 255}
		}
		label(32, y+24, []string{"LIGHT TASKBAR / 1:1 PIXELS", "DARK TASKBAR / 1:1 PIXELS"}[row], text)
		x := 40
		for i, data := range images {
			im, _ := png.Decode(bytes.NewReader(data))
			n := sizes[i]
			draw.Draw(out, image.Rect(x, y+44, x+n, y+44+n), im, image.Point{}, draw.Over)
			label(x, y+44+n+18, strconv.Itoa(n)+"px", text)
			x += n + 28
		}
		// Explicit true-size strip plus enlarged nearest-neighbor pixel grid.
		label(32, y+280, "16/20/24/32: TRUE PIXELS", text)
		for i := 0; i < 4; i++ {
			im, _ := png.Decode(bytes.NewReader(images[i]))
			n := sizes[i]
			x := 250 + i*70
			draw.Draw(out, image.Rect(x, y+259, x+n, y+259+n), im, image.Point{}, draw.Over)
		}
	}
	label(28, 800, "16px / 20px enlarged 4x (nearest neighbor, no smoothing)", ink)
	for i := 0; i < 2; i++ {
		im, _ := png.Decode(bytes.NewReader(images[i]))
		n := sizes[i]
		x := 490 + i*120
		xdraw.NearestNeighbor.Scale(out, image.Rect(x, 780, x+n*4, 780+n*4), im, im.Bounds(), draw.Over, nil)
	}
	var b bytes.Buffer
	_ = png.Encode(&b, out)
	return b.Bytes()
}
func main() {
	check := flag.Bool("check", false, "verify deterministic outputs without writing")
	flag.Parse()
	if err := run(*check); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(check bool) error {
	svg, err := os.ReadFile("assets/brand/mycast.svg")
	if err != nil {
		return err
	}
	images := make([][]byte, len(sizes))
	outputs := map[string][]byte{}
	for i, n := range sizes {
		images[i], err = raster(svg, n)
		if err != nil {
			return err
		}
		outputs[fmt.Sprintf("assets/brand/mycast-%d.png", n)] = images[i]
	}
	outputs["assets/brand/mycast.ico"] = ico(images)
	outputs["cmd/mycast/icon_windows_amd64.syso"] = coff(images)
	outputs["internal/branding/mycast-256.png"] = images[7]
	outputs["internal/branding/mycast-32.png"] = images[3]
	outputs["docs/previews/mycast-icon.png"] = preview(images)
	for path, data := range outputs {
		if check {
			current, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(current, data) {
				return fmt.Errorf("stale generated icon: %s", path)
			}
		} else {
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(path, data, 0644); err != nil {
				return err
			}
		}
	}
	fmt.Println("MyCast SVG, 8 PNG/ICO sizes, amd64 resource object and synthetic preview verified")
	return nil
}
