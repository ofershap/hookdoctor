# Security Policy

hookdoctor's default mode is fully static: it reads settings files and checks
that hook commands exist. With `-exec` it runs the hook commands it finds,
which are arbitrary code - only use `-exec` on repos you trust.

If you find a case where hookdoctor itself executes something without `-exec`,
or mishandles a crafted settings file, please report it privately via
GitHub's "Report a vulnerability" flow on the Security tab.
