<p align="center">
    <img src="image/README/1783175501783.png" alt="Docker Mailserver Mailman (Image)">
</p align="center">

# Docker Mailserver Mailman

Interface to manage docker mailserver email accounts.

Not through the setup script. Tried that, too slow. Let's do LDAP!

Much like my other projects, this was built because there either wasn't a solution, or I didn't like it.

This package is intended to do the following, some of it yet to be implemented:

* Manage Email accounts & Aliases through a local LDAP server.
* Allow to connect different distributed databases to the LDAP backend
* Managing Dovecot sieve scripts

**WARNING:** LDAP is not supposed to be exposed to the public internet. Use IP restrictions if you must.
