//reference https://github.com/trojan-gfw/igniter-go-libs/blob/master/tun2socks/tun2socks.go
package main

import (
	"C"

	"context"
	"io"
	"net"
	"strings"
	"time"

	//"github.com/Trojan-Qt5/go-tun2socks/common/log"
	_ "github.com/Trojan-Qt5/go-tun2socks/common/log/simple"
	"github.com/Trojan-Qt5/go-tun2socks/core"
	"github.com/Trojan-Qt5/go-tun2socks/proxy/socks"
	"github.com/Trojan-Qt5/go-tun2socks/tun"

	_ "github.com/p4gefau1t/trojan-go/build"
	"github.com/p4gefau1t/trojan-go/common"
	"github.com/p4gefau1t/trojan-go/log"

	"github.com/Trojan-Qt5/go-shadowsocks2/cmd/shadowsocks"

	v2ray "github.com/Trojan-Qt5/v2ray-go/core"
)

const (
	MTU = 1500
)

var (
	client     common.Runnable
	lwipWriter io.Writer
	tunDev     io.ReadWriteCloser
	ctx        context.Context
	cancel     context.CancelFunc
	isRunning  bool = false
)

//export is_tun2socks_running
func is_tun2socks_running() bool {
	return isRunning
}

//export stop_tun2socks
func stop_tun2socks() {
	log.Info("Stopping tun2socks")

	isRunning = false

	err := tunDev.Close()
	if err != nil {
		log.Fatalf("failed to close tun device: %v", err)
	}

	cancel()
}

//export run_tun2socks
func run_tun2socks(tunName *C.char, tunAddr *C.char, tunGw *C.char, tunDns *C.char, proxyServer *C.char) {

	// Open the tun device.
	dnsServers := strings.Split(C.GoString(tunDns), ",")
	var err error
	tunDev, err = tun.OpenTunDevice(C.GoString(tunName), C.GoString(tunAddr), C.GoString(tunGw), "255.255.255.0", dnsServers)
	if err != nil {
	}

	// Setup TCP/IP stack.
	lwipWriter := core.NewLWIPStack().(io.Writer)

	// Register tun2socks connection handlers.
	proxyAddr, err := net.ResolveTCPAddr("tcp", C.GoString(proxyServer))
	proxyHost := proxyAddr.IP.String()
	proxyPort := uint16(proxyAddr.Port)
	if err != nil {
		log.Info("invalid proxy server address: %v", err)
	}
	core.RegisterTCPConnHandler(socks.NewTCPHandler(proxyHost, proxyPort, nil))
	core.RegisterUDPConnHandler(socks.NewUDPHandler(proxyHost, proxyPort, 1*time.Minute, nil, nil))

	// Register an output callback to write packets output from lwip stack to tun
	// device, output function should be set before input any packets.
	core.RegisterOutputFn(func(data []byte) (int, error) {
		return tunDev.Write(data)
	})

	ctx, cancel = context.WithCancel(context.Background())

	// Copy packets from tun device to lwip stack, it's the main loop.
	go func(ctx context.Context) {
		_, err := io.CopyBuffer(lwipWriter, tunDev, make([]byte, MTU))
		if err != nil {
			log.Info(err.Error())
		}
	}(ctx)

	log.Info("Running tun2socks")

	isRunning = true

	<-ctx.Done()
}

//export startShadowsocksGo
func startShadowsocksGo(ClientAddr *C.char, ServerAddr *C.char, Cipher *C.char, Password *C.char, Plugin *C.char, PluginOptions *C.char, EnableAPI bool, APIAddress *C.char) {
	shadowsocks.StartGoShadowsocks(C.GoString(ClientAddr), C.GoString(ServerAddr), C.GoString(Cipher), C.GoString(Password), C.GoString(Plugin), C.GoString(PluginOptions), EnableAPI, C.GoString(APIAddress))
}

//export stopShadowsocksGo
func stopShadowsocksGo() {
	shadowsocks.StopGoShadowsocks()
}

//export testV2rayGo
func testV2rayGo(configFile *C.char) (bool, *C.char) {
	status, err := v2ray.TestV2ray(C.GoString(configFile))
	return status, C.CString(err)
}

//export startV2rayGo
func startV2rayGo(configFile *C.char) {
	v2ray.StartV2ray(C.GoString(configFile))
}

//export stopV2rayGo
func stopV2rayGo() {
	v2ray.StopV2ray()
}

func main() {
}
