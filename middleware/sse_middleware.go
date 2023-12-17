package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/panjf2000/ants/v2"
	"time"
)

func SSEMiddleware(workerPool *ants.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")

		clientGone := c.Request.Context().Done()

		err := workerPool.Submit(func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()

			for {
				select {
				case t := <-ticker.C:
					if _, err := c.Writer.WriteString("data: " + t.Format(time.RFC3339) + "\n\n"); err != nil {
						return
					}
					c.Writer.Flush()
				case <-clientGone:
					return
				}
			}
		})
		if err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
			return
		}

		c.Next()
	}
}
