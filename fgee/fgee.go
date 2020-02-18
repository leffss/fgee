package fgee

import (
	"html/template"
	"log"
	"path"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

var (
	ReadTimeout time.Duration = time.Second * 30
	WriteTimeout time.Duration = time.Second * 30
)

// HandlerFunc defines the request handler used by gee
type HandlerFunc func(*Context)

// Engine implement the interface of ServeHTTP
type (
	RouterGroup struct {
		prefix      string
		middlewares []HandlerFunc // support middleware
		parent      *RouterGroup  // support nesting
		engine      *Engine       // all groups share a Engine instance
	}

	Engine struct {
		*RouterGroup
		router        *router
		groups        []*RouterGroup     // store all groups
		htmlTemplates *template.Template // for html render
		funcMap       template.FuncMap   // for html render
		server *fasthttp.Server
	}
)

func SetReadTimeout(second int) {
	ReadTimeout = time.Second * time.Duration(second)
}

func SetWriteTimeout(second int) {
	WriteTimeout = time.Second * time.Duration(second)
}

// New is the constructor of gee.Engine
func New() *Engine {
	engine := &Engine{router: newRouter()}
	engine.RouterGroup = &RouterGroup{engine: engine}
	engine.groups = []*RouterGroup{engine.RouterGroup}
	return engine
}

// Default use Logger & Recovery middleware
func Default() *Engine {
	engine := New()
	engine.Use(Logger(), Recovery())
	return engine
}

// Group is defined to create a new RouterGroup
// remember all groups share the same Engine instance
func (group *RouterGroup) Group(prefix string) *RouterGroup {
	engine := group.engine
	newGroup := &RouterGroup{
		prefix: group.prefix + prefix,
		parent: group,
		engine: engine,
	}
	engine.groups = append(engine.groups, newGroup)
	return newGroup
}

// Use is defined to add middleware to the group
func (group *RouterGroup) Use(middlewares ...HandlerFunc) {
	group.middlewares = append(group.middlewares, middlewares...)
}

func (group *RouterGroup) addRoute(method string, comp string, handler HandlerFunc) {
	pattern := group.prefix + comp
	log.Printf("Route %4s - %s", method, pattern)
	group.engine.router.addRoute(method, pattern, handler)
}

// GET defines the method to add GET request
func (group *RouterGroup) GET(pattern string, handler HandlerFunc) {
	group.addRoute(fasthttp.MethodGet, pattern, handler)
}

// POST defines the method to add POST request
func (group *RouterGroup) POST(pattern string, handler HandlerFunc) {
	group.addRoute(fasthttp.MethodPost, pattern, handler)
}

// PUT defines the method to add PUT request
func (group *RouterGroup) PUT(pattern string, handler HandlerFunc) {
	group.addRoute(fasthttp.MethodPut, pattern, handler)
}


// DELETE defines the method to add DELETE request
func (group *RouterGroup) DELETE(pattern string, handler HandlerFunc) {
	group.addRoute(fasthttp.MethodDelete, pattern, handler)
}


// PATCH defines the method to add PATCH request
func (group *RouterGroup) PATCH(pattern string, handler HandlerFunc) {
	group.addRoute(fasthttp.MethodPatch, pattern, handler)
}

// HEAD defines the method to add HEAD request
func (group *RouterGroup) HEAD(pattern string, handler HandlerFunc) {
	group.addRoute(fasthttp.MethodHead, pattern, handler)
}

// OPTIONS defines the method to add OPTIONS request
func (group *RouterGroup) OPTIONS(pattern string, handler HandlerFunc) {
	group.addRoute(fasthttp.MethodOptions, pattern, handler)
}

// TRACE defines the method to add TRACE request
func (group *RouterGroup) TRACE(pattern string, handler HandlerFunc) {
	group.addRoute(fasthttp.MethodTrace, pattern, handler)
}

// Any registers a route that matches all the HTTP methods.
// GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS, TRACE.
func (group *RouterGroup) Any(pattern string, handler HandlerFunc) {
	group.GET(pattern, handler)
	group.POST(pattern, handler)
	group.PUT(pattern, handler)
	group.DELETE(pattern, handler)
	group.PATCH(pattern, handler)
	group.HEAD(pattern, handler)
	group.OPTIONS(pattern, handler)
	group.TRACE(pattern, handler)
}

//create static handler
func (group *RouterGroup) createStaticHandler(relativePath string, fs string) HandlerFunc {
	pathRewrite := fasthttp.NewPathPrefixStripper(len(relativePath))
	if relativePath[len(relativePath) - 1] == '/' {
		pathRewrite = fasthttp.NewPathPrefixStripper(len(relativePath) - 1)
	}

	fileServer := &fasthttp.FS{
		// Path to directory to serve.
		Root: fs,
		// Generate index pages if client requests directory contents.
		GenerateIndexPages: false,
		// Enable transparent compression to save network traffic.
		Compress: true,
		PathRewrite: pathRewrite,
	}
	static := fileServer.NewRequestHandler()
	return func(c *Context) {
		static(c.Ctx)
	}
}

//serve static files
func (group *RouterGroup) Static(relativePath string, root string) {
	handler := group.createStaticHandler(group.prefix + relativePath, root)
	urlPattern := path.Join(relativePath, "/*static")
	// Register GET handlers
	group.GET(urlPattern, handler)
}

// custom render function
func (engine *Engine) SetFuncMap(funcMap template.FuncMap) {
	engine.funcMap = funcMap
}

func (engine *Engine) LoadHTMLGlob(pattern string) {
	engine.htmlTemplates = template.Must(template.New("").Funcs(engine.funcMap).ParseGlob(pattern))
}

// Run defines the method to start a http server
func (engine *Engine) Run(addr string) (err error) {
	server := &fasthttp.Server{
		Handler: engine.ServeHTTP,
		ReadTimeout: ReadTimeout,
		WriteTimeout: WriteTimeout,
	}
	engine.server = server
	return server.ListenAndServe(addr)
}

func (engine *Engine) RunTLS(addr, ca, key string) (err error) {
	server := &fasthttp.Server{
		Handler: engine.ServeHTTP,
		ReadTimeout: ReadTimeout,
		WriteTimeout: WriteTimeout,
	}
	engine.server = server
	return server.ListenAndServeTLS(addr, ca, key)
}

func (engine *Engine) Shutdown() (err error) {
	return engine.server.Shutdown()
}

func (engine *Engine) ServeHTTP(fastCtx *fasthttp.RequestCtx) {
	var middlewares []HandlerFunc
	for _, group := range engine.groups {
		if strings.HasPrefix(string(fastCtx.Path()), group.prefix) {
			middlewares = append(middlewares, group.middlewares...)
		}
	}
	c := newContext(fastCtx)
	c.handlers = middlewares
	c.engine = engine
	engine.router.handle(c)
}
