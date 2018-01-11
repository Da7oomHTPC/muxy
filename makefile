default: build

execName=muxyProxy

clean:
	rm "${execName}"

build:
	GOOS=darwin GOARCH=amd64 CGO_ENALED=0 go build -ldflags="-w -s" -o "${execName}"
	chmod +x "${execName}"
