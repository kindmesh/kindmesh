run:
	go run main.go

image:
	docker build -t kindmesh -f build/Dockerfile.kindmesh .

load: build
	kind load docker-image kindmesh

image-envoy:
	docker build -t kindmesh-envoy -f build/Dockerfile.envoy .

load-envoy: build-envoy
	kind load docker-image kindmesh-envoy
