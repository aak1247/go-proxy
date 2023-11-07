package main

import (
	"io"
	"log"
	"net"
	"strconv"
	"time"
)

func NewTcpProxy(url string, localPort int) {
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(localPort))
	defer listener.Close()
	if err != nil {
		log.Fatal(err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go tcpProxy(url, conn)
	}
}

func tcpProxy(url string, client net.Conn) error {
	cs := make(chan bool)
	ss := make(chan bool)

	// 连接远程服务器
	server, err := net.Dial("tcp", url)
	if err != nil {
		log.Printf("dial error %v\n", err)
		return err
	}

	// 启动上行goroutine
	go func() {
		defer client.Close()
		if err := client.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
			log.Println("Failed to set deadline:", err)
			return
		}

		buf := make([]byte, 1024*256)
		for {
			select {
			case <-cs:
				break
			default:
				n, err := client.Read(buf)
				if err != nil {
					if err != io.EOF {
						log.Printf("Failed to read from client: %v, close connection\n", err)
						cs <- true
						ss <- true
						return
					}
				}
				if n > 0 {
					log.Printf("received from client bytes %d\n", n)
					log.Printf("Received data from client %s: %s\n", client.RemoteAddr().String(), string(buf[:n]))

					// write to server
					if n, err := server.Write(buf); err != nil {
						log.Println("Failed to write to server:\n", err)
					} else {
						log.Printf("write to server %d\n", n)
					}
					// 清空缓冲区
					buf = make([]byte, 1024*256)
				}
			}
		}
	}()

	// 启动下行goroutine
	go func() {
		defer server.Close()
		if err := server.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
			log.Println("Failed to set deadline:\n", err)
			return
		}
		buf := make([]byte, 1024*256)
		for {
			select {
			case <-ss:
				break
			default:
				n, err := server.Read(buf)
				if err != nil {
					if err != io.EOF {
						log.Printf("Failed to read from server: %v, close connection\n", err)
						ss <- true
						cs <- true
						return
					}
				}
				if n > 0 {
					log.Printf("received from server bytes %d\n", n)
					log.Printf("Received data from server %s: %s\n", server.RemoteAddr().String(), string(buf[:n]))

					// write to client
					if n, err := client.Write(buf); err != nil {
						log.Println("Failed to write to client:", err)
					} else {
						log.Printf("write to client %d", n)
					}
					// 清空缓冲区
					buf = make([]byte, 1024*256)
				}
			}
		}
	}()
	return nil
}
