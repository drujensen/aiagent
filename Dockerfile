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

# Install asdf
RUN echo "Downloading asdf version v0.16.5 for linux-arm64" && \
    curl -fSL -o /tmp/asdf.tar.gz "https://github.com/asdf-vm/asdf/releases/download/v0.16.5/asdf-v0.16.5-linux-arm64.tar.gz" && \
    mkdir -p /opt/asdf && \
    tar -xzf /tmp/asdf.tar.gz -C /opt/asdf && \
    chmod +x /opt/asdf/asdf && \
    ln -s /opt/asdf/asdf /usr/bin/asdf

# Verify asdf installation
RUN echo "Verifying asdf installation:" && /usr/bin/asdf --version

# Add asdf to PATH
ENV PATH="/usr/bin:/root/.asdf/shims:${PATH}"

# Copy .tool-versions file
COPY .tool-versions /root/app/.

# Install asdf plugins and versions
RUN /usr/bin/asdf plugin add golang
RUN /usr/bin/asdf install golang

RUN /usr/bin/asdf plugin add rust
RUN /usr/bin/asdf install rust

RUN /usr/bin/asdf plugin add zig https://github.com/zigcc/asdf-zig
RUN /usr/bin/asdf install zig

RUN /usr/bin/asdf plugin add swift https://github.com/drujensen/asdf-swift
RUN /usr/bin/asdf install swift

RUN /usr/bin/asdf plugin add java
RUN /usr/bin/asdf install java

RUN /usr/bin/asdf plugin add kotlin
RUN /usr/bin/asdf install kotlin

RUN /usr/bin/asdf plugin add dotnet
RUN /usr/bin/asdf install dotnet

RUN /usr/bin/asdf plugin add nodejs
RUN /usr/bin/asdf install nodejs

RUN /usr/bin/asdf plugin add python
RUN /usr/bin/asdf install python

RUN /usr/bin/asdf plugin add ruby
RUN /usr/bin/asdf install ruby

# Install Go and build the application
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o main ./cmd/http

EXPOSE 8080

CMD ["./main"]
