package main

import (
	"net"
	"log"
)

func StatsDaemon(addr string, ch chan *Message) (error) {
	listener, err := net.ListenPacket("udp", addr)
	if err != nil {
		return err
	}

	buf := make([]byte, 8192)

	for {
		n, addr, err := listener.ReadFrom(buf)
		if err != nil {
			log.Printf("Can't read from socket: %v", err)
			continue
		}
		msgs, err := ParseMessages(buf[:n])
		if err != nil {
			log.Printf("Parsing error: %v", err)
			continue
		}

		for _, msg := range(msgs) {
			msg.Tags = append(msg.Tags, "addr=" + addr.String())
			ch <- msg
		}
	}

	return nil
}

