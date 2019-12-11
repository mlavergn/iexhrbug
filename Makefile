###############################################
#
# Makefile
#
###############################################

.DEFAULT_GOAL := run

GOPATH = "${HOME}/iexhrbug"

build:
	-rm -f main
	GOPATH=${GOPATH} go build -o main .

clean:
	-rm -f main

run: build
	./main

test:
	open "http://localhost:8000"