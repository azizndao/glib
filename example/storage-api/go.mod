module github.com/azizndao/glib/example/storage-api

go 1.24.0

require (
	github.com/azizndao/glib/storage v0.0.0
	github.com/azizndao/glib/storage/local v0.0.0
	github.com/azizndao/glib/storage/s3 v0.0.0
)

require (
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/goccy/go-json v0.10.4 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.10 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/minio/minio-go/v7 v7.0.82 // indirect
	github.com/rs/xid v1.6.0 // indirect
	golang.org/x/crypto v0.46.0 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
)

replace (
	github.com/azizndao/glib/storage => ../../storage
	github.com/azizndao/glib/storage/local => ../../storage/local
	github.com/azizndao/glib/storage/s3 => ../../storage/s3
)
