package adapters

import (
	"bufio"
	"fmt"
	"generate-script-lambda/application/ports/outbound"
	"generate-script-lambda/domain"
	"github.com/google/uuid"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type videoCreator struct {
	logger outbound.LoggerPort
}

func NewVideoCreator(logger outbound.LoggerPort) outbound.VideoCreatorPort {
	return &videoCreator{
		logger: logger,
	}
}

func (v *videoCreator) Create(segments []domain.AudioWithImageBackground) (*outbound.VideoCreatorResponse, error) {
	sort.Sort(domain.AudioSegmentsAscByOrdinal(segments))
	durations, err := v.getAudioDurations(segments)
	if err != nil {
		return nil, err
	}

	mergedAudio, err := v.concatenateAudio(segments)
	if err != nil {
		v.logger.Error(err, "error merging audio")
		return nil, err
	}

	videoFileName, err := v.CreateVideo(mergedAudio, durations, segments)
	if err != nil {
		return nil, err
	}

	videoSegments, err := v.createVideoSegments(segments, durations)
	if err != nil {
		v.logger.Error(err, "error creating video segments")
		return nil, err
	}

	return &outbound.VideoCreatorResponse{
		VideoFileName: videoFileName,
		VideoSegments: videoSegments,
	}, nil
}

func (v *videoCreator) createVideoSegments(segments []domain.AudioWithImageBackground, durations []float64) ([]domain.VideoSegment, error) {
	videoSegments := make([]domain.VideoSegment, 0)
	for i, s := range segments {
		videoSegments = append(videoSegments, domain.VideoSegment{
			ID:       s.ID,
			Duration: durations[i],
			Ordinal:  s.Ordinal,
			Text:     s.Text,
		})
	}

	return videoSegments, nil
}

func (v *videoCreator) getAudioDurations(segments []domain.AudioWithImageBackground) ([]float64, error) {
	durations := make([]float64, 0)
	for _, s := range segments {
		cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", s.FileName)

		out, err := cmd.Output()
		if err != nil {
			v.logger.Error(err, "error getting audio duration")
			return nil, err
		}

		durationStr := strings.TrimSpace(string(out))

		re := regexp.MustCompile(`\.\d+`)
		durationStr = re.ReplaceAllString(durationStr, "")

		duration, err := strconv.ParseFloat(durationStr, 64)
		if err != nil {
			v.logger.Error(err, "error parsing audio duration")
			return nil, err
		}

		durations = append(durations, duration)
	}

	return durations, nil
}

func (v *videoCreator) concatenateAudio(segments []domain.AudioWithImageBackground) (finalFileName string, err error) {
	listFileName := uuid.NewString()
	fileList, err := os.Create("/tmp/" + listFileName)
	if err != nil {
		v.logger.Error(err, "Failed to create audio list file")
		return
	}

	if err != nil {
		v.logger.Error(err, "Failed to create video list file")
		return
	}

	defer func(fileList *os.File) {
		err := fileList.Close()
		if err != nil {
			v.logger.Error(err, "Failed to close video list file")
			return
		}
		err = os.Remove(fileList.Name())
		if err != nil {
			v.logger.Error(err, "Failed to remove video list file")
			return
		}
		for _, s := range segments {
			err = os.Remove(s.FileName)
			if err != nil {
				v.logger.Error(err, "Failed to remove segment file")
				return
			}
		}
	}(fileList)

	writer := bufio.NewWriter(fileList)
	for _, s := range segments {
		_, err = writer.WriteString("file '" + s.FileName + "'\n")
		if err != nil {
			v.logger.Error(err, "Failed to write to video list file")
			return
		}
	}
	err = writer.Flush()
	if err != nil {
		v.logger.Error(err, "Failed to flush video list file")
		return
	}
	b, err := os.ReadFile(fileList.Name())
	if err != nil {
		v.logger.Error(err, "Failed to read video list file")
		return
	}

	v.logger.Debug("List: " + string(b))

	finalFileName = "/tmp/" + uuid.NewString() + ".mp3"

	cmd := exec.Command("ffmpeg", "-f", "concat", "-safe", "0", "-i", fileList.Name(), "-c", "copy", finalFileName)
	err = cmd.Run()
	if err != nil {
		v.logger.Error(err, "Failed to concatenate videos")
		return
	}

	return
}

func (v *videoCreator) generateSlideshowCommand(segments []domain.AudioWithImageBackground, durations []float64, audioFile string, outputFile string) *exec.Cmd {
	var args []string
	for i, seg := range segments {
		args = append(args, "-loop", "1", "-t", fmt.Sprintf("%f", durations[i]), "-i", seg.BackgroundImageFileName)
	}

	args = append(args, "-i", audioFile)

	var filters []string
	var overlays []string
	lastOverlayName := "0"

	accumulatedFade := 1.0
	for i := range segments {
		fadeStart := accumulatedFade - 1
		accumulatedFade += durations[i]
		filters = append(filters, fmt.Sprintf("[%d]fade=d=1:t=in:alpha=1,setpts=PTS-STARTPTS+%f/TB[f%d];", i, fadeStart, i))
		newOverlayName := fmt.Sprintf("bg%d", i+1)

		if i == len(segments)-1 {
			overlays = append(overlays, fmt.Sprintf("[%s][f%d]overlay", lastOverlayName, i))
		} else {
			overlays = append(overlays, fmt.Sprintf("[%s][f%d]overlay[%s]", lastOverlayName, i, newOverlayName))
			lastOverlayName = newOverlayName
		}
	}

	overlayStr := strings.Join(overlays, ";") + ",format=yuv420p[v]"

	args = append(args, "-filter_complex", strings.Join(append(filters, overlayStr), " "))

	args = append(args, "-movflags", "faststart", "-map", "[v]", "-map", fmt.Sprintf("%d:a", len(segments)), "-r", "25", outputFile)

	v.logger.Debug("ffmpeg " + strings.Join(args, " "))

	return exec.Command("ffmpeg", args...)
}

func (v *videoCreator) CreateVideo(mergedAudioFileName string, durations []float64, segments []domain.AudioWithImageBackground) (string, error) {
	defer func() {
		err := os.Remove(mergedAudioFileName)
		if err != nil {
			v.logger.Error(err, "error removing audio file")
		}
		for _, s := range segments {
			err = os.Remove(s.BackgroundImageFileName)
			if err != nil {
				v.logger.Error(err, "error removing image file")
			}
		}
	}()

	outputFile := "/tmp/" + uuid.NewString() + ".mp4"

	c := v.generateSlideshowCommand(segments, durations, mergedAudioFileName, outputFile)

	err := c.Run()

	if err != nil {
		return "", err
	}

	return outputFile, nil
}
