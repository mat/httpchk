test:
	go test -v ./...

deploy:
	git push heroku master
	heroku config:set GIT_REVISION=`git describe --always` DEPLOYED_AT=`date +%s`

run_server:
	PORT=3000 go run .

build_rasperrypi:
	GOARM=5 GOARCH=arm GOOS=linux go build -o bin/raspberrypi/httpchk .

build_linux_amd64:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/linux_amd64/httpchk .
