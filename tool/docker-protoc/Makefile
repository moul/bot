IMAGE ?= 	moul/moul-bot-protoc
VERSION ?= 	1

build:
	docker build --pull -t $(IMAGE):$(VERSION) .

publish: build
	docker tag $(IMAGE):$(VERSION) $(IMAGE):latest
	docker push $(IMAGE):$(VERSION)
	docker push $(IMAGE):latest
