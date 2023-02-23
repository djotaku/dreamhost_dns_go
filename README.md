# Dreamhost DNS

This is a Go program for updating the IP addresses associated with your Dreamhost domain and subdomain names if you have a Dynamic DNS situation.

Run this script on whatever computer is functioning as the server. (ie the one behind the ever-changing IP address)

You will need a Dreamhost API key.

Create a settings.json file that looks like this:

```json
{
  "api_key": "myapikey",
  "domains": ["sub.domain1.com", "sub.domain2.com", "sub2.domain1.com"]
}
```
The settings.json file should be put into the right xdg directory for your system. The output of the program will tell you where that is. On Linux/Unix this is $HOME/.config/dreamhostdns/settings.json

From this repo you can grab the latest binary to run. Starting with the next release, binaries should be generated for Linux, Windows, and Mac on x86-64 and ARM.

# Why Go?

Originally I wrote this program in [Python](https://github.com/djotaku/dreamhost_dns), but every time I upgraded to a new version of Python, my virtual environment broke. That was really annoying for a script that I always want to work correctly. Since Go is a compiled language it doesn't require virtual environments in order to run without polluting the system files.

