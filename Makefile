
test:
	go test

update_godeps:
	godep save ./...

deploy:
	git push heroku master
	heroku config:set GIT_REVISION=`git describe --always` DEPLOYED_AT=`date +%s`

run_server:
# go run getxpath.go -port=3000
	PORT=3000 CHECKS_CSV=checks.csv go run httpchk.go

install_devtools:
	go get code.google.com/p/go.tools/cmd/vet
	go get github.com/golang/lint/golint

check:
	go vet *.go
	golint *.go

build_rasperrypi:
	GOARM=5 GOARCH=arm GOOS=linux go build httpchk.go

build_linux_amd64:
		GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/linux_amd64/httpchk httpchk.go
