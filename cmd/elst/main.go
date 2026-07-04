package main

import (
	"github.com/gin-gonic/gin"
	"github.com/vprix/vncproxy/rfb"
	"github.com/vprix/vncproxy/security"
	"github.com/vprix/vncproxy/session"
	"github.com/vprix/vncproxy/vnc"
	"golang.org/x/net/websocket"
	"io"
	"log"
	"net"
	"os"
)

func main() {
	file, err := os.Open("test.log")
	if err != nil {
		return
	}
	log.SetOutput(file)
	//logger.SetLevel(glog.LEVEL_ERRO)
	route := gin.Default()
	route.GET("/ws", func(c *gin.Context) {
		handler := websocket.Handler(func(conn *websocket.Conn) {
			defer conn.Close()
			WSNoVNCProxy(conn)
		})
		handler.ServeHTTP(c.Writer, c.Request)
	})

	err = route.Run(":18098")
	if err != nil {
		return
	}
}

func WSNoVNCProxy(conn *websocket.Conn) {
	var err error
	conn.PayloadType = websocket.BinaryFrame
	targetCfg := rfb.TargetConfig{
		Host:     "192.168.1.20",
		Port:     5901,
		Password: []byte("105771"),
	}
	securityHandlers := []rfb.ISecurityHandler{
		&security.ServerAuthVNC{Password: []byte("123456")},
	}
	svrSess := session.NewServerSession(
		rfb.OptDesktopName([]byte("Vprix VNC Proxy")),
		rfb.OptHeight(768),
		rfb.OptWidth(1024),
		rfb.OptSecurityHandlers(securityHandlers...),
		rfb.OptGetConn(func(sess rfb.ISession) (io.ReadWriteCloser, error) {
			return conn, nil
		}),
	)
	cliSess := session.NewClient(
		rfb.OptSecurityHandlers([]rfb.ISecurityHandler{&security.ClientAuthVNC{Password: targetCfg.Password}}...),
		rfb.OptGetConn(func(sess rfb.ISession) (io.ReadWriteCloser, error) {
			return net.DialTimeout(targetCfg.GetNetwork(), targetCfg.Addr(), targetCfg.GetTimeout())
		}))
	p := vnc.NewVncProxy(cliSess, svrSess)
	err = p.Start()
	if err != nil {
		return
	}
}
