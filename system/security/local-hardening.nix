{
  # Laptops join untrusted networks: never let ICMP redirects or source routes
  # change the routing table, and never send redirects ourselves.
  boot.kernel.sysctl = {
    "net.ipv4.conf.all.accept_redirects" = 0;
    "net.ipv4.conf.default.accept_redirects" = 0;
    "net.ipv4.conf.all.secure_redirects" = 0;
    "net.ipv4.conf.default.secure_redirects" = 0;
    "net.ipv4.conf.all.send_redirects" = 0;
    "net.ipv4.conf.default.send_redirects" = 0;
    "net.ipv4.conf.all.accept_source_route" = 0;
    "net.ipv6.conf.all.accept_redirects" = 0;
    "net.ipv6.conf.default.accept_redirects" = 0;
    "net.ipv6.conf.all.accept_source_route" = 0;
  };

  # Stored account hashes: root-only directory, so the set of accounts with
  # stored hashes is not listable. Also tightens directories created 0755 by
  # earlier installer versions.
  systemd.tmpfiles.rules = [
    "d /var/lib/gjallarOS/passwords 0700 root root -"
  ];
}
