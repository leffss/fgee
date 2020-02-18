package main

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"os/signal"
	"time"

	"fgee/fgee"
	"github.com/valyala/fasthttp"
)

func formatAsDate(t time.Time) string {
	year, month, day := t.Date()
	return fmt.Sprintf("%d-%02d-%02d", year, month, day)
}

type student struct {
	Name string
	Age  int8
}

func addServerName(v string) fgee.HandlerFunc {
	log.Printf("Init addServerName Middleware")
	return func(c *fgee.Context) {
		c.SetHeader("Server", v)
		c.Next()
	}
}

func main() {
	listen := ":9999"
	fgee.SetReadTimeout(5)
	fgee.SetWriteTimeout(5)
	serverName := "BWS"

	r := fgee.Default()

	r.Use(addServerName(serverName))

	r.SetFuncMap(template.FuncMap{
		"formatAsDate": formatAsDate,
	})
	r.LoadHTMLGlob("templates/*")
	r.Static("/assets/", "./static")

	g := r.Group("/g")
	{
		g.Static("/assets", "./static")
		g1 := g.Group("/g1")
		g1.Static("/test", "./static")
	}

	r.GET("/", func(c *fgee.Context) {
		test := c.Query("test")
		//c.String(http.StatusOK, "Hello Geektutu\n")
		c.JSON(fasthttp.StatusOK, fgee.H{
			"username": "leffss",
			"passwd": "123456",
			"test": test,
		})
	})

	r.GET("/t1/id", func(c *fgee.Context) {
		c.String(fasthttp.StatusOK, "id static")
	})

	r.GET("/t1/:id", func(c *fgee.Context) {
		id := c.Param("id")
		c.String(fasthttp.StatusOK, "id: %s :", id)
	})

	r.GET("/test1/:name/test2", func(c *fgee.Context) {
		x := c.Param("name")
		c.String(fasthttp.StatusOK, x)
	})

	r.GET("/test2/*name", func(c *fgee.Context) {
		x := c.Param("name")
		c.String(fasthttp.StatusOK, x)
	})

	r.GET("/re1/{id:\\d+}", func(c *fgee.Context) {
		id:= c.Param("id")
		c.String(fasthttp.StatusOK, "re1 id: %s", id)
	})

	r.GET("/re2/{id:[a-z]+}", func(c *fgee.Context) {
		id:= c.Param("id")
		c.String(fasthttp.StatusOK, "re2 id: %s", id)
	})

	r.GET("/re3/{year:[12][0-9]{3}}/{month:[1-9]{2}}/{day:[1-9]{2}}/{hour:(12|[3-9])}", func(c *fgee.Context) {
		year := c.Param("year")
		month := c.Param("month")
		day := c.Param("day")
		hour := c.Param("hour")
		c.String(fasthttp.StatusOK, "re3 year: %s, month: %s, day: %s, hour: %s", year, month, day, hour)
	})

	r.GET("/re2/{id:[a-z]+}/test", func(c *fgee.Context) {
		id:= c.Param("id")
		c.String(fasthttp.StatusOK, "re2 id: %s test", id)
	})

	// index out of range for testing Recovery()
	r.Any("/panic", func(c *fgee.Context) {
		names := []string{"geektutu"}
		c.String(fasthttp.StatusOK, names[100])
	})

	r.POST("/post", func(c *fgee.Context) {
		name := c.PostJson()
		c.String(fasthttp.StatusOK, name)
	})

	stu1 := &student{Name: "Geektutu", Age: 20}
	stu2 := &student{Name: "Jack", Age: 22}

	r.GET("/students", func(c *fgee.Context) {
		c.HTML(fasthttp.StatusOK, "arr.tmpl", fgee.H{
			"title":  "gee",
			"stuArr": [2]*student{stu1, stu2},
		})
	})

	r.GET("/date", func(c *fgee.Context) {
		c.HTML(fasthttp.StatusOK, "custom_func.tmpl", fgee.H{
			"title": "gee",
			"now":   time.Date(2019, 8, 17, 0, 0, 0, 0, time.UTC),
		})
	})

	go func() {
		log.Println("Server Start @", listen)
		//if err := r.Run(":9999"); err != nil {
		//	log.Fatalf("Server Start Error: %s\n", err)
		//}
		if err := r.RunTLS(":9999", "./server.crt", "./server.key"); err != nil {
			log.Fatalf("Server Start Error: %s\n", err)
		}
	}()

	// 等待中断信号以优雅地关闭服务器
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	<-quit
	if err := r.Shutdown(); err != nil {
		log.Fatal("Server Shutdown Error:", err)
	}
	log.Println("Server Shutdown")
}
