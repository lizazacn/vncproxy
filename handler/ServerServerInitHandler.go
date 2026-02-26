package handler

import (
	"encoding/binary"
	"github.com/lizazacn/vncproxy/rfb"
	"log/slog"
)

// ServerServerInitHandler vnc握手步骤第四步
// 1. 发送proxy服务端的参数信息，屏幕宽高，像素格式，桌面名称
type ServerServerInitHandler struct{}

func (*ServerServerInitHandler) Handle(session rfb.ISession) error {
	slog.Debug("[Proxy服务端->VNC客户端]: 执行vnc握手第四步:[ServerInit]")
	if err := binary.Write(session, binary.BigEndian, session.Options().Width); err != nil {
		return err
	}
	if err := binary.Write(session, binary.BigEndian, session.Options().Height); err != nil {
		return err
	}
	if err := binary.Write(session, binary.BigEndian, session.Options().PixelFormat); err != nil {
		return err
	}
	desktopName := session.Options().DesktopName
	size := uint32(len(session.Options().DesktopName))
	if size == 0 {
		desktopName = []byte("vprix")
		size = uint32(len(desktopName))
	}
	if err := binary.Write(session, binary.BigEndian, size); err != nil {
		return err
	}
	if err := binary.Write(session, binary.BigEndian, desktopName); err != nil {
		return err
	}
	slog.Debug("[Proxy服务端->VNC客户端]", slog.Group("ServerInit",
		"Width", session.Options().Width,
		"Height", session.Options().Height,
		"PixelFormat", session.Options().PixelFormat,
		"DesktopName", desktopName))
	return session.Flush()
}
