package config

import (
	"google.golang.org/grpc"
	"longhu.bojiu.com/pkg/viper"
	"math/rand"
	"time"

	"common.bojiu.com/discover/kit/sd/etcdv3"
)

type serverCfg struct {
	protocol   string
	ip         string
	port       int
	etcdServer []string
	regKey     string

	option etcdv3.ClientOptions
}

func NewServerCfg() *serverCfg {
	protocol := viper.Vp.GetString("ser.longhugame.protocol")
	ip := viper.Vp.GetString("ser.longhugame.ip")
	port := viper.Vp.GetInt("ser.longhugame.port")
	etcdServer := viper.Vp.GetStringSlice("ser.longhugame.etcdServer")
	regKey := viper.Vp.GetString("ser.longhugame.regKey")

	return &serverCfg{
		protocol:   protocol,
		ip:         ip,
		port:       port,
		etcdServer: etcdServer,
		regKey:     regKey,
		option: etcdv3.ClientOptions{
			// Path to trusted ca file
			CACert: "",
			// Path to certificate
			Cert: "",
			// Path to private key
			Key: "",
			// Username if required
			Username: "",
			// Password if required
			Password: "",
			// If DialTimeout is 0, it defaults to 3s
			DialTimeout: time.Second * 3,
			// If DialKeepAlive is 0, it defaults to 3s
			DialKeepAlive: time.Second * 3,
			// If passing `grpc.WithBlock`, dial connection will block until success.
			DialOptions: []grpc.DialOption{grpc.WithBlock()},
		},
	}

	//if err := viper.Vp.UnmarshalKey("ser.login", cfg); err != nil {
	//	log.ZapLog.Error("解析配置文件失败", zap.Any("err", err))
	//}
	//log.ZapLog.With(zap.Stack("trace")).Info("serverCfg")
}

func (f *serverCfg) GetProtocol() string {
	return f.protocol
}

func (f *serverCfg) GetIp() string {
	return f.ip
}

func (f *serverCfg) GetPort() int {
	return f.port
}

func (f *serverCfg) GetEtcdServer() []string {
	return f.etcdServer
}

func (f *serverCfg) GetRandEtcdServer() string {
	n := rand.Intn(len(f.etcdServer))
	return f.etcdServer[n]
}

func (f *serverCfg) GetRegKey() string {
	return f.regKey
}

func (f *serverCfg) GetOption() etcdv3.ClientOptions {
	return f.option
}
