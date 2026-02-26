package vnc

import (
	"encoding/binary"
	"github.com/lizazacn/vncproxy/messages"
	"github.com/lizazacn/vncproxy/rfb"
	"github.com/lizazacn/vncproxy/session"
	"log/slog"
	"time"
)

type Recorder struct {
	errorCh         chan error
	closed          bool
	cliSession      *session.ClientSession // 链接到vnc服务端的会话
	recorderSession *session.RecorderSession
}

func NewRecorder(recorderSess *session.RecorderSession, cliSession *session.ClientSession) *Recorder {
	recorder := &Recorder{
		recorderSession: recorderSess,
		cliSession:      cliSession,
		errorCh:         make(chan error, 32),
		closed:          false,
	}
	return recorder
}

func (that *Recorder) Start() error {
	var err error
	that.cliSession.Start()
	encS := []rfb.EncodingType{
		rfb.EncCursorPseudo,
		rfb.EncPointerPosPseudo,
		rfb.EncCopyRect,
		rfb.EncZRLE,
		rfb.EncHexTile,
		rfb.EncZlib,
		rfb.EncRRE,
	}
	err = that.cliSession.SetEncodings(encS)
	if err != nil {
		return err
	}
	// 设置参数信息
	that.recorderSession.SetProtocolVersion(that.cliSession.ProtocolVersion())
	that.recorderSession.SetWidth(that.cliSession.Options().Width)
	that.recorderSession.SetHeight(that.cliSession.Options().Height)
	that.recorderSession.SetPixelFormat(that.cliSession.Options().PixelFormat)
	that.recorderSession.SetDesktopName(that.cliSession.Options().DesktopName)
	that.recorderSession.Start()
	reqMsg := messages.FramebufferUpdateRequest{Inc: 1, X: 0, Y: 0, Width: that.cliSession.Options().Width, Height: that.cliSession.Options().Height}
	err = reqMsg.Write(that.cliSession)
	if err != nil {
		return err
	}
	var lastUpdate *time.Time
	for {
		select {
		case msg := <-that.recorderSession.Options().Output:
			slog.Debug("client message ", "received.msgType", msg.Type(), ",msg", msg)
		case msg := <-that.cliSession.Options().Output:
			if rfb.ServerMessageType(msg.Type()) == rfb.FramebufferUpdate {
				err = msg.Write(that.recorderSession)
				if err != nil {
					return err
				}
				if lastUpdate == nil {
					_ = binary.Write(that.recorderSession, binary.BigEndian, int64(0))
				} else {
					secsPassed := time.Now().UnixNano() - lastUpdate.UnixNano()
					_ = binary.Write(that.recorderSession, binary.BigEndian, secsPassed)
				}
				err = that.recorderSession.Flush()
				if err != nil {
					return err
				}
				t := time.Now()
				lastUpdate = &t
				reqMsg = messages.FramebufferUpdateRequest{Inc: 1, X: 0, Y: 0, Width: that.cliSession.Options().Width, Height: that.cliSession.Options().Height}
				err = reqMsg.Write(that.cliSession)
				if err != nil {
					return err
				}
			}
		case <-that.cliSession.Wait():
			return nil
		case <-that.recorderSession.Wait():
			return nil
		case err = <-that.cliSession.Options().ErrorCh:
			that.errorCh <- err
			that.Close()
		case err = <-that.recorderSession.Options().ErrorCh:
			that.errorCh <- err
			that.Close()
		case err = <-that.errorCh:
			return err
		}
	}
}

func (that *Recorder) Close() {
	that.closed = true
	_ = that.cliSession.Close()
	_ = that.recorderSession.Close()
}
