package videodisk

import (
	"context"
	"errors"
	"image"
	"io"
	"os"
	"strings"
	"time"

	"github.com/pion/mediadevices/pkg/frame"
	"github.com/pion/mediadevices/pkg/io/video"
	"github.com/pion/mediadevices/pkg/prop"
)

// func init() {
// 	driver.GetManager().Register(
// 		newVideo(),
// 		driver.Info{Label: "videoOnDisk", DeviceType: driver.Camera},
// 	)
// }

type savedVideo struct {
	closed <-chan struct{}
	cancel func()
	tick   *time.Ticker
	reader *Y4mReader
}

func NewVideo(videoFileName string) *savedVideo {

	splitVideoName := strings.Split(videoFileName, ".")
	if splitVideoName[len(splitVideoName)-1] != "y4m" {
		panic(errors.New("video is not .y4m"))
	}

	file, y4merr := os.Open(videoFileName)
	if y4merr != nil {
		panic(y4merr)
	}

	reader, err := NewWith(file)
	if err != nil {
		panic(err)
	}

	return &savedVideo{
		reader: reader,
	}

}

func (d *savedVideo) Open() error {
	ctx, cancel := context.WithCancel(context.Background())
	d.closed = ctx.Done()
	d.cancel = cancel
	return nil
}

func (d *savedVideo) Close() error {
	d.cancel()
	if d.tick != nil {
		d.tick.Stop()
	}
	return nil
}

func (d *savedVideo) VideoRecord(p prop.Media) (video.Reader, error) {

	tick := time.NewTicker(time.Duration(float32(time.Second) / float32(d.reader.FPS)))
	d.tick = tick
	closed := d.closed

	r := video.ReaderFunc(func() (image.Image, func(), error) {
		select {
		case <-closed:
			return nil, func() {}, io.EOF
		default:
		}

		<-tick.C

		img, error := d.reader.ParseNextFrame()

		if error != nil {
			return nil, func() {}, error
		}

		return img, func() {}, nil
	})

	return r, nil

}

func (d savedVideo) Properties() []prop.Media {
	return []prop.Media{
		{
			Video: prop.Video{
				Width:       d.reader.Width,
				Height:      d.reader.Height,
				FrameFormat: frame.FormatI420,
				FrameRate:   float32(d.reader.FPS),
			},
		},
	}
}
