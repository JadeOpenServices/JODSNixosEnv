# Hardware and profiles

The installer detects laptop versus desktop before showing profiles. Desktop
systems use the standard profile; laptops can select laptop, ThinkPad, or
Framework profiles.

Graphics and Wi-Fi are detected with `lspci`. AMD, Intel, and NVIDIA use the
matching common graphics stack. Wi-Fi modules are not force-loaded, avoiding
driver mismatches; NetworkManager Wi-Fi power saving is disabled for reliable
throughput.

Laptop profiles include battery, suspend, lid, thermal, and dock behavior.
Framework fan and charge controls are enabled only after model selection.
Model data lives in `system/hardware/framework/profiles/`.
