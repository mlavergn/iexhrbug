###############################################
#
# Makefile
#
###############################################

.DEFAULT_GOAL := run

VERSION := 1.0.0

build:
	-rm -f main
	go build -o main .
	$(MAKE) pack

win:
	-rm -f main
	GOARCH=amd64 GOOS=windows go build --ldflags "-s -w" -o main .
	$(MAKE) pack
	mv main main.exe

pack:
	zip pack index.html iexhr.crt iexhr.key
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


release:
	hub release create -m "${VERSION} - iexhrbug" -a main.exe.zip -t master "v${VERSION}"
	open "https://github.com/mlavergn/iexhrbug/releases"

#
# SSH
# recommended key 2048 (or 4096)
#

SSHKEY := 2048

ssh:
	ssh-keygen -t rsa -b $(SSHKEY) -C "$(EMAIL)"

#
# RSA
# recommended key 2048
#

#
# Subject info
#

ORG     := Example\ Inc.
DEPT    := IT
CITY    := San\ Francisco
STATE   := California
COUNTRY := US
HOST    := www.example.org
ALT     := www-alt.example.org
EMAIL   := admin@example.og
SUBJ    := '/O=$(ORG)/OU=$(DEPT)/C=${COUNTRY}/ST=$(STATE)/L=$(CITY)/CN=$(HOST)/subjectAltName=$(ALT)/emailAddress=$(EMAIL)'

subj:
	@echo $(SUBJ)

NAME := iexhr
RSAKEY := 2048

rsapriv:
	openssl genrsa -out $(NAME).key $(RSAKEY)

csr:
	openssl req -new -subj $(SUBJ) -key $(NAME).key -out $(NAME).csr

rsapub:
	openssl x509 -req -days 365 -in $(NAME).csr -signkey $(NAME).key -out $(NAME).crt

rsa: rsapriv csr rsapub	