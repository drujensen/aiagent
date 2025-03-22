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
RUN echo "Downloading asdf version v0.16.5 for linux-arm64" && \
    curl -fSL -o /tmp/asdf.tar.gz "https://github.com/asdf-vm/asdf/releases/download/v0.16.5/asdf-v0.16.5-linux-arm64.tar.gz" && \
    mkdir -p /opt/asdf && \
    tar -xzf /tmp/asdf.tar.gz -C /opt/asdf && \
    chmod +x /opt/asdf/asdf && \
    ln -s /opt/asdf/asdf /usr/bin/asdf

# Switch to ubuntu
USER ubuntu
WORKDIR /home/ubuntu

# Set up asdf for ubuntu
ENV PATH="/home/ubuntu/.asdf/shims:/home/ubuntu/.asdf/bin:${PATH}"
RUN echo '. /opt/asdf/asdf.sh' >> /home/ubuntu/.bashrc && \
    echo '. /opt/asdf/completions/asdf.bash' >> /home/ubuntu/.bashrc

# Copy .tool-versions file
COPY --chown=ubuntu:ubuntu .tool-versions ./

# Install asdf plugins and versions
RUN /usr/bin/asdf plugin add golang && \
    /usr/bin/asdf install golang && \
    /usr/bin/asdf plugin add rust && \
    /usr/bin/asdf install rust && \
    /usr/bin/asdf plugin add zig https://github.com/zigcc/asdf-zig && \
    /usr/bin/asdf install zig && \
    /usr/bin/asdf plugin add swift https://github.com/drujensen/asdf-swift && \
    /usr/bin/asdf install swift && \
    /usr/bin/asdf plugin add java && \
    /usr/bin/asdf install java && \
    /usr/bin/asdf plugin add kotlin && \
    /usr/bin/asdf install kotlin && \
    /usr/bin/asdf plugin add dotnet && \
    /usr/bin/asdf install dotnet && \
    /usr/bin/asdf plugin add nodejs && \
    /usr/bin/asdf install nodejs && \
    /usr/bin/asdf plugin add python && \
    /usr/bin/asdf install python && \
    /usr/bin/asdf plugin add ruby && \
    /usr/bin/asdf install ruby

# Copy Go project files
COPY --chown=ubuntu:ubuntu go.mod go.sum ./
RUN go mod download

# Copy .env file
COPY --chown=ubuntu:ubuntu .env ./

# Copy the rest of the application code
COPY --chown=ubuntu:ubuntu . .

# Build the Go application
RUN go build -o main ./cmd/http

EXPOSE 8080

CMD ["./main"]
