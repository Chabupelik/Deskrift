package ui

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
)

func CreateDefaultIcon() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	cyan := color.RGBA{R: 59, G: 130, B: 246, A: 255}
	dark := color.RGBA{R: 24, G: 24, B: 27, A: 255}

	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			if y >= 2 && y <= 11 && x >= 1 && x <= 14 {
				if y == 2 || y == 11 || x == 1 || x == 14 {
					img.Set(x, y, cyan)
				} else {
					img.Set(x, y, dark)
				}
			} else if y >= 12 && y <= 14 && x >= 6 && x <= 9 {
				img.Set(x, y, cyan)
			} else if y == 14 && x >= 4 && x <= 11 {
				img.Set(x, y, cyan)
			}
		}
	}

	var pngBuf bytes.Buffer
	_ = png.Encode(&pngBuf, img)
	pngBytes := pngBuf.Bytes()

	var icoBuf bytes.Buffer
	binary.Write(&icoBuf, binary.LittleEndian, uint16(0))
	binary.Write(&icoBuf, binary.LittleEndian, uint16(1))
	binary.Write(&icoBuf, binary.LittleEndian, uint16(1))

	icoBuf.WriteByte(16)
	icoBuf.WriteByte(16)
	icoBuf.WriteByte(0)
	icoBuf.WriteByte(0)
	binary.Write(&icoBuf, binary.LittleEndian, uint16(1))
	binary.Write(&icoBuf, binary.LittleEndian, uint16(32))
	binary.Write(&icoBuf, binary.LittleEndian, uint32(len(pngBytes)))
	binary.Write(&icoBuf, binary.LittleEndian, uint32(22))

	icoBuf.Write(pngBytes)
	return icoBuf.Bytes()
}

