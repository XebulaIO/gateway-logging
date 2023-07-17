docker run -it -v /c/Users/cetnf/OneDrive/Masaüstü/xebula/gateway-logging:/app -w "/app" krakend/builder:2.4.1 go build -buildmode=plugin -o decoder.so .

 docker run --rm -v C:/Users/cetnf/OneDrive/Masaüstü/xebula/gateway-logging:/app -w "/app" devopsfaith/krakend check-plugin -g 1.18 /app/go.sum -f 