.PHONY: all clean edge router udpping

BIN_DIR := bin

all: edge router udpping

edge: $(BIN_DIR)
	go build -o $(BIN_DIR)/wg-edge ./edge

router: $(BIN_DIR)
	go build -o $(BIN_DIR)/wg-router ./router

udpping: $(BIN_DIR)
	go build -o $(BIN_DIR)/udpping ./cmd/udpping

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

clean:
	rm -rf $(BIN_DIR)
