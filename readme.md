docker run -it -v "pwd":/app -w "/app" krakend/builder:2.3.2 go build -buildmode=plugin -o logger.so .

docker run --rm -v "pwd":/app -w "/app" devopsfaith/krakend:2.3.2 check-plugin  /app/go.sum -f