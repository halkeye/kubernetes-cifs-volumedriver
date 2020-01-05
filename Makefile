TAGNAME = halkeye/kubernetes-cifs-volumedriver-installer
VERSION = 0.6

build: Dockerfile
	docker build -t $(TAGNAME):$(VERSION) .

push: build
	docker push $(TAGNAME):$(VERSION)

run: build
	mkdir -p /tmp/halkeye~cifs
	docker run -v /tmp/halkeye~cifs:/flexmnt -it --rm $(TAGNAME):$(VERSION)
