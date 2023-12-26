package outbound

type CreateVideoResponse struct {
	FileName string
	Duration float64
}

type SegmentVideoCreator interface {
	Create(audioFileName string, imageFileName string) (*CreateVideoResponse, error)
}
