package grpc

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"golang.org/x/net/context"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zero-os/0-stor/server"
	"github.com/zero-os/0-stor/server/db"
	"github.com/zero-os/0-stor/server/db/badger"
	"github.com/zero-os/0-stor/server/encoding"
	pb "github.com/zero-os/0-stor/server/schema"
)

func TestCreateObject(t *testing.T) {
	require := require.New(t)

	api, clean := getTestObjectAPI(require)
	defer clean()

	data, err := encoding.EncodeNamespace(server.Namespace{Label: []byte(label)})
	require.NoError(err)
	err = api.db.Set(db.NamespaceKey([]byte(label)), data)
	require.NoError(err)

	buf := make([]byte, 1024*1024)
	_, err = rand.Read(buf)
	require.NoError(err)

	req := &pb.CreateObjectRequest{
		Label: label,
		Object: &pb.Object{
			Key:           []byte("testkey"),
			Value:         buf,
			ReferenceList: []string{"user1", "user2"},
		},
	}

	_, err = api.Create(context.Background(), req)
	require.NoError(err)

	// get data and validate it's correct
	objRawData, err := api.db.Get(db.DataKey([]byte(label), []byte("testkey")))
	require.NoError(err)
	require.NotNil(objRawData)
	obj, err := encoding.DecodeObject(objRawData)
	require.NoError(err)
	require.Equal(req.Object.Value, obj.Data)

	// get reference list, and validate it's correct
	refListRawData, err := api.db.Get(db.ReferenceListKey([]byte(label), []byte("testkey")))
	require.NoError(err)
	require.NotNil(refListRawData)
	refList, err := encoding.DecodeReferenceList(refListRawData)
	require.NoError(err)
	require.Len(refList, len(req.Object.ReferenceList))
	require.Subset(req.Object.ReferenceList, refList)
}

func TestGetObject(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	api, clean := getTestObjectAPI(require)
	defer clean()

	bufList := populateDB(t, label, api.db)

	t.Run("valid", func(t *testing.T) {
		key := []byte("testkey0")
		req := &pb.GetObjectRequest{
			Label: label,
			Key:   key,
		}

		resp, err := api.Get(context.Background(), req)
		require.NoError(err)

		obj := resp.GetObject()

		assert.Equal(key, obj.GetKey())
		assert.Equal(bufList["testkey0"], obj.GetValue())
		assert.Equal([]string{"user1", "user2"}, obj.GetReferenceList())
	})

	t.Run("non existing", func(t *testing.T) {
		req := &pb.GetObjectRequest{
			Label: label,
			Key:   []byte("notexistingkey"),
		}

		_, err := api.Get(context.Background(), req)
		assert.Equal(db.ErrNotFound, err)
	})
}

func TestExistsObject(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	api, clean := getTestObjectAPI(require)
	defer clean()

	bufList := populateDB(t, label, api.db)

	for i := 0; i < len(bufList); i++ {
		key := fmt.Sprintf("testkey%d", i)
		t.Run(key, func(t *testing.T) {
			req := &pb.ExistsObjectRequest{
				Label: label,
				Key:   []byte(key),
			}

			resp, err := api.Exists(context.Background(), req)
			require.NoError(err)
			assert.True(resp.Exists, fmt.Sprintf("Key %s should exists", key))
		})
	}

	t.Run("non exists", func(t *testing.T) {
		req := &pb.ExistsObjectRequest{
			Label: label,
			Key:   []byte("nonexists"),
		}

		resp, err := api.Exists(context.Background(), req)
		require.NoError(err)
		assert.False(resp.Exists, fmt.Sprint("Key nonexists should not exists"))
	})
}

func TestDeleteObject(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	api, clean := getTestObjectAPI(require)
	defer clean()

	populateDB(t, label, api.db)

	t.Run("valid", func(t *testing.T) {
		req := &pb.DeleteObjectRequest{
			Label: label,
			Key:   []byte("testkey1"),
		}

		_, err := api.Delete(context.Background(), req)
		require.NoError(err)

		exists, err := api.db.Exists([]byte(req.Key))
		require.NoError(err)
		assert.False(exists)
	})

	// deleting a non existing object doesn't return an error.
	t.Run("non exists", func(t *testing.T) {
		req := &pb.DeleteObjectRequest{
			Label: label,
			Key:   []byte("nonexists"),
		}

		_, err := api.Delete(context.Background(), req)
		require.NoError(err)

		exists, err := api.db.Exists([]byte(req.Key))
		require.NoError(err)
		assert.False(exists)
	})
}

func getTestObjectAPI(require *require.Assertions) (*ObjectAPI, func()) {
	tmpDir, err := ioutil.TempDir("", "0stortest")
	require.NoError(err)

	db, err := badger.New(path.Join(tmpDir, "data"), path.Join(tmpDir, "meta"))
	if err != nil {
		require.NoError(err)
	}

	clean := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return NewObjectAPI(db, 0), clean
}
