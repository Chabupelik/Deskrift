package encoder

import (
	"bytes"
	"io"
)

type Frame struct {
	IsKey bool
	Data  []byte
}

type FrameReader struct {
	r      io.Reader
	buf    []byte
	tmp    []byte
	audSeq []byte
}

func NewFrameReader(r io.Reader) *FrameReader {
	return &FrameReader{
		r:      r,
		buf:    make([]byte, 0, 512*1024),
		tmp:    make([]byte, 32*1024),
		audSeq: []byte{0x00, 0x00, 0x00, 0x01, 0x09},
	}
}

func (fr *FrameReader) NextFrame() (*Frame, error) {
	for {
		if len(fr.buf) > 5 {
			nextIdx := bytes.Index(fr.buf[5:], fr.audSeq)
			if nextIdx != -1 {
				frameEnd := nextIdx + 5
				frameBytes := make([]byte, frameEnd)
				copy(frameBytes, fr.buf[:frameEnd])

				fr.buf = fr.buf[frameEnd:]
				return &Frame{
					IsKey: hasKeyframeNAL(frameBytes),
					Data:  frameBytes,
				}, nil
			}
		}

		n, err := fr.r.Read(fr.tmp)
		if n > 0 {
			fr.buf = append(fr.buf, fr.tmp[:n]...)
		}
		if err != nil {
			if len(fr.buf) > 0 && (err == io.EOF || err == io.ErrUnexpectedEOF) {
				frameBytes := make([]byte, len(fr.buf))
				copy(frameBytes, fr.buf)
				fr.buf = nil
				return &Frame{
					IsKey: hasKeyframeNAL(frameBytes),
					Data:  frameBytes,
				}, nil
			}
			return nil, err
		}
	}
}

func hasKeyframeNAL(data []byte) bool {
	for i := 0; i+4 < len(data); i++ {
		if data[i] == 0 && data[i+1] == 0 {
			offset := 0
			if data[i+2] == 1 {
				offset = i + 3
			} else if data[i+2] == 0 && data[i+3] == 1 {
				offset = i + 4
			}
			if offset > 0 && offset < len(data) {
				nalType := data[offset] & 0x1F
				if nalType == 5 || nalType == 7 {
					return true
				}
			}
		}
	}
	return false
}

