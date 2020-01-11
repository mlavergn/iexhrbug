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

pack: build
	zip pack index.html
	printf "%010d" `stat -f%z pack.zip` >> pack.zip
	mv main main.pack; cat main.pack pack.zip > main
	chmod +x main
	rm pack.zip main.pack
	./main -install true

clean:
	-rm -f main

run: build
	./main

test:
	open "http://localhost:8000"