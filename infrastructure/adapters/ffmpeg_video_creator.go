package adapters

import (
	"generate-script-lambda/application/ports/outbound"
	"github.com/google/uuid"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

type ffmpegVideoCreator struct {
	logger         outbound.LoggerPort
	durationRegexp *regexp.Regexp
}

func NewFFMPEGVideoCreator(logger outbound.LoggerPort) outbound.SegmentVideoCreator {
	return &ffmpegVideoCreator{
		logger:         logger,
		durationRegexp: regexp.MustCompile(`\.\d+`),
	}
}

func (v *ffmpegVideoCreator) Create(audioFileName string, imageFileName string) (*outbound.CreateVideoResponse, error) {
	defer func() {
		err := os.Remove(audioFileName)
		if err != nil {
			v.logger.Error(err, "error removing audio file")
		}
		err = os.Remove(imageFileName)
		if err != nil {
			v.logger.Error(err, "error removing image file")
		}
	}()

	videoID := uuid.NewString()
	outputFile := "/tmp/" + videoID + ".mp4"
	cmd := exec.Command("ffmpeg", "-loop", "1", "-i", imageFileName, "-i", audioFileName, "-c:v", "libx264",
		"-tune", "stillimage", "-c:a", "aac", "-b:a", "192k", "-pix_fmt", "yuv420p", "-shortest", outputFile)
	err := cmd.Run()
	if err != nil {
		v.logger.Error(err, "error creating video")
		return nil, err
	}

	duration, err := v.getVideoDuration(outputFile)
	if err != nil {
		v.logger.Error(err, "error getting video duration")
		return nil, err
	}

	return &outbound.CreateVideoResponse{
		FileName: outputFile,
		Duration: duration,
	}, nil
}

func (v *ffmpegVideoCreator) getVideoDuration(filePath string) (float64, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", filePath)

	out, err := cmd.Output()
	if err != nil {
		v.logger.Error(err, "error getting video duration")
		return 0, err
	}

	durationStr := strings.TrimSpace(string(out))

	re := regexp.MustCompile(`\.\d+`)
	durationStr = re.ReplaceAllString(durationStr, "")

	duration, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		v.logger.Error(err, "error parsing video duration")
		return 0, err
	}

	return duration, nil
}
