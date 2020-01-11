###############################################
#
# Makefile
#
###############################################

.DEFAULT_GOAL := run

GOPATH = "${HOME}/iexhrbug"

deps:
	-rm -rf src/github.com/mlavergn
	GOPATH=${GOPATH} go get -d github.com/mlavergn/gopack/src/gopack

build: deps
	-rm -f main
	GOPATH=${GOPATH} go build -o main .
	$(MAKE) pack

win:
	-rm -f main
	GOPATH=${GOPATH} GOARCH=amd64 GOOS=windows go build --ldflags "-s -w" -o main .
	$(MAKE) pack
	mv main main.exe

pack:
	zip pack index.html
	printf "%010d" `stat -f%z pack.zip` >> pack.zip
	mv main main.pack; cat main.pack pack.zip > main
	chmod +x main
	rm pack.zip main.pack

clean:
	-rm -f main

run: build
	./main

test:
	open "http://localhost:8000"