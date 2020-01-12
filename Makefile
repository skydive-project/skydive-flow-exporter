BUILD := \
	allinone \
	awsflowlogs \
	core \
	dummy \
	secadvisor \
	vpclogs \

STATIC := \
	allinone \

.PHONY: all
all:
	for i in $(BUILD); do $(MAKE) -C $$i || exit; done

.PHONY: build
build:
	for i in $(BUILD); do $(MAKE) -C $$i build || exit; done

.PHONY: clean
clean:
	for i in $(BUILD); do $(MAKE) -C $$i clean || exit; done

.PHONY: static
static:
	for i in $(STATIC); do $(MAKE) -C $$i static || exit; done

include Makefile.dev
