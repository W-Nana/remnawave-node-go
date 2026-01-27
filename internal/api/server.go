package api

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/klauspost/compress/zstd"

	"github.com/remnawave/node-go/internal/config"
	apperrors "github.com/remnawave/node-go/internal/errors"
	"github.com/remnawave/node-go/internal/logger"
)

type Server struct {
	config         *config.Config
	logger         *logger.Logger
	mainServer     *http.Server
	internalServer *http.Server
	mainRouter     *gin.Engine
	internalRouter *gin.Engine
}

func NewServer(cfg *config.Config, log *logger.Logger) (*Server, error) {
	gin.SetMode(gin.ReleaseMode)

	s := &Server{
		config: cfg,
		logger: log,
	}

	s.mainRouter = s.setupMainRouter()
	s.internalRouter = s.setupInternalRouter()

	tlsConfig, err := s.buildTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to build TLS config: %w", err)
	}

	s.mainServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.NodePort),
		Handler:      s.mainRouter,
		TLSConfig:    tlsConfig,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}

	s.internalServer = &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", cfg.InternalRestPort),
		Handler: s.internalRouter,
	}

	return s, nil
}

func (s *Server) buildTLSConfig() (*tls.Config, error) {
	cert, err := tls.X509KeyPair(
		[]byte(s.config.Payload.NodeCertPEM),
		[]byte(s.config.Payload.NodeKeyPEM),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load server certificate: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM([]byte(s.config.Payload.CACertPEM)) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caCertPool,
		MinVersion:   tls.VersionTLS12,
	}, nil
}

func (s *Server) setupMainRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(s.loggingMiddleware())
	router.Use(s.zstdMiddleware())

	router.NoRoute(s.notFoundHandler())

	return router
}

func (s *Server) setupInternalRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(s.loggingMiddleware())

	internalPrefixes := []string{"/internal/get-config", "/vision/block-ip", "/vision/unblock-ip"}

	router.Use(func(c *gin.Context) {
		path := c.Request.URL.Path
		matched := false
		for _, prefix := range internalPrefixes {
			if strings.HasPrefix(path, prefix) {
				matched = true
				break
			}
		}

		if matched {
			c.Set("forwarded", true)
			c.Next()
		} else {
			c.String(404, "Cannot %s %s", c.Request.Method, c.Request.URL.Path)
			c.Abort()
		}
	})

	router.NoRoute(func(c *gin.Context) {
		if c.GetBool("forwarded") {
			destroySocket(c)
		} else {
			c.String(404, "Cannot %s %s", c.Request.Method, c.Request.URL.Path)
		}
	})

	return router
}

func (s *Server) MainRouter() *gin.Engine {
	return s.mainRouter
}

func (s *Server) InternalRouter() *gin.Engine {
	return s.internalRouter
}

func (s *Server) loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

func (s *Server) zstdMiddleware() gin.HandlerFunc {
	decoder, _ := zstd.NewReader(nil)

	return func(c *gin.Context) {
		if c.GetHeader("Content-Encoding") == "zstd" {
			body, err := io.ReadAll(c.Request.Body)
			if err != nil {
				c.AbortWithStatus(400)
				return
			}
			decompressed, err := decoder.DecodeAll(body, nil)
			if err != nil {
				c.AbortWithStatus(400)
				return
			}
			c.Request.Body = io.NopCloser(bytes.NewReader(decompressed))
			c.Request.Header.Del("Content-Encoding")
			c.Request.ContentLength = int64(len(decompressed))
		}
		c.Next()
	}
}

func (s *Server) notFoundHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		destroySocket(c)
	}
}

func (s *Server) Start() error {
	errCh := make(chan error, 2)

	go func() {
		s.logger.Info(fmt.Sprintf("Starting main HTTPS server on :%d", s.config.NodePort))
		if err := s.mainServer.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("main server error: %w", err)
		}
	}()

	go func() {
		s.logger.Info(fmt.Sprintf("Starting internal HTTP server on 127.0.0.1:%d", s.config.InternalRestPort))
		if err := s.internalServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("internal server error: %w", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	default:
		return nil
	}
}

func (s *Server) Stop() error {
	if err := s.mainServer.Close(); err != nil {
		return err
	}
	if err := s.internalServer.Close(); err != nil {
		return err
	}
	return nil
}

func destroySocket(c *gin.Context) {
	defer func() {
		recover()
		c.Abort()
	}()

	hijacker, ok := c.Writer.(http.Hijacker)
	if !ok {
		return
	}
	conn, _, err := hijacker.Hijack()
	if err != nil {
		return
	}
	conn.Close()
}

func PortGuardMiddleware(expectedPort int) gin.HandlerFunc {
	return func(c *gin.Context) {
		localAddr := c.Request.Context().Value(http.LocalAddrContextKey)
		if localAddr == nil {
			destroySocket(c)
			return
		}

		tcpAddr, ok := localAddr.(*net.TCPAddr)
		if !ok {
			destroySocket(c)
			return
		}

		if tcpAddr.Port != expectedPort || tcpAddr.IP.String() != "127.0.0.1" {
			destroySocket(c)
			return
		}

		c.Next()
	}
}

func ErrorHandler(code string, c *gin.Context) {
	errDef, ok := apperrors.GetError(code)
	if !ok {
		errDef = apperrors.ERRORS[apperrors.CodeInternalServerError]
	}

	c.JSON(errDef.HTTPCode, NewErrorResponse(
		c.Request.URL.Path,
		errDef.Message,
		errDef.Code,
	))
}
