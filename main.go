package main

import (
	"fmt"
	"image/jpeg"
	"log"
	"net"
	"os"

	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
)

func main() {
	log.SetFlags(log.Lmicroseconds)

	dec := &decoder{}
	dec.initialize()

	addr, _ := net.ResolveUDPAddr("udp", ":5000")
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	log.Printf("listening on :5000")
	log.Printf("send a stream with:")
	log.Printf("  ffmpeg -re -f lavfi -i \"testsrc=rate=1:size=320x240\" -pix_fmt yuv420p -c:v libx265 -tune zerolatency -f rtp rtp://localhost:5000")

	buf := make([]byte, 65536)
	var depacketizer codecs.H265Depacketizer
	var annexb []byte
	var curTimestamp uint32
	frameCount := 0

	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("read error: %v", err)
			continue
		}

		var pkt rtp.Packet
		if err := pkt.Unmarshal(buf[:n]); err != nil {
			continue
		}

		if pkt.Timestamp != curTimestamp {
			annexb = annexb[:0]
			curTimestamp = pkt.Timestamp
		}

		payload := make([]byte, len(pkt.Payload))
		copy(payload, pkt.Payload)

		data, err := depacketizer.Unmarshal(payload)
		if err != nil {
			continue
		}
		annexb = append(annexb, data...)

		if !pkt.Marker {
			continue
		}

		if len(annexb) == 0 {
			continue
		}

		log.Printf("AU %d: %d bytes -> decoder", frameCount, len(annexb))

		img := dec.decode(annexb)
		if img == nil {
			log.Printf("AU %d: no frame returned (decoder is buffering)", frameCount)
			frameCount++
			continue
		}

		filename := fmt.Sprintf("%d.jpg", frameCount)
		f, err := os.Create(filename)
		if err != nil {
			log.Printf("AU %d: create %s: %v", frameCount, filename, err)
			frameCount++
			continue
		}
		jpeg.Encode(f, img, &jpeg.Options{Quality: 90})
		f.Close()

		log.Printf("AU %d: decoded %dx%d -> %s", frameCount, img.Bounds().Dx(), img.Bounds().Dy(), filename)
		frameCount++
	}
}
