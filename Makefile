run:
	go run cmd/mesh-agent/main.go
dns:
	go run cmd/local-dns/main.go --conf config/Corefile
image:
	docker build -t mesh-agent -f build/Dockerfile.mesh-agent .
	docker build -t local-dns -f build/Dockerfile.local-dns .
	docker build -t envoy -f build/Dockerfile.envoy .

load: image
	kind load docker-image mesh-agent
	kind load docker-image local-dns
	kind load docker-image envoy
