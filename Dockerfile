FROM ubuntu:24.04

USER root
RUN mkdir -p /root/app
WORKDIR /root/app

# prerequisite packages
RUN apt-get update -qq && \
    apt-get upgrade -qq -y && \
    DEBIAN_FRONTEND=noninteractive apt-get install -qq -y \
            apt-transport-https \
            autoconf \
            automake \
            binutils \
            bison \
            bubblewrap \
            build-essential \
            ca-certificates \
            curl \
            file \
            git \
            gnupg2 \
            jq \
            locales \
            pkg-config \
            re2c \
            software-properties-common \
            tar \
            tree \
            time \
            tzdata \
            unzip \
            vim \
            wget \
            xorg-dev && \
    apt-get clean -qq -y && \
    apt-get autoclean -qq -y && \
    apt-get autoremove -qq -y

# locales
RUN locale-gen en_US.UTF-8
ENV LANG=en_US.UTF-8
ENV LANGUAGE=en_US:en
ENV LC_ALL=en_US.UTF-8

# libraries
RUN DEBIAN_FRONTEND=noninteractive apt-get install -qq -y \
            llvm \
            clang \
            libbz2-dev \
            libffi-dev \
            liblzma-dev \
            libncurses5-dev \
            libreadline-dev \
            libssl-dev \
            libyaml-dev \
            libsqlite3-dev \
            libxml2-dev \
            libxmlsec1-dev \
            libc6-dev \
            libz3-dev \
            libgd-dev \
            libpcre2-dev \
            libpcre3-dev \
            libonig-dev \
            libpq-dev \
            libedit-dev \
            libgdbm-dev \
            libcurl4-openssl-dev \
            libunistring-dev \
            libgc-dev \
            libpng-dev \
            libxslt-dev \
            libgmp3-dev \
            libtool \
            libncurses-dev \
            libssh-dev \
            libzip-dev \
            libevent-dev \
            libicu-dev \
            libglu1-mesa-dev \
            unixodbc-dev \
            zlib1g-dev \
            libsdl2-dev \
            libgl1-mesa-dev \
            libgmp-dev \
            libfontconfig1-dev && \
    apt-get clean -qq -y && \
    apt-get autoclean -qq -y && \
    apt-get autoremove -qq -y


# asdf languages
RUN git clone https://github.com/asdf-vm/asdf.git /root/.asdf
RUN chmod 755 /root/.asdf/asdf.sh
RUN echo "/root/.asdf/asdf.sh" >> /etc/bash.bashrc

# Add asdf and above languages to PATH
ENV PATH="${PATH}:/root/.asdf/shims:/root/.asdf/bin"

COPY .tool-versions /root/app/.

RUN asdf plugin-add golang
RUN asdf install golang

RUN asdf plugin-add rust
RUN asdf install rust

RUN asdf plugin add zig https://github.com/zigcc/asdf-zig
RUN asdf install zig

RUN asdf plugin-add swift https://github.com/drujensen/asdf-swift
RUN asdf install swift

RUN asdf plugin-add java
RUN asdf install java

RUN asdf plugin-add kotlin
RUN asdf install kotlin

RUN asdf plugin-add dotnet
RUN asdf install dotnet

RUN asdf plugin-add nodejs
RUN asdf install nodejs

RUN asdf plugin-add python
RUN asdf install python

RUN asdf plugin-add ruby
RUN asdf install ruby


#FROM golang:1.23

#WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o main ./cmd/http

EXPOSE 8080

CMD ["./main"]
