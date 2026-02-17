package main

import (
	"image"
	"runtime"
	"unsafe"
)

// #cgo pkg-config: libavcodec libavutil
// #include <libavcodec/avcodec.h>
// #include <libavutil/imgutils.h>
import "C"

type decoder struct {
	codecCtx *C.AVCodecContext
	frame    *C.AVFrame
}

func (d *decoder) initialize() {
	codec := C.avcodec_find_decoder(C.AV_CODEC_ID_H265)
	d.codecCtx = C.avcodec_alloc_context3(codec)
	d.codecCtx.flags |= C.AV_CODEC_FLAG_LOW_DELAY
	d.codecCtx.flags2 |= C.AV_CODEC_FLAG2_FAST
	d.codecCtx.thread_type = C.FF_THREAD_SLICE
	C.avcodec_open2(d.codecCtx, codec, nil)
	d.frame = C.av_frame_alloc()
}

func (d *decoder) decode(annexb []byte) *image.YCbCr {
	var pinner runtime.Pinner
	pinner.Pin(&annexb[0])
	var pkt C.AVPacket
	pkt.data = (*C.uint8_t)(unsafe.Pointer(&annexb[0]))
	pkt.size = C.int(len(annexb))
	C.avcodec_send_packet(d.codecCtx, &pkt)
	pinner.Unpin()

	if C.avcodec_receive_frame(d.codecCtx, d.frame) < 0 {
		return nil
	}

	w := int(d.frame.width)
	h := int(d.frame.height)
	yStride := int(d.frame.linesize[0])
	cStride := int(d.frame.linesize[1])

	return &image.YCbCr{
		Y:              unsafe.Slice((*byte)(unsafe.Pointer(d.frame.data[0])), yStride*h),
		Cb:             unsafe.Slice((*byte)(unsafe.Pointer(d.frame.data[1])), cStride*h/2),
		Cr:             unsafe.Slice((*byte)(unsafe.Pointer(d.frame.data[2])), cStride*h/2),
		YStride:        yStride,
		CStride:        cStride,
		SubsampleRatio: image.YCbCrSubsampleRatio420,
		Rect:           image.Rect(0, 0, w, h),
	}
}
