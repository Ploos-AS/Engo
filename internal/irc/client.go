package irc

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

type Config struct {
	Server   string
	Nick     string
	User     string
	RealName string
	TLS      bool
}

type Client struct {
	conn net.Conn
}

func Dial(cfg Config) (*Client, error) {
	if cfg.Server == "" {
		return nil, fmt.Errorf("server is required")
	}

	d := net.Dialer{Timeout: 15 * time.Second}
	var (
		conn net.Conn
		err  error
	)
	if cfg.TLS {
		host, _, splitErr := net.SplitHostPort(cfg.Server)
		if splitErr != nil {
			return nil, fmt.Errorf("invalid server address: %w", splitErr)
		}
		conn, err = tls.DialWithDialer(&d, "tcp", cfg.Server, &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12})
	} else {
		conn, err = d.Dial("tcp", cfg.Server)
	}
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	c := &Client{conn: conn}
	if err := c.writef("NICK %s", cfg.Nick); err != nil {
		conn.Close()
		return nil, err
	}
	if err := c.writef("USER %s 0 * :%s", cfg.User, cfg.RealName); err != nil {
		conn.Close()
		return nil, err
	}
	return c, nil
}

func (c *Client) Close() error { return c.conn.Close() }

func (c *Client) Run() error {
	r := bufio.NewReader(c.conn)
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("read IRC: %w", err)
		}
		line = strings.TrimRight(line, "\r\n")
		if strings.HasPrefix(line, "PING ") {
			if err := c.writef("PONG %s", strings.TrimPrefix(line, "PING ")); err != nil {
				return err
			}
		}
	}
}

func (c *Client) writef(format string, args ...any) error {
	if _, err := fmt.Fprintf(c.conn, format+"\r\n", args...); err != nil {
		return fmt.Errorf("write IRC: %w", err)
	}
	return nil
}
