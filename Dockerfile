FROM golang:1.27

WORKDIR /workspace

RUN apt-get update && apt-get install -y --no-install-recommends git curl ca-certificates build-essential vim && rm -rf /var/lib/apt/lists/*

CMD ["bash"]
