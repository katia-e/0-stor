package benchers

import (
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/coreos/etcd/embed"
	"github.com/coreos/pkg/capnslog"
	"github.com/stretchr/testify/require"
	"github.com/zero-os/0-stor/client"
	"github.com/zero-os/0-stor/client/datastor/pipeline"
	"github.com/zero-os/0-stor/client/processing"
	"github.com/zero-os/0-stor/server/api"
	"github.com/zero-os/0-stor/server/api/grpc"
	"github.com/zero-os/0-stor/server/db/badger"
)

// newEmbeddedMetaServer creates new embedded metadata (etcd) server
func newEmbeddedMetaServer() (*embeddedMetaServer, error) {
	tmpDir, err := ioutil.TempDir("", "etcd")
	if err != nil {
		return nil, err
	}

	cfg := embed.NewConfig()
	cfg.Dir = tmpDir

	capnslog.SetGlobalLogLevel(capnslog.CRITICAL)
	// listen client URL
	// we use tmpDir as unix address because it is a simple
	// yet valid way to generate random string
	lcurl, err := url.Parse("unix://" + filepath.Base(tmpDir))
	if err != nil {
		return nil, err
	}
	cfg.LCUrls = []url.URL{*lcurl}

	// listen peer url
	// same strategy with listen client URL
	lpDir, err := ioutil.TempDir("", "etcd")
	if err != nil {
		return nil, err
	}
	lpurl, err := url.Parse("unix://" + filepath.Base(lpDir))
	if err != nil {
		return nil, err
	}
	cfg.LPUrls = []url.URL{*lpurl}

	e, err := embed.StartEtcd(cfg)
	if err != nil {
		return nil, err
	}

	<-e.Server.ReadyNotify()

	conf := e.Config()

	return &embeddedMetaServer{
		lpDir:      lpDir,
		lcDir:      tmpDir,
		etcd:       e,
		listenAddr: conf.LCUrls[0].String(),
	}, nil
}

// embeddedMetaServer is embedded metadata server
// which listen on unix socket
type embeddedMetaServer struct {
	lcDir      string
	lpDir      string
	etcd       *embed.Etcd
	listenAddr string
}

// Stop stops the server and release it's resources
func (s *embeddedMetaServer) Stop() {
	s.etcd.Server.Stop()
	<-s.etcd.Server.StopNotify()
	s.etcd.Close()
	os.RemoveAll(s.lpDir)
	os.RemoveAll(s.lcDir)
}

// ListenAddrs returns listen address of this server
func (s *embeddedMetaServer) ListenAddr() string {
	return s.listenAddr
}

// newTestZstorServers returns n amount of zstor test servers
// also returns a function to clean up the servers
func newTestZstorServers(t testing.TB, n int) ([]*testZstorServer, func()) {
	require := require.New(t)

	servers := make([]*testZstorServer, n)
	dirs := make([]string, n)

	for i := 0; i < n; i++ {
		tmpDir, err := ioutil.TempDir("", "0stortest")
		require.NoError(err)
		dirs[i] = tmpDir

		db, err := badger.New(path.Join(tmpDir, "data"), path.Join(tmpDir, "meta"))
		require.NoError(err)

		server, err := grpc.New(db, nil, 4, 0)
		require.NoError(err)

		listener, err := net.Listen("tcp", "localhost:0")
		require.NoError(err, "failed to create listener on /any/ open (local) port")

		go func() {
			err := server.Serve(listener)
			if err != nil {
				panic(err)
			}
		}()

		servers[i] = &testZstorServer{Server: server, addr: listener.Addr().String()}
	}

	clean := func() {
		for _, server := range servers {
			server.Close()
		}
		for _, dir := range dirs {
			os.RemoveAll(dir)
		}
	}

	return servers, clean
}

type testZstorServer struct {
	api.Server
	addr string
}

func (ts *testZstorServer) Address() string {
	return ts.addr
}

func newDefaultZstorConfig(dataShards []string, metaShards []string, blockSize int) client.Config {
	return client.Config{
		Namespace: "namespace1",
		DataStor: client.DataStorConfig{
			Shards: dataShards,
			Pipeline: pipeline.Config{
				BlockSize: blockSize,
				Compression: pipeline.CompressionConfig{
					Mode: processing.CompressionModeDefault,
				},
				Encryption: pipeline.EncryptionConfig{
					PrivateKey: "cF0BFpIsljOS8UmaP8YRHRX0nBPVRVPw",
				},
				Distribution: pipeline.ObjectDistributionConfig{
					DataShardCount:   3,
					ParityShardCount: 1,
				},
			},
		},
		MetaStor: client.MetaStorConfig{
			Database: client.MetaStorETCDConfig{
				Endpoints: metaShards,
			},
		},
	}
}
