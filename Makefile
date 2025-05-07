build-ubuntu:
	@echo "Building Ubuntu Docker image..."
	docker buildx build --file ./dockerFile --tag my-image-name --load .
	docker run -it --rm my-image-name:latest
