# Arbot - Setup and Install

## Developer Machine Setup

Set up your development environment for building, running, and testing the Taxi Simulator server.

| Software       | Version  | Required     |
| -------------- | -------- | ------------ |
| Go             | 1.18.0+  | Yes          |
| Docker         | 17.12.0+ | No, optional |
| Docker Compose | 1.21.0+  | No, optional |

### For Ubuntu 16.04 / 18.04

1. Install Golang on your machine

   ```bash
   export ARCH=amd64 # or arm64

   sudo apt-get install -y build-essential
   sudo rm -rf /usr/local/go

   # Download
   wget https://dl.google.com/go/go1.18.3.linux-$ARCH.tar.gz

   # Extract
   sudo tar -C /usr/local -xzf go1.18.3.linux-$ARCH.tar.gz

   # Update path
   # You might want to add this to your
   # ~/.bashrc or ~/.zshrc
   export PATH=$PATH:/usr/local/go/bin
   ```

2. Configure Docker CE

   ```bash
   curl -fsSL https://get.docker.com -o get-docker.sh
   sudo sh get-docker.sh
   sudo usermod -aG docker $(whoami)
   docker login
   ```

   If you prefer to perform these steps manually:

   - https://docs.docker.com/install/linux/docker-ce/ubuntu/
   - https://docs.docker.com/install/linux/linux-postinstall/

3. Install Docker Compose

   ```bash
   sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose

   sudo chmod +x /usr/local/bin/docker-compose
   ```

4. Clone github.com/iqDF/arbot

```bash
git clone https://github.com/iqDF/arbot.git
```

### For MacOS X

1. Install homebrew if you haven't already: https://brew.sh/.
2. Install Golang on your machine

   ```bash
   brew install go
   ```

3. Configure Docker CE. Install and configure Docker CE: https://docs.docker.com/docker-for-mac/.

4. Clone github.com/iqDF/arbot

```bash
git clone https://github.com/iqDF/arbot.git
cd arbot # go to root project directory
```

## Run Tests

### Unit testing

You can invoke `Makefile` command to run all unit tests for you:

```bash
make test
```

If you prefer, you can also run it using golang testing tool:

```bash
go test ./... -short
```

## Run Server

1. Start the server as follows:
   ```bash
   cd arbot
   make run-server
   ```
   Optionally, you can also invoke golang compiler tool to run.
   ```bash
   go run main.go
   ```
2. Once finished, you can stop the server.
   ```bash
   make stop-server
   ```
