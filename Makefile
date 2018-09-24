default:
	rm -rf _gopath
	mkdir -p _gopath/src/github.com/lomik/
	ln -s ../../../.. _gopath/src/github.com/lomik/prometheus-png
	GOPATH=${PWD}/_gopath go build -v -tags cairo github.com/lomik/prometheus-png
	rm -rf _gopath
