## Meshtastic Serial Client Proxy

### About
This program lets you connect a [Meshtastic](https://meshtastic.org/) node 
to your system via a USB or serial cable and use your devices internet 
connection to connect the node to MQTT.

### IMPORTANT
This project is under heavy development. While it currently works, and 
works well, it is going to change. Right now the proxy doesn't utilize 
the settings on the node at all, but they are **required** to be configured
in order to work. You must enable the MQTT module, turn on the client proxy
setting, and configure the uplink and/or downlink setting for each channel. 

You also need to pass in the MQTT broker details and channels you want to
proxy as well. I will eliminate this in the future but if I wait any longer
to release this code and the [meshtastic-go](https://github.com/crypto-smoke/meshtastic-go)
package its likely to never see the light of day. 