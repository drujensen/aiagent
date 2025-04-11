FROM ubuntu:24.04

# Update and install packages as root
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
            gnupg \
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
            xorg-dev \
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

# Set up locales
RUN locale-gen en_US.UTF-8
ENV LANG=en_US.UTF-8
ENV LANGUAGE=en_US:en
ENV LC_ALL=en_US.UTF-8

# Install asdf
RUN ARCH=$(uname -m) && \
    echo "Detected architecture: ${ARCH}" && \
    ASDF_ARCHIVE_URL="https://github.com/asdf-vm/asdf/releases/download/v0.16.5" && \
    if [ "${ARCH}" = "aarch64" ]; then \
        ASDF_ARCHIVE_URL="${ASDF_ARCHIVE_URL}/asdf-v0.16.5-linux-arm64.tar.gz"; \
    elif [ "${ARCH}" = "x86_64" ]; then \
        ASDF_ARCHIVE_URL="${ASDF_ARCHIVE_URL}/asdf-v0.16.5-linux-amd64.tar.gz"; \
    else \
        echo "Unsupported architecture: ${ARCH}"; \
        exit 1; \
    fi && \
    echo "Downloading asdf from $ASDF_ARCHIVE_URL" && \
    curl -fSLk -o /tmp/asdf.tar.gz "${ASDF_ARCHIVE_URL}" && \
    mkdir -p /opt/asdf && \
    tar -xzf /tmp/asdf.tar.gz -C /opt/asdf && \
    chmod +x /opt/asdf/asdf && \
    ln -s /opt/asdf/asdf /usr/bin/asdf

# Switch to ubuntu
USER ubuntu
WORKDIR /home/ubuntu

# Set up asdf for ubuntu
ENV PATH="/home/ubuntu/.asdf/shims:/home/ubuntu/.asdf/bin:${PATH}"
RUN echo '. /opt/asdf/asdf' >> /home/ubuntu/.bashrc

# Copy .tool-versions file
COPY --chown=ubuntu:ubuntu .tool-versions ./

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

# Install MongoDB shell
RUN curl -fsSL https://pgp.mongodb.com/server-7.0.asc | \
    gpg --dearmor -o /usr/share/keyrings/mongodb-server-7.0.gpg
RUN echo "deb [ arch=amd64,arm64 signed-by=/usr/share/keyrings/mongodb-server-7.0.gpg ] https://repo.mongodb.org/apt/ubuntu noble/mongodb-org/7.0 multiverse" | \
    tee /etc/apt/sources.list.d/mongodb-org-7.0.list
RUN apt-get update -qq && \
    apt-get install -qq -y mongodb-mongosh && \
    && rm -rf /var/lib/apt/lists/*

# Copy Go project files
COPY --chown=ubuntu:ubuntu go.mod go.sum ./
RUN go mod download

# Copy the rest of the application code
COPY --chown=ubuntu:ubuntu . .

# Build the Go application
RUN go build -o main cmd/server/main.go

EXPOSE 8080

CMD ["./main"]
