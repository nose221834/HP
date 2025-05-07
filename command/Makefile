build-help:
	go build -o bin/help help/main.go


build-terminal:
	@echo "Building Ubuntu Docker image..."
	docker buildx build --file ./dockerFile --tag hp-terminal --load .
	docker run -it --rm hp-terminal:latest