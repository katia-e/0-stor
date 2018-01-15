package config

import (
	"os"
	"testing"

	"github.com/zero-os/0-stor/client"
	"github.com/zero-os/0-stor/client/itsyouonline"

	"github.com/stretchr/testify/require"
)

const (
	validFile              = "../../fixtures/benchmark/testconfigs/validConf.yaml"
	invalidDurOpsConfFile  = "../../fixtures/benchmark/testconfigs/invalidDurOpsConf.yaml"
	invalidKeySizeConfFile = "../../fixtures/benchmark/testconfigs/invalidKeySizeConf.yaml"
)

func TestClientConfig(t *testing.T) {
	require := require.New(t)

	yamlFile, err := os.Open(validFile)
	require.NoError(err)

	_, err = FromReader(yamlFile)
	require.NoError(err)
}

func TestInvalidClientConfig(t *testing.T) {
	require := require.New(t)

	// empty config
	var clientConf ClientConf
	require.Error(clientConf.validate())

	// invalid ops/duration
	yamlFile, err := os.Open(invalidDurOpsConfFile)
	require.NoError(err)

	_, err = FromReader(yamlFile)
	require.Error(err)

	// invalid keysize
	yamlFile, err = os.Open(invalidKeySizeConfFile)
	require.NoError(err)

	_, err = FromReader(yamlFile)
	require.Error(err)
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
	require.NotEmpty(c.Namespace, "Namespace should be set")

	const testNamespace = "test_namespace"
	c = client.Config{
		IYO: itsyouonline.Config{
			Organization:      "org",
			ApplicationID:     "some ID",
			ApplicationSecret: "some secret",
		},
		Namespace: testNamespace,
	}

	SetupClientConfig(&c)
	require.Equal(testNamespace, c.Namespace)
}
