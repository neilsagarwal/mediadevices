package videodisk

// I don't think this is the appropriate place/package to put this reader
// But not sure where to place.

import (
	"bufio"
	"errors"
	"fmt"
	"image"
	"io"
	"strconv"
	"strings"
)

const y4m_signature string = "YUV4MPEG2"

var (
	errNilStream            = errors.New("stream is nil")
	errIncompleteFileHeader = errors.New("incomplete file header")
	errSignatureMismatch    = errors.New("Y4M signature mismatch")
	errWrongColorSpace      = errors.New("only YUV420 color space is supported")
	errorFrameParsing       = errors.New("FRAME prefix is not found")
)

type Y4mReader struct {
	stream       io.Reader
	Width        int
	Height       int
	FPS          int
	bufio_reader *bufio.Reader
}

func NewWith(in io.Reader) (*Y4mReader, error) {
	if in == nil {
		return nil, errNilStream
	}

	reader := &Y4mReader{
		stream:       in,
		bufio_reader: bufio.NewReader(in),
	}

	err := reader.checkSignature()
	if err != nil {
		return nil, err
	}
	err = reader.processHeader()
	if err != nil {
		return nil, err
	}

	return reader, nil

}

func (i *Y4mReader) y_size() int {
	return i.Width * i.Height
}

func (i *Y4mReader) uv_size() int {
	return i.Width * i.Height / 4
}

// check signature
func (i *Y4mReader) checkSignature() error {
	buffer := make([]byte, len(y4m_signature))

	_, err := io.ReadFull(i.bufio_reader, buffer)

	if errors.Is(err, io.ErrUnexpectedEOF) {
		return errIncompleteFileHeader
	} else if err != nil {
		return err
	}

	if string(buffer) != y4m_signature {
		return errSignatureMismatch
	}

	fmt.Println("Signature match!")
	return nil
}

func (i *Y4mReader) processHeader() error {
	line, _ := i.bufio_reader.ReadString('\n')

	fmt.Println(line)

	for _, token := range strings.Split(line, " ") {

		// skip over empty strings
		if len(token) == 0 {
			continue
		}

		switch token[0] {
		case 'W':
			intVar, err := strconv.Atoi(token[1:])
			if err != nil {
				return err
			}
			i.Width = intVar

		case 'H':
			intVar, err := strconv.Atoi(token[1:])
			if err != nil {
				return err
			}
			i.Height = intVar

		case 'C':
			if token[:4] != "C420" {
				return errWrongColorSpace
			}

		case 'F':
			intVar, err := strconv.Atoi(strings.Split(token[1:], ":")[0])
			if err != nil {
				return err
			}
			i.FPS = intVar

		}
	}

	fmt.Println("Header processed!")
	fmt.Printf("Width: %d, Height: %d, FPS: %d\n", i.Width, i.Height, i.FPS)

	return nil
}

func (i *Y4mReader) ParseNextFrame() (image.Image, error) {
	line, err := i.bufio_reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if line[0:5] != "FRAME" {
		return nil, errorFrameParsing
	}

	img_buffer := make([]byte, i.y_size()+i.uv_size()*2)
	_, err = io.ReadFull(i.bufio_reader, img_buffer)
	if err != nil {
		return nil, err
	}

	yi := i.Width * i.Height
	cbi := yi + i.Width*i.Height/4
	cri := cbi + i.Width*i.Height/4

	if cri > len(img_buffer) {
		return nil, fmt.Errorf("frame length (%d) less than expected (%d)", len(img_buffer), cri)
	}

	return &image.YCbCr{
		Y:              img_buffer[:yi],
		YStride:        i.Width,
		Cb:             img_buffer[yi:cbi],
		Cr:             img_buffer[cbi:cri],
		CStride:        i.Width / 2,
		SubsampleRatio: image.YCbCrSubsampleRatio420,
		Rect:           image.Rect(0, 0, i.Width, i.Height),
	}, nil

}
