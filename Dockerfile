FROM ubuntu:latest


RUN apt-get update && apt-get install -y ca-certificates openssl

ARG cert_location=/usr/local/share/ca-certificates

# Get certificate from "github.com"
RUN openssl s_client -showcerts -connect github.com:443 </dev/null 2>/dev/null|openssl x509 -outform PEM > ${cert_location}/github.crt
# Get certificate from "proxy.golang.org"
RUN openssl s_client -showcerts -connect proxy.golang.org:443 </dev/null 2>/dev/null|openssl x509 -outform PEM >  ${cert_location}/proxy.golang.crt
# Update certificates
RUN update-ca-certificates

RUN apt-get install -y golang git gcc libgtk-3-dev libwebkit2gtk-4.0-dev flatpak flatpak-builder

ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH
RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" && chmod -R 777 "$GOPATH"
RUN mkdir -p /opencloud

COPY ./ /opencloud

WORKDIR /opencloud
# RUN go build
# Need to add flathub, install gnome sdk, etc.,
# then build the flatpak
# RUN flatpak install org.gnome.Sdk
RUN build/linux/package.bash

CMD ["./opencloudsave", "--help"]