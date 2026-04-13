package rtsp

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	stdjpeg "image/jpeg"
	"log"
	"sync"
	"time"

	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/bluenviron/gortsplib/v4/pkg/description"
	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/bluenviron/gortsplib/v4/pkg/format/rtpmjpeg"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

type Server struct {
	port       string
	server     *gortsplib.Server
	stream     *gortsplib.ServerStream
	quitChan   chan struct{}
	running    bool
	isOccupied bool
	mux        sync.Mutex
}

func NewServer(port string) *Server {
	return &Server{
		port:     port,
		quitChan: make(chan struct{}),
	}
}

func (s *Server) SetOccupied(isOccupied bool) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.isOccupied = isOccupied
}

func (s *Server) isOccupiedSafe() bool {
	s.mux.Lock()
	defer s.mux.Unlock()
	return s.isOccupied
}

type serverHandler struct {
	stream *gortsplib.ServerStream
}

func (h *serverHandler) OnConnOpen(ctx *gortsplib.ServerHandlerOnConnOpenCtx) {
}

func (h *serverHandler) OnConnClose(ctx *gortsplib.ServerHandlerOnConnCloseCtx) {
}

func (h *serverHandler) OnSessionOpen(ctx *gortsplib.ServerHandlerOnSessionOpenCtx) {
}

func (h *serverHandler) OnSessionClose(ctx *gortsplib.ServerHandlerOnSessionCloseCtx) {
}

func (h *serverHandler) OnDescribe(ctx *gortsplib.ServerHandlerOnDescribeCtx) (*base.Response, *gortsplib.ServerStream, error) {
	return &base.Response{
		StatusCode: base.StatusOK,
	}, h.stream, nil
}

func (h *serverHandler) OnSetup(ctx *gortsplib.ServerHandlerOnSetupCtx) (*base.Response, *gortsplib.ServerStream, error) {
	return &base.Response{
		StatusCode: base.StatusOK,
	}, h.stream, nil
}

func (h *serverHandler) OnPlay(ctx *gortsplib.ServerHandlerOnPlayCtx) (*base.Response, error) {
	return &base.Response{
		StatusCode: base.StatusOK,
	}, nil
}

func (s *Server) Start() error {
	if s.running {
		return nil
	}
	s.running = true

	mjpegFormat := &format.MJPEG{}
	desc := &description.Session{
		Medias: []*description.Media{
			{
				Type:    description.MediaTypeVideo,
				Formats: []format.Format{mjpegFormat},
			},
		},
	}

	h := &serverHandler{}
	s.server = &gortsplib.Server{
		RTSPAddress: fmt.Sprintf(":%s", s.port),
		Handler:     h,
	}

	if err := s.server.Start(); err != nil {
		return err
	}

	s.stream = gortsplib.NewServerStream(s.server, desc)
	h.stream = s.stream

	go s.runStream(mjpegFormat, desc.Medias[0])

	log.Printf("Started RTSP Server on port %s", s.port)
	return nil
}

func (s *Server) Stop() {
	if !s.running {
		return
	}
	s.running = false
	close(s.quitChan)
	if s.stream != nil {
		s.stream.Close()
	}
	if s.server != nil {
		s.server.Close()
	}
}

func addLabel(img *image.RGBA, x, y int, label string, col color.RGBA) {
	point := fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y)}
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: basicfont.Face7x13,
		Dot:  point,
	}
	d.DrawString(label)
}

func (s *Server) runStream(f *format.MJPEG, media *description.Media) {
	ticker := time.NewTicker(time.Second / 10)
	defer ticker.Stop()

	// rtp packetizer
	packetizer := &rtpmjpeg.Encoder{
	}
	packetizer.Init()

	var sequenceNumber uint16
	var timestamp uint32
	var ssrc uint32 = 12345

	for {
		select {
		case <-s.quitChan:
			return
		case <-ticker.C:
			img := image.NewRGBA(image.Rect(0, 0, 640, 480))
			draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{0, 0, 0, 255}}, image.Point{}, draw.Src)

			nowStr := time.Now().Format("15:04:05.000000")
			addLabel(img, 10, 20, nowStr, color.RGBA{255, 255, 255, 255})

			if s.isOccupiedSafe() {
				addLabel(img, 320-30, 450, "Occupied", color.RGBA{255, 0, 0, 255})
			}

			var buf bytes.Buffer
			if err := stdjpeg.Encode(&buf, img, &stdjpeg.Options{Quality: 60}); err == nil {
				packets, err := packetizer.Encode(buf.Bytes())
				if err == nil {
					for _, pkt := range packets {
						pkt.Header.SequenceNumber = sequenceNumber
						sequenceNumber++
						pkt.Header.Timestamp = timestamp
						pkt.Header.SSRC = ssrc

						s.stream.WritePacketRTP(media, pkt)
					}
					timestamp += 90000 / 10 // 90kHz / 10fps
				}
			}
		}
	}
}
