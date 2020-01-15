package fgee

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"

	"github.com/valyala/fasthttp"
)

type H map[string]interface{}

type Context struct {
	// origin objects
	Ctx *fasthttp.RequestCtx
	// request info
	Path   string
	Method string
	Params map[string]string
	// response info
	StatusCode int
	// middleware
	handlers []HandlerFunc
	index    int8
	// engine pointer
	engine *Engine
	// Keys is a key/value pair exclusively for the context of each request.
	Keys map[string]interface{}
}

func newContext(fastCtx *fasthttp.RequestCtx) *Context {
	return &Context{
		Path:   string(fastCtx.Path()),
		Method: string(fastCtx.Method()),
		Ctx:    fastCtx,
		index:  -1,
		Keys: nil,
	}
}

func (c *Context) Next() {
	c.index++
	s := int8(len(c.handlers))
	for ; c.index < s; c.index++ {
		c.handlers[c.index](c)
	}
}

func (c *Context) Fail(code int, err string) {
	c.index = int8(len(c.handlers))
	c.JSON(code, H{"message": err})
}

func (c *Context) Param(key string) string {
	value, _ := c.Params[key]
	return value
}

func (c *Context) PostForm(key string) string {
	return string(c.Ctx.FormValue(key))
}

func (c *Context) Query(key string) string {
	return string(c.Ctx.QueryArgs().Peek(key))
}

func (c *Context) PostJson() string {
	if string(c.Ctx.Request.Header.ContentType()) != "application/json" {
		return ""
	}
	return string(c.Ctx.Request.Body())
}

func (c *Context) Status(code int) {
	c.StatusCode = code
	c.Ctx.SetStatusCode(code)
}

func (c *Context) SetHeader(key string, value string) {
	c.Ctx.Response.Header.Set(key, value)
}

func (c *Context) String(code int, format string, values ...interface{}) {
	c.Status(code)
	c.Ctx.SetContentType("text/plain")
	if _, err := c.Ctx.Write([]byte(fmt.Sprintf(format, values...))); err != nil {
		log.Printf("%s", err.Error())
		c.Fail(fasthttp.StatusInternalServerError, "Internal Server Error")
	}
}

func (c *Context) JSON(code int, obj interface{}) {
	c.Status(code)
	c.Ctx.SetContentType("application/json")
	encoder := json.NewEncoder(c.Ctx)
	if err := encoder.Encode(obj); err != nil {
		log.Printf("%s", err.Error())
		c.Fail(fasthttp.StatusInternalServerError, "Internal Server Error")
	}
}

func (c *Context) Data(code int, data []byte) {
	c.Status(code)
	if _, err := c.Ctx.Write(data); err != nil {
		log.Printf("%s", err.Error())
		c.Fail(fasthttp.StatusInternalServerError, "Internal Server Error")
	}
}

// HTML template render
// refer https://golang.org/pkg/html/template/
func (c *Context) HTML(code int, name string, data interface{}) {
	c.Status(code)
	c.Ctx.SetContentType("text/html")
	if err := c.engine.htmlTemplates.ExecuteTemplate(c.Ctx, name, data); err != nil {
		log.Printf("%s", err.Error())
		c.Fail(fasthttp.StatusInternalServerError, "Internal Server Error")
	}
}

// Set is used to store a new key/value pair exclusively for this context.
// It also lazy initializes  c.Keys if it was not used previously.
func (c *Context) Set(key string, value interface{}) {
	if c.Keys == nil {
		c.Keys = make(map[string]interface{})
	}
	c.Keys[key] = value
}

// Get returns the value for the given key, ie: (value, true).
// If the value does not exists it returns (nil, false)
func (c *Context) Get(key string) (value interface{}, exists bool) {
	value, exists = c.Keys[key]
	return
}

// MustGet returns the value for the given key if it exists, otherwise it panics.
func (c *Context) MustGet(key string) interface{} {
	if value, exists := c.Get(key); exists {
		return value
	}
	panic("Key \"" + key + "\" does not exist")
}

// SetCookie adds a Set-Cookie header to the ResponseWriter's headers.
// The provided cookie must have a valid Name. Invalid cookies may be
// silently dropped.
func (c *Context) SetCookie(name, value string, maxAge int, path, domain string, secure, httpOnly bool) {
	if path == "" {
		path = "/"
	}
	cookie := fasthttp.Cookie{}
	cookie.SetKey(name)
	cookie.SetValue(value)
	cookie.SetMaxAge(maxAge)
	cookie.SetPath(path)
	cookie.SetDomain(domain)
	cookie.SetSecure(secure)
	cookie.SetHTTPOnly(httpOnly)
	c.Ctx.Response.Header.SetCookie(&cookie)
}

// Cookie returns the named cookie provided in the request or
// ErrNoCookie if not found. And return the named cookie is unescaped.
// If multiple cookies match the given name, only one cookie will
// be returned.
func (c *Context) Cookie(name string) (string, error) {
	cookie := string(c.Ctx.Request.Header.Cookie(name))
	return url.QueryUnescape(cookie)
}
