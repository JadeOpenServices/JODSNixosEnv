# Compressed swap in RAM. Evaluating this configuration takes about 3.5 GiB,
# so a 4 GiB machine without swap gets its rebuild OOM-killed (Galaxy Book
# 12-like VM, 2026-10-10). zram never writes pages to disk, so nothing in
# memory reaches an unencrypted swap. Size as Fedora since 34: RAM, at most
# 8 GiB (https://fedoraproject.org/wiki/Changes/Scale_ZRAM_to_full_memory_size).
{
  zramSwap = {
    enable = true;
    algorithm = "zstd";
    memoryPercent = 100;
    memoryMax = 8 * 1024 * 1024 * 1024;
  };
}
