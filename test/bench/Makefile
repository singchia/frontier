
.PHONY: all
all: publish

.PHONY: publish
publish:
	make -C publish
	make -C call

.PHONY: clean
clean:
	make clean -C publish
	make clean -C call

bench:
	make bench -C publish
	make bench -C call