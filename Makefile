BINARY = cameraleech
COMMIT = $(shell git rev-parse --short HEAD)
BUILTAT = $(shell date +%FT%T%z)

LDFLAGS = -ldflags "-X main.commit=${COMMIT} -X main.builtat=${BUILTAT}"

all:
	go build ${LDFLAGS} -o ${BINARY}
