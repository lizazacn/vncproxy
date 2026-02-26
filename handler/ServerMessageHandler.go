package handler

import (
	"encoding/binary"
	"fmt"
	"github.com/lizazacn/vncproxy/rfb"
	"log/slog"
)

// ServerMessageHandler vnc握手已结束，进入消息交互阶段
// 启动两个协程，
// 1. 处理proxy服务端的ServerMessage,在ServerMessageCh通道的消息都转发写入到该会话中.
// 2. 从会话中读取clientMessages，并判断是否支持该消息，如果支持则转发到ClientMessageCh通道中。如果不支持则关闭该会话并报错。
type ServerMessageHandler struct{}

func (*ServerMessageHandler) Handle(session rfb.ISession) error {
	slog.Debug("[VNC客户端->Proxy服务端]: vnc握手已结束，进入消息交互阶段[ServerMessageHandler]")

	cfg := session.Options()
	var err error
	clientMessages := make(map[rfb.ClientMessageType]rfb.Message)
	for _, m := range cfg.Messages {
		clientMessages[rfb.ClientMessageType(m.Type())] = m
	}

	// 处理proxy服务端发送给vnc客户端的消息
	go func() {
		//defer wg.Done()
		for {
			select {
			case <-session.Wait(): // 如果收到退出信号，则退出协程
				return
			case msg := <-cfg.Input:
				// 收到proxy服务端消息，则转发写入到vnc客户端会话中。
				slog.Debug("[Proxy服务端->VNC客户端] ", "消息类型", rfb.ServerMessageType(msg.Type()), "消息内容", msg.String())
				if err = msg.Write(session); err != nil {
					cfg.ErrorCh <- err
					_ = session.Close()
					return
				}
			}
		}
	}()

	// 处理vnc客户端发送给proxy服务端的消息
	go func() {
		for {
			select {
			case <-session.Wait():
				return
			default:
				// 从vnc客户端的会话中读取消息类型
				var messageType rfb.ClientMessageType
				if err = binary.Read(session, binary.BigEndian, &messageType); err != nil {
					cfg.ErrorCh <- fmt.Errorf("读取vnc客户端数据失败，err:%v", err)
					_ = session.Close()
					return
				}
				// 判断vnc客户端发送的消息类型proxy服务端是否支持。
				msg, ok := clientMessages[messageType]
				if !ok {
					cfg.ErrorCh <- fmt.Errorf("不支持的消息类型: %v", messageType)
					_ = session.Close()
					return
				}
				// 从会话中读取消息内容
				parsedMsg, e := msg.Read(session)
				if e != nil {
					cfg.ErrorCh <- fmt.Errorf("解析消息失败，err:%v", e)
					_ = session.Close()
					return
				}
				slog.Debug("[VNC客户端->Proxy服务端] ", "消息类型", rfb.ClientMessageType(msg.Type()), "消息内容", parsedMsg.String())
				cfg.Output <- parsedMsg
			}
		}
	}()
	return nil
}
