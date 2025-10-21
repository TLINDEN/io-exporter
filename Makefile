all:
	CGO_LDFLAGS='-static' go build -tags osusergo,netgo -ldflags "-extldflags=-static" -o io-exporter
