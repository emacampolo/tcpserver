package tcpserver

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"net"
)

type newLineEncodeDecoder struct{}

func (n *newLineEncodeDecoder) Encode(w io.Writer, p []byte) error {
	_, err := w.Write(p)
	return err
}

func (n *newLineEncodeDecoder) Decode(r io.Reader) ([]byte, error) {
	return bufio.NewReader(r).ReadBytes('\n')
}

func echo(ctx context.Context, message []byte) ([]byte, error) {
	return message, nil
}

func defaultConfig(config ...Config) Config {
	var cfg Config
	if len(config) > 0 {
		cfg = config[0]
	}

	if cfg.Address == "" {
		cfg.Address = "127.0.0.1:0"
	}

	if cfg.Handler == nil {
		cfg.Handler = echo
	}

	if cfg.Decoder == nil {
		cfg.Decoder = &newLineEncodeDecoder{}
	}

	if cfg.Encoder == nil {
		cfg.Encoder = &newLineEncodeDecoder{}
	}

	if cfg.ListenerAddrFunc == nil {
		cfg.ListenerAddrFunc = func(addr net.Addr) {
			slog.Info("server listening", "addr", addr.String())
		}
	}

	return cfg
}
