MORTIS (currently Jarvis) - Your everyday AI assistant 

An assitant written in Golang, with his simple architecture mortis can handle for you your calendar, files
your TV (mine as for now), your AC (mine too),

## Browser and system commands

Mortis supports browser and system actions through its modules:

- `browser.open`: open a URL or search query.
- `browser.current`: read the active tab URL in Safari or Google Chrome on macOS.
- `browser.explain`: fetch and explain the active page's HTML text using the AI client. It cannot inspect screenshots, canvas content, or pages that require a browser login.
- `system.info`: report operating system, architecture, hostname, and working directory.
- `system.open_app`: open an installed application using the host operating system.
- `system.execute`: execute an explicitly requested command with string arguments.