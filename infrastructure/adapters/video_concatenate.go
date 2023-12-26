package adapters

import (
	"bufio"
	"generate-script-lambda/application/ports/outbound"
	"generate-script-lambda/domain"
	"github.com/google/uuid"
	"os"
	"os/exec"
	"sort"
)

type ffmpegVideoConcatenate struct {
	logger outbound.LoggerPort
}

func NewFFmpegVideoConcatenate(logger outbound.LoggerPort) outbound.ConcatenateVideosPort {
	return &ffmpegVideoConcatenate{
		logger: logger,
	}
}

func (f *ffmpegVideoConcatenate) Concatenate(segments []domain.VideoSegment) (finalFileName string, err error) {
	sort.Sort(domain.VideoSegmentsAscByOrdinal(segments))
	listFileName := uuid.NewString()
	fileList, err := os.Create("/tmp/" + listFileName)

	defer func(fileList *os.File) {
		err := fileList.Close()
		if err != nil {
			f.logger.Error(err, "Failed to close video list file")
			return
		}
	}(fileList)
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			f.logger.Error(err, "Failed to remove video list file")
			return
		}
	}(fileList.Name())

	if err != nil {
		f.logger.Error(err, "Failed to create video list file")
		return
	}

	writer := bufio.NewWriter(fileList)
	for _, s := range segments {
		_, err = writer.WriteString("file '" + s.FileName + "'\n")
		if err != nil {
			f.logger.Error(err, "Failed to write to video list file")
			return
		}
	}
	err = writer.Flush()
	if err != nil {
		f.logger.Error(err, "Failed to flush video list file")
		return
	}

	finalFileName = uuid.NewString() + ".mp4"

	cmd := exec.Command("ffmpeg", "-f", "concat", "-safe", "0", "-i", fileList.Name(), "-c", "copy", finalFileName)
	err = cmd.Run()
	if err != nil {
		f.logger.Error(err, "Failed to concatenate videos")
		return
	}
	for _, s := range segments {
		err = os.Remove(s.FileName)
		if err != nil {
			f.logger.Error(err, "Failed to remove segment file")
			return
		}
	}

	return
}
