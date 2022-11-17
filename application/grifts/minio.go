package grifts

import (
	"github.com/gobuffalo/grift/grift"
	"github.com/silinternational/cover-api/storage"
)

var _ = grift.Namespace("minio", func() {
	_ = grift.Desc("minio", "seed minIO")
	_ = grift.Add("seed", func(c *grift.Context) error {
		return storage.CreateS3Bucket()
	})
})
