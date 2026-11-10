#!/bin/bash

# 1. Kill any existing broken or stuck background processes
killall -9 websockify x11vnc fluxbox Xvfb 2>/dev/null
sleep 0.5

# 2. Build a local workspace web-root directory to avoid the file index layout
mkdir -p .vnc-web
cp -r /usr/share/novnc/* .vnc-web/ 2>/dev/null || cp -r /usr/share/javascript/novnc/* .vnc-web/ 2>/dev/null
cp .vnc-web/vnc.html .vnc-web/index.html 2>/dev/null

# 3. Initialize the virtual display frame buffer
Xvfb :1 -screen 0 800x600x24 &
sleep 1

# 4. Bind the display environment variable globally
export DISPLAY=:1

# 5. Launch the window manager fully detached
fluxbox > /dev/null 2>&1 &
sleep 1

# 6. Spin up VNC server on 5900, but bind it strictly to localhost (blocks Codespaces from auto-forwarding it)
x11vnc -display :1 -nopw -forever -shared -listen 127.0.0.1 -bg > /dev/null 2>&1
sleep 1

# 7. Start the proxy web server targeting port 6080
websockify --web=.vnc-web 6080 127.0.0.1:5900 &
sleep 1

# 8. Force VS Code Codespaces to configure port 6080 correctly
gh codespace ports visibility 6080:public 2>/dev/null

echo "=========================================================="
echo "🚀 DISPLAY STACK RECONFIGURED SUCCESSFULLY!"
echo "=========================================================="
echo "👉 1. Look at your VS Code terminal panel and click the 'Ports' tab."
echo "👉 2. Click the link next to port 6080."
echo "👉 (Port 5900 is now hidden so it won't trick the browser anymore!)"
echo "=========================================================="
