package tcpserver

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
)

// Decoder is responsible for decoding the stream-based message into a meaningful object.
type Decoder interface {
	Decode(reader io.Reader) ([]byte, error)
}

// Encoder is responsible for encoding the Handler message back to the client.
type Encoder interface {
	Encode(writer io.Writer, p []byte) error
}

// The Handler type allows clients to process incoming tcp connections.
// The provided context is canceled on Shutdown.
type Handler func(ctx context.Context, message []byte) ([]byte, error)

// Server is a TCP server that listens on a TCP network address and
// invokes the Handler for each incoming connection.
type Server struct {
	address          string
	handler          Handler
	decoder          Decoder
	encoder          Encoder
	listenerAddrFunc func(addr net.Addr)

	wg        sync.WaitGroup
	isClosing atomic.Bool

	mux      sync.Mutex
	listener net.Listener

	ctx       context.Context
	ctxCancel context.CancelFunc
}

// Config is the configuration of the server. If a field is not set, a default value is used.
type Config struct {
	// Address is the TCP address to listen on.
	// By default, it listens on a random port on localhost.
	Address string

	// Handler to invoke. If nil, the server echoes the message back to the client.
	Handler Handler

	// Decoder used to decode incoming messages that are forwarded to the Handler.
	// If nil, it will use a new line decoder.
	Decoder Decoder

	// Encoder used to encode the message back to the client.
	// If nil, it will write the message as is.
	Encoder Encoder

	// ListenerFunc allows accessing the listener before the server starts serving.
	// It is called synchronously.
	// By default, it logs the listener address.
	ListenerAddrFunc func(addr net.Addr)
}

// New creates a new Server with the given config.
func New(config ...Config) *Server {
	cfg := defaultConfig(config...)
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		address:          cfg.Address,
		handler:          cfg.Handler,
		decoder:          cfg.Decoder,
		encoder:          cfg.Encoder,
		listenerAddrFunc: cfg.ListenerAddrFunc,

		ctx:       ctx,
		ctxCancel: cancel,
	}
}

// Serve starts the server and blocks until the server is closed.
func (s *Server) Serve() error {
	if s.isClosing.Load() {
		return errors.New("server is already closing")
	}

	s.mux.Lock()
	if s.listener != nil {
		s.mux.Unlock()
		return errors.New("server is already running")
	}

	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		s.mux.Unlock()
		return err
	}

	s.listener = listener

	if s.listenerAddrFunc != nil {
		s.listenerAddrFunc(listener.Addr())
	}

	s.mux.Unlock()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if s.isClosing.Load() {
				slog.Info("server is closing")
				return nil
			}

			return err
		}

		s.wg.Add(1)
		go func(c net.Conn) {
			defer s.wg.Done()

			s.serve(c)
		}(conn)
	}
}

// Addr returns the net.Addr used by the server or nil if the server is not running.
func (s *Server) Addr() net.Addr {
	if s.isClosing.Load() {
		return nil
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	if s.listener == nil {
		return nil
	}

	return s.listener.Addr()
}

func (s *Server) serve(conn net.Conn) {
	defer func() {
		if err := conn.Close(); err != nil {
			slog.Error("failed to close connection", "error", err)
		}
	}()

	message, err := s.decoder.Decode(conn)
	if err != nil {
		if errors.Is(err, io.EOF) {
			slog.Info("connection closed by client")
		} else {
			slog.Error("failed to decode message", "error", err)
		}

		return
	}

	response, err := s.handler(s.ctx, message)
	if err != nil {
		slog.Error("failed to process message", "error", err)
		return
	}

	if err := s.encoder.Encode(conn, response); err != nil {
		slog.Error("failed to encode message", "error", err)
		return
	}
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown() {
	if s.isClosing.Swap(true) {
		return
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	if s.listener == nil {
		return
	}

	if err := s.listener.Close(); err != nil {
		slog.Error("failed to close listener", "error", err)
	}

	s.ctxCancel()
	s.wg.Wait()
}
