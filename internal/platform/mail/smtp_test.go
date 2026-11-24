package mail

import (
	"bufio"
	"context"
	"net"
	"strings"
	"testing"
	"time"
)

// fakeServer speaks just enough SMTP to accept one message and hands back what the client sent in DATA.
func fakeServer(t *testing.T) (addr string, got <-chan string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	out := make(chan string, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		r := bufio.NewReader(conn)
		say := func(s string) { _, _ = conn.Write([]byte(s + "\r\n")) }
		say("220 fake ESMTP")
		var data strings.Builder
		inData := false
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimRight(line, "\r\n")
			switch {
			case inData && line == ".":
				inData = false
				out <- data.String()
				say("250 queued")
			case inData:
				data.WriteString(line + "\n")
			case strings.HasPrefix(line, "EHLO"), strings.HasPrefix(line, "HELO"):
				say("250 fake")
			case strings.HasPrefix(line, "DATA"):
				inData = true
				say("354 go ahead")
			case strings.HasPrefix(line, "QUIT"):
				say("221 bye")
				return
			default: // MAIL FROM, RCPT TO
				say("250 ok")
			}
		}
	}()
	return ln.Addr().String(), out
}

func TestSendOrderConfirmation(t *testing.T) {
	addr, got := fakeServer(t)
	host, port, _ := net.SplitHostPort(addr)
	m := NewSMTP(Config{Host: host, Port: port})
	if err := m.SendOrderConfirmation(context.Background(), 42, 5990); err != nil {
		t.Fatalf("send: %v", err)
	}
	select {
	case msg := <-got:
		for _, want := range []string{"Subject: Order 42 confirmed", "Order number: 42", "NT$5990", "To: customer@go-shop.test"} {
			if !strings.Contains(msg, want) {
				t.Errorf("message lacks %q:\n%s", want, msg)
			}
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server never received a message")
	}
}

func TestSendFailsWhenServerIsDown(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	host, port, _ := net.SplitHostPort(ln.Addr().String())
	_ = ln.Close() // nobody listens any more
	m := NewSMTP(Config{Host: host, Port: port})
	if err := m.SendOrderConfirmation(context.Background(), 1, 1); err == nil {
		t.Fatal("want an error so the outbox retries, got nil")
	}
}
