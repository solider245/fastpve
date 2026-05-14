module github.com/solider245/fastpve

go 1.24.4

require (
	github.com/kspeeder/blobDownload v0.0.0-20251124020807-3c82a6d26394
	github.com/kspeeder/docker-registry v0.0.0-20251123150517-9065e6afc698
	github.com/kspeeder/urlcache v0.0.0-20251125050822-bf2b496b4f24
	github.com/manifoldco/promptui v0.9.0
	github.com/urfave/cli/v2 v2.27.6
	github.com/urfave/cli/v3 v3.6.1
)

require (
	github.com/chzyer/readline v0.0.0-20180603132655-2972be24d48e // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.5 // indirect
	github.com/docker/cli v29.0.2+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.9.4 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/xrash/smetrics v0.0.0-20240521201337-686a1a2994c1 // indirect
	golang.org/x/sys v0.33.0 // indirect
)

//replace github.com/kspeeder/blobDownload => ../blobDownload
//replace github.com/kspeeder/docker-registry => ../docker-registry
