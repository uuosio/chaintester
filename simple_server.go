package chaintester

import (
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/apache/thrift/lib/go/thrift"
)

// ErrAbandonRequest is a special error server handler implementations can
// return to indicate that the request has been abandoned.
//
// SimpleIPCServer will check for this error, and close the client connection
// instead of writing the response/error back to the client.
//
// It shall only be used when the server handler implementation know that the
// client already abandoned the request (by checking that the passed in context
// is already canceled, for example).
var ErrAbandonRequest = errors.New("request abandoned")

// ServerConnectivityCheckInterval defines the ticker interval used by
// connectivity check in thrift compiled TProcessorFunc implementations.
//
// It's defined as a variable instead of constant, so that thrift server
// implementations can change its value to control the behavior.
//
// If it's changed to <=0, the feature will be disabled.
var ServerConnectivityCheckInterval = time.Millisecond * 5

/*
 * This is not a typical SimpleIPCServer as it is not blocked after accept a socket.
 * It is more like a TThreadedServer that can handle different connections in different goroutines.
 * This will work if golang user implements a conn-pool like thing in client side.
 */
type SimpleIPCServer struct {
	closed int32
	wg     sync.WaitGroup
	mu     sync.Mutex

	processorFactory       thrift.TProcessorFactory
	serverTransport        thrift.TServerTransport
	inputTransportFactory  thrift.TTransportFactory
	outputTransportFactory thrift.TTransportFactory
	inputProtocolFactory   thrift.TProtocolFactory
	outputProtocolFactory  thrift.TProtocolFactory

	// Headers to auto forward in THeaderProtocol
	forwardHeaders []string
	logger         thrift.Logger
	client         thrift.TTransport
	end_loop       bool

	inputTransport, outputTransport thrift.TTransport
	inputProtocol, outputProtocol   thrift.TProtocol
	headerProtocol                  *thrift.THeaderProtocol
}

func NewSimpleIPCServer2(processor thrift.TProcessor, serverTransport thrift.TServerTransport) *SimpleIPCServer {
	return NewSimpleIPCServerFactory2(thrift.NewTProcessorFactory(processor), serverTransport)
}

func NewSimpleIPCServer4(processor thrift.TProcessor, serverTransport thrift.TServerTransport, transportFactory thrift.TTransportFactory, protocolFactory thrift.TProtocolFactory) *SimpleIPCServer {
	return NewSimpleIPCServerFactory4(thrift.NewTProcessorFactory(processor),
		serverTransport,
		transportFactory,
		protocolFactory,
	)
}

func NewSimpleIPCServer6(processor thrift.TProcessor, serverTransport thrift.TServerTransport, inputTransportFactory thrift.TTransportFactory, outputTransportFactory thrift.TTransportFactory, inputProtocolFactory thrift.TProtocolFactory, outputProtocolFactory thrift.TProtocolFactory) *SimpleIPCServer {
	return NewSimpleIPCServerFactory6(thrift.NewTProcessorFactory(processor),
		serverTransport,
		inputTransportFactory,
		outputTransportFactory,
		inputProtocolFactory,
		outputProtocolFactory,
	)
}

func NewSimpleIPCServerFactory2(processorFactory thrift.TProcessorFactory, serverTransport thrift.TServerTransport) *SimpleIPCServer {
	return NewSimpleIPCServerFactory6(processorFactory,
		serverTransport,
		thrift.NewTTransportFactory(),
		thrift.NewTTransportFactory(),
		thrift.NewTBinaryProtocolFactoryDefault(),
		thrift.NewTBinaryProtocolFactoryDefault(),
	)
}

func NewSimpleIPCServerFactory4(processorFactory thrift.TProcessorFactory, serverTransport thrift.TServerTransport, transportFactory thrift.TTransportFactory, protocolFactory thrift.TProtocolFactory) *SimpleIPCServer {
	return NewSimpleIPCServerFactory6(processorFactory,
		serverTransport,
		transportFactory,
		transportFactory,
		protocolFactory,
		protocolFactory,
	)
}

func NewSimpleIPCServerFactory6(processorFactory thrift.TProcessorFactory, serverTransport thrift.TServerTransport, inputTransportFactory thrift.TTransportFactory, outputTransportFactory thrift.TTransportFactory, inputProtocolFactory thrift.TProtocolFactory, outputProtocolFactory thrift.TProtocolFactory) *SimpleIPCServer {
	return &SimpleIPCServer{
		processorFactory:       processorFactory,
		serverTransport:        serverTransport,
		inputTransportFactory:  inputTransportFactory,
		outputTransportFactory: outputTransportFactory,
		inputProtocolFactory:   inputProtocolFactory,
		outputProtocolFactory:  outputProtocolFactory,
	}
}

func (p *SimpleIPCServer) ProcessorFactory() thrift.TProcessorFactory {
	return p.processorFactory
}

func (p *SimpleIPCServer) ServerTransport() thrift.TServerTransport {
	return p.serverTransport
}

func (p *SimpleIPCServer) InputTransportFactory() thrift.TTransportFactory {
	return p.inputTransportFactory
}

func (p *SimpleIPCServer) OutputTransportFactory() thrift.TTransportFactory {
	return p.outputTransportFactory
}

func (p *SimpleIPCServer) InputProtocolFactory() thrift.TProtocolFactory {
	return p.inputProtocolFactory
}

func (p *SimpleIPCServer) OutputProtocolFactory() thrift.TProtocolFactory {
	return p.outputProtocolFactory
}

func (p *SimpleIPCServer) Listen() error {
	return p.serverTransport.Listen()
}

// SetForwardHeaders sets the list of header keys that will be auto forwarded
// while using THeaderProtocol.
//
// "forward" means that when the server is also a client to other upstream
// thrift servers, the context object user gets in the processor functions will
// have both read and write headers set, with write headers being forwarded.
// Users can always override the write headers by calling SetWriteHeaderList
// before calling thrift client functions.
func (p *SimpleIPCServer) SetForwardHeaders(headers []string) {
	size := len(headers)
	if size == 0 {
		p.forwardHeaders = nil
		return
	}

	keys := make([]string, size)
	copy(keys, headers)
	p.forwardHeaders = keys
}

// SetLogger sets the logger used by this SimpleIPCServer.
//
// If no logger was set before Serve is called, a default logger using standard
// log library will be used.
func (p *SimpleIPCServer) SetLogger(logger thrift.Logger) {
	p.logger = logger
}

func (p *SimpleIPCServer) innerAccept() (int32, error) {
	client, err := p.serverTransport.Accept()
	p.mu.Lock()
	defer p.mu.Unlock()
	closed := atomic.LoadInt32(&p.closed)
	if closed != 0 {
		return closed, nil
	}
	if err != nil {
		return 0, err
	}
	if client != nil {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			if err := p.processRequests(client); err != nil {
				p.logger(fmt.Sprintf("error processing request: %v", err))
			}
		}()
	}
	return 0, nil
}

func (p *SimpleIPCServer) AcceptOnce() (int32, error) {
	client, err := p.serverTransport.Accept()
	p.mu.Lock()
	defer p.mu.Unlock()
	closed := atomic.LoadInt32(&p.closed)
	if closed != 0 {
		return closed, nil
	}

	if err != nil {
		return 0, err
	}
	fmt.Println("+++++++++SimpleIPCServer: new client connected")
	p.client = client

	p.inputTransport, err = p.inputTransportFactory.GetTransport(client)
	if err != nil {
		return 0, err
	}
	p.inputProtocol = p.inputProtocolFactory.GetProtocol(p.inputTransport)
	// var outputTransport thrift.TTransport
	// var outputProtocol thrift.TProtocol

	// for THeaderProtocol, we must use the same protocol instance for
	// input and output so that the response is in the same dialect that
	// the server detected the request was in.
	var ok bool
	p.headerProtocol, ok = p.inputProtocol.(*thrift.THeaderProtocol)
	if ok {
		p.outputProtocol = p.inputProtocol
	} else {
		oTrans, err := p.outputTransportFactory.GetTransport(client)
		if err != nil {
			return 0, err
		}
		p.outputTransport = oTrans
		p.outputProtocol = p.outputProtocolFactory.GetProtocol(p.outputTransport)
	}

	// if inputTransport != nil {
	// 	defer inputTransport.Close()
	// }
	// if outputTransport != nil {
	// 	defer outputTransport.Close()
	// }

	return 0, nil
}

func (p *SimpleIPCServer) ProcessRequests() (int32, error) {
	if err := p.processRequests(p.client); err != nil {
		p.logger(fmt.Sprintf("error processing request: %v", err))
	}
	return 0, nil
}

func (p *SimpleIPCServer) AcceptLoop() error {
	for {
		closed, err := p.innerAccept()
		if err != nil {
			return err
		}
		if closed != 0 {
			return nil
		}
	}
}

func (p *SimpleIPCServer) Serve() error {
	if p.logger == nil {
		p.logger = thrift.StdLogger(nil)
	}

	err := p.Listen()
	if err != nil {
		return err
	}
	p.AcceptLoop()
	return nil
}

func (p *SimpleIPCServer) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if atomic.LoadInt32(&p.closed) != 0 {
		return nil
	}
	atomic.StoreInt32(&p.closed, 1)
	p.serverTransport.Interrupt()
	p.wg.Wait()
	return nil
}

// If err is actually EOF or NOT_OPEN, return nil, otherwise return err as-is.
func treatEOFErrorsAsNil(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, io.EOF) {
		return nil
	}
	var te thrift.TTransportException
	// NOT_OPEN returned by processor.Process is usually caused by client
	// abandoning the connection (e.g. client side time out, or just client
	// closes connections from the pool because of shutting down).
	// Those logs will be very noisy, so suppress those logs as well.
	if errors.As(err, &te) && (te.TypeId() == thrift.END_OF_FILE || te.TypeId() == thrift.NOT_OPEN) {
		return nil
	}
	return err
}

func (p *SimpleIPCServer) EndProcessRequests() {
	p.end_loop = true
}

func (p *SimpleIPCServer) processRequests(client thrift.TTransport) (err error) {
	defer func() {
		err = treatEOFErrorsAsNil(err)
	}()

	processor := p.processorFactory.GetProcessor(client)
	for {
		if atomic.LoadInt32(&p.closed) != 0 {
			return nil
		}

		ctx := thrift.SetResponseHelper(
			defaultCtx,
			thrift.TResponseHelper{
				THeaderResponseHelper: thrift.NewTHeaderResponseHelper(p.outputProtocol),
			},
		)
		if p.headerProtocol != nil {
			// We need to call ReadFrame here, otherwise we won't
			// get any headers on the AddReadTHeaderToContext call.
			//
			// ReadFrame is safe to be called multiple times so it
			// won't break when it's called again later when we
			// actually start to read the message.
			if err := p.headerProtocol.ReadFrame(ctx); err != nil {
				return err
			}
			ctx = thrift.AddReadTHeaderToContext(ctx, p.headerProtocol.GetReadHeaders())
			ctx = thrift.SetWriteHeaderList(ctx, p.forwardHeaders)
		}

		ok, err := processor.Process(ctx, p.inputProtocol, p.outputProtocol)

		if p.end_loop {
			p.end_loop = false
			return nil
		}

		if errors.Is(err, ErrAbandonRequest) {
			return client.Close()
		}
		if errors.As(err, new(thrift.TTransportException)) && err != nil {
			return err
		}
		var tae thrift.TApplicationException
		if errors.As(err, &tae) && tae.TypeId() == thrift.UNKNOWN_METHOD {
			continue
		}
		if !ok {
			break
		}
	}
	return nil
}
