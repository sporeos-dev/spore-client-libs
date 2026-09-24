HOST_OS := $(shell uname -s)
HOST_ARCH := $(shell uname -m)

ifeq ($(HOST_OS),Darwin)
	ifeq ($(HOST_ARCH),arm64)
		PLATFORM := macos-arm
	else ifeq ($(HOST_ARCH),x86_64)
		PLATFORM := macos-intel
	endif
else ifeq ($(HOST_OS),Linux)
	ifneq ($(filter x86_64 amd64,$(HOST_ARCH)),)
		PLATFORM := linux-intel
	else ifneq ($(filter aarch64 arm64,$(HOST_ARCH)),)
		PLATFORM := linux-arm
	endif
else ifneq ($(filter MINGW% MSYS%,$(HOST_OS)),)
	ifneq ($(filter aarch64 arm64,$(HOST_ARCH)),)
		PLATFORM := windows-arm
	else
		PLATFORM := windows-intel
	endif
endif

ifndef PLATFORM
	$(error Unsupported host platform: $(HOST_OS) $(HOST_ARCH))
endif

.PHONY: all check analyze

all: check

check analyze:
	@$(MAKE) -f Makefile.$(PLATFORM) $@
