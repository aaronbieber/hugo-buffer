default: build

build:
    go build -o hugo-buffer .

install: build
    mv hugo-buffer ~/.local/bin/hugo-buffer

clean:
    rm -f hugo-buffer
