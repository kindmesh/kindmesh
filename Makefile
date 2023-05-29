run:
	go run cmd/mesh-agent/main.go
	
dns:
	go run cmd/local-dns/main.go --conf config/Corefile

image:
	docker build -t kindmesh/mesh-agent -f build/Dockerfile.mesh-agent .
	docker build -t kindmesh/local-dns -f build/Dockerfile.local-dns .
	docker build -t kindmesh/envoy -f build/Dockerfile.envoy .

load:
	kind load docker-image kindmesh/mesh-agent
	kind load docker-image kindmesh/local-dns
	kind load docker-image kindmesh/envoy
