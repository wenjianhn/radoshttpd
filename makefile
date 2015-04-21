.PHONY:wuzei

GOPATH = $(PWD)/build:$(PWD)/Godeps/_workspace
export GOPATH
URL = github.com/thesues
REPO = radoshttpd
URLPATH = $(PWD)/build/src/$(URL)
LOGPATH=$$DESTDIR/var/log/wuzei
PIDPATH=$$DESTDIR/var/run/wuzei

wuzei:
	@[ -d $(URLPATH) ] || mkdir -p $(URLPATH)
	@ln -nsf $(PWD) $(URLPATH)/$(REPO)
	go install $(URL)/$(REPO)/wuzei

install:
	@[ -d $(LOGPATH) ]|| mkdir -p $(LOGPATH)
	@[ -d $(PIDPATH) ]|| mkdir -p $(PIDPATH)
	install -D build/bin/wuzei $$DESTDIR/usr/bin/wuzei
	install -d -m 755 $$DESTDIR/etc/wuzei
	install -p -D -m 640 package/wuzei.json $$DESTDIR/etc/wuzei/
	install -m 0755 scripts/wuzei.sh -D $$DESTDIR/etc/init.d/wuzei
	install -D -m 0644 scripts/wuzei.logrotate $$DESTDIR/etc/logrotate.d/wuzei

clean:
	rm -fr rpm-build
	rm -rf build
	rm -rf *.rpm

rpm:
	sh package/rpmbuild.sh
