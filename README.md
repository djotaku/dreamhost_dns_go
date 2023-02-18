# Dreamhost DNS

This is a Go program for updating all your Dreamhost domain and subdomain names if you have a Dynamic DNS situation.

Run this script on whatever computer is hosting the DNS.

You will need a Dreamhost API key.

Create a settings.json file that looks like this:

```json
{
  "api_key": "myapikey",
  "domains": ["sub.domain1.com", "sub.domain2.com", "sub2.domain1.com"]
}
```
It should be put into the right xdg directory for your system. The output of the program will tell you where that is. On Linux/Unix this is $HOME/.config/dreamhostdns/settings.json


# Why Go?

Originally I wrote this program in [Python](https://github.com/djotaku/dreamhost_dns), but every time I upgrade the computer to a new version of Python, my virtual environm environment broke. That was really annoying for a script that I always want to work correctly.

# TODO

Figure out how to have Github actions generate a new build when I push updates.
