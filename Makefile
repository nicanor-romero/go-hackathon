BINARY  := bin/shield
CMD     := ./shield

.PHONY: all build run clean

all: build

build:
	@mkdir -p bin
	go build -o $(BINARY) $(CMD)

run: build
	./$(BINARY)

clean:
	rm -f $(BINARY)
