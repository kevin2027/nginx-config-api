#!/bin/bash

# Specify the repository owner and name
REPO_OWNER="kevin2027"
REPO_NAME="nginx-config-api"

# Specify the installation directory
INSTALL_DIR="/usr/local/bin"

# Determine the latest release version
VERSION=$(curl -s "https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

# Construct the download URL
DOWNLOAD_URL="https://github.com/$REPO_OWNER/$REPO_NAME/releases/download/$VERSION/$REPO_NAME"

# Download and install the binary
echo "Downloading $REPO_NAME version $VERSION..."
sudo curl -L -o "$INSTALL_DIR/$REPO_NAME" "$DOWNLOAD_URL"

# Set executable permissions
sudo chmod +x "$INSTALL_DIR/$REPO_NAME"

echo "$REPO_NAME version $VERSION has been installed to $INSTALL_DIR"

# Create systemd service file
sudo tee "/etc/systemd/system/$REPO_NAME.service" > /dev/null <<EOF
[Unit]
Description=Nginx Config API Service
After=network.target

[Service]
Type=simple
ExecStart=$INSTALL_DIR/$REPO_NAME

[Install]
WantedBy=multi-user.target
EOF

# Reload systemd daemon to load the new service
sudo systemctl daemon-reload

# Enable and start the service
sudo systemctl enable $REPO_NAME
sudo systemctl start $REPO_NAME

echo "Systemd service $REPO_NAME has been enabled and started."
