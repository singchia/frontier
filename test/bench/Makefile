
.PHONY: all
all: publish

.PHONY: publish
publish:
	make -C publish
	make -C call
	make -C open

.PHONY: clean
clean:
	make clean -C publish
	make clean -C call
	make clean -C open

bench:
	make bench -C publish
	make bench -C call
	make bench -C open