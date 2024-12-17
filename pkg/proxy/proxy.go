package proxy

import (
	"accumulation/pkg/log"
	"bytes"
	"context"
	"errors"
	"fmt"
	"go.opentelemetry.io/otel/propagation"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	oteltrace "go.opentelemetry.io/otel/trace"
)

type Context struct {
	context.Context
	req        *http.Request
	queryCache url.Values
	resp       *http.Response
}

func (pc *Context) Bind(binding Unmarshaler) error {
	reader, err := pc.req.GetBody()
	if err != nil {
		return err
	}
	body, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	return binding.Unmarshal(body)
}

func (pc *Context) BindFromResp(binding Unmarshaler) error {
	body, err := io.ReadAll(pc.resp.Body)
	if err != nil {
		return err
	}
	reader := bytes.NewReader(body)
	pc.resp.Body = io.NopCloser(reader)
	return binding.Unmarshal(body)
}
func (pc *Context) Query(key string, defaultValue string) (value string) {
	if pc.queryCache == nil {
		pc.queryCache = pc.req.URL.Query()
	}
	value = defaultValue
	values, ok := pc.queryCache[key]
	if ok {
		return values[0]
	}
	return
}

type Unmarshaler interface {
	Unmarshal(data []byte) error
}
type PostFunc func(ctx *Context) error

type ReverseProxy struct {
	tracer      oteltrace.Tracer
	postFuncs   map[string]PostFunc
	proxyPort   int32
	proxyHeader string
	portRoute   PortRoute
}

func NewReverseProxy(tracer oteltrace.Tracer) *ReverseProxy {
	return &ReverseProxy{tracer: tracer, postFuncs: make(map[string]PostFunc)}
}

func (p *ReverseProxy) SetPortRoute(portRoute PortRoute) {
	p.portRoute = portRoute
}
func (p *ReverseProxy) ResetProxyPort(proxyPort int32) {
	p.proxyPort = proxyPort
}

func (p *ReverseProxy) ResetProxyHeader(proxyHeader string) {
	p.proxyHeader = proxyHeader
}

func (p *ReverseProxy) RegisterPostHandler(path string, post PostFunc) {
	p.postFuncs[path] = post
}

func (p *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now().Unix()
	ctx := r.Context()
	// 从 Request 中获取 TCP 连接信息
	host := r.Host
	if tcpConn, ok := r.Context().Value(http.LocalAddrContextKey).(net.Addr); ok {
		tcpAddr := tcpConn.(*net.TCPAddr)
		host = fmt.Sprintf("%s:%d", tcpAddr.IP.String(), tcpAddr.Port)
		if tcpAddr.IP.IsLoopback() {
			host = fmt.Sprintf("localhost:%d", tcpAddr.Port)
		}
	}
	if p.tracer != nil {
		propagation.TraceContext{}.Inject(ctx, propagation.HeaderCarrier(r.Header))
		var span oteltrace.Span
		ctx, span = p.tracer.Start(ctx, "proxy")
		defer span.End()
	}

	r = r.WithContext(ctx)
	addr, err := getIPFromHost(host)
	if err != nil {
		handlerError(w, r, err)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		handlerError(w, r, err)
		return
	}
	log.Infof(ctx, "receive   request  path is  %s,%s ,RequestURI:%s,RemoteAddr:%s,req_body:[%v]", r.URL.Path, r.Host, r.RequestURI, r.RemoteAddr, string(body))
	r.GetBody = func() (io.ReadCloser, error) {
		reader := bytes.NewReader(body)
		return io.NopCloser(reader), nil
	}
	destPort := p.proxyPort
	if p.portRoute != nil {
		port := p.portRoute(ctx, r)
		if port > 0 {
			destPort = int32(port)
		}
	}
	proxy := httputil.NewSingleHostReverseProxy(&url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%d", addr, destPort),
	})
	proxy.Transport = p
	reader := bytes.NewReader(body)
	r.Body = io.NopCloser(reader)
	proxy.ServeHTTP(w, r)
	log.Infof(ctx, "path %s time cost: %d", r.URL.Path, time.Now().Unix()-start)
}

func (p *ReverseProxy) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := http.DefaultTransport.RoundTrip(req)
	ReportOtherRequestDuration(req.RequestURI, func() int {
		if resp == nil {
			return 500
		}
		return resp.StatusCode
	}(), start)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return resp, nil
	}
	handler, ok := p.postFuncs[req.URL.Path]
	if !ok {
		return resp, nil
	}
	err = handler(&Context{Context: req.Context(), req: req, resp: resp})
	if err != nil {
		log.Errorf(req.Context(), "RoundTrip Failed to handle request: %s: %+v", req.URL.String(), err)
		return nil, err
	}
	return resp, nil
}

func getIPFromHost(host string) (string, error) {
	host, _, err := net.SplitHostPort(host)
	if err != nil {

		return "", err
	}
	return host, nil

}
func handlerError(w http.ResponseWriter, r *http.Request, err error) {
	var statusCode int
	switch {
	case errors.Is(err, context.Canceled),
		err.Error() == "client disconnected":
		statusCode = 499
	case errors.Is(err, context.DeadlineExceeded):
		statusCode = 504
	default:
		log.Errorf(context.TODO(), "Failed to handle request: %s: %+v", r.URL.String(), err)
		statusCode = 502
	}
	w.WriteHeader(statusCode)
}

type PortRoute func(context context.Context, req *http.Request) int
