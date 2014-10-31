.PHONY:wuzei

GOPATH = $(PWD)/build:$(PWD)/Godeps/_workspace
export GOPATH
URL = github.com/thesues
REPO = radoshttpd
URLPATH = $(PWD)/build/src/$(URL)

wuzei:
	@[ -d $(URLPATH) ] || mkdir -p $(URLPATH)
	@ln -nsf $(PWD) $(URLPATH)/$(REPO)
	go install $(URL)/$(REPO)/wuzei

install:
	install -D build/bin/wuzei $$DESTDIR/usr/bin/wuzei

clean:
	rm -fr rpm-build
	rm -rf build
	rm -rf *.rpm

rpm:
	sh package/rpmbuild.sh
