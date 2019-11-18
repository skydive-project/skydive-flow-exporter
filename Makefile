BUILD := \
	allinone \
	awsflowlogs \
	core \
	dummy \
	secadvisor \
	vpclogs \

STATIC := \
	allinone \

TEST := \
	secadvisor \

.PHONY: all
all:
	for i in $(BUILD); do $(MAKE) -C $$i || exit; done

.PHONY: clean
clean:
	for i in $(BUILD); do $(MAKE) -C $$i clean || exit; done

.PHONY: fmt
fmt:
	for i in $(BUILD); do $(MAKE) -C $$i fmt; done

.PHONY: test
test:
	for i in $(TEST); do $(MAKE) -C $$i test || exit; done

.PHONY: static
static:
	for i in $(STATIC); do $(MAKE) -C $$i static || exit; done
