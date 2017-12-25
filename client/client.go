package client

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/zero-os/0-stor/client/pipeline"

	"github.com/zero-os/0-stor/client/metastor/etcd"

	"github.com/zero-os/0-stor/client/datastor"
	storgrpc "github.com/zero-os/0-stor/client/datastor/grpc"
	"github.com/zero-os/0-stor/client/itsyouonline"
	"github.com/zero-os/0-stor/client/metastor"
	"github.com/zero-os/0-stor/client/pipeline/storage"

	log "github.com/Sirupsen/logrus"
)

var (
	// ErrRepairSupport is returned when data is not stored using replication or distribution
	ErrRepairSupport = fmt.Errorf("data is not stored using replication or distribution, repair impossible")
)

// Client defines 0-stor client
type Client struct {
	datastorCluster datastor.Cluster
	dataPipeline    pipeline.Pipeline

	metastorClient metastor.Client
}

// NewClientFromConfig creates new 0-stor client using the given config.
func NewClientFromConfig(cfg Config, jobCount int) (*Client, error) {
	var (
		err             error
		datastorCluster datastor.Cluster
	)
	// create datastor cluster
	if cfg.IYO != (itsyouonline.Config{}) {
		var client *itsyouonline.Client
		client, err = itsyouonline.NewClient(cfg.IYO)
		if err == nil {
			tokenGetter := jwtTokenGetterFromIYOClient(
				cfg.IYO.Organization, client)
			datastorCluster, err = storgrpc.NewCluster(cfg.DataStor.Shards, cfg.Namespace, tokenGetter)
		}
	} else {
		datastorCluster, err = storgrpc.NewCluster(cfg.DataStor.Shards, cfg.Namespace, nil)
	}
	if err != nil {
		return nil, err
	}

	// create data pipeline, using our datastor cluster
	dataPipeline, err := pipeline.NewPipeline(cfg.Pipeline, datastorCluster, jobCount)
	if err != nil {
		return nil, err
	}

	// if no metadata shards are given, return an error,
	// as we require a metastor client
	// TODO: allow a more flexible kind of metastor client configuration,
	// so we can also allow other types of metastor clients,
	// as we do really need one.
	if len(cfg.MetaStor.Shards) == 0 {
		return nil, errors.New("no metadata storage given")
	}

	// create metastor client first,
	// and than create our master 0-stor client with all features.
	metastorClient, err := etcd.NewClient(cfg.MetaStor.Shards)
	if err != nil {
		return nil, err
	}
	return NewClient(datastorCluster, metastorClient, dataPipeline)
}

// NewClient creates a 0-stor client,
// with the data (zstordb) cluster already created,
// used to read/write object data, as well as the metastor client,
// which is used to read/write the metadata of the objects.
//
// The given data pipeline is optional and a default one will be created should it not be defined.
func NewClient(dataCluster datastor.Cluster, metaClient metastor.Client, dataPipeline pipeline.Pipeline) (*Client, error) {
	if dataCluster == nil {
		panic("0-stor Client: no datastor cluster given")
	}
	if metaClient == nil {
		panic("0-stor Client: no metastor client given")
	}

	// create default pipeline if none is given
	if dataPipeline == nil {
		pipeline, err := pipeline.NewPipeline(pipeline.Config{}, dataCluster, -1)
		if err != nil {
			return nil, err
		}
		dataPipeline = pipeline
	}

	return &Client{
		datastorCluster: dataCluster,
		dataPipeline:    dataPipeline,
		metastorClient:  metaClient,
	}, nil
}

// Close the client
func (c *Client) Close() error {
	c.metastorClient.Close()
	if closer, ok := c.datastorCluster.(interface {
		Close() error
	}); ok {
		return closer.Close()
	}
	return nil
}

// Write write the value to the the 0-stors configured by the client config
func (c *Client) Write(key, value []byte) (*metastor.Metadata, error) {
	return c.WriteWithMeta(key, value, nil, nil, nil)
}

func (c *Client) WriteF(key []byte, r io.Reader) (*metastor.Metadata, error) {
	return c.writeFWithMeta(key, r, nil, nil, nil)
}

// WriteWithMeta writes the key-value to the configured pipes.
// Metadata linked list will be build if prevKey is not nil
// prevMeta is optional previous metadata, to be used in case of user already has the prev metastor.
// So the client won't need to fetch it back from the metadata server.
// prevKey still need to be set it prevMeta is set
// initialMeta is optional metadata, if user want to set his own initial metadata for example set own epoch
func (c *Client) WriteWithMeta(key, val []byte, prevKey []byte, prevMeta, md *metastor.Metadata) (*metastor.Metadata, error) {
	r := bytes.NewReader(val)
	return c.writeFWithMeta(key, r, prevKey, prevMeta, md)
}

func (c *Client) WriteFWithMeta(key []byte, r io.Reader, prevKey []byte, prevMeta, md *metastor.Metadata) (*metastor.Metadata, error) {
	return c.writeFWithMeta(key, r, prevKey, prevMeta, md)
}

func (c *Client) writeFWithMeta(key []byte, r io.Reader, prevKey []byte, prevMeta, md *metastor.Metadata) (*metastor.Metadata, error) {
	chunks, err := c.dataPipeline.Write(r)
	if err != nil {
		return nil, err
	}

	// create new metadata if not given
	if md == nil {
		now := time.Now().UnixNano()
		md = &metastor.Metadata{
			Key:            key,
			CreationEpoch:  now,
			LastWriteEpoch: now,
		}
	}

	// set/update chunks and size in metadata
	md.Chunks = chunks
	md.Size = 0
	for _, chunk := range chunks {
		md.Size += chunk.Size
	}

	err = c.linkMeta(md, prevMeta, key, prevKey)
	if err != nil {
		return md, err
	}

	return md, nil
}

// Read reads value with given key from the 0-stors configured by the client cnfig
// it will first try to get the metadata associated with key from the Metadata servers.
func (c *Client) Read(key []byte) ([]byte, error) {
	md, err := c.metastorClient.GetMetadata(key)
	if err != nil {
		return nil, err
	}

	w := &bytes.Buffer{}
	err = c.readFWithMeta(md, w)
	if err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

// ReadF similar as Read but write the data to w instead of returning a slice of bytes
func (c *Client) ReadF(key []byte, w io.Writer) error {
	md, err := c.metastorClient.GetMetadata(key)
	if err != nil {
		return err
	}
	return c.readFWithMeta(md, w)

}

// ReadWithMeta reads the value described by md
func (c *Client) ReadWithMeta(md *metastor.Metadata) ([]byte, error) {
	w := &bytes.Buffer{}
	err := c.readFWithMeta(md, w)
	if err != nil {
		return nil, err
	}

	return w.Bytes(), nil
}

func (c *Client) readFWithMeta(md *metastor.Metadata, w io.Writer) error {
	// get chunks from metadata
	chunks := make([]metastor.Chunk, len(md.Chunks))
	for index := range md.Chunks {
		chunks[index] = md.Chunks[index]
	}
	return c.dataPipeline.Read(chunks, w)
}

// Delete deletes object from the 0-stor server pointed by the key
// It also deletes the metadatastor.
func (c *Client) Delete(key []byte) error {
	meta, err := c.metastorClient.GetMetadata(key)
	if err != nil {
		log.Errorf("fail to read metadata: %v", err)
		return err
	}
	return c.DeleteWithMeta(meta)
}

// DeleteWithMeta deletes object from the 0-stor server pointed by the
// given metadata
// It also deletes the metadatastor.
func (c *Client) DeleteWithMeta(meta *metastor.Metadata) error {
	err := c.dataPipeline.Delete(meta.Chunks)
	if err != nil {
		log.Errorf("error deleting data :%v", err)
		return err
	}

	// delete metadata
	if err := c.metastorClient.DeleteMetadata(meta.Key); err != nil {
		log.Errorf("error deleting metadata :%v", err)
		return err
	}

	return nil
}

func (c *Client) Check(key []byte, fast bool) (storage.CheckStatus, error) {
	meta, err := c.metastorClient.GetMetadata(key)
	if err != nil {
		log.Errorf("fail to get metadata for check: %v", err)
		return storage.CheckStatus(0), err
	}
	return c.dataPipeline.Check(meta.Chunks, fast)
}

// Repair repairs a broken file.
// If the file is distributed and the amount of corrupted chunks is acceptable,
// we recreate the missing chunks.
// Id the file is replicated and we still have one valid replicate, we create the missing replicate
// till we reach the replication number configured in the config
// if the file as not been distributed or replicated, we can't repair it,
// or if not enough shards are available we cannot repair it either.
func (c *Client) Repair(key []byte) error {
	log.Debugf("Start repair of %x", key)
	meta, err := c.metastorClient.GetMetadata(key)
	if err != nil {
		log.Errorf("repair %x, error getting metadata :%v", key, err)
		return err
	}

	chunks, err := c.dataPipeline.Repair(meta.Chunks)
	if err != nil {
		if err == storage.ErrNotSupported {
			return ErrRepairSupport
		}
		return err
	}
	// update chunks
	meta.Chunks = chunks
	// update total size
	meta.Size = 0
	for _, chunk := range chunks {
		meta.Size += chunk.Size
	}

	// update last write epoch, as we have written while repairing
	meta.LastWriteEpoch = time.Now().UnixNano()

	if err := c.metastorClient.SetMetadata(*meta); err != nil {
		log.Errorf("error writing metadata after repair: %v", err)
		return err
	}

	return nil
}

func (c *Client) linkMeta(curMd, prevMd *metastor.Metadata, curKey, prevKey []byte) error {
	if len(prevKey) == 0 {
		return c.metastorClient.SetMetadata(*curMd)
	}

	// point next key of previous meta to new meta
	prevMd.NextKey = curKey

	// point prev key of new meta to previous one
	curMd.PreviousKey = prevKey

	// update prev meta
	if err := c.metastorClient.SetMetadata(*prevMd); err != nil {
		return err
	}

	// update new meta
	return c.metastorClient.SetMetadata(*curMd)
}
