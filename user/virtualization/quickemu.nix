{pkgs, ...}:
{
    home.packages = with pkgs; [
        quickemu
        waypipe
        swtpm
    ];

    # home.file.".config/television/config.toml".source = ./config.toml;
}
