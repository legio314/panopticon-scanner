#!/bin/bash
# Panopticon Scanner Installation Script

set -e

# Define colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Define installation paths
INSTALL_DIR="/opt/panopticon"
CONFIG_DIR="/etc/panopticon"
DATA_DIR="/var/lib/panopticon"
LOG_DIR="/var/log/panopticon"
SYSTEMD_DIR="/etc/systemd/system"

# Check if running as root
if [ "$EUID" -ne 0 ]; then
  echo -e "${RED}Please run as root or with sudo privileges${NC}"
  exit 1
fi

echo -e "${GREEN}===== Panopticon Scanner Installation Script =====${NC}"
echo -e "${YELLOW}This script will install Panopticon Scanner on your system.${NC}"
echo -e "${YELLOW}Installation directory: ${INSTALL_DIR}${NC}"
echo -e "${YELLOW}Configuration directory: ${CONFIG_DIR}${NC}"
echo -e "${YELLOW}Data directory: ${DATA_DIR}${NC}"
echo -e "${YELLOW}Log directory: ${LOG_DIR}${NC}"
echo ""

# Function to check for dependencies
check_dependencies() {
  echo -e "${GREEN}Checking dependencies...${NC}"
  
  # Check for Go
  if ! command -v go &> /dev/null; then
    echo -e "${RED}Go is not installed. Installing...${NC}"
    apt-get update
    apt-get install -y golang
  else
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    echo -e "${GREEN}Go version $GO_VERSION is installed${NC}"
  fi
  
  # Check for Node.js and npm
  if ! command -v node &> /dev/null || ! command -v npm &> /dev/null; then
    echo -e "${RED}Node.js and/or npm are not installed. Installing...${NC}"
    apt-get update
    apt-get install -y nodejs npm
  else
    NODE_VERSION=$(node -v)
    NPM_VERSION=$(npm -v)
    echo -e "${GREEN}Node.js version $NODE_VERSION and npm version $NPM_VERSION are installed${NC}"
  fi
  
  # Check for nmap
  if ! command -v nmap &> /dev/null; then
    echo -e "${RED}nmap is not installed. Installing...${NC}"
    apt-get update
    apt-get install -y nmap
  else
    NMAP_VERSION=$(nmap --version | head -n1 | awk '{print $3}')
    echo -e "${GREEN}nmap version $NMAP_VERSION is installed${NC}"
  fi
  
  # Check for SQLite
  if ! command -v sqlite3 &> /dev/null; then
    echo -e "${RED}SQLite is not installed. Installing...${NC}"
    apt-get update
    apt-get install -y sqlite3
  else
    SQLITE_VERSION=$(sqlite3 --version | awk '{print $1}')
    echo -e "${GREEN}SQLite version $SQLITE_VERSION is installed${NC}"
  fi
}

# Function to create directories
create_directories() {
  echo -e "${GREEN}Creating installation directories...${NC}"
  mkdir -p "$INSTALL_DIR"
  mkdir -p "$CONFIG_DIR"
  mkdir -p "$DATA_DIR/scans"
  mkdir -p "$DATA_DIR/backups"
  mkdir -p "$LOG_DIR"
  echo -e "${GREEN}Directories created successfully${NC}"
}

# Function to build backend
build_backend() {
  echo -e "${GREEN}Building backend...${NC}"
  
  # Navigate to backend source
  cd cmd/panopticond
  
  # Build the binary
  go build -o "../../panopticond"
  
  # Install backend binary
  cp ../../panopticond "$INSTALL_DIR/"
  chmod +x "$INSTALL_DIR/panopticond"
  
  cd ../..
  echo -e "${GREEN}Backend built successfully${NC}"
}

# Function to build frontend
build_frontend() {
  echo -e "${GREEN}Building frontend...${NC}"
  
  # Navigate to frontend source
  cd ui
  
  # Install dependencies and build
  npm install
  npm run build
  
  # Install frontend files
  mkdir -p "$INSTALL_DIR/ui"
  cp -r build/* "$INSTALL_DIR/ui/"
  
  cd ..
  echo -e "${GREEN}Frontend built successfully${NC}"
}

# Function to install configuration files
install_configs() {
  echo -e "${GREEN}Installing configuration files...${NC}"
  
  # Copy config file
  cp configs/config.yaml "$CONFIG_DIR/"
  
  # Create systemd service file
  cat > "${SYSTEMD_DIR}/panopticond.service" << EOF
[Unit]
Description=Panopticon Network Scanner Service
After=network.target

[Service]
Type=simple
User=root
ExecStart=${INSTALL_DIR}/panopticond --config=${CONFIG_DIR}/config.yaml
Restart=on-failure
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=panopticond

[Install]
WantedBy=multi-user.target
EOF
  
  # Create Desktop Entry for UI
  mkdir -p /usr/local/share/applications
  cat > /usr/local/share/applications/panopticon.desktop << EOF
[Desktop Entry]
Name=Panopticon Scanner
Comment=Network scanning and monitoring tool
Exec=${INSTALL_DIR}/panopticon-ui
Icon=${INSTALL_DIR}/ui/icon.png
Terminal=false
Type=Application
Categories=Utility;Security;
EOF
  
  echo -e "${GREEN}Configuration files installed successfully${NC}"
}

# Function to create launcher script for UI
create_ui_launcher() {
  echo -e "${GREEN}Creating UI launcher script...${NC}"
  
  cat > "${INSTALL_DIR}/panopticon-ui" << EOF
#!/bin/bash
# Launcher script for Panopticon UI

# Check if backend service is running
if ! systemctl is-active --quiet panopticond; then
  echo "Starting Panopticon backend service..."
  systemctl start panopticond
fi

# Open the UI in the default browser
xdg-open http://localhost:8080
EOF
  
  chmod +x "${INSTALL_DIR}/panopticon-ui"
  
  # Create symlink to path
  ln -sf "${INSTALL_DIR}/panopticon-ui" /usr/local/bin/panopticon-ui
  
  echo -e "${GREEN}UI launcher script created successfully${NC}"
}

# Function to set permissions
set_permissions() {
  echo -e "${GREEN}Setting file permissions...${NC}"
  
  # Set ownership
  chown -R root:root "$INSTALL_DIR"
  chown -R root:root "$CONFIG_DIR"
  
  # Set directory permissions
  chmod 755 "$INSTALL_DIR"
  chmod 755 "$CONFIG_DIR"
  chmod -R 755 "$DATA_DIR"
  chmod -R 755 "$LOG_DIR"
  
  # Make executables actually executable
  chmod +x "$INSTALL_DIR/panopticond"
  chmod +x "$INSTALL_DIR/panopticon-ui"
  
  echo -e "${GREEN}Permissions set successfully${NC}"
}

# Function to start service
start_service() {
  echo -e "${GREEN}Starting Panopticon service...${NC}"
  
  # Reload systemd configuration
  systemctl daemon-reload
  
  # Enable and start service
  systemctl enable panopticond
  systemctl start panopticond
  
  # Check service status
  sleep 2
  if systemctl is-active --quiet panopticond; then
    echo -e "${GREEN}Panopticon service started successfully${NC}"
  else
    echo -e "${RED}Failed to start Panopticon service. Check status with: systemctl status panopticond${NC}"
  fi
}

# Function to display completion message
completion_message() {
  echo -e "\n${GREEN}===== Panopticon Scanner installation complete! =====${NC}"
  echo -e "${GREEN}You can start the UI by running:${NC} panopticon-ui"
  echo -e "${GREEN}Or access it directly at:${NC} http://localhost:8080"
  echo -e "${GREEN}Configuration file:${NC} ${CONFIG_DIR}/config.yaml"
  echo -e "${GREEN}Log files:${NC} ${LOG_DIR}"
  echo -e "${GREEN}Data directory:${NC} ${DATA_DIR}"
  echo -e "\n${GREEN}Service management:${NC}"
  echo -e "  Start service:   ${YELLOW}systemctl start panopticond${NC}"
  echo -e "  Stop service:    ${YELLOW}systemctl stop panopticond${NC}"
  echo -e "  Restart service: ${YELLOW}systemctl restart panopticond${NC}"
  echo -e "  Service status:  ${YELLOW}systemctl status panopticond${NC}"
  echo -e "  View logs:       ${YELLOW}journalctl -u panopticond${NC}"
}

# Main installation process
main() {
  check_dependencies
  create_directories
  build_backend
  build_frontend
  install_configs
  create_ui_launcher
  set_permissions
  start_service
  completion_message
}

# Run the main function
main