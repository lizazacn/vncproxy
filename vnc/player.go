package vnc

import (
	"encoding/binary"
	"fmt"
	"github.com/lizazacn/vncproxy/handler"
	"github.com/lizazacn/vncproxy/messages"
	"github.com/lizazacn/vncproxy/rfb"
	"github.com/lizazacn/vncproxy/session"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"
)

type Player struct {
	svrSession    *session.ServerSession // vnc客户端连接到proxy的会话
	playerSession *session.PlayerSession
	errorCh       chan error
	closed        bool
	syncOnce      sync.Once
}

func NewPlayer(filePath string, svrSession *session.ServerSession) *Player {
	playerSession := session.NewPlayerSession(
		rfb.OptGetConn(func(sess rfb.ISession) (io.ReadWriteCloser, error) {
			if !FileExists(filePath) {
				return nil, fmt.Errorf("要读取的文件[%s]不存在", filePath)
			}
			return os.OpenFile(filePath, os.O_RDONLY, 0644)
		}),
	)

	return &Player{
		errorCh:       make(chan error, 32),
		svrSession:    svrSession,
		playerSession: playerSession,
		closed:        false,
	}
}

// Start 启动
func (that *Player) Start() error {

	err := that.svrSession.Init(rfb.OptHandlers([]rfb.IHandler{
		&handler.ServerVersionHandler{},
		&handler.ServerSecurityHandler{},
		that, // 把链接到vnc服务端的逻辑加入
		&handler.ServerClientInitHandler{},
		&handler.ServerServerInitHandler{},
		&handler.ServerMessageHandler{},
	}...))
	if err != nil {
		return err
	}

	that.svrSession.Start()
	err = <-that.errorCh
	return err
}

// Handle 建立远程链接
func (that *Player) Handle(sess rfb.ISession) error {
	that.playerSession.Start()
	that.svrSession = sess.(*session.ServerSession)
	that.svrSession.SetWidth(that.playerSession.Options().Width)
	that.svrSession.SetHeight(that.playerSession.Options().Height)
	that.svrSession.SetDesktopName(that.playerSession.Options().DesktopName)
	that.svrSession.SetPixelFormat(that.playerSession.Options().PixelFormat)

	go that.handleIO()
	return nil
}

func (that *Player) handleIO() {
	for that.closed == false {
		select {
		case <-that.svrSession.Wait():
			return
		case <-that.playerSession.Wait():
			return
		case err := <-that.svrSession.Options().ErrorCh:
			that.errorCh <- err
			that.Close()
		case err := <-that.playerSession.Options().ErrorCh:
			that.errorCh <- err
			that.Close()
		case msg := <-that.svrSession.Options().Output:
			slog.Debug("收到vnc客户端发送过来的消息", "msg", msg)
			if msg.Type() == rfb.MessageType(rfb.FramebufferUpdateRequest) {
				that.syncOnce.Do(func() {
					go that.readRbs()
				})
			}
		}
	}
}

func (that *Player) readRbs() {
	for that.closed == false {
		// 从会话中读取消息类型
		var messageType rfb.ServerMessageType
		if err := binary.Read(that.playerSession, binary.BigEndian, &messageType); err != nil {
			that.playerSession.Options().ErrorCh <- err
			return
		}
		msg := &messages.FramebufferUpdate{}
		// 读取消息内容
		parsedMsg, err := msg.Read(that.playerSession)
		if err != nil {
			that.playerSession.Options().ErrorCh <- err
			return
		}
		that.svrSession.Options().Input <- parsedMsg
		var sleep int64
		_ = binary.Read(that.playerSession, binary.BigEndian, &sleep)
		if sleep > 0 {
			time.Sleep(time.Duration(sleep))
		}
	}
}

func (that *Player) Close() {
	that.closed = true
	_ = that.svrSession.Close()
	_ = that.playerSession.Close()
}

func FileExists(path string) bool {
	// Exists checks whether given `path` exist.
	if stat, err := os.Stat(path); stat != nil && !os.IsNotExist(err) {
		return true
	}
	return false
}
