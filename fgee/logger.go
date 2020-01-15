package fgee

import (
	"log"
	"time"
)

func Logger() HandlerFunc {
	log.Printf("Init Logger Middleware")
	return func(c *Context) {
		// Start timer
		t := time.Now()
		// Process request
		c.Next()
		// Calculate resolution time
		log.Printf("%s %d %s %s %s in %v", c.Ctx.RemoteIP(), c.StatusCode, c.Method, c.Ctx.RequestURI(), c.Ctx.UserAgent(), time.Since(t))
	}
}
