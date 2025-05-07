build-ubuntu:
	@echo "Building Ubuntu Docker image..."
	docker buildx build --file ./dockerFile --tag hp-terminal --load .
	docker run -it --rm hp-terminal:latest
