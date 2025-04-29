# Building a Windows image template

Follow the how-to:
https://www.phillipsj.net/posts/building-a-windows-server-qcow2-image/

# Cloudbase-init post-configuration

The **cloudbase-init.conf** and **cloudbase-init-unattended.conf** are to be updated on template to only select specific [plugins](https://cloudbase-init.readthedocs.io/en/latest/plugins.html#user-data-main) and enforce usage of ISO-based CDROM device using the [NoCloud configuration drive](https://cloudbase-init.readthedocs.io/en/latest/services.html#nocloud-configuration-drive) service provider.

Directory offers examples of known-to-working config files.
