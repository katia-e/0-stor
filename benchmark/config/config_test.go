package config

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zero-os/0-stor/client"
	"github.com/zero-os/0-stor/client/itsyouonline"
)

const (
	validFile              = "testconfigs/validConf.yaml"
	invalidDurOpsConfFile  = "testconfigs/invalidDurOpsConf.yaml"
	invalidKeySizeConfFile = "testconfigs/invalidKeySizeConf.yaml"
)

func TestClientConfig(t *testing.T) {
	require := require.New(t)

	yamlFile, err := os.Open(validFile)
	require.NoError(err)

	clientConf, err := FromReader(yamlFile)
	require.NoError(err)

	err = yamlFile.Close()
	require.NoError(err)

	err = clientConf.Validate()
	require.NoError(err)

	for _, sc := range clientConf.Scenarios {
		err = sc.Validate()
		require.NoError(err)
	}
}

func TestInvalidClientConfig(t *testing.T) {
	require := require.New(t)

	// empty config
	var clientConf ClientConf
	require.Error(clientConf.Validate())

	// invalid ops/duration
	yamlFile, err := os.Open(invalidDurOpsConfFile)
	require.NoError(err)

	cc, err := FromReader(yamlFile)
	require.NoError(err)

	err = yamlFile.Close()
	require.NoError(err)

	sc := cc.Scenarios["bench1"]
	fmt.Println(sc.Validate())
	require.Error(sc.Validate())

	// invalid keysize
	yamlFile, err = os.Open(invalidKeySizeConfFile)
	require.NoError(err)

	cc, err = FromReader(yamlFile)
	require.NoError(err)

	err = yamlFile.Close()
	require.NoError(err)

	sc = cc.Scenarios["bench1"]
	fmt.Println(sc.Validate())
	require.Error(sc.Validate())
}

func TestSetupClientConfig(t *testing.T) {
	require := require.New(t)
	c := client.Config{
		IYO: itsyouonline.Config{
			Organization:      "org",
			ApplicationID:     "some ID",
			ApplicationSecret: "some secret",
		},
	}

	SetupClientConfig(&c)

	require.Empty(c.IYO.Organization, "IYO organization should be empty")
	require.Empty(c.IYO.ApplicationID, "IYO app ID should be empty")
	require.Empty(c.IYO.ApplicationSecret, "IYO app secret should be empty")
	require.NotEmpty(c.Namespace, "Namespace should be set")
}
