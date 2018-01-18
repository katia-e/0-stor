/*
 * Copyright (C) 2017-2018 GIG Technology NV and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package benchers

import (
	"github.com/zero-os/0-stor/client"
	"github.com/zero-os/0-stor/client/datastor"
	storgrpc "github.com/zero-os/0-stor/client/datastor/grpc"
	"github.com/zero-os/0-stor/client/datastor/pipeline"
	"github.com/zero-os/0-stor/client/itsyouonline"
	"github.com/zero-os/0-stor/client/metastor"
	metaDB "github.com/zero-os/0-stor/client/metastor/db"
	"github.com/zero-os/0-stor/client/metastor/db/etcd"
	"github.com/zero-os/0-stor/client/metastor/db/test"
	"github.com/zero-os/0-stor/client/metastor/encoding"
	"github.com/zero-os/0-stor/client/processing"

	log "github.com/Sirupsen/logrus"
)

// newClientFromConfig creates a new zstor client from provided config
// if Metastor shards are empty, it will use an in memory metadata server
func newClientFromConfig(cfg *client.Config, jobCount int, enableCaching bool) (*client.Client, error) {
	// create datastor cluster
	datastorCluster, err := createDataClusterFromConfig(cfg, enableCaching)
	if err != nil {
		return nil, err
	}

	// create data pipeline, using our datastor cluster
	dataPipeline, err := pipeline.NewPipeline(cfg.DataStor.Pipeline, datastorCluster, jobCount)
	if err != nil {
		return nil, err
	}

	// if no metadata shards are given, return an error,
	// as we require a metastor client
	// create metastor client
	metastorClient, err := createMetastorClientFromConfig(&cfg.MetaStor)
	if err != nil {
		return nil, err
	}

	return client.NewClient(metastorClient, dataPipeline), nil
}

func createDataClusterFromConfig(cfg *client.Config, enableCaching bool) (datastor.Cluster, error) {
	if cfg.IYO == (itsyouonline.Config{}) {
		// create datastor cluster without the use of IYO-backed JWT Tokens,
		// this will only work if all shards use zstordb servers that
		// do not require any authentication (run with no-auth flag)
		return storgrpc.NewCluster(cfg.DataStor.Shards, cfg.Namespace, nil)
	}

	// create IYO client
	client, err := itsyouonline.NewClient(cfg.IYO)
	if err != nil {
		return nil, err
	}

	var tokenGetter datastor.JWTTokenGetter
	// create JWT Token Getter (Using the earlier created IYO Client)
	tokenGetter, err = datastor.JWTTokenGetterUsingIYOClient(cfg.IYO.Organization, client)
	if err != nil {
		return nil, err
	}

	if enableCaching {
		// create cached token getter from this getter, using the default bucket size and count
		tokenGetter, err = datastor.CachedJWTTokenGetter(tokenGetter, -1, -1)
		if err != nil {
			return nil, err
		}
	}

	// create datastor cluster, with the use of IYO-backed JWT Tokens
	return storgrpc.NewCluster(cfg.DataStor.Shards, cfg.Namespace, tokenGetter)
}

func createMetastorClientFromConfig(cfg *client.MetaStorConfig) (*metastor.Client, error) {
	if len(cfg.Database.Endpoints) == 0 {
		// if no endpoints, return a test metadata server
		log.Info("Using in-memory metadata server")
		return createMetastorClientFromConfigAndDatabase(cfg, test.New())
	}
	log.Info("Using etcd metadata server")

	// create metastor database first,
	// so that then we can create the Metastor client itself
	// TODO: support other types of databases (e.g. badger)
	db, err := etcd.New(cfg.Database.Endpoints)
	if err != nil {
		return nil, err
	}

	// create the metastor client and the rest of its components
	return createMetastorClientFromConfigAndDatabase(cfg, db)
}

func createMetastorClientFromConfigAndDatabase(cfg *client.MetaStorConfig, db metaDB.DB) (*metastor.Client, error) {
	var (
		err    error
		config = metastor.Config{Database: db}
	)

	// create the metadata encoding func pair
	config.MarshalFuncPair, err = encoding.NewMarshalFuncPair(cfg.Encoding)
	if err != nil {
		return nil, err
	}

	if len(cfg.Encryption.PrivateKey) == 0 {
		// create potentially insecure metastor storage
		return metastor.NewClient(config)
	}

	// create the constructor which will create our encrypter-decrypter when needed
	config.ProcessorConstructor = func() (processing.Processor, error) {
		return processing.NewEncrypterDecrypter(
			cfg.Encryption.Type, []byte(cfg.Encryption.PrivateKey))
	}
	// ensure the constructor is valid,
	// as most errors (if not all) are static, and will only fail due to the given input,
	// meaning that if it can be created it now, it should be fine later on as well
	_, err = config.ProcessorConstructor()
	if err != nil {
		return nil, err
	}

	// create our full-configured metastor client,
	// including encryption support for our metadata in binary form
	return metastor.NewClient(config)
}
