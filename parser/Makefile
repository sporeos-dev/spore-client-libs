PLATFORMS := macos-arm linux-intel linux-arm windows-intel windows-arm

.PHONY: all release debug clean test clean-test \
        $(addprefix release-,$(PLATFORMS)) \
        $(addprefix debug-,$(PLATFORMS)) \
        $(addprefix clean-,$(PLATFORMS))

all: release

release: $(addprefix release-,$(PLATFORMS))

debug: $(addprefix debug-,$(PLATFORMS))

clean: $(addprefix clean-,$(PLATFORMS)) clean-test

test:
	@$(MAKE) -f Makefile.test test

clean-test:
	@$(MAKE) -f Makefile.test clean-test

release-%:
	@$(MAKE) -f Makefile.$* release

debug-%:
	@$(MAKE) -f Makefile.$* debug

clean-%:
	@$(MAKE) -f Makefile.$* clean
