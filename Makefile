UNAME := $(shell uname -s)

CC=gcc

CFLAGS := -std=c99 -Wall
CFLAGS += $(shell pkg-config --libs libgit2 jansson)
CFLAGS += $(shell pkg-config --cflags libgit2 jansson)

# Add -rpath option so that the dynamic linker knows where to find shared
# libraries and avoid having to set LD_LIBRARY_PATH.
#
# See http://stackoverflow.com/a/695684 for background on this.
ifeq ($(UNAME), Linux)
CFLAGS += -Wl,-rpath /usr/local/lib
endif

TARGET=build/gb
SRC=src/main.c

INSTALL_DIR=/usr/local/bin/gb

.PHONY: install build

all: build

build: src/main.c
	mkdir -p build/
	$(CC) -o $(TARGET) $(SRC) $(CFLAGS)

install:
	cp $(TARGET) $(INSTALL_DIR)

force:
	touch $(SRC)

# Mainly for use when developing
run: clean force build install
	gb

clean:
	@rm -rf build/

deps: clean
	@if [ `pkg-config --modversion jansson` == "2.7" ]; then \
		echo "jansson was found - skipping installation"; \
	else \
		echo "installing jansson" && \
		mkdir -p build && \
		cd build && \
		wget http://www.digip.org/jansson/releases/jansson-2.7.tar.gz && \
		tar xzf jansson-2.7.tar.gz && \
		cd jansson-2.7 && \
		./configure && \
		make && \
		make check && \
		echo "sudo password required for 'sudo make install' in jansson" && \
		sudo make install; \
	fi; \


	@if [ `pkg-config --modversion libgit2` == "0.21.3" ]; then \
		echo "libgit2 was found - skipping installation"; \
	else \
		echo "installing libgit2" && \
		mkdir -p build && \
		cd build && \
		wget https://github.com/libgit2/libgit2/archive/v0.21.3.tar.gz && \
		tar xzf v0.21.3.tar.gz && \
		cd libgit2-0.21.3 && \
		mkdir build && \
		cd build && \
		cmake .. && \
		cmake --build . --target install; \
	fi
