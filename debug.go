package rex

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/juju/loggo"
	"github.com/mailgun/manners"
)

func Inspect(v interface{}) string {
	bytes, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%#v", v)
	}
	return string(bytes)
}

func StartDebugServer(port int) *manners.GracefulServer {
	log := loggo.GetLogger("rex.debug")
	r := gin.Default()

	r.GET("/loggo", getLoggoSpec)
	r.POST("/loggo", setLoggoSpec)

	r.GET("/blockprof/:rate", func(c *gin.Context) {
		r, err := strconv.Atoi(c.Param("rate"))
		if err != nil {
			c.String(http.StatusOK, "rate invalid %s. %v", c.Param("rate"), err)
			return
		}
		runtime.SetBlockProfileRate(r)
		c.String(http.StatusOK, "new rate %d", r)
	})

	http.Handle("/", r)
	server := manners.NewWithServer(&http.Server{Handler: http.DefaultServeMux})

	log.Infof("starting debug server on port=%d", port)
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	MayPanic(err)

	go func() {
		MayPanic(server.Serve(listener))
	}()

	return server
}

func getLoggoSpec(c *gin.Context) {
	c.String(200, loggo.LoggerInfo())
}

func setLoggoSpec(c *gin.Context) {
	log := loggo.GetLogger("rex.debug")

	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	log.Infof("setting new loggo spec: %s", string(body))
	err = loggo.ConfigureLoggers(string(body))
	if err != nil {
		c.AbortWithError(400, err)
		return
	}

	c.String(200, loggo.LoggerInfo())
}
