package adapters

import (
	"generate-script-lambda/application/ports/outbound"
	"github.com/rs/zerolog"
	"os"
)

type zerologWrapper struct {
	logger zerolog.Logger
}

func NewZerologWrapper() outbound.LoggerPort {
	return &zerologWrapper{
		logger: zerolog.New(os.Stderr).With().Timestamp().Logger(),
	}
}

func (z *zerologWrapper) Info(msg string) {
	z.logger.Info().Msg(msg)
}

func (z *zerologWrapper) Error(err error, msg string) {
	z.logger.Error().Err(err).Msg(msg)
}

func (z *zerologWrapper) Debug(msg string) {
	z.logger.Debug().Msg(msg)
}

func (z *zerologWrapper) Warn(msg string) {
	z.logger.Warn().Msg(msg)
}

func (z *zerologWrapper) InfoWithFields(msg string, fields map[string]interface{}) {
	z.logger.Info().Fields(fields).Msg(msg)
}

func (z *zerologWrapper) ErrorWithFields(err error, msg string, fields map[string]interface{}) {
	z.logger.Error().Err(err).Fields(fields).Msg(msg)
}

func (z *zerologWrapper) DebugWithFields(msg string, fields map[string]interface{}) {
	z.logger.Debug().Fields(fields).Msg(msg)
}

func (z *zerologWrapper) WarnWithFields(msg string, fields map[string]interface{}) {
	z.logger.Warn().Fields(fields).Msg(msg)
}
