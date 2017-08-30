package netm

import (
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/go-errors/errors"
)

// Bootstrap 启动器
type Bootstrap struct {
	listener    net.Listener
	mu          sync.Mutex
	connTable   map[string]net.Conn
	connTableMu sync.RWMutex
	opts        *Options
	optsMu      sync.RWMutex
	handlers    []Handler
	running     bool
	grRunning   bool
}

// NewBootstrap 创建启动器
func NewBootstrap() *Bootstrap {
	b := &Bootstrap{
		opts:      &Options{},
		grRunning: true,
	}
	b.connTable = make(map[string]net.Conn)
	return b
}

// Bind 监听地址、端口
func (bootstrap *Bootstrap) Bind(host string, port int) *Bootstrap {
	bootstrap.optsMu.Lock()
	bootstrap.opts.Host = host
	bootstrap.opts.Port = port
	bootstrap.optsMu.Unlock()
	return bootstrap
}

// Sync 启动服务
func (bootstrap *Bootstrap) Sync() {
	opts := bootstrap.getOpts()
	addr := net.JoinHostPort(opts.Host, strconv.Itoa(opts.Port))

	listener, e := net.Listen("tcp", addr)
	if e != nil {
		bootstrap.Fatalf("Error listening on port: %s, %q", addr, e)
		return
	}
	bootstrap.Noticef("Listening for client connections on %s",
		net.JoinHostPort(opts.Host, strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)))
	bootstrap.Noticef("Bootstrap is ready")

	bootstrap.mu.Lock()
	if opts.Port == 0 {
		// Write resolved port back to options.
		_, port, err := net.SplitHostPort(listener.Addr().String())
		if err != nil {
			bootstrap.Fatalf("Error parsing server address (%s): %s", listener.Addr().String(), err)
			bootstrap.mu.Unlock()
			return
		}
		portNum, err := strconv.Atoi(port)
		if err != nil {
			bootstrap.Fatalf("Error parsing server address (%s): %s", listener.Addr().String(), err)
			bootstrap.mu.Unlock()
			return
		}
		opts.Port = portNum
	}
	bootstrap.listener = listener
	bootstrap.running = true
	bootstrap.mu.Unlock()

	tmpDelay := ACCEPT_MIN_SLEEP
	for bootstrap.isRunning() {
		conn, err := listener.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				bootstrap.Debugf("Temporary Client Accept Error(%v), sleeping %dms",
					ne, tmpDelay/time.Millisecond)
				time.Sleep(tmpDelay)
				tmpDelay *= 2
				if tmpDelay > ACCEPT_MAX_SLEEP {
					tmpDelay = ACCEPT_MAX_SLEEP
				}
			} else if bootstrap.isRunning() {
				bootstrap.Errorf("Accept error: %v", err)
			}
			continue
		}
		tmpDelay = ACCEPT_MIN_SLEEP

		err = bootstrap.cfgConnect(conn)
		if err != nil {
			bootstrap.Errorf("config connect error: %v", err)
			continue
		}

		// 以客户端ip,port管理连接
		remoteAddr := conn.RemoteAddr().String()
		bootstrap.connTableMu.Lock()
		bootstrap.connTable[remoteAddr] = conn
		bootstrap.connTableMu.Unlock()
		bootstrap.Debugf("Client connection created %s", remoteAddr)

		bootstrap.startGoRoutine(func() {
			bootstrap.handleConn(remoteAddr, conn)
		})
	}

	bootstrap.Noticef("Bootstrap Exiting..")
}

// 配置连接
func (bootstrap *Bootstrap) cfgConnect(conn net.Conn) error {
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		if err := tcpConn.SetKeepAlive(false); err != nil {
			return errors.Wrap(err, 0)
		}
	}

	return nil
}

// Connect 连接指定地址、端口(服务器地址管理连接)
func (bootstrap *Bootstrap) Connect(host string, port int) error {
	addr := net.JoinHostPort(host, strconv.Itoa(port))

	return bootstrap.ConnectJoinAddr(addr)
}

// Connect 使用指定地址、端口的连接字符串连接
func (bootstrap *Bootstrap) ConnectJoinAddr(addr string) error {
	_, err := bootstrap.ConnectJoinAddrAndReturn(addr)
	return err
}

// Connect 使用指定地址、端口的连接字符串进行连接并返回连接
func (bootstrap *Bootstrap) ConnectJoinAddrAndReturn(addr string) (net.Conn, error) {
	bootstrap.connTableMu.RLock()
	conn, ok := bootstrap.connTable[addr]
	bootstrap.connTableMu.RUnlock()
	if ok {
		return conn, nil
	}

	nconn, e := bootstrap.connect(addr)
	if e != nil {
		bootstrap.Fatalf("Error Connect on port: %s, %q", addr, e)
		return nil, errors.Wrap(e, 0)
	}

	bootstrap.connTableMu.Lock()
	bootstrap.connTable[addr] = nconn
	bootstrap.connTableMu.Unlock()
	bootstrap.Noticef("Connect listening on port: %s", addr)
	bootstrap.Noticef("client connections on %s", nconn.LocalAddr().String())

	bootstrap.startGoRoutine(func() {
		bootstrap.handleConn(addr, nconn)
	})

	return nconn, nil
}

func (bootstrap *Bootstrap) connect(addr string) (net.Conn, error) {
	conn, e := net.Dial("tcp", addr)
	if e != nil {
		return nil, errors.Wrap(e, 0)
	}

	return conn, nil
}

// HasConnect find connect by addr, return bool
func (bootstrap *Bootstrap) HasConnect(addr string) bool {
	bootstrap.connTableMu.RLock()
	_, ok := bootstrap.connTable[addr]
	bootstrap.connTableMu.RUnlock()
	if !ok {
		return false
	}

	return true
}

// Disconnect 关闭指定连接
func (bootstrap *Bootstrap) Disconnect(addr string) {
	bootstrap.connTableMu.RLock()
	conn, ok := bootstrap.connTable[addr]
	bootstrap.connTableMu.RUnlock()
	if ok {
		bootstrap.disconnect(addr, conn)
	}
}

func (bootstrap *Bootstrap) disconnect(addr string, conn net.Conn) {
	conn.Close()
	bootstrap.connTableMu.Lock()
	delete(bootstrap.connTable, addr)
	bootstrap.connTableMu.Unlock()
}

// Shutdown 关闭bootstrap
func (bootstrap *Bootstrap) Shutdown() {
	bootstrap.mu.Lock()
	bootstrap.running = false
	bootstrap.mu.Unlock()

	// 关闭所有连接
	bootstrap.connTableMu.Lock()
	for addr, conn := range bootstrap.connTable {
		conn.Close()
		delete(bootstrap.connTable, addr)
	}
	bootstrap.connTableMu.Unlock()
}

// Write 发送消息
func (bootstrap *Bootstrap) Write(addr string, buffer []byte) (n int, err error) {
	bootstrap.connTableMu.RLock()
	conn, ok := bootstrap.connTable[addr]
	bootstrap.connTableMu.RUnlock()
	if !ok {
		bootstrap.Fatalf("not found connect: %s", addr)
		err = errors.Errorf("not found connect %s", addr)
		return
	}

	return bootstrap.write(addr, conn, buffer)
}

func (bootstrap *Bootstrap) write(addr string, conn net.Conn, buffer []byte) (n int, e error) {
	n, e = conn.Write(buffer)
	if e != nil {
		bootstrap.disconnect(addr, conn)
		e = errors.Wrap(e, 0)
	}

	return
}

// RegisterHandler 注册连接接收数据时回调执行函数
func (bootstrap *Bootstrap) RegisterHandler(fns ...Handler) *Bootstrap {
	bootstrap.handlers = append(bootstrap.handlers, fns...)
	return bootstrap
}

func (bootstrap *Bootstrap) handleConn(addr string, conn net.Conn) {
	b := make([]byte, 1024)
	for {
		n, err := conn.Read(b)
		if err != nil {
			bootstrap.disconnect(addr, conn)
			bootstrap.Fatalf("failed handle connect: %s %s", addr, err)
			return
		}

		for _, fn := range bootstrap.handlers {
			fn(b[:n], addr, conn)
		}
	}
}

func (bootstrap *Bootstrap) startGoRoutine(fn func()) {
	if bootstrap.grRunning {
		go fn()
	}
}

func (bootstrap *Bootstrap) isRunning() bool {
	bootstrap.mu.Lock()
	defer bootstrap.mu.Unlock()
	return bootstrap.running
}

func (bootstrap *Bootstrap) getOpts() *Options {
	bootstrap.optsMu.RLock()
	opts := bootstrap.opts
	bootstrap.optsMu.RUnlock()
	return opts
}

// Size 当前连接数
func (bootstrap *Bootstrap) Size() int {
	bootstrap.connTableMu.RLock()
	defer bootstrap.connTableMu.RUnlock()
	return len(bootstrap.connTable)
}

// NewRandomConnect 连接指定地址、端口(客户端随机端口地址管理连接)。特殊业务使用
func (bootstrap *Bootstrap) NewRandomConnect(host string, port int) (net.Conn, error) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))

	nconn, e := bootstrap.connect(addr)
	if e != nil {
		bootstrap.Fatalf("Error Connect on port: %s, %q", addr, e)
		return nil, errors.Wrap(e, 0)
	}

	localAddr := nconn.LocalAddr().String()
	bootstrap.connTableMu.Lock()
	bootstrap.connTable[localAddr] = nconn
	bootstrap.connTableMu.Unlock()
	bootstrap.Noticef("Connect listening on port: %s", addr)
	bootstrap.Noticef("client connections on %s", localAddr)

	bootstrap.startGoRoutine(func() {
		bootstrap.handleConn(addr, nconn)
	})

	return nconn, nil
}
