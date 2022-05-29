package hls

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"strconv"
	"time"

	"github.com/abema/go-mp4"
	"github.com/aler9/gortsplib"
)

const (
	fmp4PTSDTSOffsetFrames = 2
)

func durationGoToMp4(v time.Duration, timescale time.Duration) int64 {
	return int64(math.Round(float64(v*timescale) / float64(time.Second)))
}

func mp4PartGenerateVideoTraf(
	w *mp4Writer,
	trackID int,
	videoSamples []*fmp4VideoSample,
	startDTS time.Duration,
	videoSampleDefaultDuration time.Duration,
) (*mp4.Trun, int, error) {
	/*
		traf
		- tfhd
		- tfdt
		- trun
	*/

	_, err := w.writeBoxStart(&mp4.Traf{}) // <traf>
	if err != nil {
		return nil, 0, err
	}

	flags := 0

	_, err = w.writeBox(&mp4.Tfhd{ // <tfhd/>
		FullBox: mp4.FullBox{
			Flags: [3]byte{2, byte(flags >> 8), byte(flags)},
		},
		TrackID: uint32(trackID),
	})
	if err != nil {
		return nil, 0, err
	}

	_, err = w.writeBox(&mp4.Tfdt{ // <tfdt/>
		FullBox: mp4.FullBox{
			Version: 1,
		},
		// sum of decode durations of all earlier samples
		BaseMediaDecodeTimeV1: uint64(durationGoToMp4(startDTS, fmp4VideoTimescale)),
	})
	if err != nil {
		return nil, 0, err
	}

	flags = 0
	flags |= 0x01  // data offset present
	flags |= 0x100 // sample duration present
	flags |= 0x200 // sample size present
	flags |= 0x400 // sample flags present
	flags |= 0x800 // sample composition time offset present or v1

	trun := &mp4.Trun{ // <trun/>
		FullBox: mp4.FullBox{
			Version: 1,
			Flags:   [3]byte{0, byte(flags >> 8), byte(flags)},
		},
		SampleCount: uint32(len(videoSamples)),
	}

	for _, e := range videoSamples {
		off := e.pts - e.dts + fmp4PTSDTSOffsetFrames*videoSampleDefaultDuration
		if off < 0 {
			return nil, 0, fmt.Errorf("detected negative offset between PTS and DTS")
		}

		flags := uint32(0)
		if !e.idrPresent {
			flags |= 1 << 16 // sample_is_non_sync_sample
		}

		trun.Entries = append(trun.Entries, mp4.TrunEntry{
			SampleDuration:                uint32(durationGoToMp4(e.duration(), fmp4VideoTimescale)),
			SampleSize:                    uint32(len(e.avcc)),
			SampleFlags:                   flags,
			SampleCompositionTimeOffsetV1: int32(durationGoToMp4(off, fmp4VideoTimescale)),
		})
	}

	trunOffset, err := w.writeBox(trun)
	if err != nil {
		return nil, 0, err
	}

	err = w.writeBoxEnd() // </traf>
	if err != nil {
		return nil, 0, err
	}

	return trun, trunOffset, nil
}

func mp4PartGenerateAudioTraf(
	w *mp4Writer,
	trackID int,
	audioTrack *gortsplib.TrackAAC,
	audioSamples []*fmp4AudioSample,
) (*mp4.Trun, int, error) {
	/*
		traf
		- tfhd
		- tfdt
		- trun
	*/

	if len(audioSamples) == 0 {
		return nil, 0, nil
	}

	_, err := w.writeBoxStart(&mp4.Traf{}) // <traf>
	if err != nil {
		return nil, 0, err
	}

	flags := 0

	_, err = w.writeBox(&mp4.Tfhd{ // <tfhd/>
		FullBox: mp4.FullBox{
			Flags: [3]byte{2, byte(flags >> 8), byte(flags)},
		},
		TrackID: uint32(trackID),
	})
	if err != nil {
		return nil, 0, err
	}

	_, err = w.writeBox(&mp4.Tfdt{ // <tfdt/>
		FullBox: mp4.FullBox{
			Version: 1,
		},
		// sum of decode durations of all earlier samples
		BaseMediaDecodeTimeV1: uint64(durationGoToMp4(audioSamples[0].pts, time.Duration(audioTrack.ClockRate()))),
	})
	if err != nil {
		return nil, 0, err
	}

	flags = 0
	flags |= 0x01  // data offset present
	flags |= 0x100 // sample duration present
	flags |= 0x200 // sample size present

	trun := &mp4.Trun{ // <trun/>
		FullBox: mp4.FullBox{
			Version: 0,
			Flags:   [3]byte{0, byte(flags >> 8), byte(flags)},
		},
		SampleCount: uint32(len(audioSamples)),
	}

	for _, e := range audioSamples {
		trun.Entries = append(trun.Entries, mp4.TrunEntry{
			SampleDuration: uint32(durationGoToMp4(e.duration(), time.Duration(audioTrack.ClockRate()))),
			SampleSize:     uint32(len(e.au)),
		})
	}

	trunOffset, err := w.writeBox(trun)
	if err != nil {
		return nil, 0, err
	}

	err = w.writeBoxEnd() // </traf>
	if err != nil {
		return nil, 0, err
	}

	return trun, trunOffset, nil
}

func mp4PartGenerate(
	videoTrack *gortsplib.TrackH264,
	audioTrack *gortsplib.TrackAAC,
	videoSamples []*fmp4VideoSample,
	audioSamples []*fmp4AudioSample,
	startDTS time.Duration,
	videoSampleDefaultDuration time.Duration,
) ([]byte, error) {
	/*
		moof
		- mfhd
		- traf (video)
		- traf (audio)
		mdat
	*/

	w := newMP4Writer()

	moofOffset, err := w.writeBoxStart(&mp4.Moof{}) // <moof>
	if err != nil {
		return nil, err
	}

	_, err = w.writeBox(&mp4.Mfhd{ // <mfhd/>
		SequenceNumber: 0,
	})
	if err != nil {
		return nil, err
	}

	trackID := 1

	var videoTrun *mp4.Trun
	var videoTrunOffset int
	if videoTrack != nil {
		var err error
		videoTrun, videoTrunOffset, err = mp4PartGenerateVideoTraf(
			w, trackID, videoSamples, startDTS, videoSampleDefaultDuration)
		if err != nil {
			return nil, err
		}

		trackID++
	}

	var audioTrun *mp4.Trun
	var audioTrunOffset int
	if audioTrack != nil {
		var err error
		audioTrun, audioTrunOffset, err = mp4PartGenerateAudioTraf(w, trackID, audioTrack, audioSamples)
		if err != nil {
			return nil, err
		}
	}

	err = w.writeBoxEnd() // </moof>
	if err != nil {
		return nil, err
	}

	mdat := &mp4.Mdat{} // <mdat/>

	dataSize := 0
	videoDataSize := 0

	if videoTrack != nil {
		for _, e := range videoSamples {
			dataSize += len(e.avcc)
		}
		videoDataSize = dataSize
	}

	if audioTrack != nil {
		for _, e := range audioSamples {
			dataSize += len(e.au)
		}
	}

	mdat.Data = make([]byte, dataSize)
	pos := 0

	if videoTrack != nil {
		for _, e := range videoSamples {
			pos += copy(mdat.Data[pos:], e.avcc)
		}
	}

	if audioTrack != nil {
		for _, e := range audioSamples {
			pos += copy(mdat.Data[pos:], e.au)
		}
	}

	mdatOffset, err := w.writeBox(mdat)
	if err != nil {
		return nil, err
	}

	if videoTrack != nil {
		videoTrun.DataOffset = int32(mdatOffset - moofOffset + 8)
		err = w.rewriteBox(videoTrunOffset, videoTrun)
		if err != nil {
			return nil, err
		}
	}

	if audioTrack != nil && audioTrun != nil {
		audioTrun.DataOffset = int32(videoDataSize + mdatOffset - moofOffset + 8)
		err = w.rewriteBox(audioTrunOffset, audioTrun)
		if err != nil {
			return nil, err
		}
	}

	return w.bytes(), nil
}

func fmp4PartName(id uint64) string {
	return "part" + strconv.FormatUint(id, 10)
}

type muxerVariantFMP4Part struct {
	videoTrack                 *gortsplib.TrackH264
	audioTrack                 *gortsplib.TrackAAC
	id                         uint64
	startDTS                   time.Duration
	videoSampleDefaultDuration time.Duration

	isIndependent    bool
	videoSamples     []*fmp4VideoSample
	audioSamples     []*fmp4AudioSample
	renderedContent  []byte
	renderedDuration time.Duration
}

func newMuxerVariantFMP4Part(
	videoTrack *gortsplib.TrackH264,
	audioTrack *gortsplib.TrackAAC,
	id uint64,
	startDTS time.Duration,
	videoSampleDefaultDuration time.Duration,
) *muxerVariantFMP4Part {
	p := &muxerVariantFMP4Part{
		videoTrack:                 videoTrack,
		audioTrack:                 audioTrack,
		id:                         id,
		startDTS:                   startDTS,
		videoSampleDefaultDuration: videoSampleDefaultDuration,
	}

	if videoTrack == nil {
		p.isIndependent = true
	}

	return p
}

func (p *muxerVariantFMP4Part) name() string {
	return fmp4PartName(p.id)
}

func (p *muxerVariantFMP4Part) reader() io.Reader {
	return bytes.NewReader(p.renderedContent)
}

func (p *muxerVariantFMP4Part) duration() time.Duration {
	if p.videoTrack != nil {
		ret := time.Duration(0)
		for _, e := range p.videoSamples {
			ret += e.duration()
		}
		return ret
	}

	return p.audioSamples[len(p.audioSamples)-1].next.pts - p.audioSamples[0].pts
}

func (p *muxerVariantFMP4Part) finalize() error {
	if len(p.videoSamples) > 0 || len(p.audioSamples) > 0 {
		var err error
		p.renderedContent, err = mp4PartGenerate(
			p.videoTrack,
			p.audioTrack,
			p.videoSamples,
			p.audioSamples,
			p.startDTS,
			p.videoSampleDefaultDuration)
		if err != nil {
			return err
		}

		p.renderedDuration = p.duration()
	}

	p.videoSamples = nil
	p.audioSamples = nil

	return nil
}

func (p *muxerVariantFMP4Part) writeH264(sample *fmp4VideoSample) {
	if sample.idrPresent {
		p.isIndependent = true
	}
	p.videoSamples = append(p.videoSamples, sample)
}

func (p *muxerVariantFMP4Part) writeAAC(sample *fmp4AudioSample) {
	p.audioSamples = append(p.audioSamples, sample)
}
