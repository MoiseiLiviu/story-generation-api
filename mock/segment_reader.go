package mock_generator

import (
	"encoding/json"
	"generate-script-lambda/application/ports/outbound"
	"os"
)

type SegmentReader interface {
	Read(fileName string) ([]MockSegment, error)
}

type fileSegmentReader struct {
	logger outbound.LoggerPort
}

func NewFileSegmentReader(logger outbound.LoggerPort) SegmentReader {
	return &fileSegmentReader{
		logger: logger,
	}
}

func (f *fileSegmentReader) Read(fileName string) ([]MockSegment, error) {
	var segments []MockSegment
	s, err := f.readJSONFile(fileName)
	if err != nil {
		return nil, err
	}
	segments = append(segments, s...)

	return segments, nil
}

func (f *fileSegmentReader) readJSONFile(fileName string) ([]MockSegment, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			f.logger.Error(err, "failed to close file")
		}
	}(file)

	var events []MockSegment
	if err := json.NewDecoder(file).Decode(&events); err != nil {
		f.logger.Error(err, "failed to decode json")
		return nil, err
	}

	return events, nil
}
