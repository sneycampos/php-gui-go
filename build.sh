#!/bin/bash

# Compile the Go program
go build -o PHP_MariaDB_Switcher

# Create the app bundle structure
mkdir -p PHP_MariaDB_Switcher.app/Contents/MacOS
mkdir -p PHP_MariaDB_Switcher.app/Contents/Resources

# Move the executable to the app bundle
mv PHP_MariaDB_Switcher PHP_MariaDB_Switcher.app/Contents/MacOS/

# Create the Info.plist file
cat <<EOF > PHP_MariaDB_Switcher.app/Contents/Info.plist
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>PHP_MariaDB_Switcher</string>
    <key>CFBundleIdentifier</key>
    <string>com.example.phpmariadbswitcher</string>
    <key>CFBundleName</key>
    <string>PHP & MariaDB Version Switcher</string>
    <key>CFBundleVersion</key>
    <string>1.0</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
</dict>
</plist>
EOF

echo "App bundle created: PHP_MariaDB_Switcher.app"
