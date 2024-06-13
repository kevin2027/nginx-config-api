#!/bin/bash

# Specify the repository owner and name
REPO_OWNER="kevin2027"
REPO_NAME="nginx-config-api"

# Specify the installation directory
INSTALL_DIR="/usr/local/bin"

# Function to detect the operating system
detect_os() {
  if [[ "$(uname)" == "Linux" ]]; then
    echo "linux"
  elif [[ "$(uname)" == "Darwin" ]]; then
    echo "darwin"
  else
    echo "Unsupported OS"
    exit 1
  fi
}

# Function to create systemd service
create_systemd_service() {
  local binary_name="$1"
  local service_name="$2"

  # Create systemd service file
  sudo tee "/etc/systemd/system/$service_name.service" > /dev/null <<EOF
[Unit]
Description=$service_name Service
After=network.target

[Service]
Type=simple
ExecStart=$INSTALL_DIR/$binary_name

[Install]
WantedBy=multi-user.target
EOF

  # Reload systemd daemon to load the new service
  sudo systemctl daemon-reload

  # Enable and start the service
  sudo systemctl enable $service_name
  sudo systemctl start $service_name

  echo "Systemd service $service_name has been enabled and started."
}

# Function to create launchd service (macOS)
create_launchd_service() {
  local binary_name="$1"
  local service_name="$2"

  # Create launchd service file
  tee "$HOME/Library/LaunchAgents/$service_name.plist" > /dev/null <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>$service_name</string>
  <key>ProgramArguments</key>
  <array>
    <string>$INSTALL_DIR/$binary_name</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
</dict>
</plist>
EOF

  # Load launchd service
  launchctl load "$HOME/Library/LaunchAgents/$service_name.plist"

  echo "Launchd service $service_name has been loaded."
}

# Determine the latest release version
VERSION=$(curl -s "https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

# Determine the binary file name based on OS
OS=$(detect_os)
case "$OS" in
  "linux")
    BINARY="$REPO_NAME"_linux_amd64
    ;;
  "darwin")
    BINARY="$REPO_NAME"_darwin_amd64
    ;;
  *)
    echo "Unsupported OS"
    exit 1
    ;;
esac

# Construct the download URL
DOWNLOAD_URL="https://github.com/$REPO_OWNER/$REPO_NAME/releases/download/$VERSION/$BINARY"

# Download and install the binary
echo "Downloading $BINARY version $VERSION..."
sudo curl -L -o "$INSTALL_DIR/$BINARY" "$DOWNLOAD_URL"

# Set executable permissions
sudo chmod +x "$INSTALL_DIR/$BINARY"

echo "$BINARY version $VERSION has been installed to $INSTALL_DIR"

# Create systemd service if on Linux
if [[ "$OS" == "linux" ]]; then
  create_systemd_service "$BINARY" "$REPO_NAME"
fi

# Create launchd service if on macOS
if [[ "$OS" == "darwin" ]]; then
  create_launchd_service "$BINARY" "$REPO_NAME"
fi
